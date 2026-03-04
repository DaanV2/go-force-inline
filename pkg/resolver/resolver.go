package resolver

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/daanv2/go-force-inline/pkg/directive"
	"golang.org/x/tools/go/packages"
)

// ResolvedEdge is a fully resolved call edge ready for profile generation.
type ResolvedEdge struct {
	CallerName      string
	CalleeName      string
	CallSiteOffset  int
	CallerStartLine int
	CalleeStartLine int
	CallLine        int
	Weight          int64
	Warnings        []string
}

// Resolve takes package patterns, loads them with full type info, finds
// directives, and resolves caller/callee linker symbols.
func Resolve(patterns []string, verbose bool) ([]ResolvedEdge, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps,
		Fset: token.NewFileSet(),
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	// Check for package load errors
	var errs []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, e.Error())
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("package errors:\n%s", strings.Join(errs, "\n"))
	}

	var edges []ResolvedEdge

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			directives := directive.FindDirectives(cfg.Fset, file)
			for _, d := range directives {
				edge, ok := resolveDirective(cfg.Fset, pkg, file, d, verbose)
				if ok {
					edges = append(edges, edge)
				}
			}
		}
	}

	return edges, nil
}

func resolveDirective(
	fset *token.FileSet,
	pkg *packages.Package,
	file *ast.File,
	d directive.DirectiveWithNode,
	verbose bool,
) (ResolvedEdge, bool) {
	if d.Node == nil {
		log.Warn("directive not followed by a call expression",
			"file", d.Directive.Pos.Filename,
			"line", d.Directive.Pos.Line)
		return ResolvedEdge{}, false
	}

	call, ok := d.Node.(*ast.CallExpr)
	if !ok {
		log.Warn("directive not followed by a call expression",
			"file", d.Directive.Pos.Filename,
			"line", d.Directive.Pos.Line)
		return ResolvedEdge{}, false
	}

	// Find the enclosing function
	callerFunc := findEnclosingFunc(fset, file, d.Directive.Pos.Line)
	if callerFunc == nil {
		log.Warn("directive not inside a function",
			"file", d.Directive.Pos.Filename,
			"line", d.Directive.Pos.Line)
		return ResolvedEdge{}, false
	}

	edge := ResolvedEdge{
		Weight: d.Directive.Weight,
	}

	// Resolve caller
	callerName, callerStartLine, callerWarnings := resolveFunc(fset, pkg, callerFunc)
	edge.CallerName = callerName
	edge.CallerStartLine = callerStartLine
	edge.Warnings = append(edge.Warnings, callerWarnings...)

	// Compute call line and offset
	callPos := fset.Position(call.Pos())
	edge.CallLine = callPos.Line
	edge.CallSiteOffset = callPos.Line - callerStartLine

	// Resolve callee
	calleeName, calleeStartLine, calleeWarnings, calleeOk := resolveCallee(fset, pkg, call)
	if !calleeOk {
		return ResolvedEdge{}, false
	}
	edge.CalleeName = calleeName
	edge.CalleeStartLine = calleeStartLine
	edge.Warnings = append(edge.Warnings, calleeWarnings...)

	// Check for chained calls
	if hasChainedCalls(call) {
		edge.Warnings = append(edge.Warnings,
			fmt.Sprintf("chained calls on line %d - ambiguous which call is targeted", callPos.Line))
	}

	if verbose {
		log.Info("resolved edge",
			"caller", edge.CallerName,
			"callee", edge.CalleeName,
			"offset", edge.CallSiteOffset,
			"weight", edge.Weight)
	}

	for _, w := range edge.Warnings {
		log.Warn(w,
			"file", d.Directive.Pos.Filename,
			"line", d.Directive.Pos.Line)
	}

	return edge, true
}

// resolveFunc resolves a function declaration to its linker symbol name.
func resolveFunc(fset *token.FileSet, pkg *packages.Package, fn *ast.FuncDecl) (string, int, []string) {
	startLine := fset.Position(fn.Pos()).Line
	var warnings []string

	obj := pkg.TypesInfo.Defs[fn.Name]
	if obj == nil {
		// Fallback: construct name manually
		name := pkg.PkgPath + "." + fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recvType := exprString(fn.Recv.List[0].Type)
			name = pkg.PkgPath + "." + recvType + "." + fn.Name.Name
		}
		return name, startLine, warnings
	}

	funcObj, ok := obj.(*types.Func)
	if !ok {
		return pkg.PkgPath + "." + fn.Name.Name, startLine, warnings
	}

	return linkerSymbol(funcObj), startLine, warnings
}

// resolveCallee resolves the called function/method to its linker symbol.
func resolveCallee(fset *token.FileSet, pkg *packages.Package, call *ast.CallExpr) (string, int, []string, bool) {
	var warnings []string

	// Get the callee identifier
	var ident *ast.Ident
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		ident = fn
	case *ast.SelectorExpr:
		ident = fn.Sel
	case *ast.IndexExpr:
		// Generic instantiation: f[T](...)
		switch inner := fn.X.(type) {
		case *ast.Ident:
			ident = inner
		case *ast.SelectorExpr:
			ident = inner.Sel
		}
	case *ast.IndexListExpr:
		// Multi-type-param generic: f[T1, T2](...)
		switch inner := fn.X.(type) {
		case *ast.Ident:
			ident = inner
		case *ast.SelectorExpr:
			ident = inner.Sel
		}
	default:
		log.Warn("cannot resolve callee - unsupported call expression type",
			"type", fmt.Sprintf("%T", call.Fun))
		return "", 0, warnings, false
	}

	if ident == nil {
		log.Warn("cannot resolve callee identifier")
		return "", 0, warnings, false
	}

	// Check for interface method call
	obj := pkg.TypesInfo.Uses[ident]
	if obj == nil {
		// Try ObjectOf as fallback
		obj = pkg.TypesInfo.ObjectOf(ident)
	}
	if obj == nil {
		log.Warn("cannot resolve callee - no type info",
			"name", ident.Name)
		return "", 0, warnings, false
	}

	funcObj, ok := obj.(*types.Func)
	if !ok {
		// Could be a builtin or type conversion
		log.Warn("callee is not a function",
			"name", ident.Name,
			"type", fmt.Sprintf("%T", obj))
		return "", 0, warnings, false
	}

	// Check if it's an interface method
	sig := funcObj.Type().(*types.Signature)
	if sig.Recv() != nil {
		recvType := sig.Recv().Type()
		if isInterface(recvType) {
			log.Warn("interface method call cannot be inlined",
				"method", funcObj.Name())
			return "", 0, warnings, false
		}
	}

	// Check for //go:noinline on the callee
	calleeDecl := findFuncDeclForObj(pkg, funcObj)
	if calleeDecl != nil {
		if hasNoInlineDirective(calleeDecl) {
			warnings = append(warnings,
				fmt.Sprintf("callee %s has //go:noinline - PGO won't override it", funcObj.Name()))
		}
	}

	name := linkerSymbol(funcObj)

	// Determine callee start line
	var calleeStartLine int
	if calleeDecl != nil {
		calleeStartLine = fset.Position(calleeDecl.Pos()).Line
	} else {
		// External function - use 0, will be set in profile as 1
		calleeStartLine = 1
	}

	// Check if caller is a closure
	if isClosure(funcObj) {
		warnings = append(warnings,
			fmt.Sprintf("callee %s appears to be a closure - name may be fragile", name))
	}

	return name, calleeStartLine, warnings, true
}

// linkerSymbol computes the linker symbol name matching ir.LinkFuncName.
func linkerSymbol(fn *types.Func) string {
	sig := fn.Type().(*types.Signature)

	if sig.Recv() == nil {
		// Package-level function
		return fn.Pkg().Path() + "." + fn.Name()
	}

	// Method
	recv := sig.Recv().Type()
	return recvLinkerName(fn.Pkg().Path(), recv, fn.Name())
}

// recvLinkerName builds the linker name for a method with the given receiver type.
func recvLinkerName(pkgPath string, recv types.Type, methodName string) string {
	switch t := recv.(type) {
	case *types.Pointer:
		named, ok := t.Elem().(*types.Named)
		if ok {
			return pkgPath + ".(*" + named.Obj().Name() + ")." + methodName
		}
	case *types.Named:
		return pkgPath + "." + t.Obj().Name() + "." + methodName
	}
	return pkgPath + "." + recv.String() + "." + methodName
}

// findEnclosingFunc finds the FuncDecl that contains the given line.
func findEnclosingFunc(fset *token.FileSet, file *ast.File, line int) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start := fset.Position(fn.Pos()).Line
		end := fset.Position(fn.Body.Rbrace).Line
		if line >= start && line <= end {
			return fn
		}
	}
	return nil
}

// findFuncDeclForObj finds the ast.FuncDecl corresponding to a types.Func
// within the same package.
func findFuncDeclForObj(pkg *packages.Package, fn *types.Func) *ast.FuncDecl {
	if fn.Pkg() == nil || fn.Pkg().Path() != pkg.PkgPath {
		return nil
	}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if obj := pkg.TypesInfo.Defs[fd.Name]; obj != nil && obj == fn {
				return fd
			}
		}
	}
	return nil
}

// hasNoInlineDirective checks if a function has //go:noinline.
func hasNoInlineDirective(fn *ast.FuncDecl) bool {
	if fn.Doc == nil {
		return false
	}
	for _, c := range fn.Doc.List {
		if strings.Contains(c.Text, "//go:noinline") {
			return true
		}
	}
	return false
}

// isInterface checks if a type is an interface type.
func isInterface(t types.Type) bool {
	t = t.Underlying()
	_, ok := t.(*types.Interface)
	return ok
}

// isClosure checks if a function object looks like a closure (contains "func" suffix pattern).
func isClosure(fn *types.Func) bool {
	name := fn.Name()
	// Closures in Go are named like "funcN" where N is a number
	if strings.HasPrefix(name, "func") && len(name) > 4 {
		for _, c := range name[4:] {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}

// hasChainedCalls checks if a call expression has call expressions as arguments.
func hasChainedCalls(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if _, ok := arg.(*ast.CallExpr); ok {
			return true
		}
	}
	return false
}

// exprString returns a simple string representation of a receiver type expr.
func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "(*" + exprString(e.X) + ")"
	case *ast.IndexExpr:
		return exprString(e.X) + "[" + exprString(e.Index) + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

package directive

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// Directive represents a parsed //pgogen:hot comment directive.
type Directive struct {
	Pos    token.Position // position of the directive comment
	Weight int64          // weight for the edge (default 10000)
}

// CallSite represents a resolved call site annotated with a directive.
type CallSite struct {
	Directive       Directive
	CallerName      string // linker symbol of the enclosing function
	CalleeName      string // linker symbol of the called function
	CallSiteOffset  int    // call_line - caller_start_line
	CallerStartLine int
	CallLine        int
	Warnings        []string
}

const defaultWeight = 10000

// ParseDirective parses a //pgogen:hot comment and returns a Directive.
// Returns ok=false if the comment is not a pgogen directive.
func ParseDirective(comment string, pos token.Position) (Directive, bool) {
	text := strings.TrimSpace(comment)
	// Strip leading "//"
	if !strings.HasPrefix(text, "//") {
		return Directive{}, false
	}
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimSpace(text)

	if !strings.HasPrefix(text, "pgogen:hot") {
		return Directive{}, false
	}

	d := Directive{
		Pos:    pos,
		Weight: defaultWeight,
	}

	rest := strings.TrimPrefix(text, "pgogen:hot")
	rest = strings.TrimSpace(rest)

	if rest != "" {
		parts := strings.FieldsSeq(rest)
		for part := range parts {
			if after, ok := strings.CutPrefix(part, "weight="); ok {
				valStr := after
				val, err := strconv.ParseInt(valStr, 10, 64)
				if err == nil && val > 0 {
					d.Weight = val
				}
			}
		}
	}

	return d, true
}

// FindDirectives scans a file's comment groups for //pgogen:hot directives
// and pairs each with the immediately following statement.
func FindDirectives(fset *token.FileSet, file *ast.File) []DirectiveWithNode {
	var results []DirectiveWithNode

	// Collect all comments that are pgogen directives
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			pos := fset.Position(c.Pos())
			d, ok := ParseDirective(c.Text, pos)
			if !ok {
				continue
			}
			results = append(results, DirectiveWithNode{
				Directive:   d,
				CommentLine: pos.Line,
			})
		}
	}

	// For each directive, find the next statement/expression on the following line
	for i := range results {
		results[i].Node = findNextCall(fset, file, results[i].CommentLine)
	}

	return results
}

// DirectiveWithNode pairs a parsed directive with the AST node it annotates.
type DirectiveWithNode struct {
	Directive   Directive
	CommentLine int
	Node        ast.Node // the call expression, or nil
}

// findNextCall walks the AST to find a call expression on the line immediately
// after the given comment line.
func findNextCall(fset *token.FileSet, file *ast.File, commentLine int) ast.Node {
	targetLine := commentLine + 1
	var found ast.Node

	ast.Inspect(file, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		if n == nil {
			return true
		}
		pos := fset.Position(n.Pos())
		if pos.Line == targetLine {
			if call, ok := n.(*ast.CallExpr); ok {
				found = call
				return false
			}
		}
		return true
	})

	return found
}

// FormatCallSite formats a call site for display.
func (cs *CallSite) FormatCallSite() string {
	return fmt.Sprintf("%s → %s (offset=%d, weight=%d)",
		cs.CallerName, cs.CalleeName, cs.CallSiteOffset, cs.Directive.Weight)
}

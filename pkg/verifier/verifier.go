package verifier

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/daanv2/go-force-inline/pkg/extensions/xio"
	"github.com/google/pprof/profile"
)

// Edge represents a call edge from a profile.
type Edge struct {
	Caller string
	Callee string
	Weight int64
}

// Verify reads a pprof profile and prints a CDF analysis of its edges,
// marking which ones fall within the hot threshold.
// When short is true, function names are shortened to just type and method.
func Verify(profilePath string, threshold float64, short bool, w io.Writer) error {
	f, err := os.Open(profilePath) //nolint:gosec // profilePath is from CLI arg, not user-controlled input
	if err != nil {
		return fmt.Errorf("opening profile: %w", err)
	}
	defer xio.CloseReport(f, nil)

	p, err := profile.Parse(f)
	if err != nil {
		return fmt.Errorf("parsing profile: %w", err)
	}

	// Build edge map from samples
	edgeMap := make(map[string]*Edge)
	for _, sample := range p.Sample {
		if len(sample.Location) < 2 || len(sample.Value) < 1 {
			continue
		}

		weight := sample.Value[0]

		// Stack: [callee, caller, ...]
		calleeLoc := sample.Location[0]
		callerLoc := sample.Location[1]

		calleeName := locationFuncName(calleeLoc)
		callerName := locationFuncName(callerLoc)

		key := callerName + " → " + calleeName
		if e, ok := edgeMap[key]; ok {
			e.Weight += weight
		} else {
			edgeMap[key] = &Edge{
				Caller: callerName,
				Callee: calleeName,
				Weight: weight,
			}
		}
	}

	// Collect and sort edges by weight (descending)
	edges := make([]*Edge, 0, len(edgeMap))
	var totalWeight int64
	for _, e := range edgeMap {
		edges = append(edges, e)
		totalWeight += e.Weight
	}

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Weight > edges[j].Weight
	})

	// Print CDF table
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(tw, "Edge\tWeight\tCDF%%\tHot?\n")
	_, _ = fmt.Fprintf(tw, "----\t------\t----\t----\n")

	var cumWeight int64
	hotThreshold := float64(totalWeight) * threshold / 100.0

	for _, e := range edges {
		cumWeight += e.Weight
		cdfPct := float64(cumWeight) / float64(totalWeight) * 100.0
		hot := "no"
		if float64(cumWeight-e.Weight) < hotThreshold {
			hot = "yes"
		}

		caller := e.Caller
		callee := e.Callee
		if short {
			caller = shortenName(caller)
			callee = shortenName(callee)
		}

		edgeStr := fmt.Sprintf("%s → %s", caller, callee)
		_, _ = fmt.Fprintf(tw, "%s\t%d\t%.1f%%\t%s\n", edgeStr, e.Weight, cdfPct, hot)
	}

	_ = tw.Flush()

	_, _ = fmt.Fprintf(w, "\nTotal weight: %d\n", totalWeight)
	_, _ = fmt.Fprintf(w, "Hot threshold: %.0f%% (cumulative weight < %.0f)\n", threshold, hotThreshold)

	return nil
}

func locationFuncName(loc *profile.Location) string {
	if len(loc.Line) > 0 && loc.Line[0].Function != nil {
		return loc.Line[0].Function.Name
	}

	return fmt.Sprintf("unknown@%d", loc.ID)
}

// shortenName strips the package path, keeping only the last package segment,
// type, and method. e.g.:
//   "github.com/user/pkg/sub.(*Type).Method" → "sub.(*Type).Method"
//   "github.com/user/pkg.Func"               → "pkg.Func"
func shortenName(name string) string {
	// Find the last '/' to get the last package segment
	slashIdx := strings.LastIndex(name, "/")
	if slashIdx >= 0 {
		return name[slashIdx+1:]
	}

	return name
}

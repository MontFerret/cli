package debugger

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/source"
)

func renderVariables(out io.Writer, variables []apidebug.Variable) {
	for _, variable := range variables {
		fmt.Fprintf(out, "  %s = %s\n", variable.Name, variable.Value.Display)
	}
}

func formatBreakpointIDs(ids []apidebug.BreakpointID) string {
	values := make([]string, 0, len(ids))

	for _, id := range ids {
		values = append(values, strconv.Itoa(int(id)))
	}

	return strings.Join(values, ", ")
}

func formatBindingMode(mode apidebug.BreakpointBindingMode) string {
	switch mode {
	case apidebug.BreakpointBindExact:
		return "exact"
	case apidebug.BreakpointBindNextExecutableInFunction:
		return "in-function"
	default:
		return "next-file"
	}
}

func formatLocation(location source.Location) string {
	if location.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", location.SourceName, location.Line, location.Column)
	}

	if location.Line > 0 {
		return fmt.Sprintf("%s:%d", location.SourceName, location.Line)
	}

	return location.SourceName
}

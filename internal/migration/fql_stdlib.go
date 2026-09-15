package migration

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/MontFerret/ferret/v2/pkg/parser/fql"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

type (
	fqlStdlibRule struct {
		target  string
		reason  string
		special func(fql.IFunctionCallContext) (target, reason string)
	}

	fqlStdlibCalls struct {
		calls   []*fql.FunctionCallContext
		locals  map[string]bool
		aliases map[string]string
	}

	fqlStdlibFinding struct {
		name   fql.IFunctionNameContext
		target string
		reason string
	}
)

const fqlAggregateReason = "legacy aggregate behavior is permissive for heterogeneous collections; " +
	"the strict math API requires a separate semantic migration"

// Only behavior-preserving call-target replacements belong here. Arrays and
// rand/range are intentionally outside this pass, including manual diagnostics.
var fqlStdlibRules = map[string]fqlStdlibRule{
	"json_parse":           {target: "encoding::json_parse"},
	"json_stringify":       {target: "encoding::json_stringify"},
	"encode_uri_component": {target: "encoding::query_escape"},
	"decode_uri_component": {target: "encoding::query_unescape"},
	"to_base64":            {target: "encoding::base64_encode"},
	"from_base64":          {target: "encoding::base64_decode"},
	"escape_html":          {target: "encoding::html_escape"},
	"unescape_html":        {target: "encoding::html_unescape"},

	"md5":          {target: "crypto::md5"},
	"sha1":         {target: "crypto::sha1"},
	"sha512":       {target: "crypto::sha512"},
	"random_token": {target: "crypto::random_token"},

	"base":     {target: "path::base"},
	"clean":    {target: "path::clean"},
	"dir":      {target: "path::dir"},
	"ext":      {target: "path::ext"},
	"is_abs":   {target: "path::is_abs"},
	"separate": {target: "path::separate"},
	"match":    {target: "path::match"},
	// There is no source-version marker: rerunning migration must preserve modern JOIN.
	"join": {reason: "join may mean legacy path::join or modern global string joining; the source version is ambiguous"},

	"values":          {target: "object::values"},
	"has":             {target: "object::has_key"},
	"zip":             {target: "object::zip"},
	"keep_keys":       {target: "object::keep_keys"},
	"merge":           {target: "object::merge"},
	"merge_recursive": {target: "object::merge_deep"},
	"keys":            {special: migrateFQLKeys},

	"now":                {target: "datetime::now"},
	"date":               {target: "datetime::parse"},
	"date_dayofweek":     {target: "datetime::day_of_week"},
	"date_year":          {target: "datetime::year"},
	"date_month":         {target: "datetime::month"},
	"date_day":           {target: "datetime::day"},
	"date_hour":          {target: "datetime::hour"},
	"date_minute":        {target: "datetime::minute"},
	"date_second":        {target: "datetime::second"},
	"date_millisecond":   {target: "datetime::millisecond"},
	"date_dayofyear":     {target: "datetime::day_of_year"},
	"date_leapyear":      {target: "datetime::is_leap_year"},
	"date_quarter":       {target: "datetime::quarter"},
	"date_days_in_month": {target: "datetime::days_in_month"},
	"date_format":        {target: "datetime::format"},
	"date_add":           {target: "datetime::add"},
	"date_subtract":      {target: "datetime::subtract"},
	"date_compare": {reason: "legacy component-range comparison semantics differ from datetime::same; " +
		"it is not a drop-in replacement"},
	"date_diff": {reason: "legacy integer/floating behavior differs from the canonical datetime::diff contract"},

	"pi":      {target: "math::pi"},
	"abs":     {target: "math::abs"},
	"acos":    {target: "math::acos"},
	"asin":    {target: "math::asin"},
	"atan":    {target: "math::atan"},
	"atan2":   {target: "math::atan2"},
	"ceil":    {target: "math::ceil"},
	"cos":     {target: "math::cos"},
	"degrees": {target: "math::degrees"},
	"exp":     {target: "math::exp"},
	"exp2":    {target: "math::exp2"},
	"floor":   {target: "math::floor"},
	"log":     {target: "math::log"},
	"log2":    {target: "math::log2"},
	"log10":   {target: "math::log10"},
	"pow":     {target: "math::pow"},
	"radians": {target: "math::radians"},
	"round":   {target: "math::round"},
	"sin":     {target: "math::sin"},
	"sqrt":    {target: "math::sqrt"},
	"tan":     {target: "math::tan"},

	"average":             {reason: fqlAggregateReason},
	"sum":                 {reason: fqlAggregateReason},
	"min":                 {reason: fqlAggregateReason},
	"max":                 {reason: fqlAggregateReason},
	"median":              {reason: fqlAggregateReason},
	"percentile":          {reason: fqlAggregateReason},
	"stddev_population":   {reason: fqlAggregateReason},
	"stddev_sample":       {reason: fqlAggregateReason},
	"variance_population": {reason: fqlAggregateReason},
	"variance_sample":     {reason: fqlAggregateReason},
}

func migrateFQLKeys(call fql.IFunctionCallContext) (string, string) {
	args := call.ArgumentList()
	if args != nil && len(args.AllExpression()) == 1 {
		return "object::keys", ""
	}

	return "", "only keys(obj) has a mechanical replacement; other arities require separate argument-aware review"
}

func planFQLStdlib(src source.Source, program *fql.ProgramContext) ([]fqlSourceEdit, []ManualAction, error) {
	var edits []fqlSourceEdit
	var actions []ManualAction
	for _, finding := range analyzeFQLStdlib(program) {
		name := finding.name
		spelling := name.GetText()
		if finding.reason != "" {
			actions = append(actions, ManualAction{
				Path:   src.Name(),
				Detail: spelling + "(...)",
				Reason: finding.reason,
				Line:   name.GetStart().GetLine(),
			})

			continue
		}

		span, ok := fqlByteSpan(src.Content(), source.Span{
			Start: name.GetStart().GetStart(),
			End:   name.GetStop().GetStop() + 1,
		})
		if !ok || span.End <= span.Start {
			return nil, nil, fmt.Errorf("locate stdlib call %s in Ferret source", spelling)
		}

		edits = append(edits, fqlSourceEdit{start: span.Start, end: span.End, text: finding.target})
	}

	return edits, actions, nil
}

// Share rule selection and resolution guards between read-only checks and edits.
// A reason takes precedence over the proposed target and requires manual review.
func analyzeFQLStdlib(program *fql.ProgramContext) []fqlStdlibFinding {
	info := fqlStdlibCalls{locals: make(map[string]bool), aliases: make(map[string]string)}
	collectFQLStdlibCalls(program, &info)

	var findings []fqlStdlibFinding
	for _, call := range info.calls {
		if namespace := call.Namespace(); namespace != nil && namespace.GetText() != "" {
			continue
		}

		name := call.FunctionName()
		spelling := name.GetText()
		rule, ok := fqlStdlibRules[strings.ToLower(spelling)]
		if !ok {
			continue
		}

		target, reason := rule.target, rule.reason
		if rule.special != nil {
			target, reason = rule.special(call)
		}

		if info.locals[strings.ToLower(spelling)] {
			reason = "a file-local function declaration or use function alias may resolve this call; the stdlib target is ambiguous"
		} else if namespace, _, _ := strings.Cut(target, "::"); namespace != "" {
			if alias, exists := info.aliases[namespace]; exists && alias != namespace {
				reason = fmt.Sprintf("use alias %q would redirect the replacement %s; the canonical target is ambiguous", namespace, target)
			}
		}

		findings = append(findings, fqlStdlibFinding{name: name, target: target, reason: reason})
	}

	return findings
}

// File-wide guards deliberately sacrifice some migrations to avoid implementing
// compiler scope resolution here. Collect declarations before considering any call.
func collectFQLStdlibCalls(node antlr.Tree, info *fqlStdlibCalls) {
	switch node := node.(type) {
	case *fql.FunctionCallContext:
		info.calls = append(info.calls, node)
	case *fql.FunctionDeclarationContext:
		info.locals[strings.ToLower(node.FunctionName().GetText())] = true
	case *fql.UseContext:
		if alias := node.GetAlias(); alias != nil {
			name := alias.GetText()
			target := strings.TrimSuffix(node.NamespaceIdentifier().GetText(), "::")
			if strings.Contains(target, "::") {
				info.locals[strings.ToLower(name)] = true
			}

			if previous, exists := info.aliases[name]; exists && previous != target {
				// Conflicting declarations must not hide a possible redirection.
				target = ""
			}

			info.aliases[name] = target
		}
	}

	for i := 0; i < node.GetChildCount(); i++ {
		collectFQLStdlibCalls(node.GetChild(i), info)
	}
}

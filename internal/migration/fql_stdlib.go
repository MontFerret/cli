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
		target    string
		namespace string
		reason    string
		special   func(fql.IFunctionCallContext) fqlStdlibRewrite
	}

	fqlStdlibCalls struct {
		calls   []*fql.FunctionCallContext
		locals  map[string]bool
		aliases map[string]string
	}

	fqlStdlibFinding struct {
		call fql.IFunctionCallContext
		fqlStdlibRewrite
	}
)

const fqlAggregateReason = "legacy aggregate behavior is permissive for heterogeneous collections; " +
	"the strict math API requires a separate semantic migration"

// Rules declare canonical namespaces separately from generated source text.
// Only behavior-preserving rewrites are automatic; semantic differences stay manual.
var fqlStdlibRules = map[string]fqlStdlibRule{
	"json_parse":           {target: "encoding::json_parse", namespace: "encoding"},
	"json_stringify":       {target: "encoding::json_stringify", namespace: "encoding"},
	"encode_uri_component": {target: "encoding::query_escape", namespace: "encoding"},
	"decode_uri_component": {target: "encoding::query_unescape", namespace: "encoding"},
	"to_base64":            {target: "encoding::base64_encode", namespace: "encoding"},
	"from_base64":          {target: "encoding::base64_decode", namespace: "encoding"},
	"escape_html":          {target: "encoding::html_escape", namespace: "encoding"},
	"unescape_html":        {target: "encoding::html_unescape", namespace: "encoding"},

	"md5":          {target: "crypto::md5", namespace: "crypto"},
	"sha1":         {target: "crypto::sha1", namespace: "crypto"},
	"sha512":       {target: "crypto::sha512", namespace: "crypto"},
	"random_token": {target: "crypto::random_token", namespace: "crypto"},

	"base":     {target: "path::base", namespace: "path"},
	"clean":    {target: "path::clean", namespace: "path"},
	"dir":      {target: "path::dir", namespace: "path"},
	"ext":      {target: "path::ext", namespace: "path"},
	"is_abs":   {target: "path::is_abs", namespace: "path"},
	"separate": {target: "path::separate", namespace: "path"},
	"match":    {target: "path::match", namespace: "path"},
	// There is no source-version marker: rerunning migration must preserve modern JOIN.
	"join": {reason: "join may mean legacy path::join or modern global string joining; the source version is ambiguous"},

	"first":          {target: "arrays::first", namespace: "arrays"},
	"flatten":        {target: "arrays::flatten", namespace: "arrays"},
	"last":           {target: "arrays::last", namespace: "arrays"},
	"sorted":         {target: "arrays::sorted", namespace: "arrays"},
	"unique":         {target: "arrays::unique", namespace: "arrays"},
	"nth":            {target: "arrays::at", namespace: "arrays"},
	"remove_values":  {target: "arrays::remove_any", namespace: "arrays"},
	"slice":          {target: "arrays::slice", namespace: "arrays"},
	"intersection":   {target: "arrays::intersection", namespace: "arrays"},
	"minus":          {target: "arrays::difference", namespace: "arrays"},
	"union":          {target: "arrays::concat", namespace: "arrays"},
	"union_distinct": {target: "arrays::union", namespace: "arrays"},
	"append":         {special: migrateFQLAppend},
	"push":           {special: migrateFQLAppend},
	"position":       {special: migrateFQLPosition},
	"remove_value":   {special: migrateFQLRemoveValue},
	"sorted_unique":  {special: migrateFQLSortedUnique},
	"shift":          {special: migrateFQLShift},
	"outersection":   {special: migrateFQLOutersection},
	"range":          {special: migrateFQLRange},
	"rand":           {special: migrateFQLRand},
	"pop":            {reason: "legacy pop returns an immutable copy without the last element; a length-dependent replacement must preserve argument evaluation count"},
	"unshift":        {reason: "legacy unshift evaluates the array before the value; arrays::concat([value], array) reverses evaluation order, and unique mode has additional semantics"},
	"remove_nth":     {reason: "legacy remove_nth delegates removal to the copied host list; arrays::remove_at returns an unchanged copy for negative or out-of-range indexes"},

	"values":          {target: "object::values", namespace: "object"},
	"has":             {target: "object::has_key", namespace: "object"},
	"zip":             {target: "object::zip", namespace: "object"},
	"keep_keys":       {target: "object::keep_keys", namespace: "object"},
	"merge":           {target: "object::merge", namespace: "object"},
	"merge_recursive": {target: "object::merge_deep", namespace: "object"},
	"keys":            {special: migrateFQLKeys},

	"now":                {target: "datetime::now", namespace: "datetime"},
	"date":               {target: "datetime::parse", namespace: "datetime"},
	"date_dayofweek":     {target: "datetime::day_of_week", namespace: "datetime"},
	"date_year":          {target: "datetime::year", namespace: "datetime"},
	"date_month":         {target: "datetime::month", namespace: "datetime"},
	"date_day":           {target: "datetime::day", namespace: "datetime"},
	"date_hour":          {target: "datetime::hour", namespace: "datetime"},
	"date_minute":        {target: "datetime::minute", namespace: "datetime"},
	"date_second":        {target: "datetime::second", namespace: "datetime"},
	"date_millisecond":   {target: "datetime::millisecond", namespace: "datetime"},
	"date_dayofyear":     {target: "datetime::day_of_year", namespace: "datetime"},
	"date_leapyear":      {target: "datetime::is_leap_year", namespace: "datetime"},
	"date_quarter":       {target: "datetime::quarter", namespace: "datetime"},
	"date_days_in_month": {target: "datetime::days_in_month", namespace: "datetime"},
	"date_format":        {target: "datetime::format", namespace: "datetime"},
	"date_add":           {target: "datetime::add", namespace: "datetime"},
	"date_subtract":      {target: "datetime::subtract", namespace: "datetime"},
	"date_compare": {reason: "legacy component-range comparison semantics differ from datetime::same; " +
		"it is not a drop-in replacement"},
	"date_diff": {special: migrateFQLDateDiff},

	"pi":      {target: "math::pi", namespace: "math"},
	"abs":     {target: "math::abs", namespace: "math"},
	"acos":    {target: "math::acos", namespace: "math"},
	"asin":    {target: "math::asin", namespace: "math"},
	"atan":    {target: "math::atan", namespace: "math"},
	"atan2":   {target: "math::atan2", namespace: "math"},
	"ceil":    {target: "math::ceil", namespace: "math"},
	"cos":     {target: "math::cos", namespace: "math"},
	"degrees": {target: "math::degrees", namespace: "math"},
	"exp":     {target: "math::exp", namespace: "math"},
	"exp2":    {target: "math::exp2", namespace: "math"},
	"floor":   {target: "math::floor", namespace: "math"},
	"log":     {target: "math::log", namespace: "math"},
	"log2":    {target: "math::log2", namespace: "math"},
	"log10":   {target: "math::log10", namespace: "math"},
	"pow":     {target: "math::pow", namespace: "math"},
	"radians": {target: "math::radians", namespace: "math"},
	"round":   {target: "math::round", namespace: "math"},
	"sin":     {target: "math::sin", namespace: "math"},
	"sqrt":    {target: "math::sqrt", namespace: "math"},
	"tan":     {target: "math::tan", namespace: "math"},

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

func planFQLStdlib(src source.Source, program *fql.ProgramContext) ([]fqlSourceEdit, []ManualAction, error) {
	var edits []fqlSourceEdit
	var actions []ManualAction
	for _, finding := range analyzeFQLStdlib(program) {
		name := finding.call.FunctionName()
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

		callEdits, err := finding.edits(src.Content(), finding.call)
		if err != nil {
			return nil, actions, err
		}

		edits = append(edits, callEdits...)
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

		rewrite := fqlStdlibRewrite{target: rule.target, reason: rule.reason}
		if rule.namespace != "" {
			rewrite.namespaces = []string{rule.namespace}
		}

		if rule.special != nil {
			rewrite = rule.special(call)
		}

		if info.locals[strings.ToLower(spelling)] {
			rewrite.reason = "a file-local function declaration or use function alias may resolve this call; the stdlib target is ambiguous"
		} else {
			for _, namespace := range rewrite.namespaces {
				if alias, exists := info.aliases[namespace]; exists && alias != namespace {
					rewrite.reason = fmt.Sprintf("use alias %q would redirect the replacement %s; the canonical target is ambiguous", namespace, rewrite.description())

					break
				}
			}
		}

		findings = append(findings, fqlStdlibFinding{call: call, fqlStdlibRewrite: rewrite})
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

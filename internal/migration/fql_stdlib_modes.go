package migration

import "github.com/MontFerret/ferret/v2/pkg/parser/fql"

func migrateFQLKeys(call fql.IFunctionCallContext) fqlStdlibRewrite {
	rewrite := fqlStdlibRewrite{target: "object::keys", namespaces: []string{"object"}}

	args := fqlCallArguments(call)
	if len(args) == 1 {
		return rewrite
	}

	if len(args) != 2 {
		return fqlStdlibRewrite{reason: "keys supports one object and an optional literal Boolean sort flag; this arity requires manual review"}
	}

	mode, ok := fqlBooleanLiteral(args[1])
	if !ok {
		return fqlStdlibRewrite{reason: "legacy keys selects sorted or unordered keys using its Boolean flag; a dynamic sort mode cannot select a fixed canonical replacement"}
	}

	rewrite.dropLast = true

	if mode {
		rewrite.wrapper = "arrays::sorted"
		rewrite.namespaces = []string{"object", "arrays"}
	}

	return rewrite
}

func migrateFQLPosition(call fql.IFunctionCallContext) fqlStdlibRewrite {
	rewrite := fqlStdlibRewrite{target: "arrays::contains", namespaces: []string{"arrays"}}

	args := fqlCallArguments(call)
	if len(args) == 2 {
		return rewrite
	}

	if len(args) != 3 {
		return fqlStdlibRewrite{reason: "position supports two arguments and an optional literal Boolean index mode; this arity requires manual review"}
	}

	mode, ok := fqlBooleanLiteral(args[2])
	if !ok {
		return fqlStdlibRewrite{reason: "legacy position returns a Boolean or an index depending on its flag; a dynamic mode cannot select arrays::contains or arrays::index_of"}
	}

	rewrite.dropLast = true

	if mode {
		rewrite.target = "arrays::index_of"
	}

	return rewrite
}

func migrateFQLAppend(call fql.IFunctionCallContext) fqlStdlibRewrite {
	rewrite := fqlStdlibRewrite{target: "arrays::append", namespaces: []string{"arrays"}}

	args := fqlCallArguments(call)
	if len(args) == 2 {
		return rewrite
	}

	if len(args) != 3 {
		return fqlStdlibRewrite{reason: "append/push supports two arguments and an optional Boolean unique mode; this arity requires manual review"}
	}

	mode, ok := fqlBooleanLiteral(args[2])
	if !ok {
		return fqlStdlibRewrite{reason: "legacy append/push has a conditional unique mode; a dynamic flag cannot select a behavior-preserving canonical call"}
	}

	if mode {
		return fqlStdlibRewrite{reason: "legacy unique append/push only suppresses the new value when already present; arrays::unique(arrays::append(...)) would also remove existing duplicates"}
	}

	rewrite.dropLast = true

	return rewrite
}

func migrateFQLRemoveValue(call fql.IFunctionCallContext) fqlStdlibRewrite {
	rewrite := fqlStdlibRewrite{target: "arrays::remove", namespaces: []string{"arrays"}}

	args := fqlCallArguments(call)
	if len(args) == 2 {
		return rewrite
	}

	if len(args) != 3 {
		return fqlStdlibRewrite{reason: "remove_value supports two arguments and an optional integer removal limit; this arity requires manual review"}
	}

	if !fqlNegativeIntegerLiteral(args[2]) {
		return fqlStdlibRewrite{reason: "legacy remove_value uses negative integer limits for unlimited removal, zero for no removal, and positive limits for bounded removal; arrays::remove has no limit mode, so only a valid negative integer literal can be dropped"}
	}

	rewrite.dropLast = true

	return rewrite
}

func migrateFQLSortedUnique(call fql.IFunctionCallContext) fqlStdlibRewrite {
	if len(fqlCallArguments(call)) != 1 {
		return fqlStdlibRewrite{reason: "sorted_unique requires exactly one array to compose arrays::sorted(arrays::unique(...))"}
	}

	return fqlStdlibRewrite{target: "arrays::unique", wrapper: "arrays::sorted", namespaces: []string{"arrays"}}
}

func migrateFQLShift(call fql.IFunctionCallContext) fqlStdlibRewrite {
	if len(fqlCallArguments(call)) != 1 {
		return fqlStdlibRewrite{reason: "immutable shift requires exactly one array to replace it with arrays::slice(array, 1)"}
	}

	return fqlStdlibRewrite{target: "arrays::slice", addLast: "1", namespaces: []string{"arrays"}}
}

func migrateFQLOutersection(call fql.IFunctionCallContext) fqlStdlibRewrite {
	count := len(fqlCallArguments(call))
	if count == 2 {
		return fqlStdlibRewrite{target: "arrays::symmetric_difference", namespaces: []string{"arrays"}}
	}

	if count < 2 {
		return fqlStdlibRewrite{reason: "outersection requires at least two arrays; only exactly two inputs have a behavior-preserving canonical replacement"}
	}

	return fqlStdlibRewrite{reason: "legacy outersection keeps values present in exactly one input; arrays::symmetric_difference uses odd-number-of-inputs semantics for three or more arrays"}
}

func migrateFQLDateDiff(call fql.IFunctionCallContext) fqlStdlibRewrite {
	args := fqlCallArguments(call)
	if len(args) != 3 && len(args) != 4 {
		return fqlStdlibRewrite{reason: "date_diff supports three arguments and an optional Boolean floating mode; this arity requires manual review"}
	}

	if len(args) == 4 {
		mode, ok := fqlBooleanLiteral(args[3])
		if ok && mode {
			return fqlStdlibRewrite{target: "datetime::diff", dropLast: true, namespaces: []string{"datetime"}}
		}
	}

	return fqlStdlibRewrite{reason: "legacy date_diff truncates toward zero when its floating flag is false or omitted, while datetime::diff always returns a Float; only a literal true floating mode has a safe replacement"}
}

func migrateFQLRange(call fql.IFunctionCallContext) fqlStdlibRewrite {
	count := len(fqlCallArguments(call))
	if count != 2 && count != 3 {
		return fqlStdlibRewrite{reason: "range requires start and end with an optional step; this arity requires manual review"}
	}

	return fqlStdlibRewrite{target: "arrays::range", namespaces: []string{"arrays"}}
}

func migrateFQLRand(call fql.IFunctionCallContext) fqlStdlibRewrite {
	count := len(fqlCallArguments(call))
	if count == 0 {
		return fqlStdlibRewrite{target: "random::float", namespaces: []string{"random"}}
	}

	if count > 2 {
		return fqlStdlibRewrite{reason: "legacy rand supports zero, one, or two arguments; this arity requires manual review"}
	}

	return fqlStdlibRewrite{reason: "parameterized legacy rand uses a historical rounded/floored range calculation (max/2 to max*2 for one argument) and has no behavior-preserving canonical call"}
}

package migration

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	ferret "github.com/MontFerret/ferret/v2"
	"github.com/MontFerret/ferret/v2/pkg/runtime"
	"github.com/MontFerret/ferret/v2/pkg/source"
	"github.com/MontFerret/ferret/v2/pkg/stdlib"
)

func TestFQLStdlibRuntimeEquivalence(t *testing.T) {
	tests := []struct {
		expression, want, check string
		calls                   []string
	}{
		{`position(observe("array", a), observe("value", 3))`, "true", "is_bool(result)", []string{"array", "value"}},
		{`position(a, 9, false)`, "false", "is_bool(result)", nil},
		{`position(a, 3, true)`, "0", "is_int(result)", nil},
		{`position(a, 9, true)`, "-1", "is_int(result)", nil},
		{`sorted_unique(observe("array", a))`, "[1,2,3]", "is_array(result)", []string{"array"}},
		{`sorted_unique([])`, "[]", "is_array(result)", nil},
		{`shift(observe("array", a))`, "[1,3,2]", "is_array(result)", []string{"array"}},
		{`shift([])`, "[]", "is_array(result)", nil},
		{`shift([1])`, "[]", "is_array(result)", nil},
		{`append(observe("array", a), observe("value", 3), false)`, "[3,1,3,2,3]", "is_array(result)", []string{"array", "value"}},
		{`push(a, 5, false)`, "[3,1,3,2,5]", "is_array(result)", nil},
		{`remove_value(observe("array", a), observe("value", 3), -17)`, "[1,2]", "is_array(result)", []string{"array", "value"}},
		{`remove_value([], 3, -1)`, "[]", "is_array(result)", nil},
		{`outersection(observe("left", a), observe("right", b))`, "[1,2,4]", "is_array(result)", []string{"left", "right"}},
		{`outersection(a, a)`, "[]", "is_array(result)", nil},
		{`union(a, b)`, "[3,1,3,2,3,4,4]", "is_array(result)", nil},
		{`union_distinct(a, b)`, "[3,1,2,4]", "is_array(result)", nil},
		{`date_diff(left, right, "seconds", true)`, "1.5", "is_float(result)", nil},
		{`date_diff(right, left, "second", (true))`, "-1.5", "is_float(result)", nil},
		{`date_diff(left, left, "hour", true)`, "0", "is_float(result)", nil},
		{`range(1, 3)`, "[1,2,3]", "is_array(result) AND is_float(arrays::first(result))", nil},
		{`range(3, 1, -1)`, "[3,2,1]", "is_array(result) AND is_float(arrays::first(result))", nil},
		{`range(0, 1, 0.5)`, "[0,0.5,1]", "is_array(result) AND is_float(arrays::first(result))", nil},
		{`rand()`, "", "is_float(result) AND result >= 0 AND result < 1", nil},
		{`sorted_unique(shift(a))`, "[1,2,3]", "is_array(result)", nil},
		{`sorted_unique(observe("array", null)) ON ERROR RETRY 1 OR RETURN []`, "[]", "is_array(result)", []string{"array", "array"}},
		{`shift(observe("array", null)) ON ERROR RETURN []`, "[]", "is_array(result)", []string{"array"}},
		{`sorted_unique(null)?`, "null", "is_none(result)", nil},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			var calls []string
			engine, err := ferret.New(ferret.WithStdlib(stdlib.Full()), ferret.WithFunctionsRegistrar(func(ns runtime.Namespace) {
				ns.Function().A2().Add("observe", func(_ context.Context, label, value runtime.Value) (runtime.Value, error) {
					calls = append(calls, string(label.(runtime.String)))

					return value, nil
				})
			}))
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if err := engine.Close(); err != nil {
					t.Errorf("close engine: %v", err)
				}
			})

			input := `let a = [3, 1, 3, 2]
let b = [3, 4, 4]
let left = datetime::parse("2026-09-15T00:00:00Z")
let right = datetime::parse("2026-09-15T00:00:01.500Z")
let result = ` + test.expression + `
return [result, ` + test.check + `, a, b, random::float()]`
			migrated, err := migrateFQLSource(source.New("query.fql", input))
			if err != nil || !migrated.Changed || len(migrated.ManualActions) != 0 {
				t.Fatalf("migration failed: %#v, %v", migrated, err)
			}

			var previous string
			for _, query := range []string{input, string(migrated.Data)} {
				calls = nil

				output, err := engine.Run(t.Context(), source.New("query.fql", query), ferret.WithSessionRandomSeed(42))
				if err != nil {
					t.Fatal(err)
				}

				var parts []json.RawMessage
				if err := json.Unmarshal(output.Content, &parts); err != nil {
					t.Fatal(err)
				}

				if len(parts) != 5 || string(parts[1]) != "true" || string(parts[2]) != "[3,1,3,2]" || string(parts[3]) != "[3,4,4]" {
					t.Fatalf("type or source immutability changed: %s", output.Content)
				}

				if test.want != "" && string(parts[0]) != test.want {
					t.Fatalf("result = %s, want %s", parts[0], test.want)
				}

				if !reflect.DeepEqual(calls, test.calls) {
					t.Fatalf("evaluation order/count = %v, want %v", calls, test.calls)
				}

				if previous != "" && previous != string(output.Content) {
					t.Fatalf("runtime result or RNG consumption changed: old=%s new=%s", previous, output.Content)
				}

				previous = string(output.Content)
			}
		})
	}
}

func TestFQLStdlibHistoricalKeysModes(t *testing.T) {
	// Core alpha.52 objects/keys_test.go requires ascending keys for true and
	// unordered membership for false. Alpha.54 no longer registers keys/2.
	for _, sorted := range []bool{false, true} {
		for _, object := range []string{`{ z: 1, a: 2, m: 3 }`, `{}`} {
			mode := "false"
			if sorted {
				mode = "true"
			}

			t.Run(object+"/"+mode, func(t *testing.T) {
				calls := 0
				engine, err := ferret.New(ferret.WithStdlib(stdlib.Full()), ferret.WithFunctionsRegistrar(func(ns runtime.Namespace) {
					ns.Function().A1().Add("observe", func(_ context.Context, value runtime.Value) (runtime.Value, error) {
						calls++

						return value, nil
					})
				}))
				if err != nil {
					t.Fatal(err)
				}

				t.Cleanup(func() {
					if err := engine.Close(); err != nil {
						t.Errorf("close engine: %v", err)
					}
				})

				input := "let obj = " + object + "\nreturn keys(observe(obj), " + mode + ")"
				result, err := migrateFQLSource(source.New("query.fql", input))
				if err != nil || !result.Changed {
					t.Fatalf("migrate keys: %#v, %v", result, err)
				}

				output, err := engine.Run(t.Context(), source.New("query.fql", string(result.Data)))
				if err != nil {
					t.Fatal(err)
				}

				var keys []string
				if err := json.Unmarshal(output.Content, &keys); err != nil {
					t.Fatal(err)
				}

				if sorted && !slices.IsSorted(keys) {
					t.Fatalf("keys are not ascending: %v", keys)
				}

				slices.Sort(keys)
				var want []string
				if object != "{}" {
					want = []string{"a", "m", "z"}
				}

				if !slices.Equal(keys, want) || calls != 1 {
					t.Fatalf("keys=%v, want=%v; evaluations=%d", keys, want, calls)
				}
			})
		}
	}
}

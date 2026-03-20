package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MontFerret/ferret/v2/pkg/runtime"
)

const paramFlag = "param"

func parseParams(flags []string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	for _, entry := range flags {
		pair := strings.SplitN(entry, ":", 2)

		if len(pair) < 2 {
			return nil, runtime.Error(runtime.ErrInvalidArgument, entry)
		}

		var value interface{}
		key := pair[0]

		err := json.Unmarshal([]byte(pair[1]), &value)

		if err != nil {
			return nil, fmt.Errorf("invalid value for parameter %q: %w", key, err)
		}

		res[key] = value
	}

	return res, nil
}

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
)

const paramFlag = "param"

func parseParams(flags []string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	for _, entry := range flags {
		key, value, err := parseParam(entry)
		if err != nil {
			return nil, err
		}

		res[key] = value
	}

	return res, nil
}

func parseParam(input string) (string, any, error) {
	name, raw, ok := strings.Cut(input, "=")
	if !ok {
		name, raw, ok = strings.Cut(input, ":")
	}

	if !ok {
		return "", nil, fmt.Errorf("invalid param %q: expected name=value", input)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("invalid param %q: parameter name cannot be empty", input)
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err == nil {
		return name, value, nil
	}

	return name, raw, nil
}

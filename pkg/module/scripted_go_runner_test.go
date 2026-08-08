package module

import (
	"context"
	"fmt"
	"strings"
)

type (
	scriptedGoRunner struct {
		responses map[string]scriptedGoResponse
	}

	scriptedGoResponse struct {
		output string
		err    error
	}
)

func (runner *scriptedGoRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	response, exists := runner.responses[key]
	if !exists {
		return nil, fmt.Errorf("unexpected Go command: %s", key)
	}

	return []byte(response.output), response.err
}

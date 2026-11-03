package runtime

import "errors"

// Consumption and Close retain the same cleanup causes; avoid rendering them twice.
func joinOutputErrors(values ...error) error {
	var result error
	for _, value := range values {
		if value != nil && !errors.Is(result, value) {
			result = errors.Join(result, value)
		}
	}

	return result
}

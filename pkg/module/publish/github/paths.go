package github

import (
	"net/url"
	"strings"
)

func apiPath(parts ...string) string {
	escaped := make([]string, len(parts))

	for index, part := range parts {
		escaped[index] = url.PathEscape(part)
	}

	return "/" + strings.Join(escaped, "/")
}

func contentPath(owner, repositoryName, filename string) string {
	parts := []string{"repos", owner, repositoryName, "contents"}
	parts = append(parts, strings.Split(filename, "/")...)

	return apiPath(parts...)
}

func splitRepository(fullName string) (string, string, bool) {
	owner, name, found := strings.Cut(fullName, "/")

	return owner, name, found && owner != "" && name != "" && !strings.Contains(name, "/")
}

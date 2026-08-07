package module

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	modulemanifest "github.com/MontFerret/specs/pkg/module"
)

var scpRemotePattern = regexp.MustCompile(`^(?:[^@/:]+@)?([^/:]+):(.+)$`)

func validatePublicationMetadata(manifest *modulemanifest.Manifest) error {
	if strings.Contains(strings.ToLower(manifest.Description), "todo") {
		return fmt.Errorf("module manifest description still contains a TODO placeholder")
	}

	if manifest.License == "LicenseRef-TODO" {
		return fmt.Errorf("module manifest license still contains the scaffold placeholder")
	}

	documentation, err := url.Parse(manifest.Documentation)
	if err != nil || documentation.Hostname() == "example.invalid" || strings.HasSuffix(documentation.Hostname(), ".invalid") {
		return fmt.Errorf("module manifest documentation still contains the scaffold placeholder")
	}

	return nil
}

func normalizeGitRemote(remote string) (string, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", fmt.Errorf("remote URL is empty")
	}

	if match := scpRemotePattern.FindStringSubmatch(remote); match != nil && !strings.Contains(remote, "://") {
		return canonicalRepositoryURL(match[1], match[2])
	}

	parsed, err := url.Parse(remote)
	if err != nil {
		return "", fmt.Errorf("parse remote URL: %w", err)
	}

	if parsed.Scheme != "https" && parsed.Scheme != "ssh" {
		return "", fmt.Errorf("remote must use HTTPS, SSH, or SCP syntax")
	}

	if parsed.Scheme == "https" && parsed.User != nil {
		return "", fmt.Errorf("HTTPS remote must not contain credentials")
	}

	if parsed.Hostname() == "" || parsed.Path == "" || parsed.Path == "/" {
		return "", fmt.Errorf("remote must include a host and repository path")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("remote must not contain a query or fragment")
	}

	return canonicalRepositoryURL(parsed.Host, parsed.Path)
}

func canonicalRepositoryURL(host, repositoryPath string) (string, error) {
	repositoryPath = strings.TrimPrefix(repositoryPath, "/")
	repositoryPath = strings.TrimSuffix(repositoryPath, "/")
	repositoryPath = strings.TrimSuffix(repositoryPath, ".git")

	for _, segment := range strings.Split(repositoryPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("remote must include a normalized repository path")
		}
	}

	cleaned := path.Clean(repositoryPath)
	if host == "" || cleaned == "." || cleaned == "" || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("remote must include a normalized repository path")
	}

	return (&url.URL{Scheme: "https", Host: host, Path: "/" + cleaned}).String(), nil
}

func validateManifestRepository(manifest *modulemanifest.Manifest, repositoryURL, sourcePath string) error {
	if manifest.Repository == nil {
		return nil
	}

	manifestRepository, err := normalizeGitRemote(manifest.Repository.URL)
	if err != nil {
		return fmt.Errorf("normalize manifest repository URL: %w", err)
	}

	if manifestRepository != repositoryURL {
		return fmt.Errorf("manifest repository %q does not match Git origin %q", manifestRepository, repositoryURL)
	}

	if manifest.Repository.Directory != sourcePath {
		return fmt.Errorf("manifest repository directory %q does not match module source path %q", manifest.Repository.Directory, sourcePath)
	}

	return nil
}

func marshalRegistryRecord(record any) ([]byte, error) {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode registry publication record: %w", err)
	}

	return append(data, '\n'), nil
}

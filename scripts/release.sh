#!/bin/bash
set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "  e.g. $0 v2.0.0"
  exit 1
fi

VERSION="$1"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
  echo "Error: version must start with 'v' followed by semver (e.g. v2.0.0)"
  exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Error: working directory has uncommitted changes"
  exit 1
fi

echo "Creating tag $VERSION..."
git tag "$VERSION"

echo "Pushing tag $VERSION..."
git push origin "$VERSION"

echo "Done. GoReleaser will build and create a draft release at:"
echo "  https://github.com/$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/' | sed 's/.*github.com[:/]\(.*\)/\1/')/releases"

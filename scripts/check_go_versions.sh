#!/bin/bash

set -e

echo "Checking Go version consistency across files..."
echo ""

# Extract Docker image name from Dockerfile
DOCKER_IMAGE=$(grep 'FROM registry.access.redhat.com/ubi9/go-toolset:' Dockerfile | head -n 1 | sed -E 's/.*FROM ([^ ]+).*/\1/')

if [ -z "$DOCKER_IMAGE" ]; then
    echo "ERROR: Could not extract Docker image from Dockerfile"
    exit 1
fi

echo "INFO: Inspecting Docker image: $DOCKER_IMAGE"

# Use skopeo to inspect the image and extract Go version from labels
DOCKER_MAJOR_MINOR=$(skopeo inspect docker://$DOCKER_IMAGE | jq -r '.Labels.version_major_minor // empty')

if [ -z "$DOCKER_MAJOR_MINOR" ]; then
    echo "ERROR: Could not extract Go version from Docker image"
    exit 1
fi

echo "INFO: Dockerfile: $DOCKER_MAJOR_MINOR"

# Extract Go version from gotests workflow
GOTESTS_VERSION=$(grep 'go-version:' .github/workflows/gotests.yaml | sed -E 's/.*go-version:\s*"?([0-9]+\.[0-9]+)"?.*/\1/')
echo "INFO: gotests.yaml: $GOTESTS_VERSION"

# Extract major.minor version from gotests (e.g., 1.24 from 1.24.0)
GOTESTS_MAJOR_MINOR=$(echo "$GOTESTS_VERSION" | cut -d. -f1,2)

echo ""
echo "Comparing major.minor versions:"
echo "  Dockerfile:   $DOCKER_MAJOR_MINOR"
echo "  gotests.yaml: $GOTESTS_MAJOR_MINOR (from $GOTESTS_VERSION)"

# Compare major.minor versions only
if [ "$DOCKER_MAJOR_MINOR" != "$GOTESTS_MAJOR_MINOR" ]; then
    echo ""
    echo "ERROR: Go version mismatch between Dockerfile and gotests.yaml!"
    echo "  Dockerfile:   $DOCKER_MAJOR_MINOR"
    echo "  gotests.yaml: $GOTESTS_MAJOR_MINOR (from $GOTESTS_VERSION)"
    echo ""
    echo "Please ensure Go major.minor versions are synchronized."
    exit 1
fi

echo ""
echo "SUCCESS: Go major.minor versions are in sync ($DOCKER_MAJOR_MINOR)"
echo "  Dockerfile:   $DOCKER_MAJOR_MINOR"
echo "  gotests.yaml: $GOTESTS_MAJOR_MINOR (from $GOTESTS_VERSION)"

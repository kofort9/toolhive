#!/bin/bash
set -e

# Script to run operator integration tests with envtest

echo "Setting up envtest environment..."

# Check if setup-envtest is installed
if ! command -v setup-envtest &> /dev/null; then
    echo "Installing setup-envtest..."
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
fi

# Download and setup envtest binaries
ENVTEST_ASSETS_DIR="/tmp/kubebuilder-envtest"
echo "Downloading envtest binaries to ${ENVTEST_ASSETS_DIR}..."
ENVTEST_K8S_VERSION=$(setup-envtest use --bin-dir "${ENVTEST_ASSETS_DIR}" | grep Path | cut -d' ' -f2)

echo "Using envtest binaries from: ${ENVTEST_K8S_VERSION}"

# Export the environment variable
export KUBEBUILDER_ASSETS="${ENVTEST_K8S_VERSION}"

# Run the tests
echo "Running operator integration tests..."
go test ./test/integration/operator/... -v -count=1 -timeout=120s "$@"
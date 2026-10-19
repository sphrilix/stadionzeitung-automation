#!/usr/bin/env bash
# Bazel workspace status: the version stamped into release binaries.
echo "STABLE_VERSION $(git describe --tags --always --dirty 2>/dev/null || echo dev)"

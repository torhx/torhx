#!/bin/bash
# Shared helpers for workspace scripts. Source this file; do not execute it.

# Go import path prefix of this repository.
REPO_MODULE="github.com/torhx/torhx"

# Run from the repository root regardless of the caller's directory.
cd "$(dirname "${BASH_SOURCE[0]}")/.." || exit 1

# Print every workspace module directory, one per line.
module_dirs() {
  local dir
  for dir in pkg/component/* pkg/fx/* pkg/module/*; do
    if [ -f "$dir/go.mod" ]; then
      echo "$dir"
    fi
  done
}

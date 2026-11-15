#!/bin/bash
# Run golangci-lint (with --fix) in every workspace module.
# Config is the repository-root .golangci.yml, found by upward search.

source "$(dirname "$0")/lib.sh"

failed=0
for dir in $(module_dirs); do
  echo "Linting $dir..."
  (cd "$dir" && golangci-lint run --fix) || failed=1
done

exit $failed

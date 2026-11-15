#!/bin/bash
# Run tests with the race detector in every workspace module.
# Extra arguments are passed to `go test`, e.g. ./scripts/test.sh -run TestFoo

source "$(dirname "$0")/lib.sh"

failed=0
for dir in $(module_dirs); do
  echo "Testing $dir..."
  (cd "$dir" && CGO_ENABLED=1 go test -race "$@" ./...) || failed=1
done

exit $failed

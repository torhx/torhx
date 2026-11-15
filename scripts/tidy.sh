#!/bin/bash
# Sync the workspace, tidy every module, and vendor workspace dependencies.
set -e

source "$(dirname "$0")/lib.sh"

modules=$(module_dirs)
if [ -z "$modules" ]; then
  echo "No modules in workspace, nothing to tidy."
  exit 0
fi

echo "Syncing workspace..."
go work sync

echo "Tidying modules..."
for dir in $modules; do
  echo "Tidying $dir..."
  go mod tidy -C "$dir"
done

echo "Vendoring workspace dependencies..."
go work vendor

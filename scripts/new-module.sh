#!/bin/bash
# Scaffold a new workspace module: go.mod, go.work entry, and CI workflow.
#
# Usage: ./scripts/new-module.sh <component|fx|module> <name>
#   ./scripts/new-module.sh component orderbook   -> pkg/component/orderbook
#   ./scripts/new-module.sh fx orderbook          -> pkg/fx/fxorderbook
#   ./scripts/new-module.sh module exchange       -> pkg/module/exchange
set -e

source "$(dirname "$0")/lib.sh"

layer=$1
name=$2

if [[ ! "$name" =~ ^[a-z][a-z0-9]*$ ]]; then
  echo "Usage: $0 <component|fx|module> <name>   (name: lowercase letters and digits)" >&2
  exit 1
fi

case "$layer" in
  component) dir="pkg/component/$name"; ci="$name" ;;
  fx)        name="fx${name#fx}"; dir="pkg/fx/$name"; ci="$name" ;;
  module)    dir="pkg/module/$name"; ci="module-$name" ;;
  *) echo "Unknown layer '$layer': expected component, fx, or module" >&2; exit 1 ;;
esac

if [ -e "$dir/go.mod" ]; then
  echo "$dir already exists" >&2
  exit 1
fi

go_version=$(awk '$1 == "go" {print $2; exit}' go.work)

mkdir -p "$dir"
(cd "$dir" && go mod init "$REPO_MODULE/$dir" && go mod edit -go="$go_version")
printf '// Package %s is a placeholder; describe the package here.\npackage %s\n' "$name" "$name" > "$dir/doc.go"
go work use "./$dir"

cat > ".github/workflows/$ci-ci.yml" <<EOF
name: "$ci-ci"

on:
  workflow_dispatch:
  push:
    branches:
      - "feat**"
      - "fix**"
      - "hotfix**"
      - "chore**"
      - "main"
    paths:
      - "$dir/**.go"
      - "$dir/go.mod"
      - "$dir/go.sum"
  pull_request:
    types:
      - opened
      - synchronize
      - reopened
    branches:
      - main
    paths:
      - "$dir/**.go"
      - "$dir/go.mod"
      - "$dir/go.sum"

jobs:
  ci:
    uses: ./.github/workflows/common-ci.yml
    secrets: inherit
    with:
      module: "$dir"
EOF

echo "Created $dir ($REPO_MODULE/$dir) and .github/workflows/$ci-ci.yml"

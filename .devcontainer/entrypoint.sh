#!/bin/bash
set -e  # Exit on any error

# Modules live under pkg/ and are joined by go.work, so there is nothing to
# download at startup. Run ./scripts/tidy.sh to sync dependencies.

# Execute the original command passed to the container
exec "$@"

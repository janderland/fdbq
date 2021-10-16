#!/usr/bin/env bash
set -exuo pipefail

# Change directory to repo root.
cd "${0%/*}"

# Connect to the test database then lint, build, & test the codebase.
docker compose run --rm build "/bin/sh" "-c" "./scripts/setup_database.sh && ./scripts/verify_codebase.sh"

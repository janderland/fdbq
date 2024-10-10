#!/usr/bin/env bash
set -euo pipefail

CLUSTER_FILE="${1:-}"

# Convert hostnames to IPs in the cluster file. Any
# parts of the cluster file of the form '{hostname}'
# will be converted to an IP address.
readonly REGEX='\{([^}]*)\}'
while [[ "$CLUSTER_FILE" =~ $REGEX ]]; do
  token="${BASH_REMATCH[0]}"
  host="${BASH_REMATCH[1]}"

  if [[ ${#host} -lt 1 ]]; then
    echo "invalid empty hostname '{}'"
    exit 1
  fi

  if ! ip="$(getent hosts "$host" | cut -d" " -f1)"; then
    echo "failed to get IP for host '$host'"
    exit 1
  fi

  CLUSTER_FILE="${CLUSTER_FILE/${token}/${ip}}"
done

echo "cluster file: $CLUSTER_FILE"
echo

# Shift the rest of the arguments up, if there are any.
shift || true

# Create the cluster file and run FQL with the
# remaining arguments.
echo "$CLUSTER_FILE" > /etc/foundationdb/fdb.cluster
/fql "$@"

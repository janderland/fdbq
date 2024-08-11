#!/usr/bin/env bash
set -eo pipefail


function print_help {
  cat << END
build.sh is a facade for docker compose. It runs a set of
optional tasks in the order specified below. This is the same
script used by CI/CD to build, test, and package FDBQ.

If the '--image build' flag is set then the script starts off by
running 'docker build' for the 'fdbq-build' docker image. The tag
is determined by the git tag/hash. This image is used to run the
'generate' and 'verify' tasks below.

If the '--generate' flag is set then the script checks if the
code generated by 'go generate ./...' is up to date.

If the '--verify' flag is set then the script builds, lints, and
tests the codebase. This task interacts with an FDB docker
container which is automatically started.

If the '--no-hado' flag is set then the 'verify' task above will
exclude linting the Dockerfile. This provides a workaround on
Arm Macs where Dockerfile linting is currently unsupported.

If the '--image fdbq' flag is set then the script runs 'docker
build' for the 'fdbq' docker image. The tag is determined by the
git tag/hash and the version of the FDB library specified in the
'.env' file.

If the '--run' flag is provided then all the args after this flag
are passed to an instance of the 'fdbq' docker image. Normally
this image expects a cluster file as the first argument but this
script takes care of starting an FDB cluster and providing the
cluster file as the first argument. Note that this is the same
FDB instance used by the 'verify' task.

  ./build.sh --run --write -q '/my/dir{"hi"}=nil'

After this, the script ends. If any of the requested tasks fail
then the script exits immediately.

Multiple image names can be specified on the '--image' flag by
separating them with commas.

  ./build.sh --image build,fdbq

When building Docker images, the dependencies of the Dockerfile
are specified in the '.env' file. When this file is changed,
you'll need to rebuild the docker images for the changes to take
effect.
END
}


# fail prints $1 to stderr and exits with code 1.

function fail {
  local RED='\033[0;31m' NO_COLOR='\033[0m'
  echo -e "${RED}ERR! ${1}${NO_COLOR}" >&2
  exit 1
}


# join_array joins the elements of the $2 array into a
# single string, placing $1 between each element.

function join_array {
  local sep="$1" out="$2"
  if shift 2; then
    for arg in "$@"; do
      out="${out}${sep}${arg}"
    done
  fi
  echo "$out"
}


# code_version returns the latest tag for the current
# Git commit. If there are no tags associated with
# the commit then the short hash is returned.

function code_version {
  local tag=""
  if tag="$(git describe --tags)"; then
    echo "$tag"
    return 0
  fi

  git rev-parse --short HEAD
}


# fdb_version returns the version of the FDB
# library specified by the env var FDB_VER.
# If FDB_VER is not defined then the .env
# file is read to obtain the version.

function fdb_version {
  if [[ -n "$FDB_VER" ]]; then
    echo "$FDB_VER"
    return 0
  fi

  local regex='FDB_VER=([^'$'\n'']*)'
  if ! [[ "$(cat .env)" =~ $regex ]]; then
    fail "Couldn't find FDB version in .env file."
  fi
  echo "${BASH_REMATCH[1]}"
}


# Change directory to repo root.

cd "${0%/*}"


# Parse the flags.

if [[ $# -eq 0 ]]; then
  print_help
  echo
  fail "At least one flag must be provided."
fi

while [[ $# -gt 0 ]]; do
  case $1 in
    --generate)
      VERIFY_GENERATION="x"
      shift 1
      ;;

    --verify)
      VERIFY_CODEBASE="x"
      shift 1
      ;;

    --no-hado)
      NO_HADO="x"
      shift 1
      ;;

    --image)
      for service in $(echo "$2" | tr "," "\n"); do
        case $service in
          build)
            IMAGE_BUILD="x"
            ;;
          fdbq)
            IMAGE_FDBQ="x"
            ;;
          *)
            fail "Invalid build target '$service'"
            ;;
        esac
      done
      shift 2
      ;;

    --help)
      print_help
      exit 0
      ;;

    --run)
      shift 1
      RUN="x"
      FDBQ_ARGS=("$@")
      shift $#
      ;;

    *)
      fail "Invalid flag '$1'"
  esac
done


# Build variables required by the docker compose command.

BUILD_TASKS=()

if [[ -n "$VERIFY_GENERATION" ]]; then
  BUILD_TASKS+=('./scripts/verify_generation.sh')
fi

if [[ -n "$VERIFY_CODEBASE" ]]; then
  BUILD_TASKS+=('./scripts/setup_database.sh')
  BUILD_TASKS+=("./scripts/verify_codebase.sh ${NO_HADO:+--no-hado}")
fi

BUILD_COMMAND="$(join_array ' && ' "${BUILD_TASKS[@]}")"
echo "BUILD_COMMAND=${BUILD_COMMAND}"
export BUILD_COMMAND

FDBQ_COMMAND=${FDBQ_ARGS[*]}
echo "FDBQ_COMMAND=${FDBQ_COMMAND}"
export FDBQ_COMMAND

DOCKER_TAG="$(code_version)_fdb.$(fdb_version)"
echo "DOCKER_TAG=${DOCKER_TAG}"
export DOCKER_TAG


# Run the requested commands.

if [[ -n "$IMAGE_BUILD" ]]; then
  (set -x; docker compose build build)
fi

if [[ -n "$BUILD_COMMAND" ]]; then
  (set -x; docker compose run build /bin/sh -c "$BUILD_COMMAND")
fi

if [[ -n "$IMAGE_FDBQ" ]]; then
  (set -x; docker compose build fdbq)
fi

if [[ -n "$RUN" ]]; then
  (set -x; docker compose run fdbq 'docker:docker@{fdb}:4500' "${FDBQ_ARGS[@]}")
fi

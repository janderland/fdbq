#!/usr/bin/env bash
set -eo pipefail


function print_help {
  echo "build.sh is a facade for 'docker compose'."
  echo "It runs a set of optional tasks in the order"
  echo "specified below. This is the same script used"
  echo "by CI/CD to build, test, and package FDBQ."
  echo
  echo "If the '--build build' flag is set then the"
  echo "script starts off by running 'docker build'"
  echo "for the 'fdbq-build' docker image. The tag"
  echo "is determined by the git hash. This image is"
  echo "used to run the 'generated' and 'verify'"
  echo "tasks below."
  echo
  echo "If the '--generated' flag is set then the"
  echo "script checks if the code generated by"
  echo "'go generate ./...' is up to date."
  echo
  echo "If the '--verify' flag is set then the script"
  echo "builds, lints, and tests the codebase. This task"
  echo "interacts with an FDB docker container which is"
  echo "started the first time this task is run."
  echo
  echo "If the '--build fdbq' flag is set then the script"
  echo "runs 'docker build' for the 'fdbq' docker image."
  echo "The tag is determined by the git tag/hash and the"
  echo "version of the FDB library specified in the '.env'"
  echo "file."
  echo
  echo "If the '--' flag is provided then all the args"
  echo "after this flag are passed to an instance of the"
  echo "'fdbq' docker image. Normally this image expects"
  echo "a cluster file as the first argument but this"
  echo "script takes care of starting an FDB cluster and"
  echo "providing the cluster file as the first argument."
  echo "Note that this is the same FDB instance used by"
  echo "the 'verify' task."
  echo
  echo "  ./docker.sh -- --write '/my/dir{\"hi\"}=nil'"
  echo
  echo "After this, the script ends. If any of the"
  echo "requested tasks fail then the script exits"
  echo "immediately."
  echo
  echo "Multiple image names can be specified on the"
  echo "'--build' flag by separating them with commas."
  echo
  echo "  ./docker.sh --build build,fdbq"
  echo
  echo "When building Docker images, the dependencies of"
  echo "the Dockerfile are specified in the '.env' file."
  echo "When this file is changed, you'll need to rebuild"
  echo "the docker images for the changes to take effect."
}


# fail print $1 to stderr and exits with code 1.

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


# escape_quotes adds an extra layer of single quotes
# around it's arguments. Any single quotes included
# in the arguments are escaped with backslashes. This
# is required because Docker interprets the CLI args
# through a shell before being passed to the fdbq
# container

function escape_quotes {
  out=()
  for arg in "$@"; do
    out+=("$(printf "'%s'" "${arg//'/\\'}")")
  done
  echo "${out[@]}"
}


# commit_hash returns the hash for the current
# Git commit.

function commit_hash {
  git rev-parse --short HEAD
}


# fdb_version returns the version of the FDB
# library specified in the .env file.

function fdb_version {
  local regex='FDB_LIB_URL=[^'$'\n'']*([0-9]+\.[0-9]+\.[0-9]+)'
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
    --generated)
      VERIFY_GENERATION="x"
      shift 1
      ;;

    --verify)
      VERIFY_CODEBASE="x"
      shift 1
      ;;

    --build)
      for service in $(echo "$2" | tr "," "\n"); do
        case $service in
          build)
            BUILD_BUILD_CONTAINER="x"
            ;;
          fdbq)
            BUILD_FDBQ_CONTAINER="x"
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

    --)
      shift 1
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
  BUILD_TASKS+=('./scripts/verify_codebase.sh')
fi

BUILD_COMMAND="$(join_array ' && ' "${BUILD_TASKS[@]}")"
echo "BUILD_COMMAND=${BUILD_COMMAND}"
export BUILD_COMMAND

BUILD_TAG="$(commit_hash)"
echo "BUILD_TAG=${BUILD_TAG}"
export BUILD_TAG

FDBQ_COMMAND="$(escape_quotes "${FDBQ_ARGS[@]}")"
echo "FDBQ_COMMAND=${FDBQ_COMMAND}"
export FDBQ_COMMAND

FDBQ_TAG="$(commit_hash)_fdb.$(fdb_version)"
echo "FDBQ_TAG=${FDBQ_TAG}"
export FDBQ_TAG


# Run the requested commands.

if [[ -n "$BUILD_BUILD_CONTAINER" ]]; then
  (set -x;
    docker compose build build
  )
fi

if [[ -n "$BUILD_COMMAND" ]]; then
  (set -x;
    docker compose up build --attach build --exit-code-from build
  )
fi

if [[ -n "$BUILD_FDBQ_CONTAINER" ]]; then
  (set -x;
    docker compose build fdbq
  )
fi

if [[ -n "$FDBQ_COMMAND" ]]; then
  (set -x;
    docker compose up fdbq --attach fdbq --exit-code-from fdbq
  )
fi

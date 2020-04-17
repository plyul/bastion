#!/bin/bash
set -euo pipefail

OUTPUT_BRANCH=0
OUTPUT_HASH=0

ARG=${1-""}
while [ "${ARG}" != "" ]; do
    case ${ARG} in
    -b | --branch)
        OUTPUT_BRANCH=1
    ;;
    -h | --hash)
        OUTPUT_HASH=1
    ;;
    *)
        echo "Wrong argument: ${ARG}"
        exit 1
    esac
    if shift
    then
        ARG=${1-""}
    else
        break
    fi
done

if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1
then
  echo "Not inside git repository"
  exit 1
fi

TAG=$(git describe --tags 2>/dev/null || true)
if [ -n "$TAG" ]
then
    VERSION=${TAG}
else
    if [ $OUTPUT_HASH -eq 1 ]
    then
        VERSION="$(git rev-parse --short HEAD)"
    else
        VERSION="latest"
    fi
fi

BRANCH=$(git symbolic-ref --short -q HEAD)
if [ -n "${BRANCH}" ] && [ ${OUTPUT_BRANCH} -eq 1 ]
then
    VERSION=${VERSION}-${BRANCH}
else
    VERSION=${VERSION}
fi

echo "${VERSION}"

#!/bin/bash
set -euo pipefail

export VERSION=$(./build/get_git_ref.sh)
export DOCKER_REPO=docker.example.com
DOCKER_USER=${DOCKER_USER:-""}
DOCKER_PASSWORD=${DOCKER_PASSWORD:-""}

if [ -z "${DOCKER_USER}" ] || [ -z "${DOCKER_PASSWORD}" ]
then
    echo "Docker credentils not set"
    exit 1
fi

docker-compose -f build/bastion-build.dockercompose build

docker login --username=${DOCKER_USER} --password=${DOCKER_PASSWORD} ${DOCKER_REPO}
docker push ${DOCKER_REPO}/bastion-server:${VERSION}
docker push ${DOCKER_REPO}/bastion-proxy:${VERSION}
docker logout ${DOCKER_REPO}

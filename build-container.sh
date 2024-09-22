#!/bin/bash

set -e

DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")

podman build -t myfans:latest -f Containerfile --platform linux/amd64,linux/arm64 dist
podman save --format docker-archive --multi-image-archive -o dist/myfans.docker myfans

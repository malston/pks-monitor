#!/bin/bash

set -eo pipefail

docker_image_name="${1:-"malston/pks-monitor"}"
docker_image_tag="${2:-"0.0.1"}"

docker build -t "${docker_image_name}:${docker_image_tag}" .

docker login
docker tag "${docker_image_name}:${docker_image_tag}" "${docker_image_name}:${docker_image_tag}"
docker push "${docker_image_name}:${docker_image_tag}"

#!/usr/bin/env bash
set -euo pipefail

image="${1:?usage: smoke-image.sh IMAGE}"
name="matchmark-smoke-$$"
volume="${name}-data"

trap 'docker rm -f "$name" >/dev/null 2>&1; docker volume rm "$volume" >/dev/null 2>&1' EXIT

docker volume create "$volume" >/dev/null
docker run -d --name "$name" -v "${volume}:/data" \
  -e ADMIN_USERNAME=smoke -e ADMIN_PASSWORD=smoke-test-password-123 "$image" >/dev/null

for _ in $(seq 1 30); do
  if docker exec "$name" wget -q -O /dev/null http://127.0.0.1:8080/; then
    docker exec "$name" test -s /data/matchmark.db
    echo "smoke test passed: $(docker exec "$name" id -u) opened /data/matchmark.db"
    exit 0
  fi
  if [ "$(docker inspect -f '{{.State.Running}}' "$name")" != "true" ]; then
    break
  fi
  sleep 1
done

docker logs "$name" >&2 || true
echo "smoke test failed" >&2
exit 1

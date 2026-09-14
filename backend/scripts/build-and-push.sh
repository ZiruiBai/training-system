#!/usr/bin/env bash
#
# Build and push the training-system image to GHCR.
#
# Run this from the repository root (the directory containing Dockerfile),
# on a machine that has the Docker CLI and a GHCR login:
#
#   echo "$GHCR_TOKEN" | docker login ghcr.io -u <github-username> --password-stdin
#   ./scripts/build-and-push.sh
#
# It tags the image with both `latest` and the exact commit SHA, so the
# security fix can always be pinned and rolled back to.
set -euo pipefail

IMAGE="ghcr.io/ziruibai/training-system"
DOCKERFILE="${DOCKERFILE:-Dockerfile}"
CONTEXT="${CONTEXT:-.}"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker CLI not found. Install Docker or build the image in CI." >&2
  exit 1
fi

# Resolve the commit SHA that will be baked into the tag.
if git rev-parse --git-dir >/dev/null 2>&1; then
  FULL_SHA="$(git rev-parse HEAD)"
  SHORT_SHA="$(echo "$FULL_SHA" | cut -c1-12)"
else
  echo "WARNING: not a git repository; falling back to a timestamp tag." >&2
  FULL_SHA="$(date -u +%Y%m%d%H%M%S)"
  SHORT_SHA="$FULL_SHA"
fi

echo "Image:      $IMAGE"
echo "Commit:     $FULL_SHA"
echo "Tags:"
echo "  $IMAGE:latest"
echo "  $IMAGE:sha-$FULL_SHA"
echo "  $IMAGE:sha-$SHORT_SHA"
echo

docker build \
  -f "$DOCKERFILE" \
  -t "$IMAGE:latest" \
  -t "$IMAGE:sha-$FULL_SHA" \
  -t "$IMAGE:sha-$SHORT_SHA" \
  "$CONTEXT"

echo
echo "Pushing..."
docker push "$IMAGE:latest"
docker push "$IMAGE:sha-$FULL_SHA"
docker push "$IMAGE:sha-$SHORT_SHA"

echo
echo "Done. Verify the login fix with:"
echo "  docker run --rm -p 8080:8080 \\"
echo "    -e APP_ENV=production \\"
echo "    -e DB_DRIVER=postgres \\"
echo "    -e DB_DSN='host=<pg> user=<u> password=<p> dbname=<db> sslmode=disable' \\"
echo "    -e SESSION_KEY='<32+ chars>' \\"
echo "    $IMAGE:sha-$SHORT_SHA"
echo
echo "  # wrong password must return 401:"
echo "  curl -i -X POST localhost:8080/api/v1/auth/login \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"Admin\",\"password\":\"wrong\"}'"
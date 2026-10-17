#!/usr/bin/env bash
# Builds the UI and a fresh Mailpit binary, then runs the Playwright tests inside Microsoft's Playwright
# image (which ships Chromium and its system libraries). Run from the host, not from inside a container.
#
#   e2e/run-docker.sh                      # all tests
#   e2e/run-docker.sh --project=webroot    # extra arguments are passed to `playwright test`
#   e2e/run-docker.sh --ui                 # Playwright UI mode, served on http://127.0.0.1:8080 (Ctrl+C to stop)
#                                          # open it in a browser, or in VS Code via "Simple Browser: Show"
#                                          # set E2E_UI_PORT to use a different port
set -euo pipefail

cd "$(dirname "$0")/.."

# the image tag must match the installed @playwright/test version
version=$(sed -n 's/.*"version": "\(.*\)".*/\1/p' node_modules/@playwright/test/package.json | head -1)
image="mcr.microsoft.com/playwright:v${version}-noble"

bindir=$(mktemp -d)
trap 'rm -rf "$bindir"' EXIT

# the UI is embedded in the binary, so rebuild it first to avoid testing stale code
npm run package
CGO_ENABLED=0 go build -o "$bindir/mailpit" .

docker_args=(--rm --init --ipc=host -u "$(id -u):$(id -g)"
	-e HOME=/tmp -e MAILPIT_BIN=/bin-e2e/mailpit
	-v "$bindir":/bin-e2e:ro -v "$PWD":/build -w /build)

if [ "${1:-}" != "--ui" ]; then
	docker run "${docker_args[@]}" "$image" npx playwright test "$@"
	exit
fi

port="${E2E_UI_PORT:-8080}"
name="mailpit-e2e-ui-$$"
# bind to 0.0.0.0 inside the container so the port mapping works, but publish only on the loopback interface:
# UI mode can run test code, so it must not be reachable from the network
docker_args+=(--name "$name" -p "127.0.0.1:${port}:${port}")
echo "Playwright UI mode will be served on http://127.0.0.1:${port} (the page shows a one-off token after redirecting)"

# UI mode ignores SIGINT inside the container, so stop the container ourselves on Ctrl+C. The container runs in the
# background so that bash handles the trap immediately instead of waiting for the foreground command to finish.
trap 'docker stop -t 3 "$name" >/dev/null 2>&1 || true' INT TERM
docker run "${docker_args[@]}" "$image" npx playwright test --ui --ui-host=0.0.0.0 --ui-port="$port" "${@:2}" &
child=$!
wait "$child" || true
# the first wait returns early when a signal is trapped, so wait again until the container has really gone
wait "$child" 2>/dev/null || true

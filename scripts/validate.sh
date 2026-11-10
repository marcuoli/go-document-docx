#!/usr/bin/env bash
set -euo pipefail
project="${DEVELOPMENT_ROOT:-${RUNNER_TEMP:?Set DEVELOPMENT_ROOT or RUNNER_TEMP}/development}/go-document-docx"
run="$(date -u +%Y%m%d-%H%M%S)"
export TMPDIR="$project/tmp/validate-$run" GOTMPDIR="$project/tmp/validate-$run/go-build"
export TEMP="$TMPDIR" TMP="$TMPDIR" GOCACHE="$project/caches/go-build" GOMODCACHE="$project/caches/go-modules"
export GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=readonly CGO_ENABLED=0
output="$project/artifacts/runs/validate-$run"
mkdir -p "$TMPDIR" "$GOTMPDIR" "$GOCACHE" "$GOMODCACHE" "$output"
cd "$(dirname "$0")/.."
test "$(go env GOVERSION)" = go1.27.3
test -z "$(gofmt -l .)"
go test -count=1 ./...
go vet ./...
go mod verify
go build ./...
for arg in "$@"; do
 case "$arg" in
 --race) CGO_ENABLED=1 go test -race -count=1 ./... ;;
 --scan) go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./... ;;
 *) printf 'Unknown option: %s\n' "$arg" >&2; exit 2 ;;
 esac
done
go build -trimpath -buildvcs=false -o "$output/report-example" ./examples/report
go version -m "$output/report-example" > "$output/build-metadata.txt"
go env -json GOVERSION GOOS GOARCH CGO_ENABLED GOWORK GOFLAGS > "$output/build-settings.json"
go list -m -json all > "$output/modules.json"
printf 'Validated source and native build. Evidence: %s\n' "$output"

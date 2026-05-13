#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
IMAGE="${IMAGE:-docker.io/library/ubuntu:24.04}"
ARCH="${ARCH:-$(uname -m)}"
MARKER="${MARKER:-GOFVML_LIVE_MARKER_2026}"
WORKDIR_IN_CONTAINER="${WORKDIR_IN_CONTAINER:-/work}"
HOST_OUTDIR="${HOST_OUTDIR:-${REPO_DIR}/validation-output/apple-container-process}"
CONTAINER_OUTDIR="${WORKDIR_IN_CONTAINER}/validation-output/apple-container-process"

case "${ARCH}" in
	arm64|aarch64)
		ARCH="arm64"
		GOFVML_BIN="dist/gofvml-linux-arm64"
		;;
	amd64|x86_64)
		ARCH="amd64"
		GOFVML_BIN="dist/gofvml-linux-amd64"
		;;
	*)
		echo "unsupported ARCH=${ARCH}; use arm64 or amd64" >&2
		exit 2
		;;
esac

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "missing required command: $1" >&2
		exit 127
	fi
}

require_command container
require_command go

echo "[live] checking Apple container runtime"
container system status >/dev/null

echo "[live] building Linux GOFVML artifacts"
mkdir -p "${REPO_DIR}/dist" "${HOST_OUTDIR}"
(
	cd "${REPO_DIR}"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o dist/gofvml-linux-arm64 ./cmd/gofvml
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/gofvml-linux-amd64 ./cmd/gofvml
)

echo "[live] running process acquisition scenario in Apple container (${ARCH})"
container run \
	--rm \
	--arch "${ARCH}" \
	--cap-add SYS_PTRACE \
	--volume "${REPO_DIR}:${WORKDIR_IN_CONTAINER}" \
	"${IMAGE}" \
	/bin/bash -lc "
set -euo pipefail
cd ${WORKDIR_IN_CONTAINER}
mkdir -p ${CONTAINER_OUTDIR}
rm -f ${CONTAINER_OUTDIR}/process.gofvml ${CONTAINER_OUTDIR}/strict.gofvml
rm -f ${CONTAINER_OUTDIR}/process.stdout ${CONTAINER_OUTDIR}/process.stderr
rm -f ${CONTAINER_OUTDIR}/strict.stdout ${CONTAINER_OUTDIR}/strict.stderr

env ${MARKER}=present /bin/sleep 120 &
target=\$!
trap 'kill \$target >/dev/null 2>&1 || true' EXIT
sleep 1
kill -0 \"\$target\"

set +e
./${GOFVML_BIN} process -progress -pid \"\$target\" -output ${CONTAINER_OUTDIR}/process.gofvml \
	> ${CONTAINER_OUTDIR}/process.stdout \
	2> ${CONTAINER_OUTDIR}/process.stderr
process_status=\$?
./${GOFVML_BIN} process -strict -progress -pid \"\$target\" -output ${CONTAINER_OUTDIR}/strict.gofvml \
	> ${CONTAINER_OUTDIR}/strict.stdout \
	2> ${CONTAINER_OUTDIR}/strict.stderr
strict_status=\$?
set -e

cat ${CONTAINER_OUTDIR}/process.stderr
cat ${CONTAINER_OUTDIR}/process.stdout
cat ${CONTAINER_OUTDIR}/strict.stderr
cat ${CONTAINER_OUTDIR}/strict.stdout

test \"\$process_status\" -eq 0
test \"\$strict_status\" -ne 0
test -s ${CONTAINER_OUTDIR}/process.gofvml
grep -a '${MARKER}' ${CONTAINER_OUTDIR}/process.gofvml >/dev/null
grep -q 'mapping read failed' ${CONTAINER_OUTDIR}/process.stdout
grep -q 'Error: read mapping chunk' ${CONTAINER_OUTDIR}/strict.stderr
echo '[live] marker recovered from process artifact'
echo '[live] non-strict warning and strict failure semantics verified'
"

echo "[live] outputs written to ${HOST_OUTDIR}"

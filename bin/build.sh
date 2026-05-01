#!/usr/bin/env bash
# https://sharats.me/posts/shell-script-best-practices/

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
    set -o xtrace
fi

if [[ "${1-}" =~ ^-*h(elp)?$ ]]; then
    echo 'Usage: ./build.sh'
    exit
fi

DIR=$(dirname "$0")
pushd "$DIR/.." &>/dev/null

go build -ldflags "-X github.com/atdrendel/bikemark/internal/version.Version=dev \
  -X github.com/atdrendel/bikemark/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/atdrendel/bikemark/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .

popd &>/dev/null

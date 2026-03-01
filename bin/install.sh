#!/usr/bin/env bash
# https://sharats.me/posts/shell-script-best-practices/

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
    set -o xtrace
fi

if [[ "${1-}" =~ ^-*h(elp)?$ ]]; then
    echo 'Usage: ./install.sh'
    exit
fi

DIR=$(dirname "$0")
pushd "$DIR/.." &>/dev/null

go build -ldflags "-X github.com/atdrendel/mdtobike/internal/version.Version=dev \
  -X github.com/atdrendel/mdtobike/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/atdrendel/mdtobike/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .

cp mdtobike /usr/local/bin/

echo "$(ls /usr/local/bin/mdtobike)"

popd &>/dev/null

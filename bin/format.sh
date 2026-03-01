#!/usr/bin/env bash
# https://sharats.me/posts/shell-script-best-practices/

set -o errexit
set -o nounset
set -o pipefail
if [[ "${TRACE-0}" == "1" ]]; then
    set -o xtrace
fi

if [[ "${1-}" =~ ^-*h(elp)?$ ]]; then
    echo 'Usage: ./format.sh'
    exit
fi

DIR=$(dirname "$0")
pushd "$DIR/.." &>/dev/null

gofmt -w .

popd &>/dev/null

#!/bin/bash
set -eu

# Logging helper functions
info()
{
    echo "INFO:" "$@" 1>&2
}

warn()
{
    echo "WARN:" "$@" 1>&2
}

error()
{
    echo "ERROR:" "$@" 1>&2
    exit 255
}

if ! [[ "$0" =~ "scripts/protoc-gen" ]]; then
	error "must be run from repository root"
fi

if ! [[ $(protoc --version) =~ 3.14.0 ]]; then
	error "could not find protoc 3.14.0, is it installed + in PATH?"
fi

GOGOPROTO_ROOT="${GOPATH}/src/github.com/gogo/protobuf"
GOGOPROTO_PATH="${GOGOPROTO_ROOT}:${GOGOPROTO_ROOT}/protobuf"

DIRS="pkg/types"

for dir in ${DIRS}; do
    pushd ${dir} 1>/dev/null 2>&1

        protoc --gogofaster_out=plugins=grpc:. -I=. \
            -I="${GOGOPROTO_PATH}" \
            *.proto

        if ! [[ $? ]]; then
            info "failed to generate $(pwd)/*.proto"
        else
            info "generated $(pwd)/*.proto"
        fi

    popd 1>/dev/null 2>&1
done


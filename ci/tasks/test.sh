#!/usr/bin/env bash

set -euo pipefail

export GOPATH=$(pwd)
export PATH="${GOPATH}/bin:$PATH"

go get github.com/onsi/ginkgo/ginkgo

cd src/github.com/pivotal-cf/service-instance-reaper

ginkgo -r -cover

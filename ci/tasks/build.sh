#!/usr/bin/env bash

set -euo pipefail

export GOPATH=$(pwd)

cd src/github.com/pivotal-cf/service-instance-reaper

go build -v

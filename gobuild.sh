#!/bin/bash

set -eux

cd ../
export GOPATH=\"$(pwd)/.gobuild\"
SRCDIR=${GOPATH}/src/github.com/smallstep/cli
[ -d ${GOPATH} ] && rm -rf ${GOPATH}
mkdir -p ${GOPATH}/{src,pkg,bin}
mkdir -p ${SRCDIR}
cp -r cli-0.0.1-rc.1/* ${SRCDIR}
cd ${SRCDIR}
make build

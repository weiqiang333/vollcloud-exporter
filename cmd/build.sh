#!/usr/bin/env bash
set -xe

export GOARCH=amd64
export GOOS=linux
export GCCGO=gc

version=$1

if [ -z $version ]; then
    version=v0.1
fi

go build -o vollcloud-exporter vollcloud-exporter.go

tar -zcvf vollcloud-exporter-linux-amd64-${version}.tar.gz \
  vollcloud-exporter config/vollcloud-exporter.yaml config/vollcloud-exporter.service README.md

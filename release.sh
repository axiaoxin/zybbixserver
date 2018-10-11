#! /usr/bin/env bash
mkdir ./zybbixserver
go build -o ./zybbixserver/zybbixserver
cp zybbixserver.json supervisor.conf monitems.json ./zybbixserver
tar czf zybbixserver.tar.gz ./zybbixserver && rm -rf ./zybbixserver

#! /usr/bin/env bash
rm -rf ./zybbixserver ./zybbixserver.tar.gz
mkdir ./zybbixserver
go build -o ./zybbixserver/zybbixserver
cp README.md zybbixserver.json supervisor.conf ./zybbixserver
tar czf zybbixserver.tar.gz ./zybbixserver && rm -rf ./zybbixserver

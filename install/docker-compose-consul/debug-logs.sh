#!/bin/zsh

curl -X POST localhost:19000/logging\?level=debug
curl -X PUT localhost:9092/logging -d '{"level":"debug"}'
curl -X PUT localhost:9091/logging -d '{"level":"debug"}'


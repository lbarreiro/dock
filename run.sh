#!/bin/bash

set -e

go build -o dock ./cmd/dock

pkill -x -u "$USER" dock 2>/dev/null || true

nohup ./dock >/tmp/dock.log 2>&1 &

sleep 1

tail -5 /tmp/dock.log

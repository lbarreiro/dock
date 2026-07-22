#!/bin/bash

set -e

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " ⚓ Dock Development"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

echo "✓ Stop"
pkill -x -u pi dock 2>/dev/null || true

sleep 1

echo "✓ Build"
cd "$(dirname "$0")"
go build -o dock ./cmd/dock

echo "✓ Start"
nohup ./dock >/tmp/dock.log 2>&1 </dev/null &

disown

sleep 1

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Listening on :8081"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

tail -5 /tmp/dock.log

echo
echo "✅ Ready"
echo

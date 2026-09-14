#!/usr/bin/env bash
# Demo reset: wipe the SQLite DB and restart the backend so all demo data
# returns to the canonical seed state — learning tasks pending, daily
# quizzes re-takeable, coaching recommendations cleared, Erin's stage-1
# quiz (score 65) preserved so her growth radar keeps 3+ real dimensions.
# Usage: bash /workspace/scripts/demo-reset.sh
set -e

pkill -f 'training-serve[r]' || true
sleep 1

rm -f /workspace/backend/var/training.db

cd /workspace/backend
setsid nohup /tmp/training-server > /tmp/demo-server.log 2>&1 < /dev/null &
sleep 3

curl -s -o /dev/null -w 'backend=%{http_code}\n' http://localhost:8080/health/live
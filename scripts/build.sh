#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

(cd web && npm install && npm run build)
go build -o tbg-rse ./cmd/tbg-rse
echo "Built ./tbg-rse"

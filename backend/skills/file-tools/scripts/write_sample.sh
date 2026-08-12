#!/usr/bin/env bash
# Write a sample text file. Usage: write_sample.sh <content> <out-path>
set -euo pipefail
content="${1:-Hello}"
out="${2:-out/sample.txt}"
mkdir -p "$(dirname "$out")"
echo "$content" > "$out"
echo "wrote $out"

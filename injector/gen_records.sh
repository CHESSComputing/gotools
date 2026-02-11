#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<EOF
Usage: $(basename "$0") -i INPUT -p PATTERN -n COUNT

Generate multiple JSON files from a template by replacing a pattern
with a unique ID in each output file.

Options:
  -i INPUT     Input JSON template file
  -p PATTERN   Pattern to replace (e.g. PAT)
  -n COUNT     Number of files to generate
  -h           Show this help message

Example:
  $(basename "$0") -i input.json -p PAT -n 3

This will generate:
  PAT-<UUID>.json (3 files)
EOF
}

# --- Parse options ---
INPUT=""
PATTERN=""
COUNT=""

while getopts ":i:p:n:h" opt; do
  case "$opt" in
    i) INPUT="$OPTARG" ;;
    p) PATTERN="$OPTARG" ;;
    n) COUNT="$OPTARG" ;;
    h)
      usage
      exit 0
      ;;
    \?)
      echo "Error: Invalid option -$OPTARG" >&2
      usage
      exit 1
      ;;
    :)
      echo "Error: Option -$OPTARG requires an argument" >&2
      usage
      exit 1
      ;;
  esac
done

# --- Validation ---
if [[ -z "$INPUT" || -z "$PATTERN" || -z "$COUNT" ]]; then
  echo "Error: missing required arguments" >&2
  usage
  exit 1
fi

if [[ ! -f "$INPUT" ]]; then
  echo "Error: input file '$INPUT' does not exist" >&2
  exit 1
fi

if ! [[ "$COUNT" =~ ^[0-9]+$ ]] || [[ "$COUNT" -le 0 ]]; then
  echo "Error: COUNT must be a positive integer" >&2
  exit 1
fi

ODIR=$PWD/data

if [ -d $ODIR ]; then
  rm -rf $ODIR
fi
mkdir -p $ODIR

for ((i=1; i<=COUNT; i++)); do
  UUID=$(uuidgen)
  OUT="${PATTERN}-${UUID}.json"

  sed "s/${PATTERN}/${PATTERN}-${UUID}/g" "$INPUT" > "$ODIR/$OUT"

  echo "Generated $ODIR/$OUT"
done


#!/bin/bash
# Run N parallel crossval dump jobs, each playing M games. Each job writes its
# games to OUT_DIR/run_NN/ and its stdout/stderr to OUT_DIR/run_NN.log.
#
# Usage: scripts/run_dumps.sh [num_dumps] [games_per_dump] [out_dir]
#   defaults: 10 100 dumps
#
# Env overrides:
#   TURNS=20         max turns per game
#   XMAGE_DIR=../xmage
#   GAME_TIMEOUT=60  per-game wall-clock seconds (passed to crossval)
#
# Note: each parallel job spawns its own JVM at -Xmx2g. 10 jobs ~= 20GB heap
# headroom (most won't actually allocate that much).

set -euo pipefail

cd "$(dirname "$0")/.."

NUM_DUMPS="${1:-10}"
GAMES_PER_DUMP="${2:-100}"
OUT_DIR="${3:-dumps}"
TURNS="${TURNS:-40}"
XMAGE_DIR="${XMAGE_DIR:-../xmage}"
GAME_TIMEOUT="${GAME_TIMEOUT:-120}"

mkdir -p "$OUT_DIR"

# Pre-build the maven classpath once. Otherwise each parallel JVM startup
# would race on `mvn dependency:build-classpath` and on writes under ~/.m2.
if [[ -z "${CROSSVAL_CLASSPATH_FILE:-}" ]]; then
    CP_FILE="$OUT_DIR/classpath.txt"
    if [[ ! -s "$CP_FILE" ]]; then
        echo "Building XMage classpath -> $CP_FILE"
        CP_FILE_ABS="$(cd "$(dirname "$CP_FILE")" && pwd)/$(basename "$CP_FILE")"
        (cd "$XMAGE_DIR" && mvn -q dependency:build-classpath -pl Mage.Tests \
            -DincludeScope=test -Dmdep.outputFile="$CP_FILE_ABS")
    fi
    export CROSSVAL_CLASSPATH_FILE="$(cd "$(dirname "$CP_FILE")" && pwd)/$(basename "$CP_FILE")"
fi

echo "Building crossval binary..."
BIN="$OUT_DIR/crossval"
go build -o "$BIN" ./cmd/crossval/

BASE_SEED="$(date +%s)"
echo "Launching $NUM_DUMPS dump(s), $GAMES_PER_DUMP games each, base seed $BASE_SEED"

PIDS=()
for ((i=1; i<=NUM_DUMPS; i++)); do
    RUN_DIR=$(printf "%s/run_%02d" "$OUT_DIR" "$i")
    LOG_FILE="${RUN_DIR}.log"
    SEED=$((BASE_SEED + i))
    mkdir -p "$RUN_DIR"
    "$BIN" \
        --xmage "$XMAGE_DIR" \
        --dump "$RUN_DIR" \
        --games "$GAMES_PER_DUMP" \
        --turns "$TURNS" \
        --seed "$SEED" \
        --game-timeout "$GAME_TIMEOUT" \
        > "$LOG_FILE" 2>&1 &
    PIDS+=($!)
    echo "  run_$(printf '%02d' "$i") pid=$! seed=$SEED log=$LOG_FILE"
done

echo "Waiting for ${#PIDS[@]} jobs..."

FAILED=0
for idx in "${!PIDS[@]}"; do
    pid="${PIDS[$idx]}"
    n=$((idx + 1))
    if wait "$pid"; then
        echo "  run_$(printf '%02d' "$n") (pid=$pid) ok"
    else
        echo "  run_$(printf '%02d' "$n") (pid=$pid) FAILED — see ${OUT_DIR}/run_$(printf '%02d' "$n").log"
        FAILED=$((FAILED + 1))
    fi
done

TOTAL_FILES="$(find "$OUT_DIR" -mindepth 2 -name 'game_*.jsonl' | wc -l | tr -d ' ')"
echo "Done. Failed jobs: $FAILED/${#PIDS[@]}. Total game files in $OUT_DIR: $TOTAL_FILES"

exit "$FAILED"

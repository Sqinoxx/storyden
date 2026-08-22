#!/bin/bash
# Repeatedly resumes --phase ocr. Each invocation is self-bounded (20s hard
# timeout per asset, clean process exit on abandonment to reclaim memory) so
# restarting after every abandonment is safe. Stops when a pass completes
# without hitting a new pathological asset, or after MAX_RUNS as a backstop.
set -u
MAX_RUNS=60
LOG=import-work/ocr-loop.log
RUN_OUT=import-work/ocr-loop-run.tmp
: > "$LOG"

for i in $(seq 1 "$MAX_RUNS"); do
  echo "=== run $i $(date) ===" >> "$LOG"
  OCR_MAX_FILE_SIZE_MB=60 OCR_PDF_DPI=150 go run ./cmd/import --phase ocr > "$RUN_OUT" 2>&1
  code=$?
  cat "$RUN_OUT" >> "$LOG"

  if [ "$code" -ne 0 ]; then
    echo "=== run $i exited $code, stopping loop ===" >> "$LOG"
    rm -f "$RUN_OUT"
    exit "$code"
  fi

  if ! grep -q "exceeded its extraction time budget" "$RUN_OUT"; then
    echo "=== run $i completed a full pass with no new abandonment, done ===" >> "$LOG"
    rm -f "$RUN_OUT"
    exit 0
  fi
done

rm -f "$RUN_OUT"
echo "=== hit MAX_RUNS=$MAX_RUNS, stopping ===" >> "$LOG"
exit 2

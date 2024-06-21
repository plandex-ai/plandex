#!/bin/bash

OUT="${PLANDEX_DEV_CLI_OUT_DIR:-/usr/local/bin}"
NAME="${PLANDEX_DEV_CLI_NAME:-plandex-dev}"
ALIAS="${PLANDEX_DEV_CLI_ALIAS:-pdxd}"

# Double quote to prevent globbing and word splitting.
sudo go build -o "$NAME" &&
    sudo rm -f "$OUT"/"$NAME" &&
    sudo cp "$NAME" "$OUT"/"$NAME" &&
    sudo ln -sf "$OUT"/"$NAME" "$OUT"/"$ALIAS" &&
    echo built "$NAME" cli and added "$ALIAS" alias to "$OUT"

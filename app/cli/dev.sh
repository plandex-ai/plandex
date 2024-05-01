#!/bin/bash

OUT="${PLANDEX_DEV_CLI_OUT_DIR:-/usr/local/bin}"
NAME="${PLANDEX_DEV_CLI_NAME:-plandex-dev}"
ALIAS="${PLANDEX_DEV_CLI_ALIAS:-pdxd}"

go build -o $NAME && \
rm -f $OUT/$NAME && \
cp $NAME $OUT/$NAME && \
ln -sf $OUT/$NAME $OUT/$ALIAS && \
echo built $NAME cli and added $ALIAS alias to $OUT

#!/bin/bash

OUT="${PLANDEX_OUT_DIR:-/usr/local/bin}"

go build && \
rm -f $OUT/plandex && \
cp plandex $OUT/plandex && \
ln -sf $OUT/plandex $OUT/pdx && \
echo built 'plandex' cli and added 'pdx' alias to $OUT

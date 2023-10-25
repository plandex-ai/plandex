#!/bin/bash

go build && \
rm -f /usr/local/bin/plandex && \
cp plandex /usr/local/bin/plandex && \
ln -sf /usr/local/bin/plandex /usr/local/bin/pdx && \
echo built 'plandex' cli and added 'pdx' alias to /usr/local/bin
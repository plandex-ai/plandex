#!/bin/bash

terminate() {
  pkill -f 'plandex-server' # Assuming plandex-server is the name of your process
  kill -TERM "$pid1" 2>/dev/null
  kill -TERM "$pid2" 2>/dev/null
}

trap terminate SIGTERM SIGINT

cd app

(cd cli && ./dev.sh)

reflex -r '^(cli|shared)/.*\.(go|mod|sum)$' -- sh -c 'cd cli && ./dev.sh' &
pid1=$!

reflex -r '^(server|shared)/.*\.(go|mod|sum)$' -s -- sh -c 'cd server && go build && ./plandex-server' &
pid2=$!

wait $pid1
wait $pid2
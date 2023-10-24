#!/bin/bash

# Function to handle termination signals, to clean up the background processes
terminate() {
    kill -TERM "$pid1" 2>/dev/null
    kill -TERM "$pid2" 2>/dev/null
}

# Trap termination signals and call the terminate function
trap terminate SIGTERM SIGINT

# rebuild cli when there are changes
reflex -r '^(cli|shared)/.*\.go$' -- sh -c 'cd cli && go build && rm /usr/local/bin/pdx && cp plandex /usr/local/bin/pdx && echo rebuilt plandex cli' &
pid1=$!

# rebuild and restart server when there are changes
reflex -r '^(server|shared)/.*\.go$' -s -- sh -c 'cd server && go build && ./plandex-server' &
pid2=$!

# Wait for both background processes to finish
wait $pid1
wait $pid2

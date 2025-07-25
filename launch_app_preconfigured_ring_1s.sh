#!/bin/bash

fast_cleanup() {
    pkill -f '^./main (verifier|user_exp|user_linear|sensor|verifier)'
    exit 0
}

trap fast_cleanup SIGINT

declare -a PIDS=()

go build -o main main.go

sleeping_time=1

./main verifier 1 &
PIDS+=($!)
sleep $sleeping_time

./main user_exp 2 1 &
PIDS+=($!)
sleep $sleeping_time

./main user_linear 3 1 2 & 
PIDS+=($!)
sleep $sleeping_time

./main sensor 4 3 1 & 
PIDS+=($!)
sleep $sleeping_time

./main verifier 5 1 4  &
PIDS+=($!)

# --- Function to clean up all child processes on script exit or Ctrl-C ---
cleanup() {
    echo -e "\n--- Ctrl-C detected! Stopping all 'main' instances... ---"
    # Iterate through the stored PIDs
    for pid in "${PIDS[@]}"; do
        # Check if the process with this PID is still running
        if kill -0 "$pid" 2>/dev/null; then
            # If it's running, send a SIGTERM signal to gracefully terminate it.
            # If it doesn't terminate, a subsequent SIGKILL might be needed (not implemented here for simplicity).
            kill "$pid"
            echo "Sent termination signal to process $pid"
        else
            echo "Process $pid already terminated or not found."
        fi
    done
    echo "--- All 'main' instances stopped. Exiting. ---"
    exit 0 # Exit the launcher script
}

# --- Set up the trap for SIGINT (Ctrl-C) ---
# When SIGINT is received, the 'cleanup' function will be executed.
trap cleanup SIGINT

while true; do
    sleep 1 # Sleep for 1 second to avoid busy-waiting
done



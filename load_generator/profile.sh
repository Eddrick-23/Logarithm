#!/bin/bash
set -e

SAMPLE_DURATION="${1:?Usage: $0 <sample_duration_seconds> <warmup_duration_seconds> <warmup_rest_seconds> <config_path>}"
WARMUP_DURATION="${2:?Usage: $0 <sample_duration_seconds> <warmup_duration_seconds> <warmup_rest_seconds> <config_path>}"
WARMUP_REST="${3:?Usage: $0 <sample_duration_seconds> <warmup_duration_seconds> <warmup_rest_seconds> <config_path>}"
CONFIG_PATH="${4:?Usage: $0 <sample_duration_seconds> <warmup_duration_seconds> <warmup_rest_seconds> <config_path>}"

# default to local host if no argument provided
TARGET_IP="${5:-localhost}"

DIR="./benchmark_results/$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
mkdir -p "$DIR"

BASE_INGESTER="http://${TARGET_IP}:6060/debug/pprof"
BASE_WORKER="http://${TARGET_IP}:6061/debug/pprof"

echo "Taking baseline snapshots..."
curl -so "$DIR/ingester_heap_baseline.pb.gz" "$BASE_INGESTER/heap" &
curl -so "$DIR/worker_heap_baseline.pb.gz"   "$BASE_WORKER/heap" &
wait

echo "Building load generator..."
mkdir -p bin && go build -o bin/main .

echo "Starting load generator..."
./bin/main \
  -config "$CONFIG_PATH" \
  -o "$DIR" \
  -duration "$SAMPLE_DURATION" \
  -warmup "$WARMUP_DURATION" \
  -rest "$WARMUP_REST" &
LOAD_GEN_PID=$!

echo "Waiting through warmup phase (${WARMUP_DURATION}s + ${WARMUP_REST}s rest)..."

for ((i=1; i<=$WARMUP_DURATION+WARMUP_REST; i++)); do
  if ! kill -0 $LOAD_GEN_PID 2>/dev/null; then
      wait $LOAD_GEN_PID || EXIT_CODE=$?
      echo "Error: Load generated process died early with exit code ${EXIT_CODE:-0}, aborting."
      exit 1
  fi
  sleep 1
done

# track nats queue depth
echo "timestamp,pending_messages" > "$DIR/queue_depth.csv"
while true; do
    PENDING=$(curl -s "http://${TARGET_IP}:8222/jsz?stream=LOGS&consumers=true" | \
        jq -r '.account_details[].stream_detail[]? | select(.name=="LOGS") | .consumer_detail[]? | select(.name=="worker") | .num_pending // 0')
    
    echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ"),${PENDING:-0}" >> "$DIR/queue_depth.csv"
    sleep 1
done &
QUEUE_PID=$!

trap 'kill $QUEUE_PID 2>/dev/null; exit' INT TERM EXIT # ensure cleanup if script exits early
disown $QUEUE_PID

echo "Starting ${SAMPLE_DURATION}s profiling window (load test is now running)..."
curl -so "$DIR/ingester_cpu.pb.gz"   "$BASE_INGESTER/profile?seconds=$SAMPLE_DURATION" &
CURL_PID1=$!
curl -so "$DIR/worker_cpu.pb.gz"     "$BASE_WORKER/profile?seconds=$SAMPLE_DURATION" &
CURL_PID2=$!
curl -so "$DIR/ingester_trace.out"   "$BASE_INGESTER/trace?seconds=$SAMPLE_DURATION" &
CURL_PID3=$!
curl -so "$DIR/worker_trace.out"     "$BASE_WORKER/trace?seconds=$SAMPLE_DURATION" &
CURL_PID4=$!
curl -so "$DIR/ingester_block.pb.gz" "$BASE_INGESTER/block?seconds=$SAMPLE_DURATION" &
CURL_PID5=$!
curl -so "$DIR/worker_block.pb.gz"   "$BASE_WORKER/block?seconds=$SAMPLE_DURATION" &
CURL_PID6=$!
curl -so "$DIR/ingester_mutex.pb.gz" "$BASE_INGESTER/mutex?seconds=$SAMPLE_DURATION" &
CURL_PID7=$!
curl -so "$DIR/worker_mutex.pb.gz"   "$BASE_WORKER/mutex?seconds=$SAMPLE_DURATION" &
CURL_PID8=$!
wait $CURL_PID1 $CURL_PID2 $CURL_PID3 $CURL_PID4 $CURL_PID5 $CURL_PID6 $CURL_PID7 $CURL_PID8  

echo "Waiting for load generator to finish..."
wait $LOAD_GEN_PID

echo "Letting system stabilise post benchmark..."
sleep 10

kill $QUEUE_PID 2>/dev/null

echo "Taking post-load snapshots..."
curl -so "$DIR/ingester_heap_after.pb.gz"    "$BASE_INGESTER/heap" &
curl -so "$DIR/ingester_allocs_after.pb.gz"  "$BASE_INGESTER/allocs" &
curl -so "$DIR/worker_heap_after.pb.gz"      "$BASE_WORKER/heap" &
curl -so "$DIR/worker_allocs_after.pb.gz"    "$BASE_WORKER/allocs" &
curl -so "$DIR/ingester_goroutine.pb.gz"     "$BASE_INGESTER/goroutine" &
curl -so "$DIR/worker_goroutine.pb.gz"       "$BASE_WORKER/goroutine" &
wait

echo "Done. Profiles saved to $DIR"
echo ""
echo "Analyse with:"
echo ""
echo "# CPU profiles"
echo "  go tool pprof -http=:8080 $DIR/ingester_cpu.pb.gz"
echo "  go tool pprof -http=:8080 $DIR/worker_cpu.pb.gz"
echo ""
echo "# Traces"
echo "  go tool trace $DIR/ingester_trace.out"
echo "  go tool trace $DIR/worker_trace.out"
echo ""
echo "# block"
echo "  go tool pprof -http=:8080 $DIR/ingester_block.pb.gz"
echo "  go tool pprof -http=:8080 $DIR/worker_block.pb.gz"
echo ""
echo "# mutexes"
echo "  go tool pprof -http=:8080 $DIR/ingester_mutex.pb.gz"
echo "  go tool pprof -http=:8080 $DIR/worker_mutex.pb.gz"
echo ""
echo "# Heap diffs (baseline vs after)"
echo "  go tool pprof -http=:8080 -diff_base=$DIR/ingester_heap_baseline.pb.gz $DIR/ingester_heap_after.pb.gz"
echo "  go tool pprof -http=:8080 -diff_base=$DIR/worker_heap_baseline.pb.gz $DIR/worker_heap_after.pb.gz"
echo ""
echo "# Allocations (GC pressure)"
echo "  go tool pprof -http=:8080 $DIR/ingester_allocs_after.pb.gz"
echo "  go tool pprof -http=:8080 $DIR/worker_allocs_after.pb.gz"
echo ""
echo "# Goroutines (leak check)"
echo "  go tool pprof -http=:8080 $DIR/ingester_goroutine.pb.gz"
echo "  go tool pprof -http=:8080 $DIR/worker_goroutine.pb.gz"

#!/usr/bin/env bash
set -euo pipefail

echo "[Test] Running unit tests (go test)"
go test ./... -v

echo "[Test] Building binary"
go build -o blockchain-app

echo "[Test] Starting blockchain node in background"
# Choose the correct binary depending on OS
if [ -f ./blockchain-app.exe ]; then
  BIN_TO_RUN=./blockchain-app.exe
else
  BIN_TO_RUN=./blockchain-app
fi
${BIN_TO_RUN} > blockchain.log 2>&1 &
APP_PID=$!
echo "[Test] PID: ${APP_PID}"

# Wait for the node to become ready before performing API calls
wait_for_node() {
  # Try both common ports (8080, 8081) in order
  for port in 8080 8081; do
    for i in $(seq 1 60); do
      if curl -sSf http://localhost:${port}/blocks >/dev/null; then
        PORT=${port}
        return 0
      fi
      sleep 1
    done
  done
  return 1
}

if ! wait_for_node; then
  echo "[Error] Node did not become ready in time"
  kill ${APP_PID} 2>/dev/null || true
  exit 1
fi

echo "[Test] Basic API checks. You can adjust endpoints as needed."
curl -s http://localhost:8080/blocks | head -n 5 || true

echo "[Test] End-to-end quick test: create wallet, send a tx, check blocks"
PORT=${PORT:-8080}
WALLET_JSON=$(curl -s -X POST http://localhost:${PORT}/wallets -H "Content-Type: application/json" -d '{}')
WALLET=$(echo "$WALLET_JSON" | sed -n 's/.*"address"\s*:\s*"\([^"}]*\)".*/\1/p')
if [ -z "$WALLET" ]; then
  echo "[Error] Failed to parse wallet address from response: $WALLET_JSON"
  kill ${APP_PID} 2>/dev/null || true
  exit 1
fi
echo "Wallet: ${WALLET}"
TX=$(curl -s -X POST http://localhost:8080/transactions -H "Content-Type: application/json" -d '{"to":"'$WALLET'","amount":1,"payload":"test"}')
echo "TX response: ${TX}"

echo "Waiting for block to be forged..."
sleep 15

echo "Balances:"
cURL_BAL=$(curl -s http://localhost:8080/wallets/${WALLET}/balance 2>/dev/null || echo '{}')
echo "${cURL_BAL}"

echo "Stopping node (PID ${APP_PID})"
kill ${APP_PID} || true
wait ${APP_PID} 2>/dev/null || true

echo "Done. Read blockchain.log for details."

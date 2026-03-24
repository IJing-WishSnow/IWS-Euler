#!/bin/sh
set -e

DEPLOYER_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
OPERATOR="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
RPC="http://localhost:8545"

# Start Anvil in background
anvil --host 0.0.0.0 &
ANVIL_PID=$!

# Wait until Anvil is accepting connections
echo "[init] Waiting for Anvil..."
until cast block-number --rpc-url $RPC > /dev/null 2>&1; do
  sleep 0.3
done
echo "[init] Anvil ready"

# Deploy MockERC20 (nonce 0 → 0x5FbDB2315678afecb367f032d93F642f64180aa3)
MOCK_ARGS=$(cast abi-encode "constructor(string,string,uint8)" "Mock USDT" "USDT" 6 | sed 's/0x//')
MOCK_INIT="0x$(cat /contracts/mock.bin)${MOCK_ARGS}"
cast send --private-key $DEPLOYER_KEY --rpc-url $RPC --create $MOCK_INIT > /dev/null
echo "[init] MockERC20 deployed: 0x5FbDB2315678afecb367f032d93F642f64180aa3"

# Deploy IWSSettlement (nonce 1 → 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512)
SETTLEMENT_ARGS=$(cast abi-encode "constructor(address)" $OPERATOR | sed 's/0x//')
SETTLEMENT_INIT="0x$(cat /contracts/settlement.bin)${SETTLEMENT_ARGS}"
cast send --private-key $DEPLOYER_KEY --rpc-url $RPC --create $SETTLEMENT_INIT > /dev/null
echo "[init] IWSSettlement deployed: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"

echo "[init] Contract deployment complete. Anvil running."

# Keep Anvil in foreground
wait $ANVIL_PID

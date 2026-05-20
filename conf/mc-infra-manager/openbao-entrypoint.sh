#!/bin/sh
# OpenBao auto init/unseal entrypoint for development environments.
# Unsealing key is stored as plain text in /openbao/data/init.txt — NOT for production use.

CONFIG_FILE=/openbao/config/config.hcl
INIT_FILE=/openbao/data/init.txt
DESIRED_TOKEN="${MC_INFRA_MANAGER_OPENBAO_VAULT_TOKEN:-dev-only-token}"
export BAO_ADDR="http://127.0.0.1:${BAO_PORT:-8200}"

# 1) Start bao server in background
bao server -config="$CONFIG_FILE" &
BAO_PID=$!

# 2) Wait until the listener responds (exit 0=unsealed, exit 2=sealed, exit 1=unreachable)
echo "[openbao-entrypoint] waiting for listener on ${BAO_ADDR}..."
while true; do
  bao status >/dev/null 2>&1
  ec=$?
  if [ "$ec" = "0" ] || [ "$ec" = "2" ]; then
    break
  fi
  sleep 1
done

# 3) Initialize on first run (plain text format — easy to grep without jq)
if [ ! -f "$INIT_FILE" ]; then
  echo "[openbao-entrypoint] first run: initializing (1-of-1 key)..."
  bao operator init -key-shares=1 -key-threshold=1 > "$INIT_FILE"
  chmod 600 "$INIT_FILE"
fi

# 4) Extract unseal key and root token from plain text output
UNSEAL_KEY=$(grep "^Unseal Key 1:" "$INIT_FILE" | awk '{print $NF}')
INIT_ROOT_TOKEN=$(grep "^Initial Root Token:" "$INIT_FILE" | awk '{print $NF}')

if [ -z "$UNSEAL_KEY" ] || [ -z "$INIT_ROOT_TOKEN" ]; then
  echo "[openbao-entrypoint] ERROR: failed to parse init file. Contents:"
  cat "$INIT_FILE"
  exit 1
fi

# 5) Unseal
bao operator unseal "$UNSEAL_KEY" >/dev/null
echo "[openbao-entrypoint] unsealed."

# 6) Create well-known service token so mc-infra-manager can connect with a fixed token ID
export BAO_TOKEN="$INIT_ROOT_TOKEN"
if ! bao token lookup "$DESIRED_TOKEN" >/dev/null 2>&1; then
  echo "[openbao-entrypoint] creating service token id=${DESIRED_TOKEN}..."
  bao token create -id="$DESIRED_TOKEN" -policy=root -orphan -ttl=0 >/dev/null
fi

echo "[openbao-entrypoint] openbao ready."
wait "$BAO_PID"

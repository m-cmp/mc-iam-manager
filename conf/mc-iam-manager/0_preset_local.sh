#!/bin/bash

# Local PC HTTP mode setup script for MC-IAM-Manager
# - No certificate generation (plain HTTP)
# - Updates .env PUBLIC_HOST variables from https:// to http://
# - Conditionally adds /etc/hosts entry for mciam.local
# - Generates HTTP-only nginx.conf from nginx.template.local.conf

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

ENV_FILE="${PROJECT_ROOT}/.env"
IAM_ENV_FILE="${SCRIPT_DIR}/.env"

TEMPLATE_FILE="${SCRIPT_DIR}/nginx.template.local.conf"
OUTPUT_FILE="${PROJECT_ROOT}/container-volume/mc-iam-manager/nginx/nginx.conf"

NGINX_DIR="${PROJECT_ROOT}/container-volume/mc-iam-manager/nginx"

echo "PROJECT_ROOT: $PROJECT_ROOT"

# =============================================================================
# 1. .env file check
# =============================================================================

if [ ! -f "$ENV_FILE" ]; then
    echo "Error: .env file not found: $ENV_FILE"
    exit 1
fi

if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "Error: nginx template file not found: $TEMPLATE_FILE"
    exit 1
fi

# =============================================================================
# 2. Load environment variables
# =============================================================================

# Use line-by-line parsing instead of source to safely handle unquoted multi-word
# values (e.g. cron schedules like "0 30 0,6 * * ?") that docker compose .env allows.
echo "Loading environment variables..."

while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" =~ ^[[:space:]]*$ ]] && continue
    if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
        declare "${BASH_REMATCH[1]}=${BASH_REMATCH[2]}"
    fi
done < "$ENV_FILE"

# =============================================================================
# 3. Validate required variables
# =============================================================================

echo "Validating required environment variables..."

REQUIRED_VARS=(
    "MC_IAM_MANAGER_PUBLIC_DOMAIN"
    "MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
    "MC_IAM_MANAGER_DATABASE_NAME"
    "MC_IAM_MANAGER_DATABASE_USER"
    "MC_IAM_MANAGER_DATABASE_PASSWORD"
    "MC_IAM_MANAGER_DATABASE_HOST"
    "MC_IAM_MANAGER_PORT"
)

MISSING_VARS=()
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -gt 0 ]; then
    echo "❌ Error: The following required environment variables are not set:"
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
    exit 1
fi

if [ -z "$MC_IAM_MANAGER_KEYCLOAK_PORT" ]; then
    MC_IAM_MANAGER_KEYCLOAK_PORT=8080
fi

echo "✅ All required environment variables loaded."
echo "  PUBLIC_DOMAIN: $MC_IAM_MANAGER_PUBLIC_DOMAIN"
echo "  KEYCLOAK_DOMAIN: $MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
echo "  MC_IAM_MANAGER_PORT: $MC_IAM_MANAGER_PORT"

# =============================================================================
# 4. Rewrite PUBLIC_HOST variables from https:// to http:// in .env files
# =============================================================================

_sedi() {
    if [[ "$(uname)" == "Darwin" ]]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

rewrite_https_to_http() {
    local env_file="$1"
    if [ ! -f "$env_file" ]; then
        return 0
    fi
    echo "Rewriting https:// → http:// in ${env_file##*/conf/docker/}..."

    local vars=(
        "MC_IAM_MANAGER_PUBLIC_HOST"
        "MC_OBSERVABILITY_GRAFANA_PUBLIC_HOST"
        "MC_COST_OPTIMIZER_FE_PUBLIC_HOST"
        "MC_WORKFLOW_MANAGER_PUBLIC_HOST"
        "MC_DATA_MANAGER_PUBLIC_HOST"
        "MC_APPLICATION_MANAGER_PUBLIC_HOST"
    )

    for var in "${vars[@]}"; do
        if grep -qE "^${var}=https://" "$env_file"; then
            _sedi "s|^${var}=https://|${var}=http://|" "$env_file"
            echo "  ✓ ${var}: https:// → http://"
        fi
    done
}

rewrite_https_to_http "$ENV_FILE"
rewrite_https_to_http "$IAM_ENV_FILE"

# =============================================================================
# 5. /etc/hosts entry for mciam.local (skip for localhost/127.0.0.1)
# =============================================================================

DOMAIN="$MC_IAM_MANAGER_PUBLIC_DOMAIN"

if [ "$DOMAIN" = "localhost" ] || [ "$DOMAIN" = "127.0.0.1" ]; then
    echo "✓ Domain is $DOMAIN — skipping /etc/hosts modification."
else
    HOSTS_FILE="/etc/hosts"
    echo "Checking $DOMAIN in $HOSTS_FILE..."

    if grep -qE "^[[:space:]]*127\.0\.0\.1[[:space:]]+${DOMAIN}[[:space:]]*$" "$HOSTS_FILE"; then
        echo "✓ $DOMAIN already exists in $HOSTS_FILE. Skipping."
    else
        echo "Removing any existing entries for $DOMAIN..."
        _sedi "/[[:space:]]*127\.0\.0\.1[[:space:]]\+${DOMAIN}[[:space:]]*$/d" "$HOSTS_FILE" 2>/dev/null || true

        echo "Adding 127.0.0.1 $DOMAIN to $HOSTS_FILE..."
        if sudo -n true 2>/dev/null; then
            echo "127.0.0.1 $DOMAIN" | sudo tee -a "$HOSTS_FILE" > /dev/null
            echo "✓ $DOMAIN added to $HOSTS_FILE."
        elif echo "127.0.0.1 $DOMAIN" >> "$HOSTS_FILE" 2>/dev/null; then
            echo "✓ $DOMAIN added to $HOSTS_FILE."
        else
            echo "⚠️  Failed to add to $HOSTS_FILE — add manually:"
            echo "    echo '127.0.0.1 $DOMAIN' | sudo tee -a $HOSTS_FILE"
        fi
    fi
fi

# =============================================================================
# 6. Create nginx output directory
# =============================================================================

echo "Creating nginx directory..."

CURRENT_USER=$(whoami)
CURRENT_GROUP=$(id -gn)

if ! mkdir -p "$NGINX_DIR" 2>/dev/null; then
    echo "❌ Error: Cannot create $NGINX_DIR"
    echo "   A previous Docker run likely left root-owned files in the parent directory."
    echo "   Please run the cleanup script first:"
    echo "       cd ${SCRIPT_DIR}/../../bin && ./cleanAll.sh"
    exit 1
fi

if [ ! -w "$NGINX_DIR" ]; then
    echo "❌ Error: $NGINX_DIR exists but is not writable by ${CURRENT_USER}."
    echo "   Please run the cleanup script first:"
    echo "       cd ${SCRIPT_DIR}/../../bin && ./cleanAll.sh"
    exit 1
fi

echo "✓ nginx directory ready: $NGINX_DIR"

# =============================================================================
# 7. Generate nginx.conf from HTTP template
# =============================================================================

echo "Generating nginx configuration file..."
echo "  Template: $TEMPLATE_FILE"
echo "  Output:   $OUTPUT_FILE"

if [ -d "$OUTPUT_FILE" ]; then
    rm -rf "$OUTPUT_FILE"
fi

# Reload env after rewrite so substitutions use updated http:// values
while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" =~ ^[[:space:]]*$ ]] && continue
    if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
        declare "${BASH_REMATCH[1]}=${BASH_REMATCH[2]}"
    fi
done < "$ENV_FILE"

if [ -n "$MC_IAM_MANAGER_PUBLIC_DOMAIN" ] && [ -n "$MC_IAM_MANAGER_KEYCLOAK_PORT" ]; then
    sed -e "s/\${MC_IAM_MANAGER_DOMAIN}/$MC_IAM_MANAGER_DOMAIN/g" \
        -e "s/\${MC_IAM_MANAGER_PORT}/$MC_IAM_MANAGER_PORT/g" \
        -e "s/\${MC_IAM_MANAGER_PUBLIC_DOMAIN}/$MC_IAM_MANAGER_PUBLIC_DOMAIN/g" \
        -e "s/\${MC_IAM_MANAGER_KEYCLOAK_DOMAIN}/$MC_IAM_MANAGER_KEYCLOAK_DOMAIN/g" \
        -e "s/\${MC_IAM_MANAGER_KEYCLOAK_PORT}/$MC_IAM_MANAGER_KEYCLOAK_PORT/g" \
        -e "s/\${MC_OBSERVABILITY_GRAFANA_PROXY_PORT}/$MC_OBSERVABILITY_GRAFANA_PROXY_PORT/g" \
        -e "s/\${MC_COST_OPTIMIZER_FE_PROXY_PORT}/$MC_COST_OPTIMIZER_FE_PROXY_PORT/g" \
        -e "s/\${MC_COST_OPTIMIZER_BE_PORT}/$MC_COST_OPTIMIZER_BE_PORT/g" \
        -e "s/\${MC_COST_OPTIMIZER_ALARM_PORT}/$MC_COST_OPTIMIZER_ALARM_PORT/g" \
        -e "s/\${MC_WORKFLOW_MANAGER_PROXY_PORT}/$MC_WORKFLOW_MANAGER_PROXY_PORT/g" \
        -e "s/\${MC_DATA_MANAGER_PROXY_PORT}/$MC_DATA_MANAGER_PROXY_PORT/g" \
        -e "s/\${MC_APPLICATION_MANAGER_PROXY_PORT}/$MC_APPLICATION_MANAGER_PROXY_PORT/g" \
        "$TEMPLATE_FILE" > "$OUTPUT_FILE"
    echo "✓ nginx.conf generated (HTTP mode)"
else
    echo "Warning: Required variables not set — copying template as-is."
    cp "$TEMPLATE_FILE" "$OUTPUT_FILE"
fi

echo ""
echo "=== Generated nginx.conf ==="
cat "$OUTPUT_FILE"

echo ""
echo "=================================================="
echo "✓ Local HTTP mode configuration completed."
echo ""
echo "  Domain : http://${MC_IAM_MANAGER_PUBLIC_DOMAIN}"
echo "  Mode   : plain HTTP (no TLS)"
echo ""
echo "Now you can run: ./mcc infra run"
echo "=================================================="

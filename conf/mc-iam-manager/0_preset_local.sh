#!/bin/bash

# localhost (plain HTTP) preset — no certs, no /etc/hosts modification

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "PROJECT_ROOT: $PROJECT_ROOT"

ENV_FILE="${PROJECT_ROOT}/.env"

if [ ! -f "$ENV_FILE" ]; then
    echo "Error: .env file not found: $ENV_FILE"
    exit 1
fi

source "$ENV_FILE"

CURRENT_USER=$(whoami)
CURRENT_GROUP=$(id -gn)

NGINX_DIR="${PROJECT_ROOT}/container-volume/mc-iam-manager/nginx"
mkdir -p "$NGINX_DIR" || { echo "Error: Failed to create $NGINX_DIR"; exit 1; }
chown -R "${CURRENT_USER}:${CURRENT_GROUP}" "${PROJECT_ROOT}/container-volume/mc-iam-manager"
echo "✓ Container volume directory created"

MC_IAM_MANAGER_PORT="${MC_IAM_MANAGER_PORT:-5005}"
MC_IAM_MANAGER_KEYCLOAK_PORT="${MC_IAM_MANAGER_KEYCLOAK_PORT:-8080}"
MC_OBSERVABILITY_GRAFANA_PROXY_PORT="${MC_OBSERVABILITY_GRAFANA_PROXY_PORT:-3010}"
MC_COST_OPTIMIZER_FE_PROXY_PORT="${MC_COST_OPTIMIZER_FE_PROXY_PORT:-3011}"
MC_COST_OPTIMIZER_FE_PORT="${MC_COST_OPTIMIZER_FE_PORT:-7780}"

OUTPUT_FILE="${NGINX_DIR}/nginx.conf"

cat > "$OUTPUT_FILE" << NGINX_EOF
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;

events {
    worker_connections 768;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

    gzip on;

    server {
        listen 80;
        server_name localhost 127.0.0.1;

        location /nginx-health {
            access_log off;
            return 200 "nginx is healthy\n";
            add_header Content-Type text/plain;
        }

        location /health {
            proxy_pass http://mc-iam-manager:${MC_IAM_MANAGER_PORT}/readyz;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_connect_timeout 10s;
            proxy_send_timeout 10s;
            proxy_read_timeout 10s;
        }

        location /auth/ {
            proxy_pass http://mc-iam-manager-kc:${MC_IAM_MANAGER_KEYCLOAK_PORT}/auth/;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_set_header X-Forwarded-Host \$host;
            proxy_set_header X-Forwarded-Server \$host;
            proxy_hide_header X-Frame-Options;
            add_header X-Frame-Options "SAMEORIGIN" always;
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }

        location / {
            proxy_pass http://mc-iam-manager:${MC_IAM_MANAGER_PORT};
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_set_header X-Forwarded-Host \$host;
            proxy_set_header X-Forwarded-Server \$host;
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }
    }

    server {
        listen 3001;
        server_name localhost 127.0.0.1;

        location / {
            resolver 127.0.0.11 valid=10s;
            set \$upstream_console mc-web-console-front;
            proxy_pass http://\$upstream_console:3001\$request_uri;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }
    }

    server {
        listen ${MC_OBSERVABILITY_GRAFANA_PROXY_PORT};
        server_name localhost 127.0.0.1;

        location / {
            resolver 127.0.0.11 valid=10s;
            set \$upstream_grafana mc-observability-grafana;
            proxy_pass http://\$upstream_grafana:3000;
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }
    }

    server {
        listen ${MC_COST_OPTIMIZER_FE_PROXY_PORT};
        server_name localhost 127.0.0.1;

        location / {
            resolver 127.0.0.11 valid=10s;
            set \$upstream_cost_fe mc-cost-optimizer-fe;
            proxy_pass http://\$upstream_cost_fe:${MC_COST_OPTIMIZER_FE_PORT};
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_connect_timeout 60s;
            proxy_send_timeout 60s;
            proxy_read_timeout 60s;
        }
    }
}
NGINX_EOF

echo "✓ Plain HTTP nginx.conf generated: $OUTPUT_FILE"
echo ""
echo "=== 생성된 nginx.conf ==="
cat "$OUTPUT_FILE"

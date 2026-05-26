#!/bin/bash

# MC-IAM-Manager Mode Configuration Script

# =============================================================================
# Usage Function
# =============================================================================
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -m, --mode <MODE>           IAM mode selection (dev|prod)"
    echo "                              dev:  Developer mode with self-signed certificate (Mode A)"
    echo "                              prod: Production mode with Let's Encrypt certificate (Mode B)"
    echo "  -d, --domain <DOMAIN>       Public domain or IP"
    echo "                              dev (local PC):   localhost (default) — plain HTTP, no certs, no hosts change"
    echo "                              dev (local PC):   mciam.local — plain HTTP (needs /etc/hosts entry)"
    echo "                              dev (remote VM):  VM public IP (e.g. 1.2.3.4) — HTTPS self-signed"
    echo "                              prod:             real FQDN required (e.g. iam.example.com)"
    echo "  -r, --run <RUN_MODE>        Service run mode (log|background|skip)"
    echo "                              log:        Run with log mode"
    echo "                              background: Run in background with monitoring"
    echo "                              skip:       Skip execution"
    echo "  -h, --help                  Display this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -m dev -r background                    # Local PC: default domain (localhost, plain HTTP)"
    echo "  $0 -m dev -d 1.2.3.4 -r background         # Remote VM: use public IP"
    echo "  $0 -m prod -d iam.example.com -r background # Remote VM: use real domain + Let's Encrypt"
    echo "  $0                                         # Interactive mode"
    exit 1
}

# =============================================================================
# Parameter Parsing
# =============================================================================
IAM_MODE=""
IAM_DOMAIN=""
RUN_MODE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--mode)
            IAM_MODE="$2"
            shift 2
            ;;
        -d|--domain)
            IAM_DOMAIN="$2"
            shift 2
            ;;
        -r|--run)
            RUN_MODE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Parameter Validation
if [ -n "$IAM_MODE" ] && [ "$IAM_MODE" != "dev" ] && [ "$IAM_MODE" != "prod" ]; then
    echo "Error: Invalid mode. Please use 'dev' or 'prod'."
    usage
fi

if [ -n "$RUN_MODE" ] && [ "$RUN_MODE" != "log" ] && [ "$RUN_MODE" != "background" ] && [ "$RUN_MODE" != "skip" ]; then
    echo "Error: Invalid run mode. Please use 'log', 'background', or 'skip'."
    usage
fi

# =============================================================================
# Container List Definition (User Configurable)
# =============================================================================

# Expected running containers (defined in docker-compose.yaml)
EXPECTED_CONTAINERS=(
    "mc-infra-connector"
    "mc-infra-manager"
    "mc-infra-manager-etcd"
    "mc-infra-manager-postgres"
    "mc-infra-manager-openbao"
    "mc-iam-manager"
    "mc-iam-manager-db"
    "mc-iam-manager-kc"
    "mc-iam-manager-nginx"
    # "mc-iam-manager-post-initial"  # Container that exits after execution
    "mc-web-console-db"
    "mc-web-console-api"
    "mc-web-console-front"
)

# Containers without Health Check (treated as successful when in Up state)
NO_HEALTH_CHECK_CONTAINERS=(
    "mc-iam-manager-nginx"
    "mc-infra-manager-openbao"
)

# =============================================================================

# Save current directory at script start
ORIGINAL_DIR="$(pwd)"

# =============================================================================
# IAM Mode Selection
# =============================================================================

# If mode is not specified via parameter, select interactively
if [ -z "$IAM_MODE" ]; then
    echo "=========================================="
    echo "MC-IAM-Manager Configuration Mode Selection"
    echo "=========================================="
    echo ""
    echo "MC-IAM-Manager can be configured in two modes:"
    echo ""
    echo "[Developer Mode - Local Authentication]"
    echo "  - localhost (default): plain HTTP, no certificates, no /etc/hosts change required"
    echo "  - IP/domain input: HTTPS with self-signed certificate"
    echo "  - Optimized for local development environment"
    echo "  - Quick setup and testing"
    echo ""
    echo "[Production Mode - CA Authentication]"
    echo "  - Uses Let's Encrypt CA certificates"
    echo "  - For use with real domains"
    echo "  - HTTPS based on security certificates"
    echo "  - Suitable for production environments"
    echo ""
    echo "=========================================="

    while true; do
        echo -n "Which mode would you like to configure? (1: Developer Mode, 2: Production Mode): "
        read -r choice

        case $choice in
            1)
                IAM_MODE="dev"
                break
                ;;
            2)
                IAM_MODE="prod"
                break
                ;;
            *)
                echo "Invalid selection. Please enter 1 or 2."
                ;;
        esac
    done
fi

# =============================================================================
# .env Bootstrap
# =============================================================================

PROJECT_ROOT_ABS="$(cd "$ORIGINAL_DIR" && pwd)"

ensure_env_file() {
    local setup_file="$1"
    local env_file="$2"
    if [ ! -f "$env_file" ]; then
        if [ -f "$setup_file" ]; then
            cp "$setup_file" "$env_file"
            echo "✓ Created $(basename "$env_file") from $(basename "$setup_file")"
        else
            echo "Error: $setup_file not found."
            exit 1
        fi
    fi
}

sync_missing_env_vars() {
    local setup_file="$1"
    local env_file="$2"

    if [ ! -f "$setup_file" ] || [ ! -f "$env_file" ]; then
        return 0
    fi

    local tmpfile
    tmpfile=$(mktemp)

    while IFS= read -r line; do
        _key="${line%%=*}"
        if ! grep -qE "^${_key}=" "$env_file"; then
            printf '%s\n' "$line" >> "$tmpfile"
        fi
    done < <(grep -E '^[A-Z_][A-Z0-9_]*=' "$setup_file")

    if [ -s "$tmpfile" ]; then
        local rel="${env_file##*/mc-iam-manager/}"
        {
            printf '\n'
            printf '# === Synced from %s by installAll.sh on %s ===\n' \
                "$(basename "$setup_file")" "$(date -Iseconds)"
            cat "$tmpfile"
        } >> "$env_file"
        echo "✓ Synced $(wc -l < "$tmpfile") missing var(s) into ${rel}"
    fi
    rm -f "$tmpfile"
}

ensure_env_file "$PROJECT_ROOT_ABS/.env.setup" "$PROJECT_ROOT_ABS/.env"

sync_missing_env_vars "$PROJECT_ROOT_ABS/.env.setup" "$PROJECT_ROOT_ABS/.env"

# =============================================================================
# Domain Configuration
# =============================================================================

if [ -z "$IAM_DOMAIN" ]; then
    echo ""
    echo "=========================================="
    echo "Public Domain Configuration"
    echo "=========================================="
    if [ "$IAM_MODE" = "dev" ]; then
        echo ""
        echo "  [Local PC - HTTP]   Just press Enter to use 'localhost' (plain HTTP, no certs)."
        echo "                      No /etc/hosts modification required."
        echo "                      Or enter 'mciam.local' for a named local domain"
        echo "                      (requires 127.0.0.1 mciam.local in /etc/hosts)."
        echo ""
        echo "  [Remote VM - HTTPS] Enter the VM's public IP (e.g. 43.202.200.215)."
        echo "                      Self-signed certificate will be issued for the IP."
        echo ""
        echo "  [With Domain]       Use Production Mode (-m prod) for Let's Encrypt cert."
        echo ""
        echo -n "Enter domain or IP [localhost]: "
        read -r IAM_DOMAIN
        IAM_DOMAIN="${IAM_DOMAIN:-localhost}"
    else
        echo ""
        echo "Mode B: Let's Encrypt certificate will be issued for this domain."
        echo "The domain must be a real FQDN with valid DNS pointing to this server."
        echo ""
        while [ -z "$IAM_DOMAIN" ]; do
            echo -n "Enter public FQDN (e.g. iam.example.com): "
            read -r IAM_DOMAIN
            if [ -z "$IAM_DOMAIN" ]; then
                echo "Domain is required for Production Mode. Please enter a valid FQDN."
            fi
        done
    fi
fi

echo ""
echo "Using domain: $IAM_DOMAIN"

apply_domain() {
    local env_file="$1"
    local domain="$2"
    sed -i "s|^MC_IAM_MANAGER_PUBLIC_DOMAIN=.*|MC_IAM_MANAGER_PUBLIC_DOMAIN=${domain}|" "$env_file"
    echo "✓ Set MC_IAM_MANAGER_PUBLIC_DOMAIN=${domain} in $(basename "$env_file")"
}

apply_domain "$PROJECT_ROOT_ABS/.env" "$IAM_DOMAIN"

# =============================================================================
# Process selected mode
case $IAM_MODE in
    dev)
        echo ""
        cd "$PROJECT_ROOT_ABS/conf/mc-iam-manager/" || {
            echo "Error: Cannot find mc-iam-manager directory."
            cd "$ORIGINAL_DIR"
            exit 1
        }

        if [ "$IAM_DOMAIN" = "localhost" ] || [ "$IAM_DOMAIN" = "127.0.0.1" ] || [ "$IAM_DOMAIN" = "mciam.local" ]; then
            echo "Local PC mode ($IAM_DOMAIN) — configuring plain HTTP, no certificates."
            echo ""

            if [ -f "0_preset_local.sh" ]; then
                chmod +x 0_preset_local.sh
                ./0_preset_local.sh
                if [ $? -eq 0 ]; then
                    echo ""
                    echo "✓ Local HTTP mode configuration completed."
                else
                    echo ""
                    echo "❌ Error occurred during local HTTP mode configuration."
                    cd "$ORIGINAL_DIR"
                    exit 1
                fi
            else
                echo "Error: Cannot find 0_preset_local.sh file."
                cd "$ORIGINAL_DIR"
                exit 1
            fi
        else
            echo "You have selected Developer Mode - Local Authentication."
            echo "Generating self-signed certificate and configuring local environment..."
            echo ""

            if [ -f "0_preset_dev.sh" ]; then
                chmod +x 0_preset_dev.sh
                ./0_preset_dev.sh
                if [ $? -eq 0 ]; then
                    echo ""
                    echo "✓ Developer mode configuration completed."
                else
                    echo ""
                    echo "❌ Error occurred during developer mode configuration."
                    cd "$ORIGINAL_DIR"
                    exit 1
                fi
            else
                echo "Error: Cannot find 0_preset_dev.sh file."
                cd "$ORIGINAL_DIR"
                exit 1
            fi
        fi
        ;;
    prod)
        echo ""
        echo "You have selected Production Mode - CA Authentication."
        echo "Generating Let's Encrypt certificate and configuring production environment..."
        echo ""

        cd "$PROJECT_ROOT_ABS" || {
            echo "Error: Cannot return to project root."
            exit 1
        }

        # Step 1: Start nginx in HTTP-only mode so certbot can serve ACME challenge
        echo "Step 1: Starting nginx (HTTP-only) for ACME challenge..."
        _NGINX_CONF_DIR="$PROJECT_ROOT_ABS/container-volume/mc-iam-manager/nginx"
        _NGINX_CONF="$_NGINX_CONF_DIR/nginx.conf"
        _NGINX_CONF_SSL_BAK="$_NGINX_CONF_DIR/nginx.conf.ssl_bak"
        _CERTBOT_WWW="$PROJECT_ROOT_ABS/container-volume/certbot/www"
        mkdir -p "$_NGINX_CONF_DIR" "$_CERTBOT_WWW"

        # Write HTTP-only nginx.conf (no SSL, just ACME challenge)
        _DOMAIN=$(grep -m1 "^MC_IAM_MANAGER_PUBLIC_DOMAIN=" "$PROJECT_ROOT_ABS/.env" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
        cat > "$_NGINX_CONF" <<HTTPONLY_EOF
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;
events { worker_connections 768; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    server {
        listen 80;
        server_name ${_DOMAIN};
        location /.well-known/acme-challenge/ { root /var/www/certbot; }
        location / { return 200 "cert pending\n"; add_header Content-Type text/plain; }
    }
}
HTTPONLY_EOF

        docker compose up -d mc-iam-manager-nginx
        echo "Waiting for nginx to be ready..."
        for i in $(seq 1 15); do
            if docker compose exec -T mc-iam-manager-nginx nginx -t >/dev/null 2>&1; then
                echo "✓ nginx is ready."
                break
            fi
            sleep 2
        done

        # Step 2: Generate Let's Encrypt certificate via certbot webroot
        echo ""
        echo "Step 2: Generating Let's Encrypt certificate..."

        docker compose -f "$PROJECT_ROOT_ABS/docker-compose.cert.yaml" --env-file "$PROJECT_ROOT_ABS/.env" up
        if [ $? -eq 0 ]; then
            echo "✓ Certificate generation completed."
            # Reclaim ownership of volume directories created by the certbot container (runs as root)
            if command -v sudo >/dev/null 2>&1; then
                sudo chown -R "$USER:$USER" "$PROJECT_ROOT_ABS/container-volume"
            fi
        else
            echo "❌ Error occurred during certificate generation."
            docker compose stop mc-iam-manager-nginx
            exit 1
        fi

        echo ""
        echo "Step 3: Configuring production mode (SSL nginx.conf)..."

        # Execute production mode script to generate SSL nginx.conf
        cd "$PROJECT_ROOT_ABS/conf/mc-iam-manager/" || {
            echo "Error: Cannot find mc-iam-manager directory."
            cd "$ORIGINAL_DIR"
            exit 1
        }

        if [ -f "0_preset_prod.sh" ]; then
            chmod +x 0_preset_prod.sh
            ./0_preset_prod.sh
            if [ $? -eq 0 ]; then
                echo ""
                echo "✓ Production mode configuration completed."
            else
                echo ""
                echo "❌ Error occurred during production mode configuration."
                cd "$ORIGINAL_DIR"
                exit 1
            fi
        else
            echo "Error: Cannot find 0_preset_prod.sh file."
            cd "$ORIGINAL_DIR"
            exit 1
        fi

        # Reload nginx to pick up SSL config
        cd "$PROJECT_ROOT_ABS" || { cd "$ORIGINAL_DIR"; exit 1; }
        echo "Reloading nginx with SSL configuration..."
        docker compose restart mc-iam-manager-nginx
        ;;
esac

# Return to original directory after all mode configurations
cd "$ORIGINAL_DIR"

echo ""
echo "======================================================"
echo "Configuration completed!"
echo "Run './installAll.sh -r background' or 'docker compose up -d' to start the service."
echo "======================================================"

# =============================================================================
# Service Run Mode Selection
# =============================================================================

# If run mode is not specified via parameter, select interactively
if [ -z "$RUN_MODE" ]; then
    echo ""
    echo "Select service run mode:"
    echo "1. Log Mode - Run with real-time logs"
    echo "2. Background Mode - Run in background with status monitoring"
    echo "3. Skip - Do not run"
    echo ""

    while true; do
        echo -n "Select run mode (1/2/3): "
        read -r run_choice

        case $run_choice in
            1)
                RUN_MODE="log"
                break
                ;;
            2)
                RUN_MODE="background"
                break
                ;;
            3)
                RUN_MODE="skip"
                break
                ;;
            *)
                echo "Invalid selection. Please enter 1, 2, or 3."
                ;;
        esac
    done
fi

# Process selected run mode
case $RUN_MODE in
    log)
        echo ""
        echo "Starting service in log mode..."
        echo "=========================================="

        cd "$PROJECT_ROOT_ABS" || {
            echo "Error: Cannot return to project root."
            exit 1
        }

        docker compose --env-file .env up || true
        ;;
    background)
        echo ""
        echo "Starting service in background mode..."
        echo "=========================================="

        cd "$PROJECT_ROOT_ABS" || {
            echo "Error: Cannot return to project root."
            exit 1
        }

        echo "Starting service in background..."
        echo "Image download and initial setup in progress..."
        echo ""

        docker compose --env-file .env up -d

        echo ""
        echo "Image download and initial setup completed."
        echo "Monitoring container status..."
        echo ""

        # Container monitoring function
        monitor_containers() {
            local all_healthy=false
            local check_count=0
            local max_checks=120  # 20 minutes (120 * 10 seconds)

            while [ "$all_healthy" = false ] && [ $check_count -lt $max_checks ]; do
                clear
                echo "=========================================="
                echo "Container Status Monitoring"
                echo "=========================================="
                echo ""

                # Get container status (sorted by name)
                local container_status=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^mc-" | sort)

                if [ -n "$container_status" ]; then
                    echo "$container_status"
                else
                    echo "Containers have not started yet..."
                    echo "Image download and initial setup in progress..."
                fi

                echo ""
                echo "=========================================="

                # Check currently running container status
                local running_containers=$(docker ps --format "{{.Names}}\t{{.Status}}" 2>/dev/null | grep -E "^mc-" | sort)
                local all_expected_running=true
                local unhealthy_count=0
                local running_count=0
                local missing_containers=()

                # Check if each expected container is running and healthy
                for container in "${EXPECTED_CONTAINERS[@]}"; do
                    if echo "$running_containers" | grep -q "^$container"; then
                        running_count=$((running_count + 1))

                        # Containers without health check are treated as successful when Up
                        local is_no_health_check=false
                        for no_health_container in "${NO_HEALTH_CHECK_CONTAINERS[@]}"; do
                            if [ "$container" = "$no_health_container" ]; then
                                is_no_health_check=true
                                break
                            fi
                        done

                        if [ "$is_no_health_check" = true ]; then
                            # Containers without health check are successful if Up
                            if echo "$running_containers" | grep "^$container" | grep -q "Up"; then
                                # Success (just increment count)
                                :
                            else
                                unhealthy_count=$((unhealthy_count + 1))
                            fi
                        else
                            # Containers with health check verify healthy status
                            if echo "$running_containers" | grep "^$container" | grep -q "unhealthy\|starting\|restarting"; then
                                unhealthy_count=$((unhealthy_count + 1))
                            fi
                        fi
                    else
                        all_expected_running=false
                        missing_containers+=("$container")
                    fi
                done

                # Display list of containers waiting to start
                if [ ${#missing_containers[@]} -gt 0 ]; then
                    echo ""
                    echo "Containers waiting to start:"
                    printf "  %s\n" "${missing_containers[@]}"
                fi

                # Check if all expected containers are running and healthy
                if [ "$all_expected_running" = true ] && [ "$unhealthy_count" -eq 0 ] && [ "$running_count" -gt 0 ]; then
                    all_healthy=true
                    echo ""
                    echo "🎉 All environments have been set up!"
                    echo ""
                    echo "Final container status:"
                    echo "$container_status"
                    echo ""
                    MC_IAM_PORT="${MC_IAM_MANAGER_PORT:-5005}"
                    MC_KC_PORT="${MC_IAM_MANAGER_KEYCLOAK_PORT:-8080}"
                    echo "  mc-iam-manager : http://localhost:${MC_IAM_PORT}/readyz"
                    echo "  Keycloak admin : http://localhost:${MC_KC_PORT}/admin/"
                    break
                else
                    echo ""
                    echo "Checking status again in 10 seconds... (${check_count}/${max_checks})"
                    check_count=$((check_count + 1))
                    sleep 10
                fi
            done

            if [ "$all_healthy" = false ]; then
                echo ""
                echo "⚠️  Some containers did not reach healthy status."
                echo "To check status: docker compose ps"
                echo "To check logs: docker logs <container_name>"
            fi
        }

        # Start container monitoring
        monitor_containers
        ;;
    skip)
        echo ""
        echo "Skipping service execution."
        echo "You can start the service later with 'docker compose up -d' command."
        ;;
esac

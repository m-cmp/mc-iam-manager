#!/bin/bash
# Renew the Let's Encrypt certificate used by mc-iam-manager-nginx (prod mode).
#
# The certificate lives in the container volume (container-volume/mc-iam-manager/certs),
# not in the host's /etc/letsencrypt, so a host certbot/certbot.timer does not renew it.
# This script runs the certbot container against that volume:
#   - nginx running     -> webroot (nginx keeps serving /.well-known/acme-challenge/)
#   - nginx not running -> standalone (certbot binds port 80 itself)
# and reloads nginx afterwards.
#
# Usage:
#   sudo scripts/renew-cert.sh                  # renew if due (within 30 days of expiry)
#   sudo scripts/renew-cert.sh --dry-run        # test against the staging server
#   sudo scripts/renew-cert.sh --force-renewal  # renew now
#   sudo scripts/renew-cert.sh --install-cron   # register twice-daily renewal in /etc/cron.d
# Extra arguments are passed to "certbot renew".

set -u

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"
ROOT_DIR="$(cd "$(dirname "$SCRIPT_PATH")/.." && pwd)"
CERT_DIR="$ROOT_DIR/container-volume/mc-iam-manager/certs"
WEBROOT_DIR="$ROOT_DIR/container-volume/certbot/www"
NGINX_CONTAINER="mc-iam-manager-nginx"
CERTBOT_IMAGE="certbot/certbot:latest"
CRON_FILE="/etc/cron.d/mc-iam-manager-cert-renew"
LOG_FILE="/var/log/mc-iam-manager-cert-renew.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [renew-cert] $*"
}

install_cron() {
    if [ ! -d /etc/cron.d ]; then
        log "WARNING: /etc/cron.d not found. Schedule this manually (twice daily): $SCRIPT_PATH"
        return 0
    fi
    if [ "$(id -u)" -ne 0 ]; then
        log "ERROR: --install-cron requires root (run with sudo)."
        return 1
    fi
    cat > "$CRON_FILE" <<EOF
# mc-iam-manager: renew the Let's Encrypt certificate for mc-iam-manager-nginx (prod mode)
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
17 3,15 * * * root $SCRIPT_PATH --quiet >> $LOG_FILE 2>&1
EOF
    chmod 644 "$CRON_FILE"
    log "Installed $CRON_FILE (log: $LOG_FILE)"
}

if [ "${1:-}" = "--install-cron" ]; then
    install_cron
    exit $?
fi

if [ ! -d "$CERT_DIR/renewal" ]; then
    log "ERROR: no certbot renewal config in $CERT_DIR. Issue the certificate first (./installAll.sh -m prod)."
    exit 1
fi

mkdir -p "$WEBROOT_DIR"

if docker ps --format '{{.Names}}' | grep -qx "$NGINX_CONTAINER"; then
    log "$NGINX_CONTAINER is running - renewing with webroot"
    docker run --rm \
        -v "$CERT_DIR:/etc/letsencrypt" \
        -v "$WEBROOT_DIR:/var/www/certbot" \
        "$CERTBOT_IMAGE" renew --non-interactive --webroot -w /var/www/certbot "$@"
    rc=$?
    if [ $rc -eq 0 ]; then
        docker exec "$NGINX_CONTAINER" nginx -s reload \
            && log "$NGINX_CONTAINER reloaded" \
            || log "WARNING: failed to reload $NGINX_CONTAINER"
    fi
else
    log "$NGINX_CONTAINER is not running - renewing with standalone (port 80)"
    docker run --rm -p 80:80 \
        -v "$CERT_DIR:/etc/letsencrypt" \
        "$CERTBOT_IMAGE" renew --non-interactive --standalone "$@"
    rc=$?
fi

# certbot runs as root; give the volume back to its owner (same as installAll.sh)
if [ "$(id -u)" -eq 0 ]; then
    chown -R "$(stat -c '%u:%g' "$ROOT_DIR")" "$CERT_DIR" "$WEBROOT_DIR" 2>/dev/null || true
fi

if [ $rc -ne 0 ]; then
    log "ERROR: certbot renew failed (exit $rc)"
fi
exit $rc

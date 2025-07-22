#!/bin/bash

# --- 스크립트 설정 및 변수 ---
# 프로젝트의 루트 디렉토리 (스크립트가 실행될 위치)
PROJECT_ROOT="$(pwd)"
DOCKER_COMPOSE_FILE="docker-compose.cert.yaml"
NGINX_CONTAINER_NAME="mcmp-nginx-cert"
CERTBOT_CONTAINER_NAME="mcmp-certbot"

# nginx의 after_nginx.conf 파일을 이용하여 dockerfiles/nginx/nginx.conf
# nginx의 conf.d/after_default.conf 파일을 이용하여 dockerfiles/nginx/conf.d/default.conf.template 파일을 생성

# 프로젝트 구조에 맞게 파일 경로 설정
NGINX_SOURCE_DIR="$PROJECT_ROOT/asset/setup/presetup/nginx"
NGINX_TARGET_DIR="$PROJECT_ROOT/dockercontainer-volume/nginx"

if [ -f "$NGINX_SOURCE_DIR/after_nginx.conf" ]; then
    echo "cp $NGINX_SOURCE_DIR/after_nginx.conf to $NGINX_TARGET_DIR/nginx.conf"
    cp "$NGINX_SOURCE_DIR/after_nginx.conf" "$NGINX_TARGET_DIR/nginx.conf"
fi
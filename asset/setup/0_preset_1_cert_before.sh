#!/bin/bash

# --- 스크립트 설정 및 변수 ---
# 프로젝트의 루트 디렉토리 (스크립트가 실행될 위치)
PROJECT_ROOT="$(pwd)"
DOCKER_COMPOSE_FILE="docker-compose.cert.yaml"
NGINX_CONTAINER_NAME="mcmp-nginx-cert"
CERTBOT_CONTAINER_NAME="mcmp-certbot"

# 환경 변수 로드 (.env 파일이 있으면)
if [ -f "$PROJECT_ROOT/.env" ]; then
  export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
  echo ">>> .env 파일 로드 완료."
else
  echo ">>> .env 파일을 찾을 수 없습니다. 도메인 및 이메일 환경 변수를 직접 설정해주세요."
  echo "    예: export KEYCLOAK_DOMAIN=yourdomain.com EMAIL=your_email@example.com"
  exit 1
fi

KEYCLOAK_DOMAIN=${KEYCLOAK_DOMAIN:-localhost} # .env에 없으면 localhost 사용
EMAIL=${EMAIL:-admin@localhost}       # .env에 없으면 admin@localhost 사용

echo "==================================================="
echo " Certbot 초기 인증서 발급 스크립트 (cert_before.sh) "
echo " 대상 도메인: ${KEYCLOAK_DOMAIN}"
echo " 이메일: ${EMAIL}"
echo "==================================================="

# --- Step 1: 대상 폴더를 만들고 파일을 배치 (초기 셋업 시만 실행) ---
echo -e "\n--- Step 1: 대상 폴더 생성 및 초기 Nginx 설정 배치 ---"

# 필수 디렉토리 생성
echo "디렉토리 생성: dockercontainer-volume/certs, dockercontainer-volume/certbot/www, dockercontainer-volume/nginx/conf.d"
mkdir -p "$PROJECT_ROOT/dockercontainer-volume/certs"
mkdir -p "$PROJECT_ROOT/dockercontainer-volume/certbot/www"
mkdir -p "$PROJECT_ROOT/dockercontainer-volume/nginx/conf.d"

# Nginx 설정 파일 복사 및 이름 변경
# before_default.conf -> before_default.conf.template 로 변경해야 함 (envsubst 사용)
# before_nginx.conf -> nginx.conf 로 복사
echo "Nginx 설정 파일 복사 및 이름 변경 확인:"

# 프로젝트 구조에 맞게 파일 경로 설정
NGINX_SOURCE_DIR="$PROJECT_ROOT/asset/setup/presetup/nginx"
NGINX_TARGET_DIR="$PROJECT_ROOT/dockercontainer-volume/nginx"

if [ -f "$NGINX_SOURCE_DIR/conf.d/before_default.conf.template" ]; then
    echo "cp $NGINX_SOURCE_DIR/conf.d/before_default.conf.template to $NGINX_TARGET_DIR/conf.d/default.conf.template"
    cp "$NGINX_SOURCE_DIR/conf.d/before_default.conf.template" "$NGINX_TARGET_DIR/conf.d/default.conf.template"
else
    echo "경고: $NGINX_SOURCE_DIR/conf.d/before_default.conf.template 파일이 없습니다. 수동으로 복사하거나 생성해주세요."
    echo "       ($NGINX_TARGET_DIR/conf.d/default.conf.template 파일이 필요합니다.)"
fi

if [ -f "$NGINX_SOURCE_DIR/before_nginx.conf" ]; then
    echo "cp $NGINX_SOURCE_DIR/before_nginx.conf to $NGINX_TARGET_DIR/nginx.conf"
    cp "$NGINX_SOURCE_DIR/before_nginx.conf" "$NGINX_TARGET_DIR/nginx.conf"
else
    echo "경고: $NGINX_SOURCE_DIR/before_nginx.conf 파일이 없습니다. 수동으로 복사하거나 생성해주세요."
    echo "       ($NGINX_TARGET_DIR/nginx.conf 파일이 필요합니다.)"
fi


echo -e "\n--- Step 1 완료: 초기 파일 배치 및 디렉토리 준비 ---"
echo "이제 Nginx 컨테이너를 시작합니다."

# Nginx 컨테이너 시작 (80 포트만 열린 상태, SSL 설정은 주석 처리된 template)
echo "docker-compose -f ${DOCKER_COMPOSE_FILE} up -d ${NGINX_CONTAINER_NAME}"
docker compose -f "${DOCKER_COMPOSE_FILE}" up -d "${NGINX_CONTAINER_NAME}"

if [ $? -ne 0 ]; then
    echo "오류: Nginx 컨테이너 시작에 실패했습니다. 로그를 확인하고 해결해주세요."
    exit 1
fi

echo -e "\n---------------------------------------------------"
echo " 사용자 작업 필요: Nginx 컨테이너가 시작되었는지 확인하세요."
echo "     'docker ps -f name=${NGINX_CONTAINER_NAME}' 명령으로 확인."
echo "     정상적으로 시작되었다면, 다음 단계를 진행합니다."
echo "---------------------------------------------------"
read -p "Nginx 컨테이너가 실행 중이면 Enter 키를 누르세요..."

# --- Step 2: Certbot 가동하여 인증서 발급 ---
echo -e "\n--- Step 2: Certbot 가동 (초기 인증서 발급) ---"
echo "docker-compose -f ${DOCKER_COMPOSE_FILE} run --rm ${CERTBOT_CONTAINER_NAME}"
docker compose -f "${DOCKER_COMPOSE_FILE}" run --rm "${CERTBOT_CONTAINER_NAME}"

if [ $? -ne 0 ]; then
    echo "오류: Certbot 인증서 발급에 실패했습니다. 로그를 확인하고 해결해주세요."
    echo "      로그 확인: 'docker-compose -f ${DOCKER_COMPOSE_FILE} logs ${CERTBOT_CONTAINER_NAME}'"
    echo "      주요 원인: DNS 설정 오류, Nginx 80포트 접근 불가, 방화벽."
    exit 1
fi

echo -e "\n--- Step 2 완료: Certbot 실행 완료 ---"
echo "Certbot이 인증서를 성공적으로 발급했습니다."
echo "인증서는 ${PROJECT_ROOT}/dockercontainer-volume/certs/live/${DOMAIN_NAME}/ 경로에 있습니다."
echo "이제 Nginx가 사용할 추가 SSL 설정 파일들을 다운로드하고 생성합니다."

# options-ssl-nginx.conf 다운로드
echo "sudo wget -O ${PROJECT_ROOT}/dockercontainer-volume/certs/options-ssl-nginx.conf https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf"
sudo wget -O "$PROJECT_ROOT/dockercontainer-volume/certs/options-ssl-nginx.conf" https://raw.githubusercontent.com/certbot/certbot/master/certbot-nginx/certbot_nginx/_internal/tls_configs/options-ssl-nginx.conf

if [ $? -ne 0 ]; then
    echo "오류: options-ssl-nginx.conf 다운로드에 실패했습니다. 인터넷 연결을 확인하세요."
    exit 1
fi

# dhparams.pem 파일 생성
echo "sudo openssl dhparam -out ${PROJECT_ROOT}/dockercontainer-volume/certs/ssl-dhparams.pem 2048 (이 작업은 시간이 걸릴 수 있습니다.)"
sudo openssl dhparam -out "$PROJECT_ROOT/dockercontainer-volume/certs/ssl-dhparams.pem" 2048

if [ $? -ne 0 ]; then
    echo "오류: ssl-dhparams.pem 생성에 실패했습니다. OpenSSL이 설치되어 있는지 확인하세요."
    exit 1
fi

echo -e "\n---------------------------------------------------"
echo " 사용자 작업 필요: Nginx 설정 파일 (default.conf.template) 수정."
echo "     '${PROJECT_ROOT}/dockercontainer-volume/nginx/conf.d/default.conf.template' 파일을 열어,"
echo "     HTTPS (443 포트) server 블록의 주석을 해제하고,"
echo "     HTTP (80 포트) server 블록의 'location /' 부분에서"
echo "     'return 301 https://\$host\$request_uri;' 라인만 활성화해주세요."
echo "     (초기 Nginx 동작 확인용 'root /usr/share/nginx/html; index index.html;' 등은 제거)"
echo "---------------------------------------------------"
read -p "Nginx 설정 파일 수정 완료 후 Enter 키를 누르세요..."

# --- Step 3: Nginx 재시작 및 인증서 적용 확인 ---
echo -e "\n--- Step 3: Nginx 재시작 및 인증서 적용 확인 ---"
echo "docker-compose -f ${DOCKER_COMPOSE_FILE} restart ${NGINX_CONTAINER_NAME}"
docker compose -f "${DOCKER_COMPOSE_FILE}" restart "${NGINX_CONTAINER_NAME}"

if [ $? -ne 0 ]; then
    echo "오류: Nginx 컨테이너 재시작에 실패했습니다. 로그를 확인하고 해결해주세요."
    exit 1
fi

echo -e "\n--- Step 3 완료: Nginx 재시작 완료 ---"
echo "이제 웹 브라우저 또는 curl로 HTTPS 접속을 확인하세요."
echo "예: https://${DOMAIN_NAME}"
echo "---------------------------------------------------"
echo "최종 확인: 웹 브라우저에서 자물쇠 아이콘이 녹색인지 확인하세요."
echo "           curl로 확인 시 'curl -v https://${DOMAIN_NAME}' 사용 (첫 접속 시 SSL 오류 무시 가능)"
echo "스크립트 실행 완료!"
echo "==================================================="
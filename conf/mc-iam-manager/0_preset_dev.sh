#!/bin/bash

# 템플릿 파일에서 환경변수를 .env 파일의 값으로 대치하는 스크립트

# 스크립트 실행 디렉토리 확인
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "PROJECT_ROOT: $PROJECT_ROOT"

# .env 파일 경로
ENV_FILE="${PROJECT_ROOT}/.env"


# 인증서 파일 생성할 경로 (Let's Encrypt 구조와 동일)
# nginx 볼륨 마운트: ./container-volume/mc-iam-manager/certs:/etc/nginx/certs
CERT_PARENT_DIR="${PROJECT_ROOT}/container-volume/mc-iam-manager"

# --- 3. 필요한 디렉토리 생성 (Let's Encrypt 구조와 동일) ---
echo "Creating necessary directories..."

# dockercontainer-volume 디렉토리 먼저 생성 (sudo 권한으로)
echo "Creating container-volume directory with proper permissions..."

# 현재 사용자 정보 가져오기
CURRENT_USER=$(whoami)
CURRENT_GROUP=$(id -gn)

echo "Current user: ${CURRENT_USER}:${CURRENT_GROUP}"

mkdir -p "${CERT_PARENT_DIR}" || { echo "Error: Failed to create ${CERT_PARENT_DIR}"; exit 1; }
chown -R "${CURRENT_USER}:${CURRENT_GROUP}" "${CERT_PARENT_DIR}" || { echo "Error: Failed to change ownership of ${CERT_PARENT_DIR}"; exit 1; }
echo "✓ Container volume directory created and permissions set"


# 템플릿 파일 경로
TEMPLATE_FILE="./nginx.template.conf"

# 출력 파일 경로 (개선된 구조)
OUTPUT_FILE="${PROJECT_ROOT}/container-volume/mc-iam-manager/nginx/nginx.conf"

# .env 파일 존재 확인
if [ ! -f "$ENV_FILE" ]; then
    echo "오류: .env 파일을 찾을 수 없습니다: $ENV_FILE"
    exit 1
fi

# 템플릿 파일 존재 확인
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "오류: nginx 템플릿 파일을 찾을 수 없습니다: $TEMPLATE_FILE"
    exit 1
fi

# .env 파일을 환경변수로 로드
echo "환경변수를 로드합니다..."

# .env 파일을 직접 소스로 불러오기
source "$ENV_FILE"

# 필수 환경변수 검증
echo "필수 환경변수를 검증합니다..."

# 검증할 필수 환경변수 목록
REQUIRED_VARS=(
    "MC_IAM_MANAGER_PUBLIC_DOMAIN"
    "MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
    "MC_IAM_MANAGER_DATABASE_NAME"
    "MC_IAM_MANAGER_DATABASE_USER"
    "MC_IAM_MANAGER_DATABASE_PASSWORD"
    "MC_IAM_MANAGER_DATABASE_HOST"
    "MC_IAM_MANAGER_PORT"
)

# 각 필수 환경변수 검증
MISSING_VARS=()
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

# 누락된 환경변수가 있으면 종료
if [ ${#MISSING_VARS[@]} -gt 0 ]; then
    echo "❌ 오류: 다음 필수 환경변수가 설정되지 않았습니다:"
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
    echo ""
    echo "해결 방법:"
    echo "1. .env 파일이 존재하는지 확인: $ENV_FILE"
    echo "2. .env 파일에 필수 환경변수들이 설정되어 있는지 확인"
    echo "3. .env_sample 파일을 참고하여 누락된 환경변수를 추가"
    exit 1
fi

# MC_IAM_MANAGER_KEYCLOAK_PORT가 설정되지 않은 경우 기본값 설정
if [ -z "$MC_IAM_MANAGER_KEYCLOAK_PORT" ]; then
    MC_IAM_MANAGER_KEYCLOAK_PORT=8080
    echo "MC_IAM_MANAGER_KEYCLOAK_PORT가 설정되지 않아 기본값 8080을 사용합니다."
fi

echo "✅ 모든 필수 환경변수가 정상적으로 로드되었습니다."
echo "읽어온 환경변수:"
echo "  PUBLIC_DOMAIN: $MC_IAM_MANAGER_PUBLIC_DOMAIN"
echo "  KEYCLOAK_DOMAIN: $MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
echo "  MC_IAM_MANAGER_KEYCLOAK_PORT: $MC_IAM_MANAGER_KEYCLOAK_PORT"
echo "  DATABASE_NAME: $MC_IAM_MANAGER_DATABASE_NAME"
echo "  DATABASE_USER: $MC_IAM_MANAGER_DATABASE_USER"
echo "  DATABASE_HOST: $MC_IAM_MANAGER_DATABASE_HOST"
echo "  MC_IAM_MANAGER_PORT: $MC_IAM_MANAGER_PORT"

# PUBLIC_DOMAIN 기준으로 인증서 디렉토리 정의 (Let's Encrypt 구조와 동일)
CERT_DIR="${CERT_PARENT_DIR}/certs/live/${MC_IAM_MANAGER_PUBLIC_DOMAIN}"

# Let's Encrypt 구조와 동일한 certs/live/도메인명 디렉토리 생성
echo "Creating certificate directory: ${CERT_DIR}"
mkdir -p "${CERT_DIR}" || { echo "Error: Failed to create ${CERT_DIR}"; exit 1; }
echo "✓ Certificate directory created successfully"


## 로컬환경(인증서) 설정
# --- 3. hosts 파일에 도메인 추가 (관리자 권한 필요) ---
HOSTS_FILE="/etc/hosts"
echo "Checking ${MC_IAM_MANAGER_PUBLIC_DOMAIN} in ${HOSTS_FILE}..."

if grep -E "^[[:space:]]*127\.0\.0\.1[[:space:]]+${MC_IAM_MANAGER_PUBLIC_DOMAIN}[[:space:]]*$" "${HOSTS_FILE}" > /dev/null; then
    echo "✓ ${MC_IAM_MANAGER_PUBLIC_DOMAIN} already exists in ${HOSTS_FILE}. Skipping."
else
    echo "Removing any existing entries for ${MC_IAM_MANAGER_PUBLIC_DOMAIN}..."
    sed -i "/[[:space:]]*127\.0\.0\.1[[:space:]]\+${MC_IAM_MANAGER_PUBLIC_DOMAIN}[[:space:]]*$/d" "${HOSTS_FILE}"

    echo "Adding 127.0.0.1 ${MC_IAM_MANAGER_PUBLIC_DOMAIN} to ${HOSTS_FILE}..."
    if echo "127.0.0.1 ${MC_IAM_MANAGER_PUBLIC_DOMAIN}" >> "${HOSTS_FILE}" 2>/dev/null; then
        echo "✓ ${MC_IAM_MANAGER_PUBLIC_DOMAIN} added successfully to ${HOSTS_FILE}."
    else
        echo "⚠️  Failed to add to ${HOSTS_FILE} — run with sudo or manually add:"
        echo "    echo '127.0.0.1 ${MC_IAM_MANAGER_PUBLIC_DOMAIN}' | sudo tee -a ${HOSTS_FILE}"
    fi
fi


# --- 4. Self-Signed Certificate 생성 ---
echo "Generating Self-Signed Certificate for ${MC_IAM_MANAGER_PUBLIC_DOMAIN}... ${CERT_DIR}"

# 기존 인증서 삭제 (새로 발급하기 위해)
if [ -f "${CERT_DIR}/privkey.pem" ]; then
    echo "Removing existing certificate files..."
    rm "${CERT_DIR}/privkey.pem" "${CERT_DIR}/fullchain.pem" 2>/dev/null
fi

openssl genrsa -out "${CERT_DIR}/privkey.pem" 2048
openssl req -new -key "${CERT_DIR}/privkey.pem" -out "${CERT_DIR}/csr.pem" -subj "/CN=${MC_IAM_MANAGER_PUBLIC_DOMAIN}"
openssl x509 -req -days 365 -in "${CERT_DIR}/csr.pem" -signkey "${CERT_DIR}/privkey.pem" -out "${CERT_DIR}/fullchain.pem"
rm "${CERT_DIR}/csr.pem" # CSR 파일 제거

if [ -f "${CERT_DIR}/fullchain.pem" ]; then
    echo "Self-Signed Certificate generated successfully at ${CERT_DIR}."
else
    echo "Failed to generate Self-Signed Certificate."
    exit 1
fi



echo "nginx 설정 파일을 생성합니다..."
echo "템플릿: $TEMPLATE_FILE"
echo "출력: $OUTPUT_FILE"

# 출력 디렉토리가 필요한 경우에만 생성 (상대 경로나 절대 경로인 경우)
OUTPUT_DIR="$(dirname "$OUTPUT_FILE")"
if [ "$OUTPUT_DIR" != "." ] && [ "$OUTPUT_DIR" != "$(pwd)" ]; then
    echo "Creating output directory: $OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"
fi

# 기존 nginx.conf 파일이 디렉토리인 경우 제거
if [ -d "$OUTPUT_FILE" ]; then
    echo "Removing existing directory: $OUTPUT_FILE"
    rm -rf "$OUTPUT_FILE"
fi

# 환경변수 대치 (한 번에 처리)
if [ -n "$MC_IAM_MANAGER_PUBLIC_DOMAIN" ] && [ -n "$MC_IAM_MANAGER_KEYCLOAK_PORT" ]; then
    sed -e "s/\${MC_IAM_MANAGER_DOMAIN}/$MC_IAM_MANAGER_DOMAIN/g" \
        -e "s/\${MC_IAM_MANAGER_PORT}/$MC_IAM_MANAGER_PORT/g" \
        -e "s/\${MC_IAM_MANAGER_PUBLIC_DOMAIN}/$MC_IAM_MANAGER_PUBLIC_DOMAIN/g" \
        -e "s/\${MC_IAM_MANAGER_KEYCLOAK_DOMAIN}/$MC_IAM_MANAGER_KEYCLOAK_DOMAIN/g" \
        -e "s/\${MC_IAM_MANAGER_KEYCLOAK_PORT}/$MC_IAM_MANAGER_KEYCLOAK_PORT/g" \
        -e "s/\${MC_OBSERVABILITY_GRAFANA_PROXY_PORT}/$MC_OBSERVABILITY_GRAFANA_PROXY_PORT/g" \
        -e "s/\${MC_COST_OPTIMIZER_FE_PROXY_PORT}/$MC_COST_OPTIMIZER_FE_PROXY_PORT/g" \
        -e "s/mciam-manager/mc-iam-manager/g" \
        -e "s/mciam-keycloak/mc-iam-manager-kc/g" \
        "$TEMPLATE_FILE" > "$OUTPUT_FILE"
    echo "✓ PUBLIC_DOMAIN 대치 완료: $MC_IAM_MANAGER_PUBLIC_DOMAIN"
    echo "✓ PORT 대치 완료: $MC_IAM_MANAGER_KEYCLOAK_PORT"
    echo "✓ 컨테이너 이름 수정 완료"
else
    echo "경고: MC_IAM_MANAGER_PUBLIC_DOMAIN 또는 MC_IAM_MANAGER_KEYCLOAK_PORT 환경변수가 설정되지 않았습니다."
    cp "$TEMPLATE_FILE" "$OUTPUT_FILE"
fi

echo "nginx 설정 파일 생성이 완료되었습니다: $OUTPUT_FILE"

# 생성된 파일의 내용 확인 (선택사항)
echo ""
echo "=== 생성된 nginx.conf 파일 내용 ==="
cat "$OUTPUT_FILE"
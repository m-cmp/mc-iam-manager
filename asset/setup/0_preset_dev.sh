#!/bin/bash

# 템플릿 파일에서 환경변수를 .env 파일의 값으로 대치하는 스크립트

# 스크립트 실행 디렉토리 확인
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

# .env 파일 경로
ENV_FILE="$PROJECT_ROOT/.env"


# 인증서 파일 생성할 경로 (Let's Encrypt 구조와 동일)
CERT_PARENT_DIR="$PROJECT_ROOT/dockercontainer-volume" # dockercontainer-volume 디렉토리

# --- 3. 필요한 디렉토리 생성 (Let's Encrypt 구조와 동일) ---
echo "Creating necessary directories..."

# dockercontainer-volume 디렉토리 먼저 생성
mkdir -p "${CERT_PARENT_DIR}" || { echo "Error: Failed to create ${CERT_PARENT_DIR}"; exit 1; }


# 템플릿 파일 경로
TEMPLATE_FILE="$PROJECT_ROOT/asset/setup/presetup/nginx/nginx.template.conf"

# 출력 파일 경로
OUTPUT_FILE="$PROJECT_ROOT/dockerfiles/nginx/nginx.conf"

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

# .env 파일을 안전하게 로드
echo "환경변수를 로드합니다..."

# .env 파일에서 필요한 변수들을 직접 읽어오기 (줄바꿈 문자 제거)
DOMAIN_NAME=$(grep "^DOMAIN_NAME=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | tr -d '\r' | xargs)
MCIAMMANAGER_PORT=$(grep "^MCIAMMANAGER_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | tr -d '\r' | xargs)

echo "읽어온 환경변수:"
echo "  DOMAIN_NAME: $DOMAIN_NAME"
echo "  MCIAMMANAGER_PORT: $MCIAMMANAGER_PORT"

# DOMAIN_NAME을 읽은 후 CERT_DIR 정의
CERT_DIR="${CERT_PARENT_DIR}/certs/live/${DOMAIN_NAME}"      # Let's Encrypt 구조와 동일한 인증서 저장 경로

# Let's Encrypt 구조와 동일한 certs/live/도메인명 디렉토리 생성
mkdir -p "${CERT_DIR}" || { echo "Error: Failed to create ${CERT_DIR}"; exit 1; }


## 로컬환경(인증서) 설정
# --- 3. hosts 파일에 도메인 추가 (관리자 권한 필요) ---
HOSTS_FILE="/etc/hosts" # hosts 파일 경로 (macOS/Linux 기준)
echo "Adding ${DOMAIN_NAME} to ${HOSTS_FILE}..."
if grep -q "127.0.0.1 ${DOMAIN_NAME}" "${HOSTS_FILE}"; then
    echo "${DOMAIN_NAME} already exists in ${HOSTS_FILE}. Skipping."
else
    # hosts 파일에 추가 (sudo 권한 필요)
    # macOS/Linux에서 이 스크립트를 직접 실행 시 sudo로 실행해야 합니다.
    echo "127.0.0.1 ${DOMAIN_NAME}" | sudo tee -a "${HOSTS_FILE}" > /dev/null
    if [ $? -eq 0 ]; then
        echo "${DOMAIN_NAME} added successfully to ${HOSTS_FILE}."
    else
        echo "Failed to add ${DOMAIN_NAME} to ${HOSTS_FILE}. Please run this script with sudo or manually add it."
        echo "Manual step: Add '127.0.0.1 ${DOMAIN_NAME}' to ${HOSTS_FILE}"
    fi
fi


# --- 4. Self-Signed Certificate 생성 ---
echo "Generating Self-Signed Certificate for ${DOMAIN_NAME}... ${CERT_DIR}"

# 기존 인증서 삭제 (새로 발급하기 위해)
if [ -f "${CERT_DIR}/privkey.pem" ]; then
    echo "Removing existing certificate files..."
    rm "${CERT_DIR}/privkey.pem" "${CERT_DIR}/fullchain.pem" 2>/dev/null
fi

openssl genrsa -out "${CERT_DIR}/privkey.pem" 2048
openssl req -new -key "${CERT_DIR}/privkey.pem" -out "${CERT_DIR}/csr.pem" -subj "/CN=${DOMAIN_NAME}"
openssl x509 -req -days 365 -in "${CERT_DIR}/csr.pem" -signkey "${CERT_DIR}/privkey.pem" -out "${CERT_DIR}/fullchain.pem"
rm "${CERT_DIR}/csr.pem" # CSR 파일 제거

if [ -f "${CERT_DIR}/fullchain.pem" ]; then
    echo "Self-Signed Certificate generated successfully at ${CERT_DIR}."
else
    echo "Failed to generate Self-Signed Certificate."
    exit 1
fi



# 출력 디렉토리 생성
OUTPUT_DIR="$(dirname "$OUTPUT_FILE")"
mkdir -p "$OUTPUT_DIR"

echo "nginx 설정 파일을 생성합니다..."
echo "템플릿: $TEMPLATE_FILE"
echo "출력: $OUTPUT_FILE"


# 템플릿 파일을 복사하고 환경변수 대치
cp "$TEMPLATE_FILE" "$OUTPUT_FILE"

# 환경변수 대치 (한 번에 처리)
if [ -n "$DOMAIN_NAME" ] && [ -n "$MCIAMMANAGER_PORT" ]; then
    # 템플릿 파일을 복사하고 환경변수를 한 번에 대치
    sed -e "s/\${DOMAIN_NAME}/$DOMAIN_NAME/g" \
        -e "s/\${PORT}/$MCIAMMANAGER_PORT/g" \
        "$TEMPLATE_FILE" > "$OUTPUT_FILE"
    echo "✓ DOMAIN_NAME 대치 완료: $DOMAIN_NAME"
    echo "✓ PORT 대치 완료: $MCIAMMANAGER_PORT"
else
    echo "경고: DOMAIN_NAME 또는 MCIAMMANAGER_PORT 환경변수가 설정되지 않았습니다."
    # 환경변수가 없으면 템플릿 파일을 그대로 복사
    cp "$TEMPLATE_FILE" "$OUTPUT_FILE"
fi

echo "nginx 설정 파일 생성이 완료되었습니다: $OUTPUT_FILE"

# 생성된 파일의 내용 확인 (선택사항)
echo ""
echo "=== 생성된 nginx.conf 파일 내용 ==="
cat "$OUTPUT_FILE"

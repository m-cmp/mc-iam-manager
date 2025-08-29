#!/bin/bash

# 템플릿 파일에서 환경변수를 .env 파일의 값으로 대치하는 스크립트

# 스크립트 실행 디렉토리 확인
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

# .env 파일 경로
ENV_FILE="$PROJECT_ROOT/.env"

# 템플릿 파일 경로
TEMPLATE_FILE="./nginx.template.conf"

# 출력 파일 경로
OUTPUT_FILE="./nginx.conf"

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

# 출력 디렉토리 생성
OUTPUT_DIR="$(dirname "$OUTPUT_FILE")"
mkdir -p "$OUTPUT_DIR"

echo "nginx 설정 파일을 생성합니다..."
echo "템플릿: $TEMPLATE_FILE"
echo "출력: $OUTPUT_FILE"

# .env 파일을 안전하게 로드
echo "환경변수를 로드합니다..."

# .env 파일에서 필요한 변수들을 직접 읽어오기
MC_IAM_MANAGER_KEYCLOAK_DOMAIN=$(grep "^MC_IAM_MANAGER_KEYCLOAK_DOMAIN=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_IAM_MANAGER_KEYCLOAK_PORT=$(grep "^MC_IAM_MANAGER_KEYCLOAK_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)

echo "읽어온 환경변수:"
echo "  MC_IAM_MANAGER_KEYCLOAK_DOMAIN: $MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
echo "  MC_IAM_MANAGER_KEYCLOAK_PORT: $MC_IAM_MANAGER_KEYCLOAK_PORT"

# 템플릿 파일을 복사하고 환경변수 대치
cp "$TEMPLATE_FILE" "$OUTPUT_FILE"

# 환경변수 대치
# ${DOMAIN_NAME} 대치
if [ -n "$MC_IAM_MANAGER_KEYCLOAK_DOMAIN" ]; then
    sed -i "s/\${MC_IAM_MANAGER_KEYCLOAK_DOMAIN}/$MC_IAM_MANAGER_KEYCLOAK_DOMAIN/g" "$OUTPUT_FILE"
    echo "✓ MC_IAM_MANAGER_KEYCLOAK_DOMAIN 대치 완료: $MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
else
    echo "경고: MC_IAM_MANAGER_KEYCLOAK_DOMAIN 환경변수가 설정되지 않았습니다."
fi

# ${PORT} 대치 (MC_IAM_MANAGER_PORT 사용)
if [ -n "$MC_IAM_MANAGER_KEYCLOAK_PORT" ]; then
    sed -i "s/\${MC_IAM_MANAGER_KEYCLOAK_PORT}/$MC_IAM_MANAGER_KEYCLOAK_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_IAM_MANAGER_KEYCLOAK_PORT 대치 완료: $MC_IAM_MANAGER_KEYCLOAK_PORT"
else
    echo "경고: MC_IAM_MANAGER_KEYCLOAK_PORT 환경변수가 설정되지 않았습니다."
fi

echo "nginx 설정 파일 생성이 완료되었습니다: $OUTPUT_FILE"

# 생성된 파일의 내용 확인 (선택사항)
echo ""
echo "=== 생성된 nginx.conf 파일 내용 ==="
cat "$OUTPUT_FILE"

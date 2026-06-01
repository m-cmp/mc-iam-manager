#!/bin/bash
set -euo pipefail

# Script to substitute environment variables in the template file with values from the .env file

# 스크립트 실행 디렉토리 확인
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

# .env 파일 경로
ENV_FILE="$PROJECT_ROOT/.env"

# 템플릿 파일 경로
TEMPLATE_FILE="${SCRIPT_DIR}/nginx.template.conf"

# 출력 파일 경로
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

# Create output directory
OUTPUT_DIR="$(dirname "$OUTPUT_FILE")"
mkdir -p "$OUTPUT_DIR" || { echo "Error: Cannot create directory: $OUTPUT_DIR (may be a permission issue)"; exit 1; }

echo "nginx 설정 파일을 생성합니다..."
echo "템플릿: $TEMPLATE_FILE"
echo "출력: $OUTPUT_FILE"

# .env 파일을 안전하게 로드
echo "환경변수를 로드합니다..."

# .env 파일에서 필요한 변수들을 직접 읽어오기
MC_IAM_MANAGER_PORT=$(grep -m1 "^MC_IAM_MANAGER_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_IAM_MANAGER_DOMAIN=$(grep -m1 "^MC_IAM_MANAGER_DOMAIN=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_IAM_MANAGER_PUBLIC_DOMAIN=$(grep -m1 "^MC_IAM_MANAGER_PUBLIC_DOMAIN=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_IAM_MANAGER_KEYCLOAK_DOMAIN=$(grep -m1 "^MC_IAM_MANAGER_KEYCLOAK_DOMAIN=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_IAM_MANAGER_KEYCLOAK_PORT=$(grep -m1 "^MC_IAM_MANAGER_KEYCLOAK_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_OBSERVABILITY_GRAFANA_PROXY_PORT=$(grep -m1 "^MC_OBSERVABILITY_GRAFANA_PROXY_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_COST_OPTIMIZER_FE_PROXY_PORT=$(grep -m1 "^MC_COST_OPTIMIZER_FE_PROXY_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_COST_OPTIMIZER_BE_PORT=$(grep -m1 "^MC_COST_OPTIMIZER_BE_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_COST_OPTIMIZER_ALARM_PORT=$(grep -m1 "^MC_COST_OPTIMIZER_ALARM_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_WORKFLOW_MANAGER_PROXY_PORT=$(grep -m1 "^MC_WORKFLOW_MANAGER_PROXY_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_DATA_MANAGER_PROXY_PORT=$(grep -m1 "^MC_DATA_MANAGER_PROXY_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)
MC_APPLICATION_MANAGER_PROXY_PORT=$(grep -m1 "^MC_APPLICATION_MANAGER_PROXY_PORT=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" | xargs)

echo "Loaded environment variables:"
echo "  MC_IAM_MANAGER_DOMAIN: $MC_IAM_MANAGER_DOMAIN"
echo "  MC_IAM_MANAGER_PORT: $MC_IAM_MANAGER_PORT"
echo "  MC_IAM_MANAGER_PUBLIC_DOMAIN: $MC_IAM_MANAGER_PUBLIC_DOMAIN"
echo "  MC_IAM_MANAGER_KEYCLOAK_DOMAIN: $MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
echo "  MC_IAM_MANAGER_KEYCLOAK_PORT: $MC_IAM_MANAGER_KEYCLOAK_PORT"
echo "  MC_OBSERVABILITY_GRAFANA_PROXY_PORT: $MC_OBSERVABILITY_GRAFANA_PROXY_PORT"
echo "  MC_COST_OPTIMIZER_FE_PROXY_PORT: $MC_COST_OPTIMIZER_FE_PROXY_PORT"
echo "  MC_COST_OPTIMIZER_BE_PORT: $MC_COST_OPTIMIZER_BE_PORT"
echo "  MC_COST_OPTIMIZER_ALARM_PORT: $MC_COST_OPTIMIZER_ALARM_PORT"
echo "  MC_WORKFLOW_MANAGER_PROXY_PORT: $MC_WORKFLOW_MANAGER_PROXY_PORT"
echo "  MC_DATA_MANAGER_PROXY_PORT: $MC_DATA_MANAGER_PROXY_PORT"
echo "  MC_APPLICATION_MANAGER_PROXY_PORT: $MC_APPLICATION_MANAGER_PROXY_PORT"

# Copy template file and substitute environment variables
cp "$TEMPLATE_FILE" "$OUTPUT_FILE" || { echo "Error: Failed to copy template file: $TEMPLATE_FILE → $OUTPUT_FILE"; exit 1; }

if [ -n "$MC_IAM_MANAGER_PORT" ]; then
    sed -i "s/\${MC_IAM_MANAGER_PORT}/$MC_IAM_MANAGER_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_IAM_MANAGER_PORT 대치 완료: $MC_IAM_MANAGER_PORT"
else
    echo "경고: MC_IAM_MANAGER_PORT 환경변수가 설정되지 않았습니다."
fi

if [ -n "$MC_IAM_MANAGER_PUBLIC_DOMAIN" ]; then
    sed -i "s/\${MC_IAM_MANAGER_PUBLIC_DOMAIN}/$MC_IAM_MANAGER_PUBLIC_DOMAIN/g" "$OUTPUT_FILE"
    echo "✓ MC_IAM_MANAGER_PUBLIC_DOMAIN 대치 완료: $MC_IAM_MANAGER_PUBLIC_DOMAIN"
else
    echo "경고: MC_IAM_MANAGER_PUBLIC_DOMAIN 환경변수가 설정되지 않았습니다."
fi

if [ -n "$MC_IAM_MANAGER_KEYCLOAK_DOMAIN" ]; then
    sed -i "s/\${MC_IAM_MANAGER_KEYCLOAK_DOMAIN}/$MC_IAM_MANAGER_KEYCLOAK_DOMAIN/g" "$OUTPUT_FILE"
    echo "✓ MC_IAM_MANAGER_KEYCLOAK_DOMAIN 대치 완료: $MC_IAM_MANAGER_KEYCLOAK_DOMAIN"
else
    echo "경고: MC_IAM_MANAGER_KEYCLOAK_DOMAIN 환경변수가 설정되지 않았습니다."
fi

if [ -n "$MC_IAM_MANAGER_KEYCLOAK_PORT" ]; then
    sed -i "s/\${MC_IAM_MANAGER_KEYCLOAK_PORT}/$MC_IAM_MANAGER_KEYCLOAK_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_IAM_MANAGER_KEYCLOAK_PORT 대치 완료: $MC_IAM_MANAGER_KEYCLOAK_PORT"
else
    echo "경고: MC_IAM_MANAGER_KEYCLOAK_PORT 환경변수가 설정되지 않았습니다."
fi

if [ -n "$MC_OBSERVABILITY_GRAFANA_PROXY_PORT" ]; then
    sed -i "s/\${MC_OBSERVABILITY_GRAFANA_PROXY_PORT}/$MC_OBSERVABILITY_GRAFANA_PROXY_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_OBSERVABILITY_GRAFANA_PROXY_PORT 대치 완료: $MC_OBSERVABILITY_GRAFANA_PROXY_PORT"
else
    echo "경고: MC_OBSERVABILITY_GRAFANA_PROXY_PORT 환경변수가 설정되지 않았습니다."
fi

if [ -n "$MC_COST_OPTIMIZER_FE_PROXY_PORT" ]; then
    sed -i "s/\${MC_COST_OPTIMIZER_FE_PROXY_PORT}/$MC_COST_OPTIMIZER_FE_PROXY_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_COST_OPTIMIZER_FE_PROXY_PORT 대치 완료: $MC_COST_OPTIMIZER_FE_PROXY_PORT"
else
    echo "경고: MC_COST_OPTIMIZER_FE_PROXY_PORT 환경변수가 설정되지 않았습니다."
fi

if [ -n "$MC_COST_OPTIMIZER_BE_PORT" ]; then
    sed -i "s/\${MC_COST_OPTIMIZER_BE_PORT}/$MC_COST_OPTIMIZER_BE_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_COST_OPTIMIZER_BE_PORT substitution done: $MC_COST_OPTIMIZER_BE_PORT"
else
    echo "Warning: MC_COST_OPTIMIZER_BE_PORT environment variable is not set."
fi

if [ -n "$MC_COST_OPTIMIZER_ALARM_PORT" ]; then
    sed -i "s/\${MC_COST_OPTIMIZER_ALARM_PORT}/$MC_COST_OPTIMIZER_ALARM_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_COST_OPTIMIZER_ALARM_PORT substitution done: $MC_COST_OPTIMIZER_ALARM_PORT"
else
    echo "Warning: MC_COST_OPTIMIZER_ALARM_PORT environment variable is not set."
fi

if [ -n "$MC_WORKFLOW_MANAGER_PROXY_PORT" ]; then
    sed -i "s/\${MC_WORKFLOW_MANAGER_PROXY_PORT}/$MC_WORKFLOW_MANAGER_PROXY_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_WORKFLOW_MANAGER_PROXY_PORT substitution done: $MC_WORKFLOW_MANAGER_PROXY_PORT"
else
    echo "Warning: MC_WORKFLOW_MANAGER_PROXY_PORT environment variable is not set."
fi

if [ -n "$MC_DATA_MANAGER_PROXY_PORT" ]; then
    sed -i "s/\${MC_DATA_MANAGER_PROXY_PORT}/$MC_DATA_MANAGER_PROXY_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_DATA_MANAGER_PROXY_PORT substitution done: $MC_DATA_MANAGER_PROXY_PORT"
else
    echo "Warning: MC_DATA_MANAGER_PROXY_PORT environment variable is not set."
fi

if [ -n "$MC_APPLICATION_MANAGER_PROXY_PORT" ]; then
    sed -i "s/\${MC_APPLICATION_MANAGER_PROXY_PORT}/$MC_APPLICATION_MANAGER_PROXY_PORT/g" "$OUTPUT_FILE"
    echo "✓ MC_APPLICATION_MANAGER_PROXY_PORT substitution done: $MC_APPLICATION_MANAGER_PROXY_PORT"
else
    echo "Warning: MC_APPLICATION_MANAGER_PROXY_PORT environment variable is not set."
fi

# Substitute container names (correct legacy names in template)
sed -i "s/mciam-manager/mc-iam-manager/g" "$OUTPUT_FILE"
sed -i "s/mciam-keycloak/mc-iam-manager-kc/g" "$OUTPUT_FILE"
echo "✓ 컨테이너 이름 수정 완료"

echo "nginx 설정 파일 생성이 완료되었습니다: $OUTPUT_FILE"

# 생성된 파일의 내용 확인 (선택사항)
echo ""
echo "=== 생성된 nginx.conf 파일 내용 ==="
cat "$OUTPUT_FILE"

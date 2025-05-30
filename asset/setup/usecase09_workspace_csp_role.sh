#!/bin/bash

# 환경 변수 로드
source ../../.env

# 로그 함수
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Platform Admin 로그인
login() {
    log "Platform Admin으로 로그인합니다..."
    login_url="$MCIAMMANAGER_HOST/api/auth/login"
    log "API 호출: $login_url"
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MCIAMMANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MCIAMMANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$login_url")
    MCIAMMANAGER_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    log "로그인 응답: $response"
    log "로그인 성공"
}

# 1. 모든 워크스페이스 역할 조회
get_workspace_roles() {
    log "모든 워크스페이스 역할을 조회합니다..."
    url="$MCIAMMANAGER_HOST/api/roles"
    log "API 호출: $url"
    response=$(curl --location --silent --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" "$url")
    log "응답: $response"
    WORKSPACE_ROLE_ID=$(echo "$response" | jq -r '.[0].id')
    log "선택된 워크스페이스 역할 ID: $WORKSPACE_ROLE_ID"
}

# 2. 모든 CSP 역할 조회
get_csp_roles() {
    log "모든 CSP 역할을 조회합니다..."
    url="$MCIAMMANAGER_HOST/api/csp-roles/all"
    log "API 호출: $url"
    response=$(curl --location --silent --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" "$url")
    log "응답: $response"
    CSP_ROLE_ID=$(echo "$response" | jq -r '.[0].id')
    log "선택된 CSP 역할 ID: $CSP_ROLE_ID"
}

# 3. 워크스페이스 역할의 CSP 역할 매핑 조회
get_role_mappings() {
    local workspaceRoleId=$1
    log "워크스페이스 역할의 CSP 역할 매핑을 조회합니다..."
    url="$MCIAMMANAGER_HOST/api/workspace-roles/$workspaceRoleId/csp-roles"
    log "API 호출: $url"
    response=$(curl --location --silent --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" "$url")
    log "응답: $response"
}

# 4. 워크스페이스 역할에 CSP 역할 매핑 생성
create_role_mapping() {
    local workspaceRoleId=$1
    local cspRoleId=$2

    log "워크스페이스 역할에 CSP 역할 매핑을 생성합니다..."
    url="$MCIAMMANAGER_HOST/api/workspace-roles/$workspaceRoleId/csp-roles"
    log "API 호출: $url"
    response=$(curl --location --silent --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data '{
            "cspType": "aws",
            "workspaceRoleId": "'"$workspaceRoleId"'",
            "cspRoleId": "'"$cspRoleId"'",
            "description": "워크스페이스 관리자용 AWS 역할 매핑"
        }' "$url")
    log "응답: $response"
}

# 5. 워크스페이스 역할에서 CSP 역할 매핑 삭제
delete_role_mapping() {
    local workspaceRoleId=$1
    local cspRoleId=$2

    log "워크스페이스 역할에서 CSP 역할 매핑을 삭제합니다..."
    url="$MCIAMMANAGER_HOST/api/workspace-roles/$workspaceRoleId/csp-roles/aws/$cspRoleId"
    log "API 호출: $url"
    response=$(curl --location --silent --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" -X DELETE "$url")
    log "응답: $response"
}

# # 메인 실행
# main() {
#     login
#     get_workspace_roles
#     get_csp_roles
#     get_role_mappings
#     create_role_mapping
#     get_role_mappings
#     delete_role_mapping
#     get_role_mappings
# }

# 메인 메뉴
while true; do
    echo
    log "=== CSP 역할 관리 테스트 ==="
    log "1. Platform Admin 로그인"
    log "2. 모든 워크스페이스 역할 조회"
    log "3. 모든 CSP 역할 조회"
    log "4. 워크스페이스 역할의 CSP 역할 매핑 조회"
    log "5. 워크스페이스 역할에 CSP 역할 매핑 생성"
    log "6. 생성된 매핑 확인"
    log "7. 워크스페이스 역할에서 CSP 역할 매핑 삭제"
    log "8. 현재 매핑 상태 확인"
    log "0. 종료"

    read -p "선택 (0-8): " choice

    case $choice in
        1)
            login
            ;;
        2)
            get_workspace_roles
            ;;
        3)
            get_csp_roles
            ;;
        4)
            read -p "워크스페이스 역할 ID를 입력하세요: " workspaceRoleId
            get_role_mappings "$workspaceRoleId"
            ;;
        5)
            read -p "워크스페이스 역할 ID를 입력하세요: " workspaceRoleId
            read -p "CSP 역할 ID를 입력하세요: " cspRoleId
            create_role_mapping "$workspaceRoleId" "$cspRoleId"
            ;;
        6)
            read -p "워크스페이스 역할 ID를 입력하세요: " workspaceRoleId
            get_role_mappings "$workspaceRoleId"
            ;;
        7)
            read -p "워크스페이스 역할 ID를 입력하세요: " workspaceRoleId
            read -p "삭제할 CSP 역할 ID를 입력하세요: " cspRoleId
            delete_role_mapping "$workspaceRoleId" "$cspRoleId"
            ;;
        8)
            read -p "워크스페이스 역할 ID를 입력하세요: " workspaceRoleId
            get_role_mappings "$workspaceRoleId"
            ;;
        0)
            log "프로그램을 종료합니다."
            exit 0
            ;;
        *)
            log "잘못된 선택입니다. 다시 시도하세요."
            ;;
    esac

    echo
    read -p "계속하려면 Enter를 누르세요..."
done
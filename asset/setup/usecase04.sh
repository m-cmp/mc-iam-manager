#!/bin/bash

# 환경 변수 로드
source ../../.env

# 로그 함수
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Platform Admin 로그인
login_as_platform_admin() {
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

# 일반 사용자 로그인
login_as_user() {
    read -p "사용자 ID: " username
    read -s -p "비밀번호: " password
    echo
    log "로그인 시도 중..."
    login_url="$MCIAMMANAGER_HOST/api/auth/login"
    log "API 호출: $login_url"
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$username"'",
        "password":"'"$password"'"
    }' "$login_url")
    MCIAMMANAGER_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    log "로그인 응답: $response"
    log "로그인 성공"
}

# 워크스페이스 조회
list_workspaces() {
    log "워크스페이스 목록 조회 중..."
    response=$(curl -s -X GET "$MCIAMMANAGER_HOST/api/workspaces" \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN")
    log "워크스페이스 목록: $response"
}

# 워크스페이스에 사용자 역할 할당
assign_workspace_role() {
    read -p "워크스페이스 ID: " workspaceId
    read -p "사용자 ID: " username
    read -p "역할 이름: " workspaceRoleName
    
    url="$MCIAMMANAGER_HOST/api/workspaces/id/$workspaceId/users/$username/roles/$workspaceRoleName"
    log "API 호출: $url"
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        -H "Content-Type: application/json" \
        -d '{"role": "'"$workspaceRoleName"'"}' \
        "$url")
    
    log "역할 할당 응답: $response"
}

# 워크스페이스에서 사용자 역할 해제
remove_workspace_role() {
    read -p "워크스페이스 ID: " workspace_id
    read -p "사용자 ID: " username
    read -p "역할 이름: " role_name
    
    log "워크스페이스 역할 해제 중..."    
    remove_url="$MCIAMMANAGER_HOST/api/workspaces/id/$workspace_id/users/$username/roles/$role_name"
    echo "Calling API: $remove_url"
    response=$(curl -s -X DELETE \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$remove_url")

    log "역할 해제 응답: $response"
}

# 사용자의 워크스페이스와 역할 목록 조회
list_user_workspaces() {
    log "사용자의 워크스페이스와 역할 목록 조회 중..."
    list_url="$MCIAMMANAGER_HOST/api/users/workspaces"
    echo "Calling API: $list_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$list_url")

    log "워크스페이스와 역할 목록: $response"
}

# 특정 사용자의 워크스페이스 역할 목록 조회
list_user_workspace_roles() {
    read -p "워크스페이스 ID: " workspace_id
    read -p "사용자 ID: " username
    
    log "사용자의 워크스페이스 역할 목록 조회 중..."
        
    list_url="$MCIAMMANAGER_HOST/api/workspaces/$workspace_id/users/$username/roles"
    echo "Calling API: $list_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$list_url")


    log "역할 목록: $response"
}

# 모든 사용자의 워크스페이스와 역할 목록 조회 (Platform Admin 전용)
list_all_users_workspaces() {
    log "모든 사용자의 워크스페이스와 역할 목록 조회 중..."

    list_url="$MCIAMMANAGER_HOST/api/workspaces/userrole"
    echo "Calling API: $list_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$list_url")

    log "모든 사용자의 워크스페이스와 역할 목록: $response"
}

# 워크스페이스에 할당된 사용자와 역할 목록 조회
list_workspace_users() {
    read -p "워크스페이스 ID: " workspace_id
    
    log "워크스페이스 사용자와 역할 목록 조회 중..."

    list_url="$MCIAMMANAGER_HOST/api/workspaces/id/$workspace_id/users"
    echo "Calling API: $list_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$list_url")

    log "사용자와 역할 목록: $response"
}

# 메인 메뉴
while true; do
    echo
    log "=== 워크스페이스 역할 관리 테스트 ==="
    log "1. Platform Admin 로그인"
    log "2. 일반 사용자 로그인"
    log "3. 워크스페이스 조회"
    log "4. 워크스페이스에 사용자 역할 할당"
    log "5. 워크스페이스에서 사용자 역할 해제"
    log "6. 사용자의 워크스페이스와 역할 목록 조회"
    log "7. 모든 사용자의 워크스페이스와 역할 목록 조회"
    log "8. 워크스페이스에 할당된 사용자와 역할 목록 조회"
    log "9. 종료"
    read -p "선택 (1-9): " choice

    case $choice in
        1)
            login_as_platform_admin
            ;;
        2)
            login_as_user
            ;;
        3)
            list_workspaces
            ;;
        4)
            assign_workspace_role
            ;;
        5)
            remove_workspace_role
            ;;
        6)
            list_user_workspaces
            ;;
        7)
            list_all_users_workspaces
            ;;
        8)
            list_workspace_users
            ;;
        9)
            log "프로그램을 종료합니다."
            exit 0
            ;;
        *)
            log "잘못된 선택입니다. 다시 시도하세요."
            ;;
    esac
done 
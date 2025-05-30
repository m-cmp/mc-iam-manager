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
    read -p "비밀번호: " password
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

# 유저에게 workspaceRole 추가 ( 관리자 기능)

# workspace-role 과 csp-role mapping 조회 ( 관리자 기능)

# workspace-role 과 csp-role mapping 추가 ( 관리자 기능)


# GetTemporaryCredentials API 호출
get_temporary_credentials() {
    read -p "워크스페이스 ID: " workspaceId
    read -p "대상 csp: " cspType
    
    url="$MCIAMMANAGER_HOST/api/workspaces/temporaryCredentials"
    log "API 호출: $url"
    log "요청 파라미터: workspaceId=${workspaceId}, cspType=${cspType}"
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"workspaceId\":${workspaceId},\"cspType\":\"${cspType}\"}" \
        "$url")
    log "임시자격증명 발급 응답: $response"
}


# 메인 메뉴
while true; do
    echo
    log "=== 임시자격증명 발급 테스트 ==="
    log "1. Platform Admin 로그인"
    log "2. 일반 사용자 로그인"
    log "3. 임시자격증명 발급"
    log "0. 종료"
    read -p "선택 (1-9): " choice

    case $choice in
        1)
            login_as_platform_admin
            ;;
        2)
            login_as_user
            ;;
        3)
            get_temporary_credentials
            ;;        
        0)
            log "프로그램을 종료합니다."
            exit 0
            ;;
        *)
            log "잘못된 선택입니다. 다시 시도하세요."
            ;;
    esac
done 
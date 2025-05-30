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

# 모든 CSP 역할 조회
get_all_csp_roles() {
    log "모든 CSP 역할 조회 중..."
    response=$(curl -s -X GET "$MCIAMMANAGER_HOST/api/csp-roles/all" \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN")
    log "CSP 역할 목록: $response"
}

# CSP 역할 목록 조회
get_csp_roles() {
    log "CSP 역할 목록 조회 중..."
    response=$(curl -s -X GET "$MCIAMMANAGER_HOST/api/csp-roles" \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN")
    log "CSP 역할 목록: $response"
}

# 새로운 CSP 역할 생성
create_csp_role() {
    log "새로운 CSP 역할 생성 중..."
    response=$(curl -s -X POST "$MCIAMMANAGER_HOST/api/csp-roles" \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "mciam-test-csp-role02",
            "description": "Test CSP Role",
            "cspType": "AWS"
            
        }')
    log "생성된 역할: $response"
    echo "$response" | jq -r '.id'
}
# "roleArn": "arn:aws:iam::050864702683:role/mciam-test-csp-role"

# CSP 역할 수정
update_csp_role() {
    local role_id=$1
    log "CSP 역할 수정 중..."
    response=$(curl -s -X PUT "$MCIAMMANAGER_HOST/api/csp-roles/$role_id" \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data '{
            "name": "test-csp-role-updated",
            "description": "Updated Test CSP Role",
            "cspType": "AWS",
            "roleArn": "arn:aws:iam::123456789012:role/test-role-updated"
        }')
    log "수정된 역할: $response"
}

# CSP 역할 삭제
delete_csp_role() {
    local role_id=$1
    log "CSP 역할 삭제 중..."
    response=$(curl -s -X DELETE "$MCIAMMANAGER_HOST/api/csp-roles/$role_id" \
        --header "Authorization: Bearer $MCIAMMANAGER_ACCESSTOKEN")
    log "삭제 응답: $response"
}

# CSP 역할에 권한 추가
add_permissions_to_csp_role() {
    local role_id=$1
    local permissions='["s3:GetObject", "s3:PutObject", "s3:ListBucket"]'
    
    log "CSP 역할에 권한 추가 중..."
    curl -s -X POST "${API_URL}/api/v1/csp-roles/${role_id}/permissions" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "${permissions}" | jq .
}

# CSP 역할에서 권한 제거
remove_permissions_from_csp_role() {
    local role_id=$1
    local permissions='["s3:PutObject"]'
    
    log "CSP 역할에서 권한 제거 중..."
    curl -s -X DELETE "${API_URL}/api/v1/csp-roles/${role_id}/permissions" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "${permissions}" | jq .
}

# CSP 역할의 권한 조회
get_csp_role_permissions() {
    local role_id=$1
    
    log "CSP 역할의 권한 조회 중..."
    curl -s -X GET "${API_URL}/api/v1/csp-roles/${role_id}/permissions" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" | jq .
}

# 메인 메뉴
while true; do
    echo
    log "=== CSP 역할 관리 테스트 ==="
    log "1. Platform Admin 로그인"
    log "2. 모든 CSP 역할 조회"
    log "3. CSP 역할 목록 조회"
    log "4. 새로운 CSP 역할 생성"
    log "5. CSP 역할 수정"
    log "6. CSP 역할 삭제"
    log "7. CSP 역할에 권한 추가"
    log "8. CSP 역할에서 권한 제거"
    log "9. CSP 역할의 권한 조회"
    log "0. 종료"
    read -p "선택 (0-9): " choice

    case $choice in
        1)
            login
            ;;
        2)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                get_all_csp_roles
            fi
            ;;
        3)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                get_csp_roles
            fi
            ;;
        4)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                role_id=$(create_csp_role)
                if [ ! -z "$role_id" ] && [ "$role_id" != "null" ]; then
                    log "생성된 역할 ID: $role_id"
                else
                    log "역할 생성 실패"
                fi
            fi
            ;;
        5)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                read -p "수정할 역할 ID: " role_id
                update_csp_role "$role_id"
            fi
            ;;
        6)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                read -p "삭제할 역할 ID: " role_id
                delete_csp_role "$role_id"
            fi
            ;;
        7)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                read -p "권한을 추가할 역할 ID: " role_id
                add_permissions_to_csp_role "$role_id"
            fi
            ;;
        8)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                read -p "권한을 제거할 역할 ID: " role_id
                remove_permissions_from_csp_role "$role_id"
            fi
            ;;
        9)
            if [ -z "$MCIAMMANAGER_ACCESSTOKEN" ]; then
                log "먼저 로그인해주세요 (옵션 1)"
            else
                read -p "권한을 조회할 역할 ID: " role_id
                get_csp_role_permissions "$role_id"
            fi
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
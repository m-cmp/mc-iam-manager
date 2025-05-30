#!/bin/bash

source ../../.env

login() {
    echo "Logging in as platformadmin..."
    login_url="$MCIAMMANAGER_HOST/api/auth/login"
    echo "Calling API: $login_url"
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MCIAMMANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MCIAMMANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$login_url")
    MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    echo "Login response: $response"
    echo "Login successful"
}

get_workspace_id() {
    local workspace_name=$1
    workspace_url="$MCIAMMANAGER_HOST/api/workspaces/name/$workspace_name"
    echo "Calling API: $workspace_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$workspace_url")
    echo "$response" | jq -r '.id'
}

assign_workspace_role() {
    local workspace_id=$1
    local username=$2
    local role=$3

    workspace_url="$MCIAMMANAGER_HOST/api/workspaces/id/$workspace_id/users/$username/roles/$role"
    echo "Calling API: $workspace_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$workspace_url")
    echo "Workspace role assignment response: $response"
}

# 자동으로 platformAdmin 로그인
login

# 워크스페이스 ID 가져오기
workspace_id=$(get_workspace_id "testws01")

# 워크스페이스 역할 할당
echo "Assigning workspace roles..."

# 프로필 1: 관리자
assign_workspace_role "$workspace_id" "testadmin01" "admin"

# 프로필 2: 운영자
assign_workspace_role "$workspace_id" "testoperator01" "operator"

# 프로필 3: 뷰어
assign_workspace_role "$workspace_id" "testviewer01" "viewer"

# 프로필 4: 재정관리자
assign_workspace_role "$workspace_id" "testbilladmin01" "billadmin"

# 프로필 5: 재정뷰어
assign_workspace_role "$workspace_id" "testbillviewer01" "billviewer"

echo "Workspace roles assigned successfully" 
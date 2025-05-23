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

create_workspace_role() {
    local name=$1
    local description=$2

    json_data=$(jq -n --arg name "$name" --arg description "$description" \
        '{name: $name, description: $description}')
    
    workspace_roles_url="$MCIAMMANAGER_HOST/api/workspace-roles"
    echo "Calling API: $workspace_roles_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$workspace_roles_url")
    echo "Workspace role creation response: $response"
    echo "$response" | jq -r '.id'
}

map_workspace_csp_roles() {
    local workspace_role_id=$1
    local csp_type=$2
    local csp_role_arn=$3
    local idp_identifier=$4
    local description=$5

    json_data=$(jq -n --arg csp_type "$csp_type" --arg csp_role_arn "$csp_role_arn" \
        --arg idp_identifier "$idp_identifier" --arg description "$description" \
        '{csp_type: $csp_type, csp_role_arn: $csp_role_arn, idp_identifier: $idp_identifier, description: $description}')
    
    mapping_url="$MCIAMMANAGER_HOST/api/workspace-roles/$workspace_role_id/csp-role-mappings"
    echo "Calling API: $mapping_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$mapping_url")
    echo "Workspace-CSP role mapping response: $response"
}

delete_workspace_csp_role_mapping() {
    local workspace_role_id=$1
    local csp_type=$2
    local csp_role_arn=$3

    mapping_url="$MCIAMMANAGER_HOST/api/workspace-roles/$workspace_role_id/csp-role-mappings/$csp_type/$csp_role_arn"
    echo "Calling API: $mapping_url"
    response=$(curl -s -X DELETE \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$mapping_url")
    echo "Delete workspace-CSP role mapping response: $response"
}

# 자동으로 platformAdmin 로그인
login

# 새로운 워크스페이스 역할 추가
echo "Creating new workspace role..."
observer_role_id=$(create_workspace_role "observer" "Observer role for monitoring")

# 워크스페이스 역할과 CSP 역할 매핑
echo "Mapping workspace role to CSP role..."
map_workspace_csp_roles "$observer_role_id" "aws" "arn:aws:iam::ACCOUNT_ID:role/MCMP_observer" \
    "arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME" \
    "Mapping for Workspace Observer to AWS MCMP_observer"

# 워크스페이스 역할과 CSP 역할 매핑 해제
echo "Unmapping workspace role from CSP role..."
delete_workspace_csp_role_mapping "$observer_role_id" "aws" "arn:aws:iam::ACCOUNT_ID:role/MCMP_observer"

echo "Workspace role management completed successfully" 
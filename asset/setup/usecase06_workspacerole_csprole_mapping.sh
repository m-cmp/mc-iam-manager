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

get_workspace_roles() {
    echo "Getting workspace roles..."
    workspace_roles_url="$MCIAMMANAGER_HOST/api/workspace-roles"
    echo "Calling API: $workspace_roles_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$workspace_roles_url")
    echo "Raw API Response:"
    echo "$response"
    echo "-------------------"
    echo "Workspace roles:"
    echo "$response" | jq '.'
    return 0
}

get_allcsp_roles() {
    echo "Getting all CSP roles..."
    csp_roles_url="$MCIAMMANAGER_HOST/api/csp-roles/all"
    echo "Calling API: $csp_roles_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$csp_roles_url")
    echo "Raw API Response:"
    echo "$response"
    echo "-------------------"
    echo "CSP roles:"
    echo "$response" | jq '.'
    return 0
}

get_csp_roles() {
    echo "Getting CSP roles..."
    csp_roles_url="$MCIAMMANAGER_HOST/api/csp-roles"
    echo "Calling API: $csp_roles_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$csp_roles_url")
    echo "Raw API Response:"
    echo "$response"
    echo "-------------------"
    echo "CSP roles:"
    echo "$response" | jq '.'
    return 0
}

create_csp_role() {
    local name=$1
    local description=$2
    echo "Creating csp role: $name"
    json_data=$(jq -n --arg name "$name" --arg description "$description" \
        '{name: $name, description: $description}')
    
    csp_roles_url="$MCIAMMANAGER_HOST/api/csp-roles"
    echo "Calling API: $csp_roles_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$csp_roles_url")
    echo "CSP role creation response: $response"
}

map_workspace_csp_roles() {
    local workspace_role_id=$1
    local csp_type=$2
    local csp_role_arn=$3
    local idp_identifier=$4
    local description=$5

    echo "Mapping workspace role $workspace_role_id to CSP role $csp_role_arn"
    json_data=$(jq -n --arg csp_type "$csp_type" --arg csp_role_arn "$csp_role_arn" \
        --arg idp_identifier "$idp_identifier" --arg description "$description" \
        '{csp_type: $csp_type, csp_role_arn: $csp_role_arn, idp_identifier: $idp_identifier, description: $description}')
    
    mapping_url="$MCIAMMANAGER_HOST/api/workspace-roles/$workspace_role_id/csp-role-mappings"
    echo "Calling API: $mapping_url"
    echo "Request data: $json_data"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$mapping_url")
    echo "Workspace-CSP role mapping response: $response"
}

map_all_workspace_csp_roles() {
    echo "Mapping all workspace roles to CSP roles..."

    # admin -> mcmp_admin
    map_workspace_csp_roles "1" "aws" "arn:aws:iam::${AWS_ACCOUNT_ID}:role/MCMP_admin" \
        "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${KEYCLOAK_CLIENT_PATH}" \
        "Mapping for Workspace Admin to AWS MCMP_admin"

    # operator -> mcmp_operator
    map_workspace_csp_roles "2" "aws" "arn:aws:iam::${AWS_ACCOUNT_ID}:role/MCMP_operator" \
        "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${KEYCLOAK_CLIENT_PATH}" \
        "Mapping for Workspace Operator to AWS MCMP_operator"

    # viewer -> mcmp_viewer
    map_workspace_csp_roles "3" "aws" "arn:aws:iam::${AWS_ACCOUNT_ID}:role/MCMP_viewer" \
        "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${KEYCLOAK_CLIENT_PATH}" \
        "Mapping for Workspace Viewer to AWS MCMP_viewer"

    # billadmin -> mcmp_billadmin
    map_workspace_csp_roles "4" "aws" "arn:aws:iam::${AWS_ACCOUNT_ID}:role/MCMP_billadmin" \
        "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${KEYCLOAK_CLIENT_PATH}" \
        "Mapping for Workspace Bill Admin to AWS MCMP_billadmin"

    # billviewer -> mcmp_billviewer
    map_workspace_csp_roles "5" "aws" "arn:aws:iam::${AWS_ACCOUNT_ID}:role/MCMP_billviewer" \
        "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${KEYCLOAK_CLIENT_PATH}" \
        "Mapping for Workspace Bill Viewer to AWS MCMP_billviewer"

    echo "Workspace-CSP role mapping completed successfully"
}

# 메인 메뉴
while true; do
    echo
    echo "=== MCMP IAM Manager - Workspace Role Management ==="
    echo "1. Login as platformadmin"
    echo "2. Get workspace roles"
    echo "3. Get ALL CSP roles"
    echo "4. Get MCMP CSP roles"
    echo "5. Create csp role"
    echo "6. Map workspace role to CSP role"
    echo "7. Map all workspace roles to CSP roles"
    echo "0. Exit"
    echo "================================================"
    echo -n "Enter your choice: "
    read choice

    case $choice in
        1)
            login
            ;;
        2)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                get_workspace_roles
            fi
            ;;
        3)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                get_allcsp_roles
            fi
            ;;
        4)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                get_csp_roles
            fi
            ;;
        5)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                echo -n "Enter role name: "
                read role_name
                echo -n "Enter role description: "
                read role_description
                create_csp_role "$role_name" "$role_description"
            fi
            ;;
        6)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                echo -n "Enter workspace role ID: "
                read workspace_role_id
                echo -n "Enter CSP type (e.g., aws): "
                read csp_type
                echo -n "Enter CSP role ARN: "
                read csp_role_arn
                echo -n "Enter IDP identifier: "
                read idp_identifier
                echo -n "Enter description: "
                read description
                map_workspace_csp_roles "$workspace_role_id" "$csp_type" "$csp_role_arn" "$idp_identifier" "$description"
            fi
            ;;
        7)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                map_all_workspace_csp_roles
            fi
            ;;
        0)
            echo "Exiting..."
            exit 0
            ;;
        *)
            echo "Invalid option. Please try again."
            ;;
    esac
done 
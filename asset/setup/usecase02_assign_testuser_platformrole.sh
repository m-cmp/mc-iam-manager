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

add_user() {
    echo "Select user profile to create:"
    echo "1. 관리자(admin) - testadmin01"
    echo "2. 운영자(operator) - testoperator01"
    echo "3. 뷰어(viewer) - testviewer01"
    echo "4. 재정관리자(billadmin) - testbilladmin01"
    echo "5. 재정뷰어(billviewer) - testbillviewer01"
    
    read -p "Enter your choice (1-5): " profile_choice
    
    case $profile_choice in
        1)
            username="testadmin01"
            email="testadmin01@test.com"
            firstName="ta"
            lastName="01"
            ;;
        2)
            username="testoperator01"
            email="testoperator01@test.com"
            firstName="to"
            lastName="01"
            ;;
        3)
            username="testviewer01"
            email="testviewer01@test.com"
            firstName="tv"
            lastName="01"
            ;;
        4)
            username="testbilladmin01"
            email="testbilladmin01@test.com"
            firstName="tba"
            lastName="01"
            ;;
        5)
            username="testbillviewer01"
            email="testbillviewer01@test.com"
            firstName="tbv"
            lastName="01"
            ;;
        *)
            echo "Invalid choice"
            return
            ;;
    esac
    
    json_data=$(jq -n --arg username "$username" --arg email "$email" --arg firstName "$firstName" --arg lastName "$lastName" \
        '{username: $username, email: $email, firstName: $firstName, lastName: $lastName}')
    user_url="$MCIAMMANAGER_HOST/api/users"
    echo "Calling API: $user_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$user_url")
    echo "User addition response: $response"
    echo "User added successfully"
}

assign_platform_role() {
    local username=$1
    local role=$2

    json_data=$(jq -n --arg username "$username" --arg role "$role" \
        '{username: $username, role: $role}')
    
    platform_role_url="$MCIAMMANAGER_HOST/api/users/assign/platform-roles"
    echo "Calling API: $platform_role_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$platform_role_url")
    echo "Platform role assignment response: $response"
}

assign_workspace_role() {
    read -p "Enter user ID: " user_id
    read -p "Enter workspace ID: " workspace_id
    read -p "Enter workspace role (admin/operator/viewer): " workspace_role
    
    json_data=$(jq -n --arg user_id "$user_id" --arg workspace_id "$workspace_id" --arg role "$workspace_role" \
        '{user_id: $user_id, workspace_id: $workspace_id, role: $role}')
    workspace_role_url="$MCIAMMANAGER_HOST/api/users/assign/workspace-roles"
    echo "Calling API: $workspace_role_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$workspace_role_url")
    echo "Workspace role assignment response: $response"
    echo "Workspace role assigned successfully"
}

# 자동으로 platformAdmin 로그인
login

# 플랫폼 역할 할당
echo "Assigning platform roles..."

# 프로필 1: 관리자
assign_platform_role "testadmin01" "admin"

# 프로필 2: 운영자
assign_platform_role "testoperator01" "operator"

# 프로필 3: 뷰어
assign_platform_role "testviewer01" "viewer"

# 프로필 4: 재정관리자
assign_platform_role "testbilladmin01" "billadmin"

# 프로필 5: 재정뷰어
assign_platform_role "testbillviewer01" "billviewer"

echo "Platform roles assigned successfully"

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

list_workspaces() {
    list_url="$MCIAMMANAGER_HOST/api/workspaces"
    echo "Calling API: $list_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$list_url")
    echo "Workspace list response:"
    echo "$response" | jq '.'
}

list_all_workspaces() {
    list_url="$MCIAMMANAGER_HOST/api/workspaces/all"
    echo "Calling API: $list_url"
    response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$list_url")
    echo "All workspaces with projects response:"
    echo "$response" | jq '.'
}

create_workspace() {
    local name=$1
    local description=$2

    # 워크스페이스 이름 중복 체크
    check_url="$MCIAMMANAGER_HOST/api/workspaces/name/$name"
    check_response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$check_url")
    
    if [ "$(echo "$check_response" | jq -r '.id')" != "null" ]; then
        echo "Error: Workspace with name '$name' already exists"
        return 1
    fi

    json_data=$(jq -n --arg name "$name" --arg description "$description" \
        '{name: $name, description: $description}')
    
    workspace_url="$MCIAMMANAGER_HOST/api/workspaces"
    echo "Calling API: $workspace_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$workspace_url")
    echo "Workspace creation response: $response"
    echo "$response" | jq -r '.id'
}

create_project() {
    local name=$1
    local description=$2

    # 프로젝트 이름 중복 체크
    check_url="$MCIAMMANAGER_HOST/api/projects/name/$name"
    check_response=$(curl -s -X GET \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$check_url")
    
    if [ "$(echo "$check_response" | jq -r '.id')" != "null" ]; then
        echo "Error: Project with name '$name' already exists"
        return 1
    fi

    json_data=$(jq -n --arg name "$name" --arg description "$description" \
        '{name: $name, description: $description}')
    
    project_url="$MCIAMMANAGER_HOST/api/projects"
    echo "Calling API: $project_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$project_url")
    echo "Project creation response: $response"
    echo "$response" | jq -r '.id'
}

add_project_to_workspace() {
    local workspace_id=$1
    local project_id=$2

    add_url="$MCIAMMANAGER_HOST/api/workspaces/id/$workspace_id/projects/$project_id"
    echo "Calling API: $add_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$add_url")
    echo "Add project to workspace response: $response"
}

remove_project_from_workspace() {
    local workspace_id=$1
    local project_id=$2

    remove_url="$MCIAMMANAGER_HOST/api/workspaces/id/$workspace_id/projects/$project_id"
    echo "Calling API: $remove_url"
    response=$(curl -s -X DELETE \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$remove_url")
    echo "Remove project from workspace response: $response"
}

# 자동으로 platformAdmin 로그인
login

# 단계별 실행을 위한 메뉴
while true; do
    echo "=== Workspace and Project Mapping Menu ==="
    echo "1. List Workspaces"
    echo "2. List All Workspaces with Projects"
    echo "3. Create Workspace"
    echo "4. Create Project"
    echo "5. Add Project to Workspace"
    echo "6. Remove Project from Workspace"
    echo "7. Exit"
    echo "========================================"
    read -p "Select an option (1-7): " choice

    case $choice in
        1)
            list_workspaces
            ;;
        2)
            list_all_workspaces
            ;;
        3)
            read -p "Enter workspace name: " ws_name
            read -p "Enter workspace description: " ws_desc
            create_workspace "$ws_name" "$ws_desc"
            ;;
        4)
            read -p "Enter project name: " prj_name
            read -p "Enter project description: " prj_desc
            create_project "$prj_name" "$prj_desc"
            ;;
        5)
            read -p "Enter workspace ID: " ws_id
            read -p "Enter project ID: " prj_id
            add_project_to_workspace "$ws_id" "$prj_id"
            ;;
        6)
            read -p "Enter workspace ID: " ws_id
            read -p "Enter project ID: " prj_id
            remove_project_from_workspace "$ws_id" "$prj_id"
            ;;
        7)
            echo "Exiting..."
            exit 0
            ;;
        *)
            echo "Invalid option. Please try again."
            ;;
    esac

    echo
    read -p "Press Enter to continue..."
done 
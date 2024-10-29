#!/bin/bash
source ./.env

# -f 옵션 체크
force_mode=false
while getopts "f" opt; do
    case $opt in
        f) force_mode=true ;;
        *) echo "Usage: $0 [-f]"; exit 1 ;;
    esac
done

login(){
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MCIAMMANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MCIAMMANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$MCIAMMANAGER_HOST/api/auth/login")

    MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
        echo "Login failed."
        $force_mode || exit 1
    else
        echo "Login successful."
    fi
}

initRoleData(){
    IFS=',' read -r -a roles <<< "$PREDEFINED_ROLE"
    local first_role=true

    for role in "${roles[@]}"; do
        json_data=$(jq -n --arg name "$role" --arg description "$role Role" \
        '{name: $name, description: $description}')
        
        local response=$(curl -s -w "\n%{http_code}" --location \
        --header 'Content-Type: application/json' \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --data "$json_data" \
        "$MCIAMMANAGER_HOST/api/role")

        local http_code=$(echo "$response" | tail -n1)
        local json_data=$(echo "$response" | head -n -1)

        if [ "$http_code" -ne 200 ]; then
            echo "Failed to create role: $role"
            $force_mode || exit 1
        else
            echo "Role created successfully: $role"
            
            # 첫 번째 요청에서만 ROLE_ID에 id 저장
            if [ "$first_role" = true ]; then
                ROLE_ID=$(echo "$json_data" | jq -r '.id')
                echo "First Role ID saved as ROLE_ID: $ROLE_ID"
                first_role=false
            fi
        fi
    done
}

initMenuDatafromMenuYaml(){
    wget -q -O ./mcwebconsoleMenu.yaml "$MCWEBCONSOLE_MENUYAML"
    if [ $? -ne 0 ]; then
        echo "Failed to download mcwebconsoleMenu.yaml"
        $force_mode || exit 1
    else
        echo "Downloaded mcwebconsoleMenu.yaml successfully."
    fi

    response=$(curl -s -o /dev/null -w "%{http_code}" --location \
    "$MCIAMMANAGER_HOST/api/resource/file/framework/mc-web-console/menu" \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --form "file=@./mcwebconsoleMenu.yaml")

    if [ "$response" -ne 200 ]; then
        echo "Failed to upload mcwebconsoleMenu.yaml"
        $force_mode || exit 1
    else
        echo "Uploaded mcwebconsoleMenu.yaml successfully."
    fi
}

initMenuPermissionCSV(){
    wget -q -O ./permission.csv "$MCWEBCONSOLE_MENU_PERMISSIONS"
    if [ $? -ne 0 ]; then
        echo "Failed to download permission.csv"
        $force_mode || exit 1
    else
        echo "Downloaded permission.csv successfully."
    fi

    response=$(curl -s -o /dev/null -w "%{http_code}" --location \
    "$MCIAMMANAGER_HOST/api/permission/file/framework/all" \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --form "file=@./permission.csv")

    if [ "$response" -ne 200 ]; then
        echo "Failed to upload permission.csv"
        $force_mode || exit 1
    else
        echo "Uploaded permission.csv successfully."
    fi
}

createWorkspace() {
    local response=$(curl -s -w "\n%{http_code}" --location \
    "$MCIAMMANAGER_HOST/api/ws" \
    --header "Content-Type: application/json" \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --data '{
      "name": "workspace1",
      "description": "workspace1 desc"
    }')

    local http_code=$(echo "$response" | tail -n1)
    local json_data=$(echo "$response" | head -n -1)

    echo $json_data $http_code
    
    # 상태 코드에 따른 처리
    if [ "$http_code" -ne 200 ]; then
        echo "Failed to create workspace"
        $force_mode || exit 1
    else
        WORKSPACE_ID=$(echo "$json_data" | jq -r '.id')
        echo "Workspace created successfully. ID: $WORKSPACE_ID"
    fi
}

createProject() {
    local response=$(curl -s -w "\n%{http_code}" --location \
    "$MCIAMMANAGER_HOST/api/prj" \
    --header "Content-Type: application/json" \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --data '{
      "name": "project1",
      "description": "project1 desc"
    }')

    local http_code=$(echo "$response" | tail -n1)
    local json_data=$(echo "$response" | head -n -1)

    echo $json_data $http_code

    if [ "$http_code" -ne 200 ]; then
        echo "Failed to create project"
        $force_mode || exit 1
    else
        PROJECT_ID=$(echo "$json_data" | jq -r '.id')
        echo "Project created successfully. ID: $PROJECT_ID"
    fi
}

assignProjectToWorkspace() {
    local workspace_id="$WORKSPACE_ID"
    local project_id="$PROJECT_ID"

    json_data=$(jq -n --arg workspaceId "$workspace_id" --arg projectId "$project_id" \
    '{workspaceId: $workspaceId, projectIds: [$projectId]}')

    local response=$(curl -s -w "\n%{http_code}" --location \
    --location "$MCIAMMANAGER_HOST/api/wsprj" \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --data "$json_data")

    local http_code=$(echo "$response" | tail -n1)
    local json_data=$(echo "$response" | head -n -1)

    echo $json_data $http_code

    if [ "$http_code" -ne 200 ]; then
        echo "Failed to create project"
        $force_mode || exit 1
    else
        echo "Project Worksapce mapping created successfully. " 
    fi
}

assignUserRoleToWorkspace() {
    local workspace_id="$WORKSPACE_ID"
    local role_id="$ROLE_ID"
    local user_id="$MCIAMMANAGER_PLATFORMADMIN_ID"

    json_data=$(jq -n --arg workspaceId "$workspace_id" --arg roleId "$role_id" --arg userId "$user_id" \
    '{workspaceId: $workspaceId, roleId: $roleId, userId: $userId}')

    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --data "$json_data" \
    "$MCIAMMANAGER_HOST/api/wsuserrole")

    if [ "$response" -ne 200 ]; then
        echo "Failed to assign user role to workspace"
        $force_mode || exit 1
    else
        echo "User role assigned to workspace successfully"
    fi
}

login

initRoleData

initMenuDatafromMenuYaml
initMenuPermissionCSV

createWorkspace
createProject

assignProjectToWorkspace
assignUserRoleToWorkspace
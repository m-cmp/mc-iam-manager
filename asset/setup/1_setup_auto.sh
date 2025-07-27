#!/bin/bash

source ../../.env

# 자동화된 설정 함수
auto_setup() {
    echo "=== Starting automated setup process ==="
    
    # 1. 플랫폼 어드민 초기화
    echo "Step 1: Initializing platform admin..."
    init_platform_admin
    if [ $? -ne 0 ]; then
        echo "ERROR: Platform admin initialization failed"
        return 1
    fi
    echo "✓ Platform admin initialized successfully"
    
    # 2. 로그인
    echo "Step 2: Logging in..."
    login
    if [ $? -ne 0 ]; then
        echo "ERROR: Login failed"
        return 1
    fi
    echo "✓ Login successful"
    
    # 3. 역할 데이터 초기화
    echo "Step 3: Initializing predefined roles..."
    init_predefined_roles
    if [ $? -ne 0 ]; then
        echo "ERROR: Role initialization failed"
        return 1
    fi
    echo "✓ Predefined roles initialized successfully"
    
    # 4. 메뉴 데이터 초기화
    echo "Step 4: Initializing menu data..."
    init_menu
    if [ $? -ne 0 ]; then
        echo "ERROR: Menu initialization failed"
        return 1
    fi
    echo "✓ Menu data initialized successfully"
    
    # 5. API 리소스 데이터 초기화
    echo "Step 5: Initializing API resources..."
    init_api_resources
    if [ $? -ne 0 ]; then
        echo "ERROR: API resources initialization failed"
        return 1
    fi
    echo "✓ API resources initialized successfully"
    
    # 6. 프로젝트 동기화
    echo "Step 6: Syncing projects..."
    sync_projects
    if [ $? -ne 0 ]; then
        echo "ERROR: Project sync failed"
        return 1
    fi
    echo "✓ Projects synced successfully"
    
    # 7. 워크스페이스-프로젝트 매핑
    echo "Step 7: Mapping workspace to all projects..."
    map_workspace_projects
    if [ $? -ne 0 ]; then
        echo "ERROR: Workspace-project mapping failed"
        return 1
    fi
    echo "✓ Workspace-project mapping completed successfully"
    
    echo "=== Automated setup completed successfully ==="
}

init_platform_admin() {
    echo "Initializing platform admin..."
    
    # 환경 변수 사용
    json_data=$(jq -n \
        --arg email "$MC_IAM_MANAGER_PLATFORMADMIN_EMAIL" \
        --arg password "$MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD" \
        --arg username "$MC_IAM_MANAGER_PLATFORMADMIN_ID" \
        '{email: $email, password: $password, username: $username}')
    
    response=$(curl -s -X POST \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MC_IAM_MANAGER_HOST/api/initial-admin")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to make request to platform admin API"
        return 1
    fi
    
    echo "Platform admin initialization response: $response"
    
    # 성공 여부 확인 (응답에 에러가 없는지 확인)
    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Platform admin initialization failed"
        return 1
    fi
    
    return 0
}

login() {
    echo "Logging in with platform admin credentials from .env file..."
    
    # 환경 변수에서 플랫폼 어드민 정보 사용
    if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ID" ] || [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD" ]; then
        echo "ERROR: Platform admin credentials not found in .env file"
        echo "Please check MC_IAM_MANAGER_PLATFORMADMIN_ID and MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD in .env"
        return 1
    fi
    
    echo "Using platform admin ID: $MC_IAM_MANAGER_PLATFORMADMIN_ID"
    
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MC_IAM_MANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$MC_IAM_MANAGER_HOST/api/auth/login")
    
    echo "Login response: $response"
    
    # 디버깅: jq가 설치되어 있는지 확인
    if ! command -v jq &> /dev/null; then
        echo "ERROR: jq is not installed. Please install jq first."
        return 1
    fi
    
    # 디버깅: 응답이 유효한 JSON인지 확인
    if ! echo "$response" | jq . > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON response"
        echo "Raw response: $response"
        return 1
    fi
    
    # 디버깅: access_token 필드가 있는지 확인
    if ! echo "$response" | jq -e '.access_token' > /dev/null 2>&1; then
        echo "ERROR: access_token field not found in response"
        echo "Available fields:"
        echo "$response" | jq 'keys'
        return 1
    fi
    
    MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    
    # 디버깅: 토큰이 제대로 추출되었는지 확인
    if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ] || [ "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" = "null" ]; then
        echo "ERROR: Failed to extract access token"
        echo "Extracted token: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
        return 1
    fi
    
    echo "Access token extracted successfully: ${MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN:0:20}..."
    echo "Login successful"
    return 0
}

init_predefined_roles() {
    echo "Initializing platform roles..."
    IFS=',' read -ra ROLES <<< "$PREDEFINED_ROLE"
    for role in "${ROLES[@]}"; do
        echo "Creating role: $role"
        json_data=$(jq -n --arg name "$role" --arg description "$role Role" \
            '{name: $name, description: $description, role_types: ["workspace", "platform"]}')
        response=$(curl -s -X POST \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
            --data "$json_data" \
            "$MC_IAM_MANAGER_HOST/api/roles")
        
        # 응답 검증
        if [ $? -ne 0 ]; then
            echo "ERROR: Failed to create role: $role"
            return 1
        fi
        
        echo "Response for role $role: $response"
        
        # 성공 여부 확인
        if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
            echo "ERROR: Failed to create role: $role"
            return 1
        fi
    done
    echo "Platform roles initialized"
    return 0
}

init_menu() {
    echo "Initializing menu data..."
    wget -q -O ./menu.yaml "$MCWEBCONSOLE_MENUYAML"
    
    # wget 성공 여부 확인
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to download menu.yaml"
        return 1
    fi
    
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/setup/initial-menus")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to initialize menu data"
        return 1
    fi
    
    echo "Menu initialization response: $response"
    
    # 성공 여부 확인
    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Menu initialization failed"
        return 1
    fi
    
    echo "Menu data initialized"
    return 0
}

init_api_resources() {
    echo "Initializing API resources..."
    wget -q -O ./api.yaml "$MCADMINCLI_APIYAML"
    
    # wget 성공 여부 확인
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to download api.yaml"
        return 1
    fi
    
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/setup/sync-mcmp-apis")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to initialize API resources"
        return 1
    fi
    
    echo "API resources initialization response: $response"
    
    # 성공 여부 확인
    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: API resources initialization failed"
        return 1
    fi
    
    echo "API resources initialized"
    return 0
}

init_cloud_resources() {
    echo "Initializing cloud resources..."
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: multipart/form-data' \
        --form "file=@./cloud-resource.yaml" \
        "$MC_IAM_MANAGER_HOST/api/resource/file/framework/all")
    echo "Cloud resources initialization response: $response"
    echo "Cloud resources initialized"
}

map_api_cloud_resources() {
    echo "Mapping API-Cloud resources..."
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/resource/mapping/api-cloud")
    echo "API-Cloud resources mapping response: $response"
    echo "API-Cloud resources mapping completed"
}


map_workspace_csp_roles() {
    echo "Mapping workspace roles to CSP IAM roles..."
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/workspace-roles/csp-mapping")
    echo "Workspace-CSP role mapping response: $response"
    echo "Workspace-CSP role mapping completed"
}


sync_projects() {
    echo "Syncing projects with mc-infra-manager..."
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/projects/sync")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to sync projects"
        return 1
    fi
    
    echo "Project sync response: $response"
    
    # 성공 여부 확인
    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Project sync failed"
        return 1
    fi
    
    echo "Project sync completed"
    return 0
}

map_workspace_projects() {
    echo "Getting workspace list..."
    
    # 워크스페이스 목록 가져오기
    workspace_response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data '{}' \
        "$MC_IAM_MANAGER_HOST/api/workspaces/list")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to get workspace list"
        return 1
    fi
    
    echo "Workspace list response: $workspace_response"
    
    # 첫 번째 워크스페이스 ID 추출
    workspace_id=$(echo "$workspace_response" | jq -r '.[0].id // empty')
    
    if [ -z "$workspace_id" ] || [ "$workspace_id" = "null" ]; then
        echo "ERROR: No workspace found or failed to extract workspace ID"
        return 1
    fi
    
    echo "Using workspace ID: $workspace_id"
    
    # 프로젝트 목록 가져오기
    echo "Getting project list..."
    project_response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data '{}' \
        "$MC_IAM_MANAGER_HOST/api/projects/list")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to get project list"
        return 1
    fi
    
    echo "Project list response: $project_response"
    
    # 모든 프로젝트 ID 추출
    project_ids=$(echo "$project_response" | jq -r '[.[].id]')
    
    if [ -z "$project_ids" ] || [ "$project_ids" = "[]" ]; then
        echo "WARNING: No projects found to assign to workspace"
        return 0
    fi
    
    echo "Found project IDs: $project_ids"
    
    # 워크스페이스에 모든 프로젝트 매핑
    json_data=$(jq -n --arg workspace_id "$workspace_id" --argjson project_ids "$project_ids" \
        '{workspaceId: $workspace_id, projectIds: $project_ids}')
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MC_IAM_MANAGER_HOST/api/workspaces/assign/projects")
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to map workspace to projects"
        return 1
    fi
    
    echo "Workspace-Project mapping response: $response"
    
    # 성공 여부 확인
    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Workspace-project mapping failed"
        return 1
    fi
    
    echo "Workspace-Project mapping completed for workspace ID: $workspace_id"
    return 0
}

# 자동 설정 실행
echo "Starting automated setup process..."
auto_setup

# 자동 설정 완료 후 종료
if [ $? -eq 0 ]; then
    echo "Setup completed successfully!"
    exit 0
else
    echo "Setup failed with errors!"
    exit 1
fi 
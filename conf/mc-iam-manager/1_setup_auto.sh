#!/bin/bash

# UID 변수 충돌 방지 - .env.setup 파일 사용
source ../../.env.setup

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
    echo "=== Starting Platform Admin Initialization ==="
    echo "Target URL: $MC_IAM_MANAGER_HOST/api/initial-admin"
    echo "Admin Email: $MC_IAM_MANAGER_PLATFORMADMIN_EMAIL"
    echo "Admin Username: $MC_IAM_MANAGER_PLATFORMADMIN_ID"
    
    # 환경 변수 사용
    json_data=$(jq -n \
        --arg email "$MC_IAM_MANAGER_PLATFORMADMIN_EMAIL" \
        --arg password "$MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD" \
        --arg username "$MC_IAM_MANAGER_PLATFORMADMIN_ID" \
        '{email: $email, password: $password, username: $username}')
    
    echo "Request JSON: $json_data"
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MC_IAM_MANAGER_HOST/api/initial-admin")
    
    # HTTP 상태 코드와 응답 본문 분리
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo $response | sed -e 's/HTTPSTATUS\:.*//g')
    
    echo "Platform admin init HTTP Status: $http_code"
    echo "Platform admin init Response Body: $response_body"
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to make request to platform admin API"
        echo "curl exit code: $?"
        return 1
    fi
    
    # HTTP 상태 코드 확인
    if [ "$http_code" != "200" ] && [ "$http_code" != "201" ]; then
        echo "ERROR: Platform admin initialization failed with HTTP status $http_code"
        return 1
    fi
    
    # JSON 응답 검증
    if ! echo "$response_body" | jq . > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON response from platform admin API"
        echo "Raw response: $response_body"
        return 1
    fi
    
    # 성공 여부 확인 (응답에 에러가 없는지 확인)
    if echo "$response_body" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Platform admin initialization failed with error in response"
        echo "Error details:"
        echo "$response_body" | jq '.error'
        return 1
    fi
    
    echo "✓ Platform admin initialized successfully"
    return 0
}

login() {
    echo "=== Starting Login Process ==="
    echo "Target URL: $MC_IAM_MANAGER_HOST/api/auth/login"
    
    # 환경 변수에서 플랫폼 어드민 정보 사용
    if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ID" ] || [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD" ]; then
        echo "ERROR: Platform admin credentials not found in .env file"
        echo "Please check MC_IAM_MANAGER_PLATFORMADMIN_ID and MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD in .env"
        return 1
    fi
    
    echo "Using platform admin ID: $MC_IAM_MANAGER_PLATFORMADMIN_ID"
    
    # 로그인 요청 JSON 생성
    login_json=$(jq -n \
        --arg id "$MC_IAM_MANAGER_PLATFORMADMIN_ID" \
        --arg password "$MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD" \
        '{id: $id, password: $password}')
    
    echo "Login request JSON: $login_json"
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" --location --header 'Content-Type: application/json' \
        --data "$login_json" \
        "$MC_IAM_MANAGER_HOST/api/auth/login")
    
    # HTTP 상태 코드와 응답 본문 분리
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo $response | sed -e 's/HTTPSTATUS\:.*//g')
    
    echo "Login HTTP Status: $http_code"
    echo "Login Response Body: $response_body"
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to make login request"
        echo "curl exit code: $?"
        return 1
    fi
    
    # HTTP 상태 코드 확인
    if [ "$http_code" != "200" ]; then
        echo "ERROR: Login failed with HTTP status $http_code"
        return 1
    fi
    
    # 디버깅: jq가 설치되어 있는지 확인
    if ! command -v jq &> /dev/null; then
        echo "ERROR: jq is not installed. Please install jq first."
        return 1
    fi
    
    # 디버깅: 응답이 유효한 JSON인지 확인
    if ! echo "$response_body" | jq . > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON response"
        echo "Raw response: $response_body"
        return 1
    fi
    
    # 디버깅: access_token 필드가 있는지 확인
    if ! echo "$response_body" | jq -e '.access_token' > /dev/null 2>&1; then
        echo "ERROR: access_token field not found in response"
        echo "Available fields:"
        echo "$response_body" | jq 'keys'
        return 1
    fi
    
    MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response_body" | jq -r '.access_token')"
    
    # 디버깅: 토큰이 제대로 추출되었는지 확인
    if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ] || [ "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" = "null" ]; then
        echo "ERROR: Failed to extract access token"
        echo "Extracted token: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
        return 1
    fi
    
    echo "✓ Access token extracted successfully: ${MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN:0:20}..."
    echo "✓ Login successful"
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
    echo "=== Starting Project Sync Process ==="
    echo "Target URL: $MC_IAM_MANAGER_HOST/api/setup/sync-projects"
    echo "Access Token: ${MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN:0:20}..."
    
    # mc-infra-manager 상태 확인
    echo "Checking mc-infra-manager availability..."
    infra_response=$(curl -s -w "HTTPSTATUS:%{http_code}" "http://mc-infra-manager:1323/tumblebug/readyz")
    infra_http_code=$(echo $infra_response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    infra_body=$(echo $infra_response | sed -e 's/HTTPSTATUS\:.*//g')
    
    echo "mc-infra-manager health check - HTTP Status: $infra_http_code"
    echo "mc-infra-manager health check - Response: $infra_body"
    
    if [ "$infra_http_code" != "200" ]; then
        echo "ERROR: mc-infra-manager is not healthy (HTTP $infra_http_code)"
        echo "This may cause project sync to fail"
    fi
    
    # 프로젝트 동기화 요청
    echo "Making project sync request..."
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/setup/sync-projects")
    
    # HTTP 상태 코드와 응답 본문 분리
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo $response | sed -e 's/HTTPSTATUS\:.*//g')
    
    echo "Project sync HTTP Status: $http_code"
    echo "Project sync Response Body: $response_body"
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to make request to project sync API"
        echo "curl exit code: $?"
        return 1
    fi
    
    # HTTP 상태 코드 확인
    if [ "$http_code" != "200" ]; then
        echo "ERROR: Project sync failed with HTTP status $http_code"
        return 1
    fi
    
    # JSON 응답 검증
    if ! echo "$response_body" | jq . > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON response from project sync API"
        echo "Raw response: $response_body"
        return 1
    fi
    
    # 성공 여부 확인
    if echo "$response_body" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Project sync failed with error in response"
        echo "Error details:"
        echo "$response_body" | jq '.error'
        return 1
    fi
    
    # 성공 시 상세 정보 출력
    echo "✓ Project sync completed successfully"
    echo "Response details:"
    echo "$response_body" | jq .
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
    
    # 모든 프로젝트 ID 추출 (문자열로 변환)
    project_ids=$(echo "$project_response" | jq -r '[.[].id | tostring]')
    
    if [ -z "$project_ids" ] || [ "$project_ids" = "[]" ]; then
        echo "WARNING: No projects found to assign to workspace"
        return 0
    fi
    
    echo "Found project IDs: $project_ids"
    
    # 워크스페이스에 모든 프로젝트 매핑
    json_data=$(jq -n --arg workspace_id "$workspace_id" --argjson project_ids "$project_ids" \
        '{workspaceId: $workspace_id, projectIds: $project_ids}')
    
    echo "Workspace-Project mapping request JSON: $json_data"
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MC_IAM_MANAGER_HOST/api/workspaces/assign/projects")
    
    # HTTP 상태 코드와 응답 본문 분리
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    response_body=$(echo $response | sed -e 's/HTTPSTATUS\:.*//g')
    
    echo "Workspace-Project mapping HTTP Status: $http_code"
    echo "Workspace-Project mapping Response Body: $response_body"
    
    # 응답 검증
    if [ $? -ne 0 ]; then
        echo "ERROR: Failed to make request to workspace-project mapping API"
        echo "curl exit code: $?"
        return 1
    fi
    
    # HTTP 상태 코드 확인
    if [ "$http_code" != "200" ]; then
        echo "ERROR: Workspace-project mapping failed with HTTP status $http_code"
        return 1
    fi
    
    # JSON 응답 검증
    if ! echo "$response_body" | jq . > /dev/null 2>&1; then
        echo "ERROR: Invalid JSON response from workspace-project mapping API"
        echo "Raw response: $response_body"
        return 1
    fi
    
    # 성공 여부 확인
    if echo "$response_body" | jq -e '.error' > /dev/null 2>&1; then
        echo "ERROR: Workspace-project mapping failed with error in response"
        echo "Error details:"
        echo "$response_body" | jq '.error'
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
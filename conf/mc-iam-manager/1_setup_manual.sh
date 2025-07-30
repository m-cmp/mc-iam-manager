#!/bin/bash

source ../../.env

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
    echo "Platform admin initialization response: $response"
}

login() {
    read -p "Enter the platformadmin ID: " MC_IAM_MANAGER_PLATFORMADMIN_ID
    read -s -p "Enter the platformadmin password: " MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD
    echo
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
        echo "Response for role $role: $response"
    done
    echo "Platform roles initialized"
}

init_menu() {
    echo "Initializing menu data..."
    wget -q -O ./menu.yaml "$MCWEBCONSOLE_MENUYAML"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/setup/initial-menus")
    echo "Menu initialization response: $response"
    echo "Menu data initialized"
}

init_api_resources() {
    echo "Initializing API resources..."
    wget -q -O ./api.yaml "$MCADMINCLI_APIYAML"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MC_IAM_MANAGER_HOST/api/setup/sync-mcmp-apis")
    echo "API resources initialization response: $response"
    echo "API resources initialized"
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
    read -p "Enter workspace ID: " workspace_id
    json_data=$(jq -n --arg workspace_id "$workspace_id" --arg all_projects "true" \
        '{workspace_id: $workspace_id, all_projects: $all_projects}')
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MC_IAM_MANAGER_HOST/api/workspaces/projects/mapping")
    echo "Workspace-Project mapping response: $response"
    echo "Workspace-Project mapping completed"
}

while true; do
    echo "Select an option:"
    echo "0. Exit"
    echo "1. Init Platform And PlatformAdmin"
    echo "2. PlatformAdmin Login"
    echo "3. Init Role Data"
    echo "4. Init Menu Data"
    echo "5. Init API Resource Data"
    echo "6. Init Cloud Resource Data"
    echo "7. Map API-Cloud Resources"
    echo "8. Init Workspace Roles"
    echo "9. Map Workspace-CSP Roles"
    echo "10. Sync Projects"
    echo "11. Map Workspace-All Projects"
    
    read -p "Enter your choice (0-8): " choice
    
    case $choice in
        0)
            echo "Exiting..."
            exit 0
            ;;
        1)
            init_platform_admin
            ;;
        2)
            login
            ;;
        3)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 2)"
                echo "Current token value: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
            else
                init_predefined_roles
            fi
            ;;
        4)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 2)"
                echo "Current token value: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
            else
                init_menu
            fi
            ;;
        5)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 2)"
                echo "Current token value: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
            else
                init_api_resources
            fi
            ;;
        7)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 2)"
                echo "Current token value: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
            else
                init_cloud_resources
            fi
            ;;
        8)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 2)"
                echo "Current token value: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
            else
                map_api_cloud_resources
            fi
            ;;
        9)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 2)"
                echo "Current token value: '$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN'"
            else
                map_workspace_csp_roles
            fi
            ;;
        10)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                sync_projects
            fi
            ;;
        11)
            if [ -z "$MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                map_workspace_projects
            fi
            ;;
        *)
            echo "Invalid option. Please try again."
            ;;
    esac
    
    echo
done 
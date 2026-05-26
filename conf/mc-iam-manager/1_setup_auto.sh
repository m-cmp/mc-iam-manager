#!/bin/bash

# UID 변수 충돌 방지 - .env 파일이 있으면 source (없으면 env_file 주입값 사용)
source .env 2>/dev/null || true

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

    # 5-1. 프레임워크 서비스 URL 등록 (sync-projects 전 서비스 레지스트리 선행 등록)
    echo "Step 5-1: Registering framework service URLs..."
    register_framework_services
    if [ $? -ne 0 ]; then
        echo "ERROR: Framework service registration failed"
        return 1
    fi
    echo "✓ Framework services registered successfully"

    # 6. 프로젝트 동기화
    echo "Step 6: Syncing projects..."
    sync_projects
    if [ $? -ne 0 ]; then
        echo "ERROR: Project sync failed"
        return 1
    fi
    echo "✓ Projects synced successfully"
    
    # 7. Keycloak client redirect URI 설정
    echo "Step 7: Configuring Keycloak client redirect URIs..."
    configure_keycloak_client_uris
    if [ $? -ne 0 ]; then
        echo "WARNING: Keycloak client redirect URI configuration failed (non-fatal)"
    else
        echo "✓ Keycloak client redirect URIs configured successfully"
    fi

    # 8. 워크스페이스-프로젝트 매핑
    echo "Step 8: Mapping workspace to all projects..."
    map_workspace_projects
    if [ $? -ne 0 ]; then
        echo "ERROR: Workspace-project mapping failed"
        return 1
    fi
    echo "✓ Workspace-project mapping completed successfully"

    # add_sample_userrole_mapping
    # if [ $? -ne 0 ]; then
    #     echo "ERROR: sample user role mapping failed"
    #     return 1
    # fi
    
    # echo "✓ sample user role mapping completed successfully"
    
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
    wget -q -O ./menu.yaml "$MC_WEB_CONSOLE_MENUYAML"
    
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
    if [ -n "$MCADMINCLI_APIYAML" ]; then
        wget -q -O ./api.yaml "$MCADMINCLI_APIYAML" && echo "  Downloaded api.yaml from $MCADMINCLI_APIYAML" || {
            echo "  WARNING: Failed to download api.yaml from $MCADMINCLI_APIYAML — using local copy"
        }
    else
        echo "  MCADMINCLI_APIYAML not set — using local api.yaml"
    fi
    if [ ! -f ./api.yaml ]; then
        echo "ERROR: api.yaml not found"
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

register_framework_services() {
    echo "Registering framework service URLs to mcmp_api_services..."

    # api.yaml의 services 섹션에 정의된 서비스들을 POST /api/mcmp-apis 로 등록
    # sync-mcmp-apis는 serviceActions(권한)만 등록하고 service URL 레지스트리는 미처리하므로 별도 등록 필요

    register_service() {
        local name="$1"
        local version="$2"
        local base_url="$3"
        local auth_type="${4:-none}"
        local auth_user="${5:-}"
        local auth_pass="${6:-}"

        json_data=$(jq -n \
            --arg name "$name" \
            --arg version "$version" \
            --arg baseUrl "$base_url" \
            --arg authType "$auth_type" \
            --arg authUser "$auth_user" \
            --arg authPass "$auth_pass" \
            --argjson isActive true \
            '{name: $name, version: $version, baseUrl: $baseUrl, authType: $authType, authUser: $authUser, authPass: $authPass, isActive: $isActive}')

        response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
            --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
            --header 'Content-Type: application/json' \
            --data "$json_data" \
            "$MC_IAM_MANAGER_HOST/api/mcmp-apis")

        http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
        response_body=$(echo $response | sed -e 's/HTTPSTATUS\:.*//g')

        if [ "$http_code" = "201" ]; then
            echo "  ✓ Registered: $name ($base_url)"
        elif [ "$http_code" = "409" ]; then
            # 기존 레코드가 있으면 base_url, version, auth 자격증명 모두 업데이트
            PGPASSWORD="$MC_IAM_MANAGER_DATABASE_PASSWORD" psql \
                -h "$MC_IAM_MANAGER_DATABASE_HOST" \
                -p "${MC_IAM_MANAGER_DATABASE_PORT:-5432}" \
                -U "$MC_IAM_MANAGER_DATABASE_USER" \
                -d "$MC_IAM_MANAGER_DATABASE_NAME" \
                -c "UPDATE mcmp_api_services SET base_url='$base_url', version='$version', auth_type='$auth_type', auth_user='$auth_user', auth_pass='$auth_pass', updated_at=NOW() WHERE name='$name';" \
                -q 2>/dev/null \
            && echo "  ✓ Updated: $name ($base_url)" \
            || echo "  ✓ Already registered: $name (psql unavailable, skipped)"
        else
            echo "  ✗ Failed to register $name (HTTP $http_code): $response_body"
            return 1
        fi
        return 0
    }

    # api.yaml의 services 섹션에 정의된 모든 framework 등록
    # mc-iam-manager 자신은 service URL registry 등록 대상에서 제외
    local failed=0
    local current_svc=""
    local current_version=""
    local current_baseurl=""
    local current_auth_type=""

    while IFS= read -r line; do
        # 서비스 이름 감지 (2칸 들여쓰기 + 콜론으로 끝나는 라인)
        if echo "$line" | grep -qE "^  [a-z].*:$"; then
            # 직전 서비스 처리
            if [ -n "$current_svc" ] && [ "$current_svc" != "mc-iam-manager" ]; then
                auth_user=""
                auth_pass=""
                if [ "$current_svc" = "mc-infra-manager" ]; then
                    auth_user="${MC_INFRA_MANAGER_API_USERNAME}"
                    auth_pass="${MC_INFRA_MANAGER_API_PASSWORD}"
                elif [ "$current_svc" = "mc-infra-connector" ]; then
                    auth_user="${MC_INFRA_CONNECTOR_API_USERNAME}"
                    auth_pass="${MC_INFRA_CONNECTOR_API_PASSWORD}"
                fi
                register_service "$current_svc" "$current_version" "$current_baseurl" \
                    "${current_auth_type:-none}" "$auth_user" "$auth_pass" || failed=1
            fi
            current_svc=$(echo "$line" | sed 's/^  //; s/:$//')
            current_version=""
            current_baseurl=""
            current_auth_type=""
        elif echo "$line" | grep -q "^    version:"; then
            current_version=$(echo "$line" | awk '{print $2}' | tr -d '"')
        elif echo "$line" | grep -q "^    baseurl:"; then
            current_baseurl=$(echo "$line" | awk '{print $2}')
        elif echo "$line" | grep -q "^      type:"; then
            current_auth_type=$(echo "$line" | awk '{print $2}' | tr -d '"')
        fi
    done < <(sed -n '/^services:/,/^serviceActions:/p' ./api.yaml | head -n -1)

    # 마지막 서비스 처리
    if [ -n "$current_svc" ] && [ "$current_svc" != "mc-iam-manager" ]; then
        auth_user=""
        auth_pass=""
        if [ "$current_svc" = "mc-infra-manager" ]; then
            auth_user="${MC_INFRA_MANAGER_API_USERNAME}"
            auth_pass="${MC_INFRA_MANAGER_API_PASSWORD}"
        elif [ "$current_svc" = "mc-infra-connector" ]; then
            auth_user="${MC_INFRA_CONNECTOR_API_USERNAME}"
            auth_pass="${MC_INFRA_CONNECTOR_API_PASSWORD}"
        fi
        register_service "$current_svc" "$current_version" "$current_baseurl" \
            "${current_auth_type:-none}" "$auth_user" "$auth_pass" || failed=1
    fi

    if [ $failed -ne 0 ]; then
        return 1
    fi

    echo "Framework service registration completed"
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

configure_keycloak_client_uris() {
    echo "=== Configuring Keycloak Client Redirect URIs ==="

    if [ -z "$MC_IAM_MANAGER_PUBLIC_HOST" ]; then
        echo "ERROR: MC_IAM_MANAGER_PUBLIC_HOST is not set — cannot configure redirect URIs"
        return 1
    fi

    PUBLIC_HOST="$MC_IAM_MANAGER_PUBLIC_HOST"
    echo "Public host: $PUBLIC_HOST"

    # Keycloak admin token 발급
    KC_ADMIN_TOKEN=$(curl -s -X POST \
        "${MC_IAM_MANAGER_KEYCLOAK_HOST}/realms/master/protocol/openid-connect/token" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" \
        -d "username=${MC_IAM_MANAGER_KEYCLOAK_ADMIN}" \
        -d "password=${MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD}" \
        | jq -r '.access_token' 2>/dev/null)

    if [ -z "$KC_ADMIN_TOKEN" ] || [ "$KC_ADMIN_TOKEN" = "null" ]; then
        echo "ERROR: Failed to obtain Keycloak admin token"
        return 1
    fi
    echo "  ✓ Keycloak admin token obtained"

    KC_ADMIN_URL="${MC_IAM_MANAGER_KEYCLOAK_HOST}/admin/realms/${MC_IAM_MANAGER_KEYCLOAK_REALM}"

    # mciamClient, mciam-oidc-Client 두 클라이언트 설정
    for CLIENT_NAME in "$MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME" "$MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_NAME"; do
        [ -z "$CLIENT_NAME" ] && continue

        # client ID (UUID) 조회
        CLIENT_ID=$(curl -s \
            "${KC_ADMIN_URL}/clients?clientId=${CLIENT_NAME}" \
            -H "Authorization: Bearer ${KC_ADMIN_TOKEN}" \
            | jq -r '.[0].id' 2>/dev/null)

        if [ -z "$CLIENT_ID" ] || [ "$CLIENT_ID" = "null" ]; then
            echo "  ⚠️  Client not found: $CLIENT_NAME — skipping"
            continue
        fi

        # 현재 client 설정 조회 후 redirect URI 갱신
        CURRENT=$(curl -s "${KC_ADMIN_URL}/clients/${CLIENT_ID}" \
            -H "Authorization: Bearer ${KC_ADMIN_TOKEN}")

        FRONT_HOST="${MC_IAM_MANAGER_PUBLIC_DOMAIN}${MC_WEB_CONSOLE_FRONT_PORT:+:${MC_WEB_CONSOLE_FRONT_PORT}}"
        FRONT_SCHEME=$(echo "$PUBLIC_HOST" | grep -o 'https\?')
        FRONT_URI="${FRONT_SCHEME}://${FRONT_HOST}"

        UPDATED=$(echo "$CURRENT" | jq \
            --arg h "$PUBLIC_HOST" \
            --arg f "$FRONT_URI" \
            '.rootUrl = $h | .baseUrl = $h | .redirectUris = [$h + "/*", $f + "/*"] | .webOrigins = [$h, $f]')

        HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
            "${KC_ADMIN_URL}/clients/${CLIENT_ID}" \
            -H "Authorization: Bearer ${KC_ADMIN_TOKEN}" \
            -H "Content-Type: application/json" \
            -d "$UPDATED")

        if [ "$HTTP" = "204" ]; then
            echo "  ✓ Updated: $CLIENT_NAME → redirectUris=[${PUBLIC_HOST}/*, ${FRONT_URI}/*]"
        else
            echo "  ✗ Failed to update $CLIENT_NAME (HTTP $HTTP)"
        fi
    done
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

# add_sample_userrole_mapping() {
#     echo "Adding test users..."

#     #admin
#     role_id="1"
#     #platform admin user
#     user_id="1"
#     role_type="platform"
#     # ws01
#     workspace_id="1"

#     json_data=$(jq -n --arg role_id "$role_id" --arg user_id "$user_id" --arg role_type "$role_type" --arg workspace_id "$workspace_id" \
#         '{role_id: $role_id, user_id: $user_id, role_type: $role_type, workspace_id: $workspace_id}')
#     response=$(curl -s -X POST \
#         --header "Authorization: Bearer $MC_IAM_MANAGER_PLATFORMADMIN_ACCESSTOKEN" \
#         --header 'Content-Type: application/json' \
#         --data "$json_data" \
#         "$MC_IAM_MANAGER_HOST_FOR_INIT/api/roles/assign/platform-role")

#     # 응답 검증
#     if [ $? -ne 0 ]; then
#         echo "ERROR: Failed to sync projects"
#         return 1
#     fi
    
#     echo "Test user addition response: $response"


#     echo "Test user addition completed"
#     return 0
# }

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
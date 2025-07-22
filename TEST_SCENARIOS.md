# MC-IAM-Manager 테스트 시나리오

## 사전 준비

1.  Keycloak에 `admin` (platformadmin 역할), `testuser1` 사용자 생성 및 활성화.
2.  DB 초기화 (`mcmp_table.sql`, `mcmp_init_data.sql` 실행).
3.  테스트용 CSP 역할 매핑 데이터 추가.
    *   `admin` (ID: 1 가정) 및 `viewer` (ID: 3 가정) 워크스페이스 역할과 테스트용 AWS IAM 역할 매핑 추가.
        ```sql
        -- 예시: mcmp_init_data.sql 또는 직접 실행
        -- Admin Role Mapping
        INSERT INTO mcmp_workspace_role_csp_role_mapping (workspace_role_id, csp_type, csp_role_arn, idp_identifier, description)
        VALUES (1, 'aws', 'arn:aws:iam::ACCOUNT_ID:role/MCMP_admin', 'arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME', 'Mapping for Workspace Admin to AWS MCMP_admin')
        ON CONFLICT (workspace_role_id, csp_type, csp_role_arn) DO NOTHING;

        -- Viewer Role Mapping
        INSERT INTO mcmp_workspace_role_csp_role_mapping (workspace_role_id, csp_type, csp_role_arn, idp_identifier, description)
        VALUES (3, 'aws', 'arn:aws:iam::ACCOUNT_ID:role/MCMP_viewer', 'arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME', 'Mapping for Workspace Viewer to AWS MCMP_viewer')
        ON CONFLICT (workspace_role_id, csp_type, csp_role_arn) DO NOTHING;
        ```
        (실제 `ACCOUNT_ID`, `KEYCLOAK_HOSTNAME`, 역할 이름(`MCMP_admin`, `MCMP_viewer`)으로 변경 필요)
4.  애플리케이션 실행 (`docker-compose up -d`).

## 시나리오 1: 사용자 로그인 및 기본 정보 확인

1.  **사용자 로그인:** `testuser1` 사용자로 로그인 API 호출.
    *   **API:** `POST /api/auth/login`
    *   **Body:** `{"id": "testuser1", "password": "password"}`
    *   **Expected:** 200 OK, 응답 본문에 Access Token (`access_token`) 포함. (이후 요청에 사용)
2.  **내 워크스페이스/역할 확인:** `testuser1`의 토큰으로 API 호출.
    *   **API:** `GET /api/users/workspaces`
    *   **Header:** `Authorization: Bearer <testuser1_access_token>`
    *   **Expected:** 200 OK, 빈 배열 `[]` 반환 (아직 할당된 워크스페이스 없음).

## 시나리오 2: 관리자의 워크스페이스 생성 및 사용자 역할 할당

1.  **관리자 로그인:** `admin` 사용자로 로그인.
    *   **API:** `POST /api/auth/login`
    *   **Body:** `{"id": "admin", "password": "password"}`
    *   **Expected:** 200 OK, Access Token (`admin_access_token`) 획득.
2.  **워크스페이스 생성:** `admin` 토큰으로 새 워크스페이스 생성.
    *   **API:** `POST /api/workspaces`
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   **Body:** `{"name": "TestWorkspace1", "description": "테스트용 워크스페이스"}`
    *   **Expected:** 201 Created, 응답 본문에 생성된 워크스페이스 정보 (ID 포함) 반환. 생성자(`admin`)는 자동으로 해당 워크스페이스의 `admin` 역할 할당됨. (생성된 워크스페이스 ID를 `<ws1_id>`로 저장)
3.  **사용자에게 역할 할당:** `admin` 토큰으로 `testuser1`에게 `TestWorkspace1`의 `viewer` 역할 할당. (viewer 역할 ID는 3 가정)
    *   **API:** `POST /api/workspaces/<ws1_id>/users/<testuser1_db_id>/roles/3` (`<testuser1_db_id>`는 DB에서 확인 필요)
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   **Expected:** 204 No Content.
4.  **할당 확인 (관리자):** `admin` 토큰으로 워크스페이스 사용자/역할 목록 조회.
    *   **API:** `GET /api/workspaces/<ws1_id>/users`
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   **Expected:** 200 OK, `testuser1`이 `viewer` 역할로 포함된 목록 반환.
5.  **할당 확인 (사용자):** `testuser1` 토큰으로 내 워크스페이스/역할 확인.
    *   **API:** `GET /api/users/workspaces`
    *   **Header:** `Authorization: Bearer <testuser1_access_token>`
    *   **Expected:** 200 OK, `TestWorkspace1`과 `viewer` 역할 정보 반환.

## 시나리오 3: 워크스페이스 접근 권한 확인

1.  **관리자 - 전체 목록 조회:** `admin` 토큰으로 워크스페이스 목록 조회.
    *   **API:** `GET /api/workspaces`
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   **Expected:** 200 OK, `TestWorkspace1` 포함된 전체 목록 반환 (`list_all` 권한).
2.  **일반 사용자 - 할당된 목록 조회:** `testuser1` 토큰으로 워크스페이스 목록 조회.
    *   **API:** `GET /api/workspaces`
    *   **Header:** `Authorization: Bearer <testuser1_access_token>`
    *   **Expected:** 200 OK, `TestWorkspace1`만 포함된 목록 반환 (`list_all` 권한 없음).
3.  **일반 사용자 - 할당된 워크스페이스 상세 조회:** `testuser1` 토큰으로 `TestWorkspace1` 상세 조회.
    *   **API:** `GET /api/workspaces/<ws1_id>`
    *   **Header:** `Authorization: Bearer <testuser1_access_token>`
    *   **Expected:** 200 OK, `TestWorkspace1` 정보 반환 (`read` 권한).
4.  **일반 사용자 - 미할당 워크스페이스 상세 조회 (실패):** (관리자가 다른 워크스페이스 `<ws2_id>` 생성 가정) `testuser1` 토큰으로 `<ws2_id>` 상세 조회 시도.
    *   **API:** `GET /api/workspaces/<ws2_id>`
    *   **Header:** `Authorization: Bearer <testuser1_access_token>`
    *   **Expected:** 403 Forbidden.

## 시나리오 4: MCMP API 호출 (RPT 사용)

1.  **관리자 - 역할 권한 확인/추가:** `admin` 워크스페이스 역할에 `mc-infra-manager:vm:read` 권한이 있는지 확인하고 없으면 추가.
    *   **API:** `GET /api/roles/workspace/1/mciam-permissions` (admin 역할 ID: 1 가정)
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   (결과 확인 후 필요시) **API:** `POST /api/roles/workspace/1/mciam-permissions/mc-infra-manager:vm:read`
    *   **Header:** `Authorization: Bearer <admin_access_token>`
2.  **관리자 - Action Ticket 발급:** `admin` 사용자가 `TestWorkspace1` 컨텍스트에서 `vm:read` 권한에 대한 티켓 요청.
    *   **API:** `POST /api/auth/action-ticket`
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   **Body:** `{"workspaceId": "<ws1_id>", "permissions": ["mc-infra-manager:vm:read"]}`
    *   **Expected:** 200 OK, 응답 본문에 RPT (`access_token`) 포함. (이후 요청에 사용, `<rpt_token>`으로 저장)
3.  **MCMP API 호출:** 발급받은 RPT로 MCMP API 호출 시도.
    *   **API:** `POST /api/mcmp-apis/call`
    *   **Header:** `Authorization: Bearer <rpt_token>`
    *   **Body:** `{"serviceName": "mc-infra-manager", "actionName": "vm_read", "parameters": {"vmId": "test-vm"}}` (파라미터는 예시)
    *   **Expected:** 200 OK (또는 대상 API의 실제 응답). `McmpApiAuthMiddleware`에서 RPT 검증 통과.

## 시나리오 5: CSP 임시 자격 증명 발급

1.  **관리자 - CSP 역할 매핑 확인/추가:** `admin` 워크스페이스 역할과 AWS `MCMP_admin` 역할 매핑 확인/추가. (사전 준비 단계에서 추가했거나 이 API 사용)
    *   **API:** `GET /api/workspace-roles/1/csp-role-mappings` (admin 역할 ID: 1 가정)
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   (결과 확인 후 필요시) **API:** `POST /api/workspace-roles/1/csp-role-mappings`
    *   **Header:** `Authorization: Bearer <admin_access_token>`
    *   **Body:** `{"cspType": "aws", "cspRoleArn": "arn:aws:iam::ACCOUNT_ID:role/MCMP_admin", "idpIdentifier": "arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME"}`
2.  **관리자 - 임시 자격 증명 요청:** `admin` 사용자가 `TestWorkspace1` 컨텍스트에서 AWS 임시 자격 증명 요청 (매핑된 `MCMP_admin` 역할 사용).
    *   **API:** `POST /api/csp/credentials`
    *   **Header:** `Authorization: Bearer <admin_access_token>` (원본 OIDC 토큰 사용)
    *   **Body:** `{"workspaceId": "<ws1_id>", "cspType": "aws"}`
    *   **Expected:** 200 OK, 응답 본문에 AWS 임시 자격 증명 (AccessKeyId, SecretAccessKey, SessionToken, Expiration) 포함.

---

## 시나리오 6: 역할 기반 메뉴 접근 및 MCMP API 호출 제어 (Admin vs Viewer)

**목표:** 관리자(Admin)와 조회자(Viewer) 역할에 따라 접근 가능한 메뉴가 다르고, MCMP API 호출 시 필요한 권한(RPT 기반)이 올바르게 제어되는지 확인합니다.

**사전 준비:**

1.  **사용자 생성:**
    *   `admin_user` (ID/PW: admin/admin)
    *   `viewer_user` (ID/PW: viewer/viewer)
2.  **역할 정의:**
    *   플랫폼 역할: `platform_admin` (ID: 1 가정)
    *   워크스페이스 역할: `ws_admin` (ID: 1 가정), `ws_viewer` (ID: 3 가정)
3.  **워크스페이스 생성:** `workspace_A` (ID: 1 가정)
4.  **권한 정의 (DB 및 Keycloak UMA):**
    *   메뉴 보기 권한 (DB `mcmp_mciam_permissions`): `menu:menu:view:settings`, `menu:menu:view:users`, `menu:menu:view:workspaces`, `menu:menu:view:projects` (메뉴 로딩 시 자동 생성 가정)
    *   워크스페이스 관리 권한 (DB `mcmp_mciam_permissions`): `mc-iam-manager:workspace:list_all`, `mc-iam-manager:workspace:read`
    *   MCMP API 권한 (Keycloak UMA Resource/Scope): `mc-infra-manager#GetVm`, `mc-infra-manager#Start-Vm`
5.  **역할-권한 매핑 (DB `mcmp_mciam_role_permissions` 및 Keycloak UMA Policy):**
    *   `platform_admin` 역할 (ID: 1): `menu:menu:view:settings`, `menu:menu:view:users`, `menu:menu:view:workspaces`, `mc-iam-manager:workspace:list_all` 권한 할당.
    *   `ws_admin` 역할 (ID: 1): Keycloak UMA에서 `mc-infra-manager#Start-Vm` 권한 부여.
    *   `ws_viewer` 역할 (ID: 3): `menu:menu:view:projects` 권한 할당, Keycloak UMA에서 `mc-infra-manager#GetVm` 권한 부여.
6.  **사용자-역할 할당:**
    *   `admin_user`: `platform_admin` 역할 할당 (DB `mcmp_user_platform_roles`). `workspace_A`에 `ws_admin` 역할로 할당 (DB `mcmp_user_workspace_roles`).
    *   `viewer_user`: `workspace_A`에 `ws_viewer` 역할로 할당 (DB `mcmp_user_workspace_roles`).

---

### 시나리오 실행 (curl 예시)

**(변수 설정 - 각 터미널에서)**

```bash
# 공통
IAM_MANAGER_URL="http://localhost:8082" # 실제 환경에 맞게 수정
WORKSPACE_ID=1 # 테스트용 워크스페이스 ID

# 터미널 1 (Admin)
ADMIN_USER="admin_user"
ADMIN_PASS="admin"
ADMIN_ACCESS_TOKEN=""
ADMIN_RPT_TOKEN=""

# 터미널 2 (Viewer)
VIEWER_USER="viewer_user"
VIEWER_PASS="viewer"
VIEWER_ACCESS_TOKEN=""
VIEWER_RPT_TOKEN=""
```

---

**터미널 1: 관리자 (admin\_user)**

1.  **로그인:**
    ```bash
    ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "${IAM_MANAGER_URL}/api/auth/login" \
      -H "Content-Type: application/json" \
      -d "{\"id\": \"${ADMIN_USER}\", \"password\": \"${ADMIN_PASS}\"}")
    ADMIN_ACCESS_TOKEN=$(echo $ADMIN_LOGIN_RESPONSE | jq -r '.access_token')
    echo "Admin Access Token: $ADMIN_ACCESS_TOKEN"
    # 예상 결과: 토큰 출력
    ```

2.  **메뉴 조회:**
    ```bash
    curl -s -X GET "${IAM_MANAGER_URL}/api/users/menus" \
      -H "Authorization: Bearer ${ADMIN_ACCESS_TOKEN}" | jq .
    # 예상 결과: settings, users, workspaces, projects 등 포함된 전체 메뉴 트리 JSON 출력
    ```

3.  **워크스페이스 목록 조회:**
    ```bash
    curl -s -X GET "${IAM_MANAGER_URL}/api/workspaces" \
      -H "Authorization: Bearer ${ADMIN_ACCESS_TOKEN}" | jq .
    # 예상 결과: workspace_A 포함된 전체 워크스페이스 목록 JSON 출력
    ```

4.  **MCMP API 호출 (VM 생성 - 권한 있음):**
    *   **Action Ticket(RPT) 발급 요청:**
        ```bash
        ADMIN_RPT_RESPONSE=$(curl -s -X POST "${IAM_MANAGER_URL}/api/auth/workspace-ticket" \
          -H "Authorization: Bearer ${ADMIN_ACCESS_TOKEN}" \
          -H "Content-Type: application/json" \
          -d "{\"workspaceId\": ${WORKSPACE_ID}, \"permissions\": [\"mc-infra-manager#Start-Vm\"]}")
        ADMIN_RPT_TOKEN=$(echo $ADMIN_RPT_RESPONSE | jq -r '.access_token')
        echo "Admin RPT Token: $ADMIN_RPT_TOKEN"
        # 예상 결과: RPT 토큰 출력 (null이 아니어야 함)
        ```
    *   **MCMP API 호출:**
        ```bash
        # 실제 nsId, mciId, VM 생성 파라미터로 대체 필요
        curl -s -X POST "${IAM_MANAGER_URL}/api/mcmp-apis/call" \
          -H "Authorization: Bearer ${ADMIN_RPT_TOKEN}" \
          -H "Content-Type: application/json" \
          -d '{
                "serviceName": "mc-infra-manager",
                "actionName": "Start-Vm",
                "requestParams": {
                  "pathParams": {"nsId": "test-ns", "mciId": "test-mci"},
                  "body": {"name": "test-vm-by-admin", "imageId": "...", "specId": "..."}
                }
              }' | jq .
        # 예상 결과: 200 OK 또는 외부 API 성공 응답 JSON 출력
        ```

---

**터미널 2: 조회자 (viewer\_user)**

1.  **로그인:**
    ```bash
    VIEWER_LOGIN_RESPONSE=$(curl -s -X POST "${IAM_MANAGER_URL}/api/auth/login" \
      -H "Content-Type: application/json" \
      -d "{\"id\": \"${VIEWER_USER}\", \"password\": \"${VIEWER_PASS}\"}")
    VIEWER_ACCESS_TOKEN=$(echo $VIEWER_LOGIN_RESPONSE | jq -r '.access_token')
    echo "Viewer Access Token: $VIEWER_ACCESS_TOKEN"
    # 예상 결과: 토큰 출력
    ```

2.  **메뉴 조회:**
    ```bash
    curl -s -X GET "${IAM_MANAGER_URL}/api/users/menus" \
      -H "Authorization: Bearer ${VIEWER_ACCESS_TOKEN}" | jq .
    # 예상 결과: projects 메뉴만 포함된 제한적인 메뉴 트리 JSON 출력
    ```

3.  **워크스페이스 목록 조회:**
    ```bash
    curl -s -X GET "${IAM_MANAGER_URL}/api/workspaces" \
      -H "Authorization: Bearer ${VIEWER_ACCESS_TOKEN}" | jq .
    # 예상 결과: workspace_A만 포함된 워크스페이스 목록 JSON 출력
    ```

4.  **MCMP API 호출 (VM 생성 - 권한 없음):**
    *   **Action Ticket(RPT) 발급 요청:**
        ```bash
        curl -s -X POST "${IAM_MANAGER_URL}/api/auth/workspace-ticket" \
          -H "Authorization: Bearer ${VIEWER_ACCESS_TOKEN}" \
          -H "Content-Type: application/json" \
          -d "{\"workspaceId\": ${WORKSPACE_ID}, \"permissions\": [\"mc-infra-manager#Start-Vm\"]}" | jq .
        # 예상 결과: 403 Forbidden 오류 JSON 출력
        ```
    *   **(RPT 발급 실패 확인)**

5.  **MCMP API 호출 (VM 조회 - 권한 있음):**
    *   **Action Ticket(RPT) 발급 요청:**
        ```bash
        VIEWER_RPT_RESPONSE=$(curl -s -X POST "${IAM_MANAGER_URL}/api/auth/workspace-ticket" \
          -H "Authorization: Bearer ${VIEWER_ACCESS_TOKEN}" \
          -H "Content-Type: application/json" \
          -d "{\"workspaceId\": ${WORKSPACE_ID}, \"permissions\": [\"mc-infra-manager#GetVm\"]}")
        VIEWER_RPT_TOKEN=$(echo $VIEWER_RPT_RESPONSE | jq -r '.access_token')
        echo "Viewer RPT Token: $VIEWER_RPT_TOKEN"
        # 예상 결과: RPT 토큰 출력 (null이 아니어야 함)
        ```
    *   **MCMP API 호출:**
        ```bash
        # 실제 nsId, mciId, vmId로 대체 필요
        curl -s -X POST "${IAM_MANAGER_URL}/api/mcmp-apis/call" \
          -H "Authorization: Bearer ${VIEWER_RPT_TOKEN}" \
          -H "Content-Type: application/json" \
          -d '{
                "serviceName": "mc-infra-manager",
                "actionName": "GetVm",
                "requestParams": {
                  "pathParams": {"nsId": "test-ns", "mciId": "test-mci", "vmId": "test-vm-by-admin"}
                }
              }' | jq .
        # 예상 결과: 200 OK 또는 외부 API 성공 응답 JSON 출력 (VM 정보)
        ```

---

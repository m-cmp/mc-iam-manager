# MC-IAM-MANAGER 개발 가이드라인
- 참조 project : https://github.com/m-cmp/mc-iam-manager 
- 참조 project의 기능을 재현.

## 1. 프로젝트 개요
- Keycloak 기반의 백엔드 서비스 개발
- 사용자, 권한, 역할 관리 기능 구현
- RESTful API 설계 및 구현

## 2. 기술 스택
- Go 1.21
- Echo Framework v4
- PostgreSQL
- Keycloak
- Docker & Docker Compose
- Swagger/OpenAPI

## 3. 프로젝트 구조
```
.
├── asset/
│   ├── menu/         # 메뉴 YAML 파일 (초기 등록/동기화용)
│   ├── sql/         # sql (테이블 정의 및 마이그레이션)
│   │   ├── tables.sql  # 최종 스키마 정의 (참고용, 실제 적용은 마이그레이션으로)
│   │   └── migrations/ # DB 마이그레이션 파일 (mcmp_ 접두사 적용됨)
│   ├── uml/         # uml
├── dockerfiles/    # Docker 관련 파일
│   ├── keycloak/         # keycloak
│   ├── mc-iam-manager/         # mc-iam-manager by src
│   ├── nginx/         # nginx
│   ├── postgres/         # postgres
├── src/
│   ├── config/         # 설정 관련 코드
│   ├── docs/         # API 문서(원본)
│   ├── handler/        # HTTP 요청 처리 (e.g., user_handler.go)
│   ├── middleware/     # 미들웨어
│   ├── model/         # 데이터 모델 (e.g., model/mcmpapi/)
│   ├── repository/    # 데이터베이스 작업 (e.g., mcmpapi_repository.go)
│   ├── service/       # 비즈니스 로직 (e.g., user_service.go, mcmpapi_service.go)
│   └── main.go        # 애플리케이션 진입점
├── migrations/        # 데이터베이스 마이그레이션
├── docs/             # API 문서(복사본)
└── docker/           # Docker 관련 파일
```

## 4. 개발 단계

### 4.1 초기 설정
1. 환경 변수 설정
   - `.env` 파일 생성
   - 필요한 환경 변수 정의
   - 데이터베이스 연결 정보 설정
   - Keycloak 설정 정보 추가
   - MCMP API YAML 파일 경로 설정 (`MCADMINCLI_APIYAML` in `.env`)

#### 4.1.1 메뉴 YAML 파일 준비 (선택 사항)
- 메뉴 데이터를 YAML 파일로부터 초기 등록하거나 동기화하려면 `asset/menu/menu.yaml` 파일을 준비합니다.
- 파일 소스는 `.env` 파일의 `MCWEBCONSOLE_MENUYAML` 환경 변수를 통해 지정하거나, 기본 URL(`https://raw.githubusercontent.com/m-cmp/mc-web-console/refs/heads/main/conf/webconsole_menu_resources.yaml`)을 사용할 수 있습니다.
- 파일 다운로드 예시: `curl -L [소스_URL_또는_경로] -o asset/menu/menu.yaml --create-dirs`

2. 데이터베이스 설정 (2025-04-18 업데이트)
   - PostgreSQL 컨테이너 실행
   - **스키마 생성:** `asset/sql/mcmp_table.sql` 파일을 사용하여 데이터베이스 스키마를 생성합니다. 이 파일은 `src/model/`의 Go 모델을 기반으로 생성된 최신 DDL을 포함합니다. (기존 마이그레이션 방식은 제거되었습니다.)
     ```bash
     psql -h <호스트> -p <포트> -U <사용자> -d <데이터베이스명> -f asset/sql/mcmp_table.sql
     ```
   - **초기 데이터 설정:** 필요한 초기 데이터(기본 역할, 권한, 메뉴 등)는 `asset/sql/mcmp_init_data.sql` 파일을 사용하여 로드합니다.
     ```bash
     psql -h <호스트> -p <포트> -U <사용자> -d <데이터베이스명> -f asset/sql/mcmp_init_data.sql
     ```

3. Keycloak 설정
   - Keycloak 컨테이너 실행
   - Realm 생성
   - 클라이언트 설정 (`mciamClient` 등)
   - 사용자 및 역할 설정
   - **서비스 계정 역할 설정 (중요):**
     - `mciamClient` (또는 `.env`의 `MCIAMMANAGER_KEYCLOAK_CLIENTID` 클라이언트)의 서비스 계정이 활성화되어 있는지 확인합니다.
     - 해당 서비스 계정에 필요한 역할(Role)을 부여해야 합니다.
     - 사용자 정보 조회 (`GET /users` 등)에는 **`realm-management` 클라이언트의 `view-users` 역할**이 필요합니다.
     - 사용자 생성/수정/삭제 (`POST /users`, `PUT /users/{id}`, `DELETE /users/{id}` 등)에는 **`realm-management` 클라이언트의 `manage-users` 역할**이 필요합니다. (다른 관리 API 사용 시 추가 역할 필요)

### 4.2 핵심 기능 구현

#### 4.2.1 MCMP API 동기화 (`mcmp_api_services`, `mcmp_api_actions` 테이블)
- **동기화 트리거:** `POST /api/v1/mcmp-apis/sync` API를 호출하여 동기화를 시작합니다.
- **YAML 소스:** `.env` 파일의 `MCADMINCLI_APIYAML` 환경 변수에 지정된 URL에서 `mcmp_api.yaml` (또는 지정된 파일)을 다운로드합니다.
- **동기화 로직:**
    - YAML 파일을 파싱하여 서비스 및 액션 정보를 읽습니다.
    - 각 서비스에 대해 `name`과 `version`을 기준으로 DB에 이미 존재하는지 확인합니다.
    - DB에 존재하지 않는 새로운 버전의 서비스 정보만 `mcmp_api_services` 테이블에 추가합니다.
    - 새로 추가된 서비스에 해당하는 액션 정보만 `mcmp_api_actions` 테이블에 추가합니다.
    - 기존 버전의 서비스 및 액션 정보는 변경되거나 삭제되지 않습니다.

#### 4.2.2 사용자 관리 (`mcmp_users` 테이블 및 Keycloak)
- **사용자 등록:** Keycloak 자체 등록 기능 사용 (초기 상태: 비활성).
- **사용자 승인:**
    - 관리자(admin/platformadmin)가 Keycloak 콘솔 또는 `POST /api/v1/users/{id}/approve` API를 통해 사용자를 활성화(`enabled=true`). (`{id}`는 Keycloak User ID)
- **로그인 및 동기화:**
    - 사용자가 `POST /api/v1/auth/login`으로 로그인 시도.
    - Keycloak 인증 성공 후, 사용자의 `enabled` 상태 확인. 비활성 시 403 Forbidden 반환.
    - 활성 상태이면, 로컬 `mcmp_users` DB에 해당 사용자 정보가 있는지 확인하고 없으면 Keycloak 정보를 바탕으로 생성 (DB 동기화 - `UserService.SyncUser` 또는 `GetUserDbIDByKcID` 호출 시 트리거될 수 있음).
    - 동기화 후 로그인 토큰 발급.
- **관리자용 사용자 생성:** `POST /api/v1/users` API (admin/platformadmin 권한 필요)는 Keycloak에 즉시 활성 상태로 사용자를 생성하고 로컬 DB에도 동기화.
- **기타 관리:** 사용자 조회(`GET /api/v1/users`, `GET /api/v1/users/{id}`, `GET /api/v1/users/username/{username}`), 수정(`PUT /api/v1/users/{id}`), 삭제(`DELETE /api/v1/users/{id}`) API 제공 (적절한 권한 필요).
- **내 정보 조회:**
    - 내 워크스페이스/역할 목록 조회: `GET /api/v1/user/workspaces`
    - 내 메뉴 트리 조회: `GET /api/v1/user/menus`
- 로컬 DB(`mcmp_users`)에는 DB 자체 ID(`id`), Keycloak User ID (`kc_id`), 사용자 이름(`username`), 추가 정보(예: `description`)가 저장됩니다. (Email, FirstName, LastName 등은 Keycloak에서 관리)
- **최고 관리자 동기화:** 애플리케이션 시작 시 `.env`의 `MCIAMMANAGER_PLATFORMADMIN_ID` 사용자를 확인하고, 로컬 DB에 동기화하며 'platformadmin' 역할을 부여합니다 (`UserService.SyncPlatformAdmin` 로직).

#### 4.2.2 워크스페이스 관리 (`mcmp_workspaces` 테이블)
- 워크스페이스 CRUD API 구현 (`/api/v1/workspaces`, `/api/v1/workspaces/{id}`)
- 워크스페이스 이름으로 조회 API 구현 (`/api/v1/workspaces/name/{name}`)
- 워크스페이스-프로젝트 연결/해제 API 구현 (`POST`/`DELETE /api/v1/workspaces/{id}/projects/{projectId}`)
- 워크스페이스별 프로젝트 목록 조회 API 구현 (`GET /api/v1/workspaces/{id}/projects`)
- 워크스페이스별 사용자 및 역할 목록 조회 API 구현 (`GET /api/v1/workspaces/{id}/users`)

#### 4.2.3 프로젝트 관리 (`mcmp_projects` 테이블)
- 프로젝트 CRUD API 구현 (`/api/v1/projects`, `/api/v1/projects/{id}`)
  - **참고:** 프로젝트 생성(`POST /api/v1/projects`) 시, 해당 프로젝트는 `.env` 파일의 `DEFAULT_WORKSPACE_NAME` 환경 변수에 지정된 이름의 워크스페이스(기본값: "default")에 자동으로 할당됩니다.
- 프로젝트 이름으로 조회 API 구현 (`/api/v1/projects/name/{name}`)
- 프로젝트-워크스페이스 연결/해제 API 구현 (`POST`/`DELETE /api/v1/projects/{id}/workspaces/{workspaceId}`)
- 워크스페이스와 프로젝트는 M:N 관계 (`mcmp_workspace_projects` 매핑 테이블 사용)
- **프로젝트 동기화 API:** `POST /api/v1/projects/sync`
  - `mc-infra-manager`의 네임스페이스 목록을 조회합니다.
  - 로컬 DB(`mcmp_projects`)에 존재하지 않는 네임스페이스를 새로운 프로젝트로 생성합니다.
  - 동기화 과정에서 **새로 생성되었거나, 기존에 존재했지만 어떤 워크스페이스에도 할당되지 않은 프로젝트**를 `.env` 파일의 `DEFAULT_WORKSPACE_NAME` 환경 변수에 지정된 이름의 워크스페이스(기본값: "default")에 자동으로 할당합니다.

#### 4.2.4 역할 관리 (`mcmp_platform_roles`, `mcmp_workspace_roles`, `mcmp_user_workspace_roles` 테이블)
- 플랫폼 역할 CRUD API 구현 (`/api/v1/platform-roles`)
- 워크스페이스 역할 CRUD API 구현 (`/api/v1/workspace-roles`) - 워크스페이스 역할 자체는 특정 워크스페이스에 종속되지 않음.
- 사용자에게 플랫폼 역할 할당/제거 API 구현 (현재 미구현, 필요시 추가)
- 사용자에게 워크스페이스 역할 할당/제거 API 구현 (`POST`/`DELETE /api/v1/workspaces/{workspaceId}/users/{userId}/roles/{roleId}`) - `{userId}`는 DB ID(`id`) 사용.
- 역할 기반 접근 제어 (미들웨어 등에서 활용)

#### 4.2.5 권한 관리 (MC-IAM 내부 권한) (`mcmp_mciam_permissions`, `mciam_role_mciam_permissions` 테이블)
- **개념:** MCMP API 호출, UI 메뉴 접근, 워크스페이스 접근 등 `mc-iam-manager` 내부 작업에 대한 권한을 관리합니다. 리소스 유형(`framework_id`, `resource_type_id`)과 액션(`action`)을 기반으로 정의됩니다 (예: `mc-infra-manager:vm:create`, `mc-iam-manager:workspace:read`).
- **권한 관리 API:** `/api/mciam-permissions` 엔드포인트를 통해 CRUD 관리.
- **역할-권한 매핑 API:** `/api/roles/{roleType}/{roleId}/mciam-permissions/{permissionId}` 엔드포인트 (POST/DELETE)를 통해 역할(플랫폼/워크스페이스)에 MC-IAM 권한을 매핑합니다.
- **권한 검증 (MCMP API):** `/api/mcmp-apis/call` 요청 시 전달된 RPT(Action Ticket) 내의 권한 정보를 확인하여 검증합니다 (`McmpApiAuthMiddleware`).
- **권한 검증 (워크스페이스 API):** 워크스페이스 관련 API(`ListWorkspaces`, `CreateWorkspace`, `GetWorkspaceByID` 등) 핸들러 내부에서 사용자의 플랫폼 역할 또는 특정 워크스페이스 역할에 부여된 MC-IAM 권한(`list_all`, `list_assigned`, `create`, `read` 등)을 확인합니다.

#### 4.2.6 역할 - CSP 역할 매핑 (`mcmp_workspace_role_csp_role_mapping` 테이블)
- **개념:** `mc-iam-manager`의 워크스페이스 역할을 실제 CSP의 IAM 역할(예: AWS Role ARN)에 매핑하여 CSP 임시 자격 증명 발급에 사용합니다.
- **API:** `/api/workspace-roles/{roleId}/csp-role-mappings` 엔드포인트를 통해 CRUD 관리.

#### 4.2.7 Keycloak 연동 및 권한 검증 (UMA 기반 - MC-IAM 권한용)
- **목표:** MCMP API 호출 권한 검증에 Keycloak UMA 활용.
- **핵심 메커니즘:**
    1.  **DB-Keycloak 그룹 동기화:** 사용자-워크스페이스 역할 할당 변경 시 Keycloak 그룹 정보 업데이트.
    2.  **Keycloak UMA 설정:** MC-IAM 권한(`mcmp_mciam_permissions`) 기반으로 리소스/스코프/정책/퍼미션 설정.
    3.  **Action Ticket (RPT) 발급:** `/api/auth/action-ticket` API 호출 시 Keycloak UMA 평가 후 RPT 발급.
    4.  **MCMP API 호출 권한 검증:** `/api/mcmp-apis/call` 미들웨어에서 RPT 검증.

#### 4.2.8 CSP 임시 자격 증명 발급
- **API:** `/api/csp/credentials` (POST)
- **요청:** 원본 OIDC 토큰 (헤더), `workspaceId`, `cspType` (본문).
- **로직:**
    1. OIDC 토큰 검증 및 사용자 확인.
    2. 사용자의 해당 워크스페이스 역할 목록 조회 (DB).
    3. 역할 목록과 `cspType`을 사용하여 `mcmp_workspace_role_csp_role_mapping` 테이블에서 매핑된 `csp_role_arn` 및 `idp_identifier` 조회.
    4. 조회된 정보와 **원본 OIDC 토큰**을 사용하여 해당 CSP의 STS API (`AssumeRoleWithWebIdentity` 등) 호출.
    5. 발급된 임시 자격 증명 반환.

#### 4.2.9 메뉴 관리 (`mcmp_menu` 테이블)

#### 4.2.6 메뉴 관리 (`mcmp_menu` 테이블)
- 메뉴 데이터는 PostgreSQL 데이터베이스의 `mcmp_menu` 테이블에 저장 및 관리됩니다.
- **내 메뉴 트리 조회:** `GET /api/v1/user/menus` API는 현재 로그인한 사용자의 Platform Role에 따라 접근 가능한 메뉴 목록을 트리 구조(`[]model.MenuTreeNode`)로 반환합니다. 하위 메뉴 접근 권한이 있으면 상위 메뉴도 포함됩니다.
- **전체 메뉴 트리 조회 (관리자용):** `GET /api/v1/menus/all` API로 모든 메뉴를 트리 구조로 조회합니다.
- **개별 메뉴 조회:** `GET /api/v1/menus/{id}` API로 특정 메뉴 정보를 조회합니다.
- **메뉴 생성/수정/삭제:** `POST /api/v1/menus`, `PUT /api/v1/menus/{id}` (부분 업데이트 지원), `DELETE /api/v1/menus/{id}` API를 통해 메뉴를 직접 관리할 수 있습니다.
- **YAML 파일/URL 등록/동기화:** `POST /api/v1/menus/register-from-yaml` API를 호출합니다.
    - `filePath` 쿼리 파라미터가 있으면 해당 로컬 경로의 YAML 파일을 읽어 DB에 Upsert합니다.
    - `filePath` 파라미터가 없으면, `.env` 파일의 `MCWEBCONSOLE_MENUYAML` 환경 변수를 확인합니다.
        - 값이 URL이면 해당 URL에서 YAML을 다운로드하여 `asset/menu/menu.yaml`에 저장한 후, 이 파일을 읽어 DB에 Upsert합니다.
        - 값이 로컬 경로이면 해당 경로의 파일을 읽어 DB에 Upsert합니다.
        - 값이 없거나 URL이 아니면 기본 로컬 경로(`asset/menu/menu.yaml`)를 읽어 DB에 Upsert합니다.
- **YAML 본문 등록/동기화:** `POST /api/v1/menus/register-from-body` API를 호출하여 요청 본문에 포함된 YAML 텍스트 내용을 읽어 DB에 Upsert할 수 있습니다. (Content-Type: text/plain, text/yaml, application/yaml 등)
- 테이블 스키마는 `asset/sql/mcmp_table.sql` 파일로 관리됩니다. (FK 제약 조건 지연 설정 포함)

### 4.3 API 문서화 (Swagger)
- 이 프로젝트는 `swaggo/swag` 라이브러리를 사용하여 Go 소스 코드의 주석으로부터 Swagger/OpenAPI 문서를 자동으로 생성합니다.
- API 문서는 핸들러(`src/handler/`), 모델(`src/model/`), 메인 함수(`src/main.go`) 등의 코드에 작성된 특정 형식의 주석(`// @Summary`, `// @Param`, `// @Success`, `// @Router` 등)을 기반으로 생성됩니다.
- **문서 업데이트 방법:**
    1. 관련 Go 소스 코드의 주석을 수정합니다.
    2. 프로젝트 루트 디렉토리에서 다음 명령어를 실행하여 Swagger 문서를 업데이트합니다:
       ```bash
       swag init -g src/main.go -o src/docs
       ```
       (참고: `-g` 플래그는 주석 파싱 시작점을 지정하고, `-o` 플래그는 생성된 파일(`docs.go`, `swagger.json`, `swagger.yaml`)이 저장될 디렉토리를 지정합니다. 현재 프로젝트에서는 `src/docs`를 사용합니다.)
    3. 생성된 `src/docs` 디렉토리의 파일들을 Git에 커밋합니다. (프로젝트 루트의 `docs` 디렉토리는 현재 사용되지 않거나 중복일 수 있으므로 확인 및 정리가 필요할 수 있습니다.)
- **문서 확인:** 애플리케이션 실행 후 `/swagger/index.html` 경로로 접속하여 Swagger UI를 통해 API 문서를 확인할 수 있습니다.

## 5. 코딩 표준

### 5.1 일반 규칙
- Go 코드 스타일 가이드 준수
- 의미 있는 변수명과 함수명 사용
- 적절한 주석 작성
- 에러 처리 일관성 유지

### 5.2 패키지 구조
- 단일 책임 원칙 준수
- 의존성 주입 사용
- 인터페이스 기반 설계
- 모듈화된 구조 유지

### 5.2.1 의존성 관리 및 초기화 (2025-04-18 업데이트)
- **초기화 흐름:**
    - `main.go`: 애플리케이션 시작 시 필요한 전역 설정(DB 연결, Keycloak 클라이언트/설정)을 초기화합니다 (`config` 패키지 활용).
    - `main.go`: 각 HTTP 요청을 처리할 핸들러(`src/handler/`)를 초기화합니다. 이때, 핸들러 생성자에는 **`*gorm.DB` 인스턴스만** 전달하는 것을 원칙으로 합니다. (Keycloak 관련 객체는 전달하지 않습니다.)
    - 핸들러 (`src/handler/`): 생성자 내부에서 자신이 직접 사용하는 서비스(`src/service/`)를 초기화합니다. 서비스 생성자에는 `*gorm.DB` 인스턴스를 전달합니다.
    - 서비스 (`src/service/`): 생성자 내부에서 자신이 직접 사용하는 리포지토리(`src/repository/`)를 초기화합니다. 대부분의 리포지토리 생성자는 `*gorm.DB`를 받습니다. (단, `PermissionRepository`처럼 트랜잭션만 받는 경우는 예외)
- **DB 트랜잭션:**
    - DB 트랜잭션은 **서비스 계층**에서 시작하고 관리하는 것을 원칙으로 합니다 (`db.WithContext(ctx)` 또는 `db.Begin()`).
    - 생성된 트랜잭션 객체(`*gorm.DB`)를 리포지토리 메소드에 전달하여 해당 트랜잭션 내에서 DB 작업을 수행합니다.
    - 리포지토리는 전달받은 트랜잭션 객체를 사용하여 `tx.Create()`, `tx.Find()` 등을 호출합니다.
- **Keycloak 연동:**
    - Keycloak API 호출 로직은 `src/service/keycloak_service.go`의 `KeycloakService`에 중앙화되어 있습니다.
    - `KeycloakService`는 상태를 가지지 않으며(stateless), 내부 메소드에서 `config` 패키지의 전역 변수(`config.KC`, `config.KC.Client`)를 직접 참조하여 Keycloak API를 호출합니다.
    - Keycloak 기능이 필요한 서비스(`UserService`, `HealthCheckService`)나 핸들러(`AuthHandler`)는 해당 기능이 필요한 **메소드 내에서** `service.NewKeycloakService()`를 호출하여 임시 인스턴스를 생성하고 사용합니다. (의존성 주입 X)
- **컨텍스트(Context):**
    - `context.Context`는 핸들러에서 시작되어 서비스 계층으로 전달됩니다.
    - 서비스 계층은 전달받은 컨텍스트를 사용하여 DB 트랜잭션을 생성(`db.WithContext(ctx)`)하거나 Keycloak API 호출 시 전달합니다.
    - 리포지토리 계층은 일반적으로 컨텍스트 대신 트랜잭션 객체를 전달받습니다.
    - **`AuthMiddleware`**는 성공적으로 토큰을 검증한 후, 요청 컨텍스트(`c.Set()`)에 다음 정보를 저장합니다:
        - `"token_claims"`: 디코딩된 JWT 클레임 (`*jwt.MapClaims` 타입)
        - `"access_token"`: 원본 액세스 토큰 문자열
        - `"kcUserId"`: 토큰의 Subject 클레임 (Keycloak User ID)
    - 핸들러에서는 `c.Get("kcUserId").(string)`과 같이 컨텍스트에서 사용자 ID를 가져와 사용할 수 있습니다.

### 5.3 테스트
- 단위 테스트 작성
- 테스트 커버리지 유지
- 테스트 케이스 문서화
- CI/CD 파이프라인 통합

## 6. 보안 가이드라인

### 6.1 인증
- JWT 토큰 사용
- 토큰 갱신 메커니즘
- 세션 관리
- 로그아웃 처리

### 6.2 권한
- RBAC (Role-Based Access Control) 구현
- 최소 권한 원칙 적용
- 권한 검증 미들웨어
- 감사 로그 기록

### 6.3 데이터 보안
- 민감한 데이터 암호화
- SQL 인젝션 방지
- XSS 방지
- CSRF 보호

## 7. 배포 가이드라인

### 7.1 Docker 배포
- 멀티 스테이지 빌드 (`dockerfiles/mc-iam-manager/Dockerfile.mciammanager` 참조)
- 최적화된 이미지 크기
- 환경별 설정 관리 (`.env` 파일 사용)
- 볼륨 마운트 설정 (`docker-compose.<scenario>.yaml` 참조)
- **실행 방법:**
    - 프로젝트 루트 디렉토리에서 다음 단계를 따릅니다.
    1.  실행하려는 시나리오에 해당하는 `.yaml` 파일을 `docker-compose.yaml`로 복사하거나 이름을 변경합니다.
        -   `docker-compose.standalone.yaml` (mc-iam-manager 단독)
        -   `docker-compose.with-db.yaml` (DB 포함)
        -   `docker-compose.with-keycloak.yaml` (Keycloak, Nginx, Certbot 포함)
        -   `docker-compose.all.yaml` (모든 서비스 포함)
        ```bash
        # 예시: with-db 시나리오 실행 준비
        cp docker-compose.with-db.yaml docker-compose.yaml
        ```
    2.  필요한 환경 변수가 `.env` 파일에 올바르게 설정되었는지 확인합니다.
    3.  다음 명령어를 사용하여 서비스를 시작합니다.
        ```bash
        docker-compose up -d
        ```
    - 서비스 중지: `docker-compose down`
- **필수 설정 (`.env`):**
    - `PORT`: mc-iam-manager 실행 포트 (기본값: 8082)
    - `IAM_POSTGRES_USER`, `IAM_POSTGRES_PASSWORD`, `IAM_POSTGRES_DB`: PostgreSQL 접속 정보
    - `KEYCLOAK_ADMIN`, `KEYCLOAK_ADMIN_PASSWORD`: Keycloak 관리자 정보 (Docker Compose 및 내부 Keycloak API 호출 시 사용)
    - `KEYCLOAK_HOST`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENTID`, `KEYCLOAK_CLIENTSECRET`: Keycloak 연동 정보
    - `DOMAIN_NAME`: Nginx 및 Keycloak에서 사용할 도메인 이름 (기본값: localhost)
    - `DEFAULT_WORKSPACE_NAME`: 할당되지 않은 프로젝트가 속할 기본 워크스페이스 이름 (기본값: "default")
    - `MCADMINCLI_APIYAML`: MCMP API 정의 YAML 경로/URL
    - `.env_sample` 파일을 참고하여 필요한 모든 변수를 설정해야 합니다.
- **Keycloak UMA 설정:** (별도 문서 또는 Keycloak 관리 콘솔 참고)
    - `mciamClient` 클라이언트의 Authorization 활성화.
    - 리소스, 스코프, 그룹 기반 정책, 퍼미션 설정 필요 (MC-IAM 권한 기준).
- **볼륨 및 설정 파일 경로:**
    - 컨테이너 데이터 볼륨: `./dockercontainer-volume/` 하위 (postgres, keycloak, certs, certbot)
    - 서비스 설정 파일: `./dockerfiles/<service_name>/` 하위 (postgres, nginx)

### 7.2 모니터링
- 로깅 설정
- 메트릭 수집
- 알림 설정
- 성능 모니터링

### 7.3 백업 및 복구
- 데이터베이스 백업
- 설정 백업
- 복구 절차
- 장애 대응 계획

## 8. 유지보수 가이드라인

### 8.1 버전 관리
- 시맨틱 버저닝 사용
- 변경 로그 관리
- 브랜치 전략
- 릴리스 프로세스

### 8.2 문서화
- API 문서 유지보수
- 코드 문서화
- 운영 문서
- 문제 해결 가이드

### 8.3 성능 최적화
- 쿼리 최적화
- 캐싱 전략
- 리소스 사용 최적화
- 부하 테스트

## 1. 권한 모델

### 1.1 리소스 유형 (Resource Type)
- 프레임워크별 리소스 유형 관리
- 테이블: `mcmp_resource_types`
- 주요 필드:
  - `framework_id`: 프레임워크 식별자 (e.g., "mc-iam-manager", "mc-infra-manager")
  - `id`: 프레임워크 내 유니크한 식별자
  - `name`: 표시 이름
  - `description`: 설명

### 1.2 MC-IAM 권한
- 테이블: `mcmp_mciam_permissions`
- 권한 ID 형식: `{framework_id}:{resource_type_id}:{action}`
- 예시:
  - `mc-iam-manager:workspace:create`
  - `mc-iam-manager:workspace:read`
  - `mc-infra-manager:vm:create`
  - `mc-infra-manager:vm:read`

### 1.3 역할 및 권한 매핑
- 워크스페이스 역할 - MC-IAM 권한 매핑
  - 테이블: `mcmp_mciam_role_permissions`
  - 역할 타입: 'workspace'
  - 권한 할당/제거 API: `/api/roles/{roleType}/{roleId}/mciam-permissions/{permissionId}`

- 워크스페이스 역할 - CSP 역할 매핑
  - 테이블: `mcmp_workspace_role_csp_role_mapping`
  - CSP 타입별 역할 매핑 관리
  - API: `/api/workspace-roles/{roleId}/csp-role-mappings`

## 2. 워크스페이스 접근 제어

### 2.1 워크스페이스 권한
- `mc-iam-manager:workspace:list_all`: 모든 워크스페이스 목록 조회 (플랫폼 관리자)
- `mc-iam-manager:workspace:list_assigned`: 할당된 워크스페이스 목록 조회
- `mc-iam-manager:workspace:create`: 워크스페이스 생성
- `mc-iam-manager:workspace:read`: 워크스페이스 상세 조회

### 2.2 API 권한 검증
- 워크스페이스 목록 조회 (`/api/workspaces`)
  - `list_all` 또는 `list_assigned` 권한 필요
- 워크스페이스 생성 (`/api/workspaces`)
  - `create` 권한 필요
- 워크스페이스 상세 조회 (`/api/workspaces/{id}`)
  - `read` 권한 필요

## 3. MCMP API 호출 권한 관리

### 3.1 Action Ticket (RPT) 발급
- API: `POST /api/auth/action-ticket`
- 요청 본문:
```json
{
  "service_name": "string",
  "action_name": "string"
}
```

### 3.2 MCMP API 호출 권한 검증
- 미들웨어: `McmpApiAuthMiddleware`
- 권한 형식: `{service_name}#{action_name}`
- RPT 토큰의 `authorization.permissions` 클레임 검증

## 4. CSP 임시 자격 증명 발급

### 4.1 API 엔드포인트
- `POST /api/csp/credentials`
- 요청 본문:
```json
{
  "csp_type": "string",
  "workspace_id": "string"
}
```

### 4.2 권한 검증
- 워크스페이스 역할 기반 CSP 역할 매핑 확인
- 임시 자격 증명 발급

## 5. 환경 설정

### 5.1 필수 환경 변수
```env
# 데이터베이스
DB_HOST=
DB_PORT=
DB_USER=
DB_PASSWORD=
DB_NAME=

# Keycloak
KEYCLOAK_URL=
KEYCLOAK_REALM=
KEYCLOAK_CLIENT_ID=
KEYCLOAK_CLIENT_SECRET=
```

### 5.2 Keycloak UMA 설정
- Resources: MCMP API 액션
- Scopes: 액션 이름
- Policies: 역할 기반
- Permissions: 역할-액션 매핑

## 6. API 문서
- Swagger UI: `http://localhost:8082/swagger/index.html`
- API 그룹:
  - 인증 API
  - 사용자 관리 API
  - 워크스페이스 관리 API
  - 역할 관리 API
  - 권한 관리 API
  - MCMP API
  - CSP 자격 증명 API

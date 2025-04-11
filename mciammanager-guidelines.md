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
│   ├── handler/        # HTTP 요청 처리
│   ├── middleware/     # 미들웨어
│   ├── model/         # 데이터 모델
│   ├── repository/    # 데이터베이스 작업
│   ├── service/       # 비즈니스 로직 (e.g., user_service.go)
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

#### 4.1.1 메뉴 YAML 파일 준비 (선택 사항)
- 메뉴 데이터를 YAML 파일로부터 초기 등록하거나 동기화하려면 `asset/menu/menu.yaml` 파일을 준비합니다.
- 파일 소스는 `.env` 파일의 `MCWEBCONSOLE_MENUYAML` 환경 변수를 통해 지정하거나, 기본 URL(`https://raw.githubusercontent.com/m-cmp/mc-web-console/refs/heads/main/conf/webconsole_menu_resources.yaml`)을 사용할 수 있습니다.
- 파일 다운로드 예시: `curl -L [소스_URL_또는_경로] -o asset/menu/menu.yaml --create-dirs`

2. 데이터베이스 설정
   - PostgreSQL 컨테이너 실행
   - 마이그레이션 실행 (`/asset/sql/migrations` 참조, 모든 테이블에 `mcmp_` 접두사 적용 및 역할/권한 구조 개선 마이그레이션 포함)
   - 초기 데이터 설정 (필요시 마이그레이션(`000005_...`) 사용)

3. Keycloak 설정
   - Keycloak 컨테이너 실행
   - Realm 생성
   - 클라이언트 설정 (`mciamClient` 등)
   - 사용자 및 역할 설정
   - **서비스 계정 역할 설정 (중요):**
     - `mciamClient` (또는 `.env`의 `MCIAMMANAGER_KEYCLOAK_CLIENTID` 클라이언트)의 서비스 계정이 활성화되어 있는지 확인합니다.
     - 해당 서비스 계정에 필요한 역할(Role)을 부여해야 합니다. 특히, 사용자 정보 조회가 필요한 기능(예: `GET /users`)을 사용하려면 **"Service Account Roles"** 탭에서 **`realm-management` 클라이언트의 `view-users` 역할**을 할당해야 합니다. (다른 관리 API 사용 시 추가 역할 필요)

### 4.2 핵심 기능 구현

#### 4.2.1 사용자 관리 (`mcmp_users` 테이블 및 Keycloak)
- **사용자 등록:** Keycloak 자체 등록 기능 사용 (초기 상태: 비활성).
- **사용자 승인:**
    - 관리자(admin/platform_superadmin)가 Keycloak 콘솔 또는 `POST /api/users/{kc_id}/approve` API를 통해 사용자를 활성화(`enabled=true`).
- **로그인 및 동기화:**
    - 사용자가 `POST /api/auth/login`으로 로그인 시도.
    - Keycloak 인증 성공 후, 사용자의 `enabled` 상태 확인. 비활성 시 403 Forbidden 반환.
    - 활성 상태이면, 로컬 `mcmp_users` DB에 해당 사용자 정보가 있는지 확인하고 없으면 Keycloak 정보를 바탕으로 생성 (DB 동기화).
    - 동기화 후 로그인 토큰 발급.
- **관리자용 사용자 생성:** `POST /api/users` API (admin/platform_superadmin 권한 필요)는 Keycloak에 즉시 활성 상태로 사용자를 생성하고 로컬 DB에도 동기화.
- **기타 관리:** 사용자 조회(`GET /users`, `GET /users/{id}`, `GET /users/username/{username}`), 수정(`PUT /users/{id}`), 삭제(`DELETE /users/{id}`) API 제공 (적절한 권한 필요).
- 로컬 DB(`mcmp_users`)에는 Keycloak User ID (`kc_id`), 사용자 이름(`username`), 추가 정보(예: `description`)가 저장됩니다. (Email, FirstName, LastName 등은 Keycloak에서 관리)
- **최고 관리자 동기화:** 애플리케이션 시작 시 `.env`의 `MCIAMMANAGER_PLATFORMADMIN_ID` 사용자를 확인하고, 로컬 DB에 동기화하며 'platform_superadmin' 역할을 부여합니다 (`UserService.SyncPlatformAdmin` 로직).

#### 4.2.2 워크스페이스 관리 (`mcmp_workspaces` 테이블)
- 워크스페이스 CRUD API 구현 (`/workspaces`, `/workspaces/{id}`)
- 워크스페이스 이름으로 조회 API 구현 (`/workspaces/name/{name}`)
- 워크스페이스-프로젝트 연결/해제 API 구현 (`/workspaces/{id}/projects/{projectId}`)

#### 4.2.3 프로젝트 관리 (`mcmp_projects` 테이블)
- 프로젝트 CRUD API 구현 (`/projects`, `/projects/{id}`)
- 프로젝트 이름으로 조회 API 구현 (`/projects/name/{name}`)
- 프로젝트-워크스페이스 연결/해제 API 구현 (`/projects/{id}/workspaces/{workspaceId}`)
- 워크스페이스와 프로젝트는 M:N 관계 (`mcmp_workspace_projects` 매핑 테이블 사용)

#### 4.2.4 역할 관리 (`mcmp_platform_roles`, `mcmp_workspace_roles` 테이블)
- 플랫폼 역할 CRUD API 구현
- 워크스페이스 역할 CRUD API 구현
- 사용자에게 플랫폼/워크스페이스 역할 할당/제거 API 구현 (`mcmp_user_platform_roles`, `mcmp_user_workspace_roles` 테이블 사용)
- 역할 기반 접근 제어 (미들웨어 등에서 활용)

#### 4.2.5 권한 관리 (`mcmp_permissions`, `mcmp_role_permissions` 테이블)
- 권한 CRUD API 구현
- 역할(플랫폼/워크스페이스)에 권한 할당/제거 API 구현 (`/api/roles/{roleType}/{roleId}/permissions/{permissionId}`)
- 권한 검증 로직 구현

#### 4.2.6 메뉴 관리 (`mcmp_menu` 테이블)
- 메뉴 데이터는 PostgreSQL 데이터베이스의 `mcmp_menu` 테이블에 저장 및 관리됩니다.
- **사용자 메뉴 트리 조회:** `GET /menus` API는 현재 로그인한 사용자의 Platform Role에 따라 접근 가능한 메뉴 목록을 트리 구조(`[]model.MenuTreeNode`)로 반환합니다. 하위 메뉴 접근 권한이 있으면 상위 메뉴도 포함됩니다.
- **개별 메뉴 조회:** `GET /menus/{id}` API로 특정 메뉴 정보를 조회합니다.
- **메뉴 생성/수정/삭제:** `POST /menus`, `PUT /menus/{id}` (부분 업데이트 지원), `DELETE /menus/{id}` API를 통해 메뉴를 직접 관리할 수 있습니다.
- **YAML 파일/URL 등록/동기화:** `POST /menus/register-from-yaml` API를 호출합니다.
    - `filePath` 쿼리 파라미터가 있으면 해당 로컬 경로의 YAML 파일을 읽어 DB에 Upsert합니다.
    - `filePath` 파라미터가 없으면, `.env` 파일의 `MCWEBCONSOLE_MENUYAML` 환경 변수를 확인합니다.
        - 값이 URL이면 해당 URL에서 YAML을 다운로드하여 `asset/menu/menu.yaml`에 저장한 후, 이 파일을 읽어 DB에 Upsert합니다.
        - 값이 로컬 경로이면 해당 경로의 파일을 읽어 DB에 Upsert합니다.
        - 값이 없거나 URL이 아니면 기본 로컬 경로(`asset/menu/menu.yaml`)를 읽어 DB에 Upsert합니다.
- **YAML 본문 등록/동기화:** `POST /menus/register-from-body` API를 호출하여 요청 본문에 포함된 YAML 텍스트 내용을 읽어 DB에 Upsert할 수 있습니다. (Content-Type: text/plain, text/yaml, application/yaml 등)
- 테이블 스키마 및 변경 사항은 `/asset/sql/migrations` 디렉토리의 마이그레이션 파일을 통해 관리됩니다. (FK 제약 조건 지연 설정 포함)

### 4.3 API 문서화
- Swagger/OpenAPI 문서 작성
- API 엔드포인트 설명
- 요청/응답 예제
- 에러 코드 정의

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
- 멀티 스테이지 빌드
- 최적화된 이미지 크기
- 환경별 설정 관리
- 볼륨 마운트 설정

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

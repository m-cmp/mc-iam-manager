# MC-IAM-Manager 개발 지침서

## 문서 정보
- **버전**: 1.0
- **작성일**: 2026-02-09
- **대상**: MC-IAM-Manager 개발자
- **목적**: 신규 기능 개발 시 참고할 표준 프로세스 정의

---

## 목차
1. [개발 환경 설정](#1-개발-환경-설정)
2. [브랜치 전략 및 Worktree](#2-브랜치-전략-및-worktree)
3. [요구사항 분석](#3-요구사항-분석)
4. [설계 및 구현](#4-설계-및-구현)
5. [테스트](#5-테스트)
6. [문서화](#6-문서화)
7. [코드 리뷰 및 병합](#7-코드-리뷰-및-병합)
8. [체크리스트](#8-체크리스트)

---

## 1. 개발 환경 설정

### 1.1 필수 도구

#### 시스템 요구사항
```bash
OS: Linux (WSL2 권장) 또는 macOS
Go: 1.25.0 이상
Git: 2.30 이상
```

#### 개발 도구
```bash
# Go 개발 도구
go install github.com/swaggo/swag/cmd/swag@latest

# GitHub CLI (선택)
# Ubuntu/Debian
sudo apt install gh

# macOS
brew install gh
```

### 1.2 프로젝트 클론
```bash
# 메인 프로젝트
git clone https://github.com/MZC-CSC/mc-iam-manager.git
cd mc-iam-manager

# 의존성 설치
cd src
go mod download
```

### 1.3 환경 변수 설정
```bash
# .env 파일 생성
cp .env_sample .env

# 필수 환경 변수 설정
vi .env
```

**주요 환경 변수:**
- `KC_HOST`: Keycloak 호스트
- `KC_PORT`: Keycloak 포트
- `KC_REALM`: Realm 이름
- `DB_HOST`: PostgreSQL 호스트
- `DB_PORT`: PostgreSQL 포트

---

## 2. 브랜치 전략 및 Worktree

### 2.1 브랜치 전략

#### 브랜치 종류
```
main           - 프로덕션 릴리스
├─ development - 개발 통합 브랜치
   ├─ feature/xxx  - 신규 기능 개발
   ├─ bugfix/xxx   - 버그 수정
   └─ hotfix/xxx   - 긴급 수정
```

#### 브랜치 네이밍 규칙
```bash
# 기능 개발
feature/기능명-간단설명
예: feature/user-signup-approval

# 버그 수정
bugfix/이슈번호-버그설명
예: bugfix/123-login-error

# 긴급 수정
hotfix/이슈번호-수정내용
예: hotfix/456-security-patch
```

### 2.2 Git Worktree 사용

#### Worktree란?
- 하나의 저장소에서 여러 브랜치를 동시에 작업
- 브랜치 전환 없이 독립적인 작업 공간 유지

#### Worktree 생성
```bash
# 현재 위치 확인
pwd
# /home/user/project/mc-iam-manager

# development 브랜치 기반으로 새 feature 브랜치 생성
git worktree add -b feature/새기능명 ../mc-iam-manager-새기능명 development

# 결과:
# /home/user/project/mc-iam-manager (원본, development)
# /home/user/project/mc-iam-manager-새기능명 (worktree, feature/새기능명)
```

#### Worktree 확인
```bash
git worktree list
```

#### Worktree에서 작업
```bash
# worktree로 이동
cd ../mc-iam-manager-새기능명

# 브랜치 확인
git branch --show-current
# feature/새기능명

# 작업 수행
# ... 코드 수정 ...

# 커밋
git add .
git commit -m "작업 내용"

# 푸시
git push origin feature/새기능명
```

#### Worktree 제거
```bash
# 작업 완료 후
cd ../mc-iam-manager
git worktree remove ../mc-iam-manager-새기능명

# 또는 디렉토리 삭제 후
rm -rf ../mc-iam-manager-새기능명
git worktree prune
```

---

## 3. 요구사항 분석

### 3.1 요구사항 수집

#### 요구사항 문서 양식
```markdown
# FR-XXX: 기능명

## 개요
- 목적:
- 우선순위: 높음/중간/낮음

## 기능 요구사항
| ID | 요구사항 | 상세 설명 |
|----|---------|----------|
| FR-XXX-01 | ... | ... |

## 비기능 요구사항
- 성능:
- 보안:
- 확장성:

## 제약사항
```

### 3.2 현재 구현 상태 분석

#### 분석 체크리스트
```bash
# 1. 관련 API 확인
grep -r "기능명" src/handler/

# 2. 데이터 모델 확인
grep -r "type 모델명" src/model/

# 3. 서비스 로직 확인
grep -r "func.*기능명" src/service/

# 4. 라우트 확인
grep "기능명" src/main.go
```

#### Gap 분석 템플릿
| 요구사항 | 현재 구현 | 문제점 | 해결방안 | 우선순위 |
|---------|----------|--------|----------|---------|
| ... | ... | ... | ... | 높음/중간/낮음 |

### 3.3 구현 계획 수립

#### 작업 분해 (WBS)
```
Phase 1: 데이터 모델
  - Task 1-1: Request 모델 추가
  - Task 1-2: Response 모델 추가
  - Task 1-3: Error 모델 추가

Phase 2: 비즈니스 로직
  - Task 2-1: Service 구현
  - Task 2-2: Handler 구현
  - Task 2-3: Validation 추가

Phase 3: 통합
  - Task 3-1: 라우트 등록
  - Task 3-2: Middleware 적용
  - Task 3-3: 권한 검증

Phase 4: 문서화
  - Task 4-1: Swagger 업데이트
  - Task 4-2: Service Actions 생성
  - Task 4-3: 가이드 문서 작성
```

---

## 4. 설계 및 구현

### 4.1 코드 구조

#### 디렉토리 구조
```
src/
├── config/          # 설정 관리
├── constants/       # 상수 정의
├── handler/         # HTTP 핸들러
├── middleware/      # 미들웨어
├── model/          # 데이터 모델
├── repository/      # 데이터베이스 레포지토리
├── service/        # 비즈니스 로직
├── utils/          # 유틸리티 함수
└── main.go         # 진입점
```

#### 레이어 구조
```
Client Request
    ↓
Handler (HTTP 처리, 검증)
    ↓
Service (비즈니스 로직)
    ↓
Repository/Keycloak (데이터 접근)
    ↓
Database/Keycloak
```

### 4.2 모델 구현

#### Request 모델
**위치**: `src/model/request.go`

**가이드라인:**
- 명확한 네이밍: `XxxRequest`
- validation 태그 추가
- JSON 태그 camelCase
- 주석으로 설명 추가

**예시:**
```go
// SignupRequest represents the signup form data
type SignupRequest struct {
    Email        string `json:"email" validate:"required,email"`
    Password     string `json:"password" validate:"required,min=8"`
    FirstName    string `json:"firstName" validate:"required"`
    LastName     string `json:"lastName" validate:"required"`
    Organization string `json:"organization,omitempty"` // 선택 필드
}
```

**Validation 태그:**
- `required`: 필수 입력
- `email`: 이메일 형식
- `min=n`: 최소 길이
- `max=n`: 최대 길이
- `oneof=a b c`: 열거형 값

#### Response 모델
**위치**: `src/model/response.go` 또는 `src/model/error.go`

**가이드라인:**
- 성공/실패를 명확히 구분
- 에러 코드 상수 정의
- 필드별 에러 메시지 지원

**예시:**
```go
type ErrorResponse struct {
    Success bool              `json:"success"`
    Error   string            `json:"error"`
    Fields  map[string]string `json:"fields,omitempty"`
    Code    string            `json:"code,omitempty"`
}

// Error codes
const (
    ErrCodeValidation = "VALIDATION_FAILED"
    ErrCodeNotFound   = "NOT_FOUND"
    ErrCodeConflict   = "CONFLICT"
)
```

### 4.3 Validation 구현

#### Validator 유틸리티
**위치**: `src/utils/validator.go`

**구현 예시:**
```go
package utils

import (
    "fmt"
    "strings"
    "github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
    validate = validator.New()
}

func ValidateStruct(s interface{}) error {
    return validate.Struct(s)
}

func FormatValidationErrorMap(err error) map[string]string {
    errMap := make(map[string]string)
    if err == nil {
        return errMap
    }

    validationErrs, ok := err.(validator.ValidationErrors)
    if !ok {
        errMap["error"] = "유효하지 않은 입력입니다"
        return errMap
    }

    for _, e := range validationErrs {
        field := strings.ToLower(e.Field())
        errMap[field] = getFieldErrorMessage(e)
    }
    return errMap
}

func getFieldErrorMessage(e validator.FieldError) string {
    switch e.Tag() {
    case "required":
        return fmt.Sprintf("%s은(는) 필수 입력입니다", e.Field())
    case "email":
        return "유효한 이메일 형식이 아닙니다"
    case "min":
        return fmt.Sprintf("최소 %s자 이상이어야 합니다", e.Param())
    default:
        return fmt.Sprintf("%s 검증 실패", e.Field())
    }
}
```

### 4.4 Service 구현

#### 인터페이스 정의
**위치**: `src/service/xxx_service.go`

**가이드라인:**
- 인터페이스 먼저 정의
- 에러 핸들링 명확히
- 트랜잭션 고려
- 로깅 추가

**예시:**
```go
type KeycloakService interface {
    CreatePendingUser(ctx context.Context, req *model.SignupRequest) (string, error)
    // ... 다른 메서드들
}

type keycloakService struct {
    // 의존성 주입
}

func NewKeycloakService() KeycloakService {
    return &keycloakService{}
}

func (s *keycloakService) CreatePendingUser(ctx context.Context, req *model.SignupRequest) (string, error) {
    // 1. 입력 검증
    if config.KC == nil || config.KC.Client == nil {
        return "", fmt.Errorf("keycloak configuration not initialized")
    }

    // 2. Admin 토큰 획득
    token, err := config.KC.LoginAdmin(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get admin token: %w", err)
    }

    // 3. 비즈니스 로직
    username := strings.Split(req.Email, "@")[0]
    keycloakUser := gocloak.User{
        Username:  &username,
        Email:     &req.Email,
        Enabled:   gocloak.BoolP(false),
        // ...
    }

    // 4. Keycloak API 호출
    kcId, err := config.KC.Client.CreateUser(ctx, token.AccessToken, config.KC.Realm, keycloakUser)
    if err != nil {
        // 에러 처리
        if strings.Contains(err.Error(), "409") {
            return "", fmt.Errorf("이미 사용 중인 이메일입니다")
        }
        return "", fmt.Errorf("failed to create user: %w", err)
    }

    // 5. 추가 작업 (비밀번호 설정 등)
    err = config.KC.Client.SetPassword(ctx, token.AccessToken, kcId, config.KC.Realm, req.Password, false)
    if err != nil {
        // 롤백
        config.KC.Client.DeleteUser(ctx, token.AccessToken, config.KC.Realm, kcId)
        return "", fmt.Errorf("failed to set password: %w", err)
    }

    // 6. 로깅
    log.Printf("[INFO] User created successfully: %s", kcId)

    return kcId, nil
}
```

### 4.5 Handler 구현

#### Handler 함수 템플릿
**위치**: `src/handler/xxx_handler.go`

**구조:**
```go
// FunctionName godoc
// @Summary 간단한 설명
// @Description 상세 설명
// @Tags 태그명
// @Accept json
// @Produce json
// @Param 파라미터명 파라미터위치 파라미터타입 필수여부 "설명"
// @Success 200 {object} 응답타입
// @Failure 400 {object} 에러타입
// @Security BearerAuth
// @Router /api/경로 [method]
// @Id operationId
func (h *Handler) FunctionName(c echo.Context) error {
    // 1. 권한 확인 (필요시)
    requiredRoles := []string{"admin", "platformAdmin"}
    if !checkRoleFromContext(c, requiredRoles) {
        return c.JSON(http.StatusForbidden, map[string]string{
            "error": "Forbidden: Administrator access required",
        })
    }

    // 2. 파라미터 파싱
    param := c.Param("id")

    // 3. Request Body 바인딩
    var req model.XxxRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request format",
        })
    }

    // 4. 유효성 검증
    if err := utils.ValidateStruct(req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "error":  "입력값 검증에 실패했습니다",
            "fields": utils.FormatValidationErrorMap(err),
        })
    }

    // 5. 서비스 호출
    result, err := h.service.DoSomething(c.Request().Context(), &req)
    if err != nil {
        // 에러 타입별 처리
        if errors.Is(err, repository.ErrNotFound) {
            return c.JSON(http.StatusNotFound, map[string]string{
                "error": "리소스를 찾을 수 없습니다",
            })
        }
        if strings.Contains(err.Error(), "conflict") {
            return c.JSON(http.StatusConflict, map[string]string{
                "error": "이미 존재하는 리소스입니다",
            })
        }
        log.Printf("[ERROR] FunctionName failed: %v", err)
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "서버 오류가 발생했습니다",
        })
    }

    // 6. 성공 응답
    return c.JSON(http.StatusOK, map[string]interface{}{
        "success": true,
        "data":    result,
    })
}
```

#### HTTP 상태 코드 가이드
| 코드 | 의미 | 사용 시점 |
|------|------|----------|
| 200 | OK | 조회/수정 성공 |
| 201 | Created | 생성 성공 |
| 204 | No Content | 삭제 성공 |
| 400 | Bad Request | 입력값 검증 실패 |
| 401 | Unauthorized | 인증 실패 |
| 403 | Forbidden | 권한 없음 |
| 404 | Not Found | 리소스 없음 |
| 409 | Conflict | 중복/충돌 |
| 500 | Internal Server Error | 서버 오류 |

### 4.6 라우트 등록

#### Public 라우트
**위치**: `src/main.go`

```go
// 인증이 필요없는 경로
skipAuthPaths := []string{
    "/readyz",
    "/api/auth/login",
    "/api/auth/signup",  // 추가
}
```

#### Protected 라우트
```go
// 인증 필요 (기본)
api := e.Group("/api")

// 그룹별 라우트
auth := api.Group("/auth")
{
    auth.POST("/signup", userHandler.SignupUser)  // Public
    auth.POST("/login", authHandler.Login)
}

users := api.Group("/users")
{
    // platformAdmin만 접근 가능
    users.PUT("/id/:userId/password",
        userHandler.ResetUserPassword,
        middleware.PlatformRoleMiddleware(middleware.Manage))
}
```

---

## 5. 테스트

### 5.1 빌드 테스트

#### 빌드 명령
```bash
cd src
go build -o ../mc-iam-manager .
```

#### 빌드 확인
```bash
# 바이너리 생성 확인
ls -lh ../mc-iam-manager

# 실행 가능 여부 확인
../mc-iam-manager --version
```

#### 일반적인 빌드 오류
| 오류 | 원인 | 해결 |
|------|------|------|
| `undefined: xxx` | import 누락 | import 추가 |
| `cannot find package` | 의존성 누락 | `go mod tidy` |
| `syntax error` | 문법 오류 | 코드 수정 |

### 5.2 Swagger 문서 생성

#### 생성 명령
```bash
cd src
swag init --parseDependency --parseInternal
```

#### 생성 파일 확인
```bash
ls -lh docs/
# docs.go
# swagger.json
# swagger.yaml
```

#### 새 API 포함 확인
```bash
# operationId 확인
grep "operationId: 함수명" docs/swagger.yaml

# API 경로 확인
grep "/api/경로:" docs/swagger.yaml

# Request 모델 확인
jq '.definitions["model.모델명"]' docs/swagger.json
```

#### Swagger 주석 체크리스트
- [ ] @Summary 작성
- [ ] @Description 작성
- [ ] @Tags 지정
- [ ] @Accept, @Produce 지정
- [ ] @Param 모두 정의
- [ ] @Success, @Failure 정의
- [ ] @Security (인증 필요시)
- [ ] @Router 경로 및 메서드
- [ ] @Id operationId (함수명과 동일)

### 5.3 Service Actions 생성

#### frameworks.yaml 확인
**위치**: `tool/swagger-to-actions/frameworks.yaml`

```yaml
frameworks:
  - name: mc-iam-manager
    version: "0.3.0"
    repository: https://github.com/m-cmp/mc-iam-manager
    swagger: ../../src/docs/swagger.yaml  # 경로 확인 중요
```

#### 생성 명령
```bash
cd tool/swagger-to-actions
go run . -c frameworks.yaml
```

#### 생성 결과 확인
```bash
# 파일 생성 확인
ls -lh service-actions.yaml

# 새 API 포함 확인
grep "함수명:" service-actions.yaml

# 액션 수 확인
grep "method:" service-actions.yaml | wc -l
```

#### 파일 복사
```bash
# asset 디렉토리로 복사
cp service-actions.yaml ../../asset/mcmpapi/
cp frameworks.yaml ../../asset/mcmpapi/

# conf 디렉토리로 복사
cp service-actions.yaml ../../conf/mc-iam-manager/
cp frameworks.yaml ../../conf/mc-iam-manager/
```

### 5.4 수동 테스트 시나리오

#### 테스트 환경 준비
```bash
# 1. Docker Compose 실행
docker-compose up -d

# 2. 애플리케이션 실행
./mc-iam-manager

# 3. 관리자 토큰 발급
ADMIN_TOKEN=$(curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"id": "admin", "password": "admin123"}' \
  | jq -r '.access_token')
```

#### 기본 테스트 케이스
```bash
# TC-001: 정상 케이스
curl -X POST http://localhost:5000/api/새경로 \
  -H "Content-Type: application/json" \
  -d '{"field": "value"}'

# TC-002: 검증 실패
curl -X POST http://localhost:5000/api/새경로 \
  -H "Content-Type: application/json" \
  -d '{"field": "invalid"}'

# TC-003: 권한 없음 (인증 필요 API)
curl -X POST http://localhost:5000/api/새경로 \
  -H "Content-Type: application/json" \
  -d '{"field": "value"}'
```

---

## 6. 문서화

### 6.1 필수 문서

#### 요구사항 분석서
**파일명**: `docs/features/FR-XXX-ANALYSIS.md`

**구조:**
```markdown
# FR-XXX 기능명 - 요구사항 분석서

## 1. 개요
## 2. 요구사항 분석
### 2.1 FR-XXX-01: 세부 요구사항
#### 요구사항
#### 현재 구현 상태
#### Gap 분석
#### 구현 방안
## 3. 구현 우선순위
## 4. 위험 요소
## 5. 결론
```

#### 구현 상세 문서
**파일명**: `docs/features/FR-XXX-IMPLEMENTATION.md`

**구조:**
```markdown
# FR-XXX 기능명 - 구현 상세 문서

## 1. 구현 개요
## 2. 데이터 모델
## 3. Validation
## 4. Service 구현
## 5. Handler 구현
## 6. 라우팅
## 7. API 명세
## 8. Swagger 문서
## 9. Service Actions
## 10. 빌드 및 배포
```

#### 테스트 결과서
**파일명**: `docs/features/FR-XXX-TEST-RESULTS.md`

**구조:**
```markdown
# FR-XXX 기능명 - 테스트 결과서

## 1. 테스트 개요
## 2. 빌드 테스트
## 3. Swagger 문서 생성 테스트
## 4. Service Actions 생성 테스트
## 5. API 기능 테스트 시나리오
### TC-001: 테스트 케이스명
#### 요청
#### 예상 결과
#### 실제 결과
## 6. 테스트 결과 요약
## 7. 발견된 이슈
## 8. 권장사항
```

### 6.2 코드 주석

#### Go 주석 가이드
```go
// 패키지 주석 (package.go 파일 상단)
// Package handler provides HTTP handlers for the API.
package handler

// 구조체 주석
// UserHandler handles user-related HTTP requests.
type UserHandler struct {
    userService service.UserService
}

// 함수 주석 (Swagger 주석 포함)
// SignupUser godoc
// @Summary User signup
// @Description Public user signup (no authentication required)
// ... (Swagger 태그들)
func (h *UserHandler) SignupUser(c echo.Context) error {
    // 구현
}

// 상수 주석
const (
    // ErrCodeValidation indicates validation error
    ErrCodeValidation = "VALIDATION_FAILED"
)
```

---

## 7. 코드 리뷰 및 병합

### 7.1 커밋 메시지 규칙

#### Conventional Commits
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type:**
- `feat`: 새 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포맷팅
- `refactor`: 리팩토링
- `test`: 테스트 추가
- `chore`: 빌드, 설정 변경

**예시:**
```bash
# 기능 구현
git commit -m "feat(auth): implement user signup API

Add public user signup endpoint with validation:
- Create users in pending state (enabled=false)
- Validate email format and password length
- Store organization info in Keycloak attributes

API: POST /api/auth/signup"

# 문서 추가
git commit -m "docs(features): add FR-004 documentation

Add three comprehensive documents:
- Requirements analysis
- Implementation details
- Test results"

# Swagger 업데이트
git commit -m "docs(swagger): update API documentation

Update swagger for new APIs:
- SignupUser (POST /api/auth/signup)
- ResetUserPassword (PUT /api/users/id/:userId/password)

Total actions: 133 → 165 (+32)"
```

### 7.2 커밋 전 체크리스트

#### 코드 품질
- [ ] 빌드 성공 (`go build`)
- [ ] 린터 통과 (`golangci-lint run`)
- [ ] 테스트 통과 (`go test ./...`)
- [ ] Swagger 생성 성공
- [ ] Service Actions 생성 성공

#### 코드 리뷰
- [ ] 불필요한 주석/로그 제거
- [ ] 디버그 코드 제거
- [ ] TODO 주석 정리
- [ ] 하드코딩된 값 확인
- [ ] 에러 처리 확인

#### 문서
- [ ] Swagger 주석 작성
- [ ] README 업데이트 (필요시)
- [ ] 변경사항 문서화

### 7.3 Pull Request 생성

#### PR 템플릿
```markdown
## 📝 변경 내용
<!-- 무엇을 변경했는지 간단히 설명 -->

## 🎯 관련 이슈
<!-- 관련 이슈 번호 (있는 경우) -->
Closes #123

## 🔧 변경 유형
- [ ] 새 기능 (feat)
- [ ] 버그 수정 (fix)
- [ ] 문서 변경 (docs)
- [ ] 리팩토링 (refactor)
- [ ] 테스트 추가 (test)

## 📋 변경 세부사항
### API
- `POST /api/auth/signup` - 사용자 가입 신청
- `PUT /api/users/id/:userId/password` - 비밀번호 재설정

### 주요 변경사항
1. SignupRequest, ResetPasswordRequest 모델 추가
2. Validation 유틸리티 구현
3. CreatePendingUser, ResetPassword 서비스 추가
4. Public 라우트 추가

## ✅ 테스트
- [x] 빌드 테스트 통과
- [x] Swagger 문서 생성
- [x] Service Actions 업데이트
- [ ] 수동 테스트 (배포 후)

## 📸 스크린샷 (선택)
<!-- API 테스트 결과, Swagger UI 등 -->

## 📚 문서
- [요구사항 분석서](./docs/features/FR-004-ANALYSIS.md)
- [구현 상세 문서](./docs/features/FR-004-IMPLEMENTATION.md)
- [테스트 결과서](./docs/features/FR-004-TEST-RESULTS.md)

## ⚠️ 주의사항
<!-- 리뷰어가 특히 확인해야 할 사항 -->
```

#### GitHub CLI로 PR 생성
```bash
# PR 생성
gh pr create \
  --base development \
  --title "feat(auth): implement user signup and password reset" \
  --body "$(cat PR_TEMPLATE.md)"

# 또는 interactive 모드
gh pr create --base development
```

#### Web UI로 PR 생성
1. GitHub 저장소 방문
2. "Pull requests" 탭
3. "New pull request" 버튼
4. Base: `development`, Compare: `feature/브랜치명`
5. 제목 및 설명 작성
6. "Create pull request"

### 7.4 코드 리뷰 가이드

#### 리뷰어 체크리스트
**기능:**
- [ ] 요구사항 충족 여부
- [ ] 에러 처리 적절성
- [ ] 에지 케이스 고려

**코드 품질:**
- [ ] 네이밍 규칙 준수
- [ ] 중복 코드 없음
- [ ] 복잡도 적절
- [ ] 주석 충분

**보안:**
- [ ] 입력값 검증
- [ ] SQL Injection 방어
- [ ] 권한 확인
- [ ] 민감 정보 노출 방지

**성능:**
- [ ] N+1 쿼리 없음
- [ ] 불필요한 반복 없음
- [ ] 적절한 인덱스

**문서:**
- [ ] Swagger 주석 완성
- [ ] README 업데이트
- [ ] 변경사항 문서화

---

## 8. 체크리스트

### 8.1 개발 시작 전
- [ ] 요구사항 명확히 이해
- [ ] 기존 코드 분석 완료
- [ ] 구현 계획 수립
- [ ] Worktree 생성 및 브랜치 생성

### 8.2 구현 중
- [ ] 데이터 모델 정의
- [ ] Validation 추가
- [ ] Service 구현
- [ ] Handler 구현
- [ ] 라우트 등록
- [ ] Swagger 주석 작성
- [ ] 에러 처리 구현
- [ ] 로깅 추가

### 8.3 구현 완료 후
- [ ] 빌드 테스트 성공
- [ ] Swagger 문서 생성
- [ ] Service Actions 생성
- [ ] 수동 테스트 수행
- [ ] 코드 리뷰 자가 점검

### 8.4 커밋 전
- [ ] 불필요한 코드 제거
- [ ] 주석 정리
- [ ] 린터 통과
- [ ] 커밋 메시지 작성

### 8.5 문서화
- [ ] 요구사항 분석서 작성
- [ ] 구현 상세 문서 작성
- [ ] 테스트 결과서 작성
- [ ] README 업데이트 (필요시)

### 8.6 PR 생성 전
- [ ] 모든 변경사항 커밋
- [ ] 원격 저장소에 푸시
- [ ] PR 템플릿 작성
- [ ] 리뷰어 지정

---

## 부록 A: 자주 사용하는 명령어

### Git
```bash
# Worktree 관련
git worktree add -b feature/xxx ../프로젝트명-xxx development
git worktree list
git worktree remove ../프로젝트명-xxx

# 브랜치 관련
git checkout -b feature/xxx
git branch --show-current
git status

# 커밋 관련
git add .
git commit -m "message"
git push origin feature/xxx

# 정보 확인
git log --oneline
git diff
git diff --staged
```

### Go
```bash
# 빌드
go build -o 바이너리명 .

# 테스트
go test ./...
go test -v -cover ./...

# 의존성 관리
go mod download
go mod tidy
go mod vendor

# 린터
golangci-lint run
```

### Swagger
```bash
# 설치
go install github.com/swaggo/swag/cmd/swag@latest

# 문서 생성
swag init --parseDependency --parseInternal

# 검증
swag fmt
```

### Docker
```bash
# 실행
docker-compose up -d

# 중지
docker-compose down

# 로그 확인
docker-compose logs -f 서비스명

# 상태 확인
docker-compose ps
```

---

## 부록 B: 트러블슈팅

### 빌드 오류

#### "undefined: xxx"
**원인**: Import 누락
**해결**:
```go
import "github.com/m-cmp/mc-iam-manager/utils"
```

#### "no required module provides package"
**원인**: 의존성 누락
**해결**:
```bash
go mod tidy
go mod download
```

### Swagger 오류

#### "cannot find type definition"
**원인**: 모델이 swagger에 포함되지 않음
**해결**:
```bash
swag init --parseDependency --parseInternal
```

#### "operationId 중복"
**원인**: 동일한 @Id 사용
**해결**: 각 핸들러 함수에 고유한 @Id 지정

### Service Actions 오류

#### "파일을 찾을 수 없습니다"
**원인**: frameworks.yaml의 swagger 경로 오류
**해결**:
```yaml
swagger: ../../src/docs/swagger.yaml  # 경로 확인
```

#### "API가 포함되지 않음"
**원인**: Swagger 문서 미생성
**해결**:
```bash
cd src
swag init --parseDependency --parseInternal
cd ../tool/swagger-to-actions
go run . -c frameworks.yaml
```

---

## 부록 C: 참고 자료

### 공식 문서
- [Echo Framework](https://echo.labstack.com/)
- [GORM](https://gorm.io/)
- [Keycloak](https://www.keycloak.org/documentation)
- [Swagger](https://swagger.io/docs/)
- [Go Validator](https://github.com/go-playground/validator)

### 내부 문서
- [API 명세서](../src/docs/swagger.yaml)
- [데이터베이스 스키마](./DATABASE_SCHEMA.md)
- [배포 가이드](./DEPLOYMENT.md)

### 코딩 스타일
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

## 변경 이력
| 버전 | 날짜 | 작성자 | 변경 내용 |
|------|------|--------|----------|
| 1.0 | 2026-02-09 | Development Team | 초안 작성 |

---

## 피드백
이 문서에 대한 피드백이나 개선 제안은 팀 리드에게 전달하거나 GitHub Issue로 등록해주세요.

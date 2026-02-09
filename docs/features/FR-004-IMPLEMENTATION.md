# FR-004 사용자 가입 및 승인 기능 - 구현 상세 문서

## 문서 정보
- **작성일**: 2026-02-09
- **버전**: 1.0
- **브랜치**: feature/user-signup-approval
- **커밋**: 387dcf74

---

## 1. 구현 개요

### 1.1 구현 목표
- Public 사용자 가입 API 구현 (인증 불필요)
- 승인 대기 상태로 사용자 생성 (enabled=false)
- 관리자용 비밀번호 재설정 API 구현
- 입력값 검증 및 한글 에러 메시지

### 1.2 구현 범위
| 구분 | 내용 |
|------|------|
| 신규 API | 2개 (SignupUser, ResetUserPassword) |
| 신규 파일 | 2개 (error.go, validator.go) |
| 수정 파일 | 6개 (models, services, handlers, routes) |
| 코드 라인 | +353, -6 |

---

## 2. 데이터 모델

### 2.1 SignupRequest
**파일**: `src/model/request.go`

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

**필드 설명**:
- `email`: 이메일 주소 (필수, 이메일 형식)
- `password`: 비밀번호 (필수, 최소 8자)
- `firstName`: 이름 (필수)
- `lastName`: 성 (필수)
- `organization`: 소속 조직 (선택)

**검증 규칙**:
- `required`: 필수 입력
- `email`: 유효한 이메일 형식
- `min=8`: 최소 8자 이상

---

### 2.2 ResetPasswordRequest
**파일**: `src/model/request.go`

```go
// ResetPasswordRequest represents the password reset request
type ResetPasswordRequest struct {
    NewPassword string `json:"newPassword" validate:"required,min=8"`
}
```

**필드 설명**:
- `newPassword`: 새 비밀번호 (필수, 최소 8자)

---

### 2.3 ErrorResponse
**파일**: `src/model/error.go` (신규)

```go
type ErrorResponse struct {
    Success bool              `json:"success"`
    Error   string            `json:"error"`
    Fields  map[string]string `json:"fields,omitempty"`
    Code    string            `json:"code,omitempty"`
}

// Error codes
const (
    ErrCodeDuplicateEmail = "DUPLICATE_EMAIL"
    ErrCodeValidation     = "VALIDATION_FAILED"
    ErrCodeServerError    = "SERVER_ERROR"
)
```

**필드 설명**:
- `success`: 요청 성공 여부
- `error`: 에러 메시지
- `fields`: 필드별 에러 메시지 (검증 실패 시)
- `code`: 에러 코드

---

### 2.4 User 모델 확장
**파일**: `src/model/user.go`

```go
type User struct {
    // Keycloak 정보
    Username     string `json:"username" gorm:"column:username;size:255;not null;unique"`
    Email        string `json:"email" gorm:"-"`
    FirstName    string `json:"firstName,omitempty" gorm:"-"`
    LastName     string `json:"lastName,omitempty" gorm:"-"`
    Enabled      bool   `json:"enabled" gorm:"-"`
    Organization string `json:"organization,omitempty" gorm:"-"` // 추가됨
    // ...
}
```

**변경사항**:
- `Organization` 필드 추가
- `gorm:"-"` 태그로 DB에는 저장하지 않음 (Keycloak attributes에만 저장)

---

## 3. Validation 유틸리티

### 3.1 Validator
**파일**: `src/utils/validator.go` (신규)

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

// ValidateStruct validates a struct using validator tags
func ValidateStruct(s interface{}) error {
    return validate.Struct(s)
}

// FormatValidationErrorMap returns field-specific error messages
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
    field := e.Field()

    switch e.Tag() {
    case "required":
        return fmt.Sprintf("%s은(는) 필수 입력입니다", field)
    case "email":
        return "유효한 이메일 형식이 아닙니다"
    case "min":
        if e.Param() == "8" {
            return "비밀번호는 8자 이상이어야 합니다"
        }
        return fmt.Sprintf("%s은(는) 최소 %s자 이상이어야 합니다", field, e.Param())
    default:
        return fmt.Sprintf("%s 검증 실패", field)
    }
}
```

**주요 기능**:
- 구조체 태그 기반 검증
- 한글 에러 메시지 반환
- 필드별 에러 매핑

---

## 4. Keycloak Service

### 4.1 CreatePendingUser
**파일**: `src/service/keycloak_service.go`

```go
// CreatePendingUser creates a user in pending state (enabled=false) with password
func (s *keycloakService) CreatePendingUser(ctx context.Context, req *model.SignupRequest) (string, error) {
    if config.KC == nil || config.KC.Client == nil {
        return "", fmt.Errorf("keycloak configuration not initialized")
    }

    token, err := config.KC.LoginAdmin(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get admin token: %w", err)
    }

    // Generate username from email (before @)
    username := strings.Split(req.Email, "@")[0]

    keycloakUser := gocloak.User{
        Username:      &username,
        Email:         &req.Email,
        FirstName:     &req.FirstName,
        LastName:      &req.LastName,
        Enabled:       gocloak.BoolP(false),       // 승인 대기 상태
        EmailVerified: gocloak.BoolP(false),       // 이메일 미확인
        Attributes: &map[string][]string{
            "organization": {req.Organization},    // 조직 정보 저장
        },
    }

    kcId, err := config.KC.Client.CreateUser(ctx, token.AccessToken, config.KC.Realm, keycloakUser)
    if err != nil {
        if strings.Contains(err.Error(), "409") {
            return "", fmt.Errorf("이미 사용 중인 이메일입니다")
        }
        return "", fmt.Errorf("failed to create user: %w", err)
    }

    // 비밀번호 설정
    err = config.KC.Client.SetPassword(ctx, token.AccessToken, kcId, config.KC.Realm, req.Password, false)
    if err != nil {
        // 사용자 생성은 성공했으나 비밀번호 설정 실패 - 사용자 삭제
        config.KC.Client.DeleteUser(ctx, token.AccessToken, config.KC.Realm, kcId)
        return "", fmt.Errorf("failed to set password: %w", err)
    }

    return kcId, nil
}
```

**구현 로직**:
1. Admin 토큰 획득
2. 이메일에서 username 자동 생성 (@ 앞부분)
3. Keycloak User 생성
   - enabled=false (승인 대기)
   - EmailVerified=false
   - Organization을 attributes에 저장
4. 비밀번호 설정 (temporary=false)
5. 실패 시 롤백 (사용자 삭제)

**에러 처리**:
- 409 Conflict → "이미 사용 중인 이메일입니다"
- 비밀번호 설정 실패 → 생성된 사용자 삭제 후 에러 반환

---

### 4.2 ResetPassword
**파일**: `src/service/keycloak_service.go`

```go
// ResetPassword resets a user's password
func (s *keycloakService) ResetPassword(ctx context.Context, kcUserID, newPassword string) error {
    if config.KC == nil || config.KC.Client == nil {
        return fmt.Errorf("keycloak configuration not initialized")
    }

    // Admin 토큰 획득
    adminToken, err := config.KC.LoginAdmin(ctx)
    if err != nil {
        return fmt.Errorf("failed to get admin token: %w", err)
    }

    // 사용자 존재 확인
    existingUser, err := config.KC.Client.GetUserByID(ctx, adminToken.AccessToken, config.KC.Realm, kcUserID)
    if err != nil {
        return fmt.Errorf("failed to get user: %w", err)
    }
    if existingUser == nil {
        return fmt.Errorf("user not found: %s", kcUserID)
    }

    // 비밀번호 재설정 (temporary=false: 영구 비밀번호)
    err = config.KC.Client.SetPassword(ctx, adminToken.AccessToken, kcUserID, config.KC.Realm, newPassword, false)
    if err != nil {
        return fmt.Errorf("failed to reset password: %w", err)
    }

    log.Printf("[INFO] Password reset successfully for user: %s", kcUserID)
    return nil
}
```

**구현 로직**:
1. Admin 토큰 획득
2. 사용자 존재 확인
3. 비밀번호 재설정 (temporary=false)
4. 성공 로그 기록

---

## 5. User Service

### 5.1 SignupUser
**파일**: `src/service/user_service.go`

```go
// SignupUser creates a user in pending state (enabled=false)
func (s *UserService) SignupUser(ctx context.Context, req *model.SignupRequest) (string, error) {
    ks := NewKeycloakService()

    // Keycloak에 pending 상태로 사용자 생성
    kcId, err := ks.CreatePendingUser(ctx, req)
    if err != nil {
        return "", err
    }

    // 로컬 DB 동기화는 승인 시에 수행
    return kcId, nil
}
```

**설계 결정**:
- 가입 신청 시점에는 Keycloak에만 생성
- 로컬 DB 동기화는 관리자 승인 시에 수행 (`ApproveUser` 메서드)

---

### 5.2 ResetUserPassword
**파일**: `src/service/user_service.go`

```go
// ResetUserPassword resets a user's password
func (s *UserService) ResetUserPassword(ctx context.Context, kcUserID, newPassword string) error {
    ks := NewKeycloakService()
    return ks.ResetPassword(ctx, kcUserID, newPassword)
}
```

---

## 6. Handlers

### 6.1 SignupUser Handler
**파일**: `src/handler/user_handler.go`

```go
// SignupUser godoc
// @Summary User signup
// @Description Public user signup (no authentication required)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.SignupRequest true "Signup Info"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/auth/signup [post]
// @Id SignupUser
func (h *UserHandler) SignupUser(c echo.Context) error {
    var req model.SignupRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request format",
        })
    }

    // Validation
    if err := utils.ValidateStruct(req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "error":  "입력값 검증에 실패했습니다",
            "fields": utils.FormatValidationErrorMap(err),
        })
    }

    // Create user in pending state
    kcId, err := h.userService.SignupUser(c.Request().Context(), &req)
    if err != nil {
        if strings.Contains(err.Error(), "이미 사용 중인") {
            return c.JSON(http.StatusConflict, map[string]string{
                "error": err.Error(),
            })
        }
        log.Printf("[ERROR] SignupUser failed: %v", err)
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "가입 신청에 실패했습니다. 잠시 후 다시 시도해주세요",
        })
    }

    return c.JSON(http.StatusCreated, map[string]interface{}{
        "success":     true,
        "message":     "가입 신청이 완료되었습니다. 관리자 승인 후 로그인 가능합니다.",
        "kcId":        kcId,
        "redirectUrl": "/login",
    })
}
```

**HTTP 상태 코드**:
- 201 Created: 가입 신청 성공
- 400 Bad Request: 입력값 검증 실패
- 409 Conflict: 중복 이메일
- 500 Internal Server Error: 서버 오류

**응답 형식**:
```json
// 성공
{
  "success": true,
  "message": "가입 신청이 완료되었습니다. 관리자 승인 후 로그인 가능합니다.",
  "kcId": "abc123-def456",
  "redirectUrl": "/login"
}

// 검증 실패
{
  "error": "입력값 검증에 실패했습니다",
  "fields": {
    "email": "유효한 이메일 형식이 아닙니다",
    "password": "비밀번호는 8자 이상이어야 합니다"
  }
}

// 중복 이메일
{
  "error": "이미 사용 중인 이메일입니다"
}
```

---

### 6.2 ResetUserPassword Handler
**파일**: `src/handler/user_handler.go`

```go
// ResetUserPassword godoc
// @Summary Reset user password
// @Description Reset a user's password (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID (DB)"
// @Param request body model.ResetPasswordRequest true "New Password"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/password [put]
// @Id ResetUserPassword
func (h *UserHandler) ResetUserPassword(c echo.Context) error {
    // 관리자 권한 확인
    requiredRoles := []string{"platformAdmin"}
    if !checkRoleFromContext(c, requiredRoles) {
        return c.JSON(http.StatusForbidden, map[string]string{
            "error": "Forbidden: Platform Administrator access required",
        })
    }

    // User ID 파싱
    userIDStr := c.Param("userId")
    userID, err := strconv.ParseUint(userIDStr, 10, 32)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid user ID",
        })
    }
    userIDInt := uint(userID)

    // 요청 바인딩
    var req model.ResetPasswordRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request format",
        })
    }

    // 유효성 검증
    if err := utils.ValidateStruct(req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "error":  "입력값 검증에 실패했습니다",
            "fields": utils.FormatValidationErrorMap(err),
        })
    }

    // 사용자 조회 (DB)
    user, err := h.userService.GetUserByID(c.Request().Context(), userIDInt)
    if err != nil {
        return c.JSON(http.StatusNotFound, map[string]string{
            "error": "User not found",
        })
    }

    // 비밀번호 재설정
    err = h.userService.ResetUserPassword(c.Request().Context(), user.KcId, req.NewPassword)
    if err != nil {
        log.Printf("[ERROR] ResetUserPassword failed: %v", err)
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "비밀번호 변경에 실패했습니다",
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "success": true,
        "message": "비밀번호가 성공적으로 변경되었습니다",
    })
}
```

**HTTP 상태 코드**:
- 200 OK: 비밀번호 재설정 성공
- 400 Bad Request: 입력값 검증 실패
- 403 Forbidden: 권한 없음
- 404 Not Found: 사용자 없음
- 500 Internal Server Error: 서버 오류

---

## 7. 라우팅

### 7.1 Public Route 추가
**파일**: `src/main.go`

```go
// 인증이 필요없는 경로 목록
skipAuthPaths := []string{
    "/readyz",
    "/api/initial-admin",
    "/api/auth/login",
    "/api/auth/logout",
    "/api/auth/refresh",
    "/api/auth/certs",
    "/api/auth/temp-credential-csps",
    "/api/auth/validate",
    "/api/auth/signup",  // 추가됨
}
```

---

### 7.2 라우트 등록
**파일**: `src/main.go`

```go
// 인증 라우트
auth := api.Group("/auth")
{
    auth.POST("/login", authHandler.Login)
    auth.POST("/logout", authHandler.Logout)
    auth.POST("/refresh", authHandler.RefreshToken)
    auth.GET("/certs", authHandler.AuthCerts)
    auth.GET("/temp-credential-csps", authHandler.GetTempCredentialProviders)
    auth.POST("/validate", authHandler.Validate)
    auth.POST("/signup", userHandler.SignupUser) // 추가됨
}

// 사용자 라우트
users := api.Group("/users")
{
    // ... 기존 라우트들 ...
    users.PUT("/id/:userId/password", userHandler.ResetUserPassword, middleware.PlatformRoleMiddleware(middleware.Manage)) // 추가됨
}
```

---

## 8. API 명세

### 8.1 POST /api/auth/signup

**Request:**
```json
{
  "email": "user@example.com",
  "password": "SecurePass123",
  "firstName": "길동",
  "lastName": "홍",
  "organization": "테스트조직"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "가입 신청이 완료되었습니다. 관리자 승인 후 로그인 가능합니다.",
  "kcId": "abc123-def456",
  "redirectUrl": "/login"
}
```

**Errors:**
- 400: 입력값 검증 실패
- 409: 이메일 중복
- 500: 서버 오류

---

### 8.2 PUT /api/users/id/{userId}/password

**Headers:**
```
Authorization: Bearer {access_token}
```

**Request:**
```json
{
  "newPassword": "NewSecure123"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "비밀번호가 성공적으로 변경되었습니다"
}
```

**Errors:**
- 400: 입력값 검증 실패
- 403: 권한 없음
- 404: 사용자 없음
- 500: 서버 오류

---

## 9. Swagger 문서

### 9.1 자동 생성
**실행:**
```bash
cd src
swag init --parseDependency --parseInternal
```

**생성 파일:**
- `src/docs/swagger.yaml`
- `src/docs/swagger.json`
- `src/docs/docs.go`

### 9.2 operationId
- `SignupUser` - 함수명과 동일
- `ResetUserPassword` - 함수명과 동일

---

## 10. Service Actions 업데이트

### 10.1 생성 도구
**위치**: `tool/swagger-to-actions/`

**실행:**
```bash
cd tool/swagger-to-actions
go run . -c frameworks.yaml
```

### 10.2 frameworks.yaml 수정
```yaml
- name: mc-iam-manager
  version: "0.3.0"
  repository: https://github.com/m-cmp/mc-iam-manager
  swagger: ../../src/docs/swagger.yaml  # 경로 수정됨
```

### 10.3 결과
**service-actions.yaml에 추가됨:**
```yaml
SignupUser:
    method: post
    resourcePath: /api/auth/signup
    description: Public user signup (no authentication required)

ResetUserPassword:
    method: put
    resourcePath: /api/users/id/{userId}/password
    description: Reset a user's password (admin only)
```

**통계:**
- mc-iam-manager: 133 → 165 actions (+32)
- 전체: 275 → 307 actions (+32)

---

## 11. 빌드 및 배포

### 11.1 빌드
```bash
cd src
go build -o mc-iam-manager .
```

**결과:**
- 바이너리 크기: 48MB
- 빌드 시간: ~10초

### 11.2 실행
```bash
./mc-iam-manager
```

**환경 변수:**
- `.env` 파일 필요
- Keycloak 연결 정보
- PostgreSQL 연결 정보

---

## 12. 코드 품질

### 12.1 코드 메트릭
| 항목 | 값 |
|------|-----|
| 신규 파일 | 2개 |
| 수정 파일 | 6개 |
| 추가 라인 | +353 |
| 삭제 라인 | -6 |
| 순증가 | +347 |

### 12.2 테스트 커버리지
- 단위 테스트: 미구현 (수동 테스트로 대체)
- 통합 테스트: 수동 수행

---

## 13. 보안 고려사항

### 13.1 구현된 보안 기능
✅ 비밀번호 최소 길이 검증 (8자)
✅ 이메일 형식 검증
✅ enabled=false로 기본 생성 (관리자 승인 필수)
✅ platformAdmin 권한 확인 (비밀번호 재설정)
✅ 비밀번호 해시화 (Keycloak에서 처리)

### 13.2 권장 추가 보안 기능
⚠️ Rate Limiting (선택 사항)
⚠️ 이메일 인증
⚠️ 비밀번호 복잡도 강화
⚠️ CAPTCHA

---

## 14. 향후 개선사항

### 14.1 기능 개선
- [ ] 이메일 인증 기능
- [ ] 비밀번호 찾기 기능
- [ ] 소셜 로그인 연동
- [ ] 2FA (Two-Factor Authentication)

### 14.2 성능 개선
- [ ] Rate Limiting 구현
- [ ] 캐싱 전략
- [ ] DB 인덱스 최적화

### 14.3 코드 품질
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 자동화
- [ ] 코드 커버리지 70% 이상

---

## 부록 A: 파일 목록

### 신규 파일
1. `src/model/error.go` - 에러 응답 모델
2. `src/utils/validator.go` - 검증 유틸리티

### 수정 파일
1. `src/model/request.go` - SignupRequest, ResetPasswordRequest 추가
2. `src/model/user.go` - Organization 필드 추가
3. `src/service/keycloak_service.go` - CreatePendingUser, ResetPassword 추가
4. `src/service/user_service.go` - SignupUser, ResetUserPassword 추가
5. `src/handler/user_handler.go` - SignupUser, ResetUserPassword 핸들러 추가
6. `src/main.go` - 라우트 및 skipAuthPaths 추가

---

## 부록 B: 참고 문서
- [요구사항 분석서](./FR-004-ANALYSIS.md)
- [테스트 결과서](./FR-004-TEST-RESULTS.md)
- [Swagger API 문서](../../src/docs/swagger.yaml)

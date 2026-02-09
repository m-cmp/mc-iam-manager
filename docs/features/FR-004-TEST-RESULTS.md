# FR-004 사용자 가입 및 승인 기능 - 테스트 결과서

## 문서 정보
- **작성일**: 2026-02-09
- **버전**: 1.0
- **테스트 환경**: Development (Worktree)
- **브랜치**: feature/user-signup-approval

---

## 1. 테스트 개요

### 1.1 테스트 목적
- 사용자 가입 API의 정상 동작 검증
- 입력값 검증 로직 확인
- 에러 처리 및 메시지 확인
- 관리자 승인 프로세스 검증
- 비밀번호 재설정 기능 검증

### 1.2 테스트 범위
| 구분 | 내용 |
|------|------|
| API 테스트 | 2개 엔드포인트 |
| 빌드 테스트 | 컴파일 및 바이너리 생성 |
| Swagger 테스트 | 문서 자동 생성 |
| Service Actions 테스트 | YAML 생성 및 검증 |

### 1.3 테스트 방법
- ✅ 빌드 테스트 (자동)
- ✅ Swagger 생성 테스트 (자동)
- ✅ Service Actions 생성 테스트 (자동)
- ⚠️ API 기능 테스트 (수동 시나리오 제공)

---

## 2. 빌드 테스트

### 2.1 테스트 환경
```
OS: Linux 6.6.87.2-microsoft-standard-WSL2
Go Version: 1.25.0
Platform: linux/amd64
```

### 2.2 테스트 실행
```bash
cd /home/nobang/ai_workspace/m-cmp-iam-manager/mc-iam-manager-signup/src
go build -o ../mc-iam-manager .
```

### 2.3 테스트 결과
✅ **성공**

| 항목 | 결과 |
|------|------|
| 컴파일 오류 | 없음 |
| 링킹 오류 | 없음 |
| 바이너리 생성 | 성공 |
| 바이너리 크기 | 48MB |
| 빌드 시간 | ~10초 |

### 2.4 생성된 바이너리
```
-rwxr-xr-x 1 nobang nobang 48M Feb  9 11:13 mc-iam-manager
```

### 2.5 실행 가능 여부
```bash
./mc-iam-manager --version
```

**결과:**
```
2026/02/09 11:13:51 common.go:143: ❌ .env 파일을 상위 디렉토리에서 로드하는데 실패했습니다
2026/02/09 11:13:51 common.go:148: ✅ .env 파일을 현재 디렉토리에서 로드했습니다
2026/02/09 11:13:51 main.go:63: === Application Starting ===
2026/02/09 11:13:51 database.go:32: Database config - Host: mc-iam-manager-db
```

✅ **바이너리 실행 가능 확인**

---

## 3. Swagger 문서 생성 테스트

### 3.1 테스트 실행
```bash
cd src
~/go/bin/swag init --parseDependency --parseInternal
```

### 3.2 테스트 결과
✅ **성공**

**출력:**
```
2026/02/09 11:24:22 Generate swagger docs....
2026/02/09 11:24:22 Generate general API Info, search dir:./
2026/02/09 11:24:31 Generating model.SignupRequest
2026/02/09 11:24:31 Generating model.ResetPasswordRequest
2026/02/09 11:24:31 create docs.go at docs/docs.go
2026/02/09 11:24:32 create swagger.json at docs/swagger.json
2026/02/09 11:24:32 create swagger.yaml at docs/swagger.yaml
```

### 3.3 생성된 파일
| 파일 | 크기 | 설명 |
|------|------|------|
| docs/swagger.yaml | 175KB | OpenAPI 3.0 YAML |
| docs/swagger.json | 179KB | OpenAPI 3.0 JSON |
| docs/docs.go | - | Go 임베드 코드 |

### 3.4 API 포함 여부 검증

#### SignupUser 확인
```bash
grep -A 5 "/api/auth/signup:" docs/swagger.yaml
```

**결과:**
```yaml
/api/auth/signup:
  post:
    consumes:
    - application/json
    description: Public user signup (no authentication required)
    operationId: SignupUser
```

✅ **SignupUser API 포함 확인**

#### ResetUserPassword 확인
```bash
grep "operationId: ResetUserPassword" docs/swagger.yaml
```

**결과:**
```
operationId: ResetUserPassword
```

✅ **ResetUserPassword API 포함 확인**

### 3.5 Request Model 검증

#### SignupRequest
```bash
jq '.definitions["model.SignupRequest"]' docs/swagger.json
```

**결과:**
```json
{
  "type": "object",
  "required": [
    "email",
    "firstName",
    "lastName",
    "password"
  ],
  "properties": {
    "email": {
      "type": "string"
    },
    "firstName": {
      "type": "string"
    },
    "lastName": {
      "type": "string"
    },
    "organization": {
      "description": "선택 필드",
      "type": "string"
    },
    "password": {
      "type": "string",
      "minLength": 8
    }
  }
}
```

✅ **검증 규칙 반영 확인**
- required 필드: email, firstName, lastName, password
- password minLength: 8
- organization: optional

---

## 4. Service Actions 생성 테스트

### 4.1 frameworks.yaml 수정
**변경 사항:**
```yaml
# 이전
swagger: ../../docs/swagger.yaml

# 이후
swagger: ../../src/docs/swagger.yaml
```

### 4.2 테스트 실행
```bash
cd tool/swagger-to-actions
go run . -c frameworks.yaml
```

### 4.3 테스트 결과
✅ **성공**

**출력:**
```
[INFO] Running in config mode with: frameworks.yaml
[SUCCESS] Successfully generated: service-actions.yaml
[INFO] Total frameworks: 4
[INFO]   - mc-iam-manager: 165 actions
[INFO]   - mc-application-manager: 66 actions
[INFO]   - mc-cost-optimizer: 14 actions
[INFO]   - mc-workflow-manager: 62 actions
[INFO] Total actions: 307
```

### 4.4 액션 수 변화
| 구분 | 이전 | 이후 | 변화 |
|------|------|------|------|
| mc-iam-manager | 133 | 165 | +32 |
| 전체 | 275 | 307 | +32 |

### 4.5 새 API 포함 여부 검증

```bash
grep -A 3 "SignupUser:\|ResetUserPassword:" service-actions.yaml
```

**결과:**
```yaml
ResetUserPassword:
    method: put
    resourcePath: /api/users/id/{userId}/password
    description: Reset a user's password (admin only)
SignupUser:
    method: post
    resourcePath: /api/auth/signup
    description: Public user signup (no authentication required)
```

✅ **새 API 모두 포함 확인**

---

## 5. API 기능 테스트 시나리오

### 5.1 사전 준비
```bash
# 1. Keycloak 및 PostgreSQL 실행
docker-compose up -d

# 2. 애플리케이션 실행
./mc-iam-manager

# 3. 관리자 토큰 발급
ADMIN_TOKEN=$(curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"id": "admin", "password": "admin123"}' \
  | jq -r '.access_token')
```

---

### 5.2 TC-001: 정상 가입 신청

#### 테스트 케이스
```
ID: TC-001
제목: 정상 가입 신청
우선순위: 높음
전제조건: 없음
```

#### 요청
```bash
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234!",
    "firstName": "길동",
    "lastName": "홍",
    "organization": "테스트조직"
  }'
```

#### 예상 결과
**Status Code:** 201 Created

**Response Body:**
```json
{
  "success": true,
  "message": "가입 신청이 완료되었습니다. 관리자 승인 후 로그인 가능합니다.",
  "kcId": "abc123-def456",
  "redirectUrl": "/login"
}
```

**Keycloak 상태:**
- enabled: false
- EmailVerified: false
- attributes.organization: "테스트조직"

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.3 TC-002: 이메일 형식 오류

#### 테스트 케이스
```
ID: TC-002
제목: 잘못된 이메일 형식
우선순위: 높음
전제조건: 없음
```

#### 요청
```bash
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "invalid-email",
    "password": "Test1234!",
    "firstName": "길동",
    "lastName": "홍"
  }'
```

#### 예상 결과
**Status Code:** 400 Bad Request

**Response Body:**
```json
{
  "error": "입력값 검증에 실패했습니다",
  "fields": {
    "email": "유효한 이메일 형식이 아닙니다"
  }
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.4 TC-003: 비밀번호 길이 오류

#### 테스트 케이스
```
ID: TC-003
제목: 비밀번호 8자 미만
우선순위: 높음
전제조건: 없음
```

#### 요청
```bash
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "123",
    "firstName": "길동",
    "lastName": "홍"
  }'
```

#### 예상 결과
**Status Code:** 400 Bad Request

**Response Body:**
```json
{
  "error": "입력값 검증에 실패했습니다",
  "fields": {
    "password": "비밀번호는 8자 이상이어야 합니다"
  }
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.5 TC-004: 필수 필드 누락

#### 테스트 케이스
```
ID: TC-004
제목: firstName 누락
우선순위: 높음
전제조건: 없음
```

#### 요청
```bash
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234!",
    "lastName": "홍"
  }'
```

#### 예상 결과
**Status Code:** 400 Bad Request

**Response Body:**
```json
{
  "error": "입력값 검증에 실패했습니다",
  "fields": {
    "firstname": "FirstName은(는) 필수 입력입니다"
  }
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.6 TC-005: 중복 이메일

#### 테스트 케이스
```
ID: TC-005
제목: 이미 존재하는 이메일로 가입 시도
우선순위: 높음
전제조건: test@example.com이 이미 등록됨
```

#### 요청
```bash
# 첫 번째 가입 (성공)
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234!",
    "firstName": "길동",
    "lastName": "홍"
  }'

# 두 번째 가입 시도 (실패 예상)
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test5678!",
    "firstName": "철수",
    "lastName": "김"
  }'
```

#### 예상 결과
**Status Code:** 409 Conflict

**Response Body:**
```json
{
  "error": "이미 사용 중인 이메일입니다"
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.7 TC-006: 가입 후 로그인 차단

#### 테스트 케이스
```
ID: TC-006
제목: enabled=false 상태에서 로그인 시도
우선순위: 높음
전제조건: test@example.com 가입 완료 (미승인)
```

#### 요청
```bash
# 가입
curl -X POST http://localhost:5000/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234!",
    "firstName": "길동",
    "lastName": "홍"
  }'

# 로그인 시도 (실패 예상)
curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test@example.com",
    "password": "Test1234!"
  }'
```

#### 예상 결과
**Status Code:** 401 Unauthorized

**Response Body:**
```json
{
  "error": "Authentication failed: Account is disabled"
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.8 TC-007: 관리자 승인 프로세스

#### 테스트 케이스
```
ID: TC-007
제목: 관리자 승인 후 로그인 성공
우선순위: 높음
전제조건: test@example.com 가입 완료 (미승인), 관리자 토큰 발급
```

#### 요청
```bash
# 1. 사용자 ID 확인
USER_ID=$(curl -X POST http://localhost:5000/api/users/list \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}' \
  | jq -r '.users[0].id')

# 2. 승인 처리
curl -X POST http://localhost:5000/api/users/id/$USER_ID/status \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "approved"}'

# 3. 로그인 시도 (성공 예상)
curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test@example.com",
    "password": "Test1234!"
  }'
```

#### 예상 결과
**승인 API:**
- Status Code: 204 No Content

**로그인 API:**
- Status Code: 200 OK
- Response Body:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 300
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.9 TC-008: 비밀번호 재설정 (관리자)

#### 테스트 케이스
```
ID: TC-008
제목: 관리자가 사용자 비밀번호 재설정
우선순위: 높음
전제조건: test@example.com 계정 존재, 관리자 토큰 발급
```

#### 요청
```bash
# 1. 사용자 ID 확인
USER_ID=$(curl -X POST http://localhost:5000/api/users/list \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}' \
  | jq -r '.users[0].id')

# 2. 비밀번호 재설정
curl -X PUT http://localhost:5000/api/users/id/$USER_ID/password \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "newPassword": "NewPassword123!"
  }'

# 3. 새 비밀번호로 로그인 시도
curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test@example.com",
    "password": "NewPassword123!"
  }'
```

#### 예상 결과
**비밀번호 재설정 API:**
- Status Code: 200 OK
- Response Body:
```json
{
  "success": true,
  "message": "비밀번호가 성공적으로 변경되었습니다"
}
```

**로그인 API:**
- Status Code: 200 OK
- 액세스 토큰 반환

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.10 TC-009: 비밀번호 재설정 권한 확인

#### 테스트 케이스
```
ID: TC-009
제목: 일반 사용자가 비밀번호 재설정 시도 (실패)
우선순위: 중간
전제조건: 일반 사용자 토큰 발급
```

#### 요청
```bash
# 1. 일반 사용자 로그인
USER_TOKEN=$(curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user@example.com",
    "password": "user123"
  }' \
  | jq -r '.access_token')

# 2. 비밀번호 재설정 시도 (실패 예상)
curl -X PUT http://localhost:5000/api/users/id/1/password \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "newPassword": "NewPassword123!"
  }'
```

#### 예상 결과
**Status Code:** 403 Forbidden

**Response Body:**
```json
{
  "error": "Forbidden: Platform Administrator access required"
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

### 5.11 TC-010: 비밀번호 재설정 검증 실패

#### 테스트 케이스
```
ID: TC-010
제목: 8자 미만 비밀번호로 재설정 시도
우선순위: 높음
전제조건: 관리자 토큰 발급
```

#### 요청
```bash
curl -X PUT http://localhost:5000/api/users/id/1/password \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "newPassword": "short"
  }'
```

#### 예상 결과
**Status Code:** 400 Bad Request

**Response Body:**
```json
{
  "error": "입력값 검증에 실패했습니다",
  "fields": {
    "newpassword": "비밀번호는 8자 이상이어야 합니다"
  }
}
```

#### 실제 결과
⚠️ **수동 테스트 필요**

---

## 6. 테스트 결과 요약

### 6.1 자동 테스트 결과
| 테스트 | 결과 | 비고 |
|--------|------|------|
| 빌드 테스트 | ✅ 성공 | 바이너리 생성 확인 |
| Swagger 생성 | ✅ 성공 | 2개 API 포함 확인 |
| Service Actions 생성 | ✅ 성공 | 165 actions 생성 |
| Request Model 검증 | ✅ 성공 | 검증 규칙 반영 확인 |

### 6.2 수동 테스트 시나리오
| TC ID | 테스트 케이스 | 상태 |
|-------|--------------|------|
| TC-001 | 정상 가입 신청 | ⚠️ 수동 테스트 필요 |
| TC-002 | 이메일 형식 오류 | ⚠️ 수동 테스트 필요 |
| TC-003 | 비밀번호 길이 오류 | ⚠️ 수동 테스트 필요 |
| TC-004 | 필수 필드 누락 | ⚠️ 수동 테스트 필요 |
| TC-005 | 중복 이메일 | ⚠️ 수동 테스트 필요 |
| TC-006 | 가입 후 로그인 차단 | ⚠️ 수동 테스트 필요 |
| TC-007 | 관리자 승인 프로세스 | ⚠️ 수동 테스트 필요 |
| TC-008 | 비밀번호 재설정 | ⚠️ 수동 테스트 필요 |
| TC-009 | 비밀번호 재설정 권한 | ⚠️ 수동 테스트 필요 |
| TC-010 | 비밀번호 재설정 검증 | ⚠️ 수동 테스트 필요 |

### 6.3 전체 테스트 통계
```
자동 테스트: 4/4 성공 (100%)
수동 테스트: 0/10 완료 (0%)
전체: 4/14 완료 (28.6%)
```

---

## 7. 발견된 이슈

### 7.1 이슈 목록
| ID | 제목 | 심각도 | 상태 |
|----|------|--------|------|
| - | - | - | 없음 |

**현재까지 발견된 이슈 없음**

---

## 8. 테스트 환경

### 8.1 시스템 환경
```
OS: Linux 6.6.87.2-microsoft-standard-WSL2
Platform: WSL2
Architecture: x86_64
```

### 8.2 소프트웨어 버전
```
Go: 1.25.0
Keycloak: (환경에 따름)
PostgreSQL: (환경에 따름)
```

### 8.3 브랜치 정보
```
Repository: https://github.com/MZC-CSC/mc-iam-manager
Branch: feature/user-signup-approval
Commit: 387dcf74
```

---

## 9. 권장사항

### 9.1 테스트 자동화
**우선순위: 높음**

다음 항목에 대한 자동화 테스트 작성 권장:
- [ ] 단위 테스트 (유틸리티 함수)
- [ ] 통합 테스트 (API 엔드포인트)
- [ ] E2E 테스트 (전체 플로우)

### 9.2 성능 테스트
**우선순위: 중간**

다음 시나리오에 대한 성능 테스트 권장:
- [ ] 동시 가입 신청 (100명)
- [ ] Rate Limiting 효과 측정
- [ ] DB 부하 테스트

### 9.3 보안 테스트
**우선순위: 높음**

다음 보안 항목 검증 권장:
- [ ] SQL Injection 방어
- [ ] XSS 방어
- [ ] CSRF 방어
- [ ] Rate Limiting 효과

---

## 10. 결론

### 10.1 테스트 완료 항목
✅ 빌드 및 컴파일 검증
✅ Swagger 문서 자동 생성
✅ Service Actions YAML 생성
✅ Request Model 검증 규칙 반영

### 10.2 미완료 항목
⚠️ API 기능 테스트 (10개 시나리오)
⚠️ 통합 테스트 자동화
⚠️ 성능 테스트
⚠️ 보안 테스트

### 10.3 최종 평가
**현재 상태**: 개발 완료, 수동 테스트 대기

**배포 가능 여부**: ⚠️ 조건부 가능
- 자동 테스트 통과
- 수동 기능 테스트 필요

**권장 조치**:
1. 수동 테스트 10개 시나리오 실행
2. 이슈 발견 시 수정 및 재테스트
3. 통합 테스트 자동화 구현 (향후)

---

## 부록 A: 테스트 스크립트

### A.1 전체 테스트 실행 스크립트
```bash
#!/bin/bash
# test-all.sh

set -e

echo "=== FR-004 테스트 시작 ==="

# 1. 빌드 테스트
echo "[1/4] 빌드 테스트..."
cd src
go build -o ../mc-iam-manager .
echo "✅ 빌드 성공"

# 2. Swagger 생성
echo "[2/4] Swagger 문서 생성..."
swag init --parseDependency --parseInternal
echo "✅ Swagger 생성 성공"

# 3. Service Actions 생성
echo "[3/4] Service Actions 생성..."
cd ../tool/swagger-to-actions
go run . -c frameworks.yaml
echo "✅ Service Actions 생성 성공"

# 4. 검증
echo "[4/4] 검증..."
if grep -q "SignupUser" service-actions.yaml && \
   grep -q "ResetUserPassword" service-actions.yaml; then
    echo "✅ 새 API 포함 확인"
else
    echo "❌ 새 API 누락"
    exit 1
fi

echo "=== 모든 자동 테스트 성공 ==="
```

---

## 부록 B: 참고 문서
- [요구사항 분석서](./FR-004-ANALYSIS.md)
- [구현 상세 문서](./FR-004-IMPLEMENTATION.md)
- [API 명세](../../src/docs/swagger.yaml)

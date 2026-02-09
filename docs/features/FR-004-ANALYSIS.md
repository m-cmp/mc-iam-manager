# FR-004 사용자 가입 및 승인 기능 - 요구사항 분석서

## 문서 정보
- **작성일**: 2026-02-09
- **버전**: 1.0
- **작성자**: Development Team
- **대상 시스템**: MC-IAM-Manager
- **브랜치**: feature/user-signup-approval

---

## 1. 개요

### 1.1 목적
MC-IAM-Manager에 신규 사용자 가입 및 관리자 승인 기능을 추가하여, 일반 사용자가 스스로 가입 신청하고 관리자 승인 후 시스템을 사용할 수 있도록 한다.

### 1.2 시스템 구조
- **프레임워크**: Echo v4
- **IDP**: Keycloak
- **데이터베이스**: PostgreSQL + GORM
- **아키텍처**: main.go → handler → service → keycloak/repository

---

## 2. 요구사항 분석

### 2.1 FR-004-01: 사용자 가입 신청

#### 요구사항
| 항목 | 내용 |
|------|------|
| 기능 | 신규 사용자가 이메일, 비밀번호, 이름, 조직 정보를 입력하여 가입 신청 |
| 우선순위 | 높음 |
| 입력 | 이메일, 비밀번호, 이름, 조직(선택) |
| 출력 | 가입 신청 성공 메시지 또는 에러 메시지 |
| 제약조건 | - 이메일은 유효한 형식<br>- 이메일은 고유<br>- 비밀번호 8자 이상<br>- 이름 필수<br>- 조직 선택 |

#### 현재 구현 상태 분석
**기존 API**: `POST /api/users`
- ❌ **admin 권한 필요** - Public API가 아님
- ❌ **enabled=true** - 가입 즉시 활성화됨
- ❌ **비밀번호 설정 주석 처리** - 비밀번호 입력 불가
- ❌ **Organization 필드 부재**
- ❌ **유효성 검증 부재**

#### Gap 분석
| 요구사항 | 현재 구현 | 문제점 | 우선순위 |
|---------|----------|--------|---------|
| 인증 없이 가입 신청 | admin 권한 필요 | Public API 필요 | 높음 |
| enabled=false | enabled=true | 승인 대기 상태 필요 | 높음 |
| 비밀번호 입력 | 주석 처리 | 활성화 필요 | 높음 |
| 조직 필드 | 없음 | 추가 필요 | 중간 |
| 유효성 검증 | 없음 | 추가 필요 | 높음 |

#### 구현 방안
**1. 새로운 Public API 추가**
- 엔드포인트: `POST /api/auth/signup`
- 인증 불필요 (skipAuthPaths에 추가)
- Request Model: `SignupRequest`

**2. User 모델 확장**
- Organization 필드 추가 (Keycloak attributes에 저장)

**3. Keycloak Service 확장**
- `CreatePendingUser()` 메서드 추가
  - enabled=false
  - EmailVerified=false
  - 비밀번호 설정
  - Organization을 attributes에 저장

---

### 2.2 FR-004-02: 가입 신청 폼 유효성 검증

#### 요구사항
| 항목 | 내용 |
|------|------|
| 기능 | 사용자 입력값 실시간 유효성 검증 |
| 우선순위 | 높음 |
| 제약조건 | 이메일 형식, 비밀번호 길이, 필수 필드 검증 |

#### 현재 구현 상태 분석
- ❌ validation 태그 없음
- ❌ 이메일 형식 검증 없음
- ❌ 비밀번호 길이 검증 없음
- ❌ 필드별 에러 메시지 없음

#### 구현 방안
**1. Validation 태그 추가**
```go
type SignupRequest struct {
    Email    string `validate:"required,email"`
    Password string `validate:"required,min=8"`
    // ...
}
```

**2. Validator 유틸리티 구현**
- 파일: `src/utils/validator.go`
- 한글 에러 메시지 지원
- 필드별 에러 반환

---

### 2.3 FR-004-03: 가입 신청 상태 관리

#### 요구사항
가입 신청 시 사용자 계정은 승인 대기 상태(pending)로 생성

#### Gap 분석
- 현재: enabled=true (즉시 활성화)
- 목표: enabled=false (승인 대기)

#### 구현 방안
`CreatePendingUser`에서:
- enabled=false
- EmailVerified=false
- 로그인 차단 (Keycloak에서 자동 처리)

---

### 2.4 FR-004-05: 가입 신청 에러 처리

#### 요구사항
명확한 에러 메시지 제공

#### 현재 구현 상태 분석
- ❌ 모든 에러를 500으로 반환
- ❌ 한글 메시지 없음
- ❌ 필드별 에러 구분 없음

#### 구현 방안
**1. ErrorResponse 모델**
```go
type ErrorResponse struct {
    Success bool
    Error   string
    Fields  map[string]string
    Code    string
}
```

**2. HTTP 상태 코드 구분**
- 400: 유효성 검증 실패
- 409: 중복 이메일
- 500: 서버 오류

---

### 2.5 FR-004-07: 인증 없이 가입 신청 API 호출

#### 요구사항
Public API로 구현

#### 구현 방안
**1. skipAuthPaths 추가**
```go
skipAuthPaths := []string{
    // ...
    "/api/auth/signup",
}
```

**2. Rate Limiting (선택사항)**
- 1분에 5회 제한 권장

---

### 2.6 FR-004-08: 관리자 승인 처리

#### 현재 구현 상태 분석
✅ **완전히 구현됨** - 수정 불필요

**기존 API**: `POST /api/users/id/{userId}/status`
- ✅ platformAdmin 권한 확인
- ✅ enabled=true로 변경
- ✅ 로컬 DB 동기화
- ✅ 에러 처리

---

### 2.7 FR-004-09: 관리자용 비밀번호 초기화

#### 요구사항
| 항목 | 내용 |
|------|------|
| 기능 | 관리자가 사용자 비밀번호 변경 |
| 우선순위 | 높음 |
| 제약조건 | 관리자만 접근, 비밀번호 8자 이상 |

#### 현재 구현 상태 분석
- ⚠️ SetPassword API 존재하지만 엔드포인트 없음

#### 구현 방안
**1. 새 API 추가**
- 엔드포인트: `PUT /api/users/id/{userId}/password`
- 권한: platformAdmin
- Request: `ResetPasswordRequest`

**2. Keycloak Service 확장**
```go
func ResetPassword(ctx, kcUserID, newPassword string) error {
    // SetPassword 호출
}
```

---

## 3. 구현 우선순위

### Phase 1: Core Signup API (필수)
1. ✅ SignupRequest, ResetPasswordRequest 모델 추가
2. ✅ User 모델에 Organization 필드 추가
3. ✅ CreatePendingUser 함수 구현
4. ✅ SignupUser 서비스 및 핸들러 구현
5. ✅ Public 라우트 추가

### Phase 2: Validation & Error Handling (필수)
6. ✅ Validator 유틸리티 구현
7. ✅ ErrorResponse 모델 추가
8. ✅ 필드별 검증 및 에러 메시지

### Phase 3: Password Reset API (필수)
9. ✅ ResetPassword 함수 구현
10. ✅ ResetUserPassword 핸들러 구현
11. ✅ 라우트 추가

### Phase 4: Documentation (완료)
12. ✅ Swagger 문서 생성
13. ✅ service-actions.yaml 업데이트

---

## 4. 위험 요소 및 제약사항

### 4.1 보안 고려사항
| 위험 | 대응 방안 | 상태 |
|------|----------|------|
| 무제한 가입 신청 | Rate Limiting 구현 (선택) | 권장사항 |
| 이메일 검증 부재 | EmailVerified=false 설정 | ✅ 완료 |
| 약한 비밀번호 | 8자 이상 검증 | ✅ 완료 |

### 4.2 기술적 제약사항
- Keycloak에서 enabled=false인 사용자는 로그인 불가 (자동 차단)
- Organization은 DB가 아닌 Keycloak attributes에 저장
- 승인 시 로컬 DB 동기화 필요

---

## 5. 기대 효과

### 5.1 사용자 측면
- ✅ 관리자 개입 없이 가입 신청 가능
- ✅ 명확한 한글 에러 메시지
- ✅ 조직 정보 입력으로 사용자 분류 가능

### 5.2 관리자 측면
- ✅ 가입 승인 프로세스 체계화
- ✅ 비밀번호 재설정 기능으로 사용자 지원 강화

### 5.3 시스템 측면
- ✅ API 일관성 향상 (Swagger 문서 자동 생성)
- ✅ 확장 가능한 검증 시스템 구축

---

## 6. 결론

### 6.1 구현 완료 항목
- ✅ FR-004-01: 사용자 가입 신청
- ✅ FR-004-02: 폼 유효성 검증
- ✅ FR-004-03: 가입 신청 상태 관리
- ✅ FR-004-05: 에러 처리
- ✅ FR-004-07: Public API
- ✅ FR-004-08: 관리자 승인 (기존 구현 유지)
- ✅ FR-004-09: 비밀번호 재설정

### 6.2 미구현 항목 (프론트엔드)
- FR-004-04: 성공 처리 (UI 리다이렉트)
- FR-004-06: 로그인 페이지 링크

### 6.3 선택적 개선사항
- Rate Limiting 구현
- 이메일 인증 기능
- 비밀번호 복잡도 검증 강화

---

## 부록 A: 관련 문서
- [구현 상세 문서](./FR-004-IMPLEMENTATION.md)
- [테스트 결과서](./FR-004-TEST-RESULTS.md)
- [API 명세](../../src/docs/swagger.yaml)

# Swagger-to-Actions

여러 프레임워크의 Swagger/OpenAPI 스펙을 수집하여 통합된 `serviceActions` YAML 파일을 생성하는 CLI 도구입니다.

## 기능

- 다중 프레임워크 Swagger 스펙 통합
- Swagger 2.0 및 OpenAPI 3.0+ 지원
- JSON 및 YAML 형식 자동 감지
- 로컬 파일 및 원격 URL 지원
- 단일/다중 모드 CLI 제공

## 설치

```bash
cd tool/swagger-to-actions
go build -o swagger-to-actions
```

## 사용법

### 다중 프레임워크 모드 (권장)

설정 파일을 사용하여 여러 프레임워크를 한 번에 처리합니다:

```bash
./swagger-to-actions -c frameworks.yaml
```

### 단일 프레임워크 모드

단일 Swagger 파일을 변환합니다:

```bash
./swagger-to-actions -i ./swagger.yaml -o ./actions.yaml -s mc-iam-manager
```

### Append 모드

기존 파일에 새 프레임워크를 추가합니다:

```bash
./swagger-to-actions -i ./new-swagger.yaml -o ./existing.yaml -s new-service --append
```

### 상세 출력

```bash
./swagger-to-actions -c frameworks.yaml -v
```

## CLI 플래그

| 플래그 | 단축 | 설명 | 필수 |
|--------|------|------|------|
| `--config` | `-c` | 프레임워크 설정 파일 경로 | 조건부 |
| `--input` | `-i` | 단일 Swagger 파일/URL | 조건부 |
| `--output` | `-o` | 출력 YAML 파일 경로 | O |
| `--service` | `-s` | 서비스명 (단일 모드) | 조건부 |
| `--append` | `-a` | 기존 파일에 추가 | X |
| `--verbose` | `-v` | 상세 출력 | X |
| `--timeout` | `-t` | HTTP 타임아웃 (초) | X |

## 설정 파일 형식

```yaml
# 출력 파일 경로
output: ./service-actions.yaml

# HTTP 타임아웃 (초)
timeout: 30

# 상세 출력
verbose: false

# 프레임워크 목록
frameworks:
  - name: mc-iam-manager
    repository: https://github.com/m-cmp/mc-iam-manager
    swagger: ../docs/swagger.yaml   # 로컬 경로

  - name: mc-infra-connector
    repository: https://github.com/cloud-barista/cb-spider
    swagger: https://example.com/swagger.yaml   # 원격 URL
```

## 출력 형식

```yaml
serviceActions:
  mc-iam-manager:
    mciamLogin:
      method: post
      resourcePath: /api/auth/login
      description: "Authenticate user and issue JWT token."
    mciamLogout:
      method: post
      resourcePath: /api/auth/logout
      description: "Invalidate the user's refresh token."

  mc-infra-connector:
    GetDriver:
      method: get
      resourcePath: /driver/{DriverName}
      description: "Retrieve details of a specific Cloud Driver."
```

## 종료 코드

| 코드 | 설명 |
|------|------|
| 0 | 성공 |
| 1 | 인자/설정 오류 |
| 2 | 입력 파일/URL 오류 |
| 3 | 파싱 오류 |
| 4 | 출력 쓰기 오류 |

## 설계 원칙

### 하드코딩 금지

- 프레임워크 정보는 `frameworks.yaml`에서 동적으로 로드
- 새 프레임워크 추가/제거 시 코드 수정 불필요

## 관련 문서

- [Specification](../../../mc-iam-manager-spec/specs/005-swagger-to-actions/spec.md)
- [Implementation Plan](../../../mc-iam-manager-spec/specs/005-swagger-to-actions/plan.md)
- [Task Breakdown](../../../mc-iam-manager-spec/specs/005-swagger-to-actions/tasks.md)

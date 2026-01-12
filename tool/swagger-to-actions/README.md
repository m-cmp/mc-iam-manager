# Swagger-to-Actions

여러 프레임워크의 Swagger/OpenAPI 스펙을 수집하여 통합된 `serviceActions` YAML 파일을 생성하는 CLI 도구입니다.

## 기능

- 다중 프레임워크 Swagger 스펙 통합
- Swagger 2.0 및 OpenAPI 3.0+ 지원
- JSON 및 YAML 형식 자동 감지
- 로컬 파일 및 원격 URL 지원
- 단일/다중 모드 CLI 제공
- **버전 관리**: 프레임워크별 버전 정보 및 메타데이터 포함

---

## 빠른 시작

### 1. 빌드

```bash
cd mc-iam-manager/tool/swagger-to-actions
go build -o swagger-to-actions .
```

### 2. 실행

```bash
# 설정 파일로 실행
./swagger-to-actions -c frameworks.yaml

# 또는 단일 파일 변환
./swagger-to-actions -i ../../docs/swagger.yaml -o ./output.yaml -s mc-iam-manager
```

---

## 상세 사용법

### 방법 1: 다중 프레임워크 모드 (권장)

여러 프레임워크의 Swagger 스펙을 하나의 파일로 통합합니다.

#### Step 1: 설정 파일 작성

`frameworks.yaml` 파일을 생성합니다:

```yaml
output: ./service-actions.yaml

frameworks:
  - name: mc-iam-manager
    version: "0.3.0"                    # 버전 정보
    repository: https://github.com/m-cmp/mc-iam-manager
    swagger: ../../docs/swagger.yaml

  - name: mc-infra-connector
    version: "0.9.8"                    # 버전 정보
    repository: https://github.com/cloud-barista/cb-spider
    swagger: https://raw.githubusercontent.com/cloud-barista/cb-spider/v0.9.8/api-runtime/rest-runtime/docs/swagger.yaml
```

#### Step 2: 실행

```bash
./swagger-to-actions -c frameworks.yaml
```

#### Step 3: 결과 확인

`service-actions.yaml` 파일이 생성됩니다:

```yaml
serviceActions:
  mc-iam-manager:
    _meta:                              # 메타데이터 (버전 추적용)
      version: "0.3.0"
      repository: https://github.com/m-cmp/mc-iam-manager
      generatedAt: "2026-01-06T10:00:00Z"
    mciamLogin:
      method: post
      resourcePath: /api/auth/login
      description: "Authenticate user and issue JWT token."
  mc-infra-connector:
    _meta:
      version: "0.9.8"
      repository: https://github.com/cloud-barista/cb-spider
      generatedAt: "2026-01-06T10:00:00Z"
    GetDriver:
      method: get
      resourcePath: /driver/{DriverName}
      description: "Retrieve details of a specific Cloud Driver."
```

---

### 방법 2: 단일 프레임워크 모드

단일 Swagger 파일만 변환합니다.

```bash
./swagger-to-actions -i ./swagger.yaml -o ./actions.yaml -s my-service
```

**필수 옵션**:
- `-i`: 입력 Swagger 파일 경로 또는 URL
- `-o`: 출력 파일 경로
- `-s`: 서비스 이름 (serviceActions 하위 키로 사용됨)

---

### 방법 3: 기존 파일에 추가 (Append 모드)

이미 존재하는 serviceActions 파일에 새 프레임워크를 추가합니다.

```bash
./swagger-to-actions -i ./new-swagger.yaml -o ./existing-actions.yaml -s new-service --append
```

---

### 방법 4: 원격 URL에서 가져오기

원격 Swagger 스펙을 직접 변환합니다.

```bash
./swagger-to-actions \
  -i https://raw.githubusercontent.com/cloud-barista/cb-spider/master/api-runtime/rest-runtime/docs/swagger.yaml \
  -o ./spider-actions.yaml \
  -s mc-infra-connector
```

---

## CLI 플래그

| 플래그 | 단축 | 설명 | 기본값 | 필수 |
|--------|------|------|--------|------|
| `--config` | `-c` | 프레임워크 설정 파일 경로 | - | 조건부* |
| `--input` | `-i` | 단일 Swagger 파일/URL | - | 조건부* |
| `--output` | `-o` | 출력 YAML 파일 경로 | - | O |
| `--service` | `-s` | 서비스명 (단일 모드) | - | 조건부* |
| `--version` | `-V` | 서비스 버전 (단일 모드) | - | X |
| `--repository` | `-r` | 저장소 URL (단일 모드) | - | X |
| `--append` | `-a` | 기존 파일에 추가 | false | X |
| `--verbose` | `-v` | 상세 출력 | false | X |
| `--timeout` | `-t` | HTTP 타임아웃 (초) | 30 | X |

> *조건부: `-c` 또는 `-i`/-s` 중 하나는 필수

---

## 설정 파일 형식 (frameworks.yaml)

```yaml
# 출력 파일 경로 (설정 파일 기준 상대 경로)
output: ./service-actions.yaml

# HTTP 타임아웃 (초)
timeout: 30

# 상세 출력 여부
verbose: false

# 프레임워크 목록
frameworks:
  # 로컬 파일 예제
  - name: mc-iam-manager                              # 서비스명 (필수)
    version: "0.3.0"                                  # 버전 (선택, DB 추적용)
    repository: https://github.com/m-cmp/mc-iam-manager  # 저장소 URL (선택)
    swagger: ../../docs/swagger.yaml                  # Swagger 경로 (필수)

  # 원격 URL 예제 (버전별 태그 사용)
  - name: mc-infra-connector
    version: "0.9.8"
    repository: https://github.com/cloud-barista/cb-spider
    swagger: https://raw.githubusercontent.com/cloud-barista/cb-spider/v0.9.8/api-runtime/rest-runtime/docs/swagger.yaml

  # 여러 프레임워크 추가 가능
  - name: mc-infra-manager
    version: "0.9.22"
    repository: https://github.com/cloud-barista/cb-tumblebug
    swagger: https://raw.githubusercontent.com/cloud-barista/cb-tumblebug/v0.9.22/src/api/rest/docs/swagger.yaml
```

### 경로 규칙

- **상대 경로**: 설정 파일 위치 기준으로 해석
- **절대 경로**: 그대로 사용
- **URL**: HTTP/HTTPS로 시작하면 원격에서 가져옴

---

## 출력 형식

```yaml
serviceActions:
  <서비스명>:
    _meta:                            # 메타데이터 (버전 추적용)
      version: <버전>
      repository: <저장소 URL>
      generatedAt: <생성 시간>
    <operationId>:
      method: <HTTP 메서드>
      resourcePath: <API 경로>
      description: "<설명>"
```

### 예제 출력

```yaml
serviceActions:
  mc-iam-manager:
    _meta:
      version: "0.3.0"
      repository: https://github.com/m-cmp/mc-iam-manager
      generatedAt: "2026-01-06T10:00:00Z"
    mciamLogin:
      method: post
      resourcePath: /api/auth/login
      description: "Authenticate user and issue JWT token."
    mciamLogout:
      method: post
      resourcePath: /api/auth/logout
      description: "Invalidate the user's refresh token."
    mciamGetUserById:
      method: get
      resourcePath: /api/users/{userId}
      description: "Retrieve user details by ID."

  mc-infra-connector:
    _meta:
      version: "0.9.8"
      repository: https://github.com/cloud-barista/cb-spider
      generatedAt: "2026-01-06T10:00:00Z"
    GetDriver:
      method: get
      resourcePath: /driver/{DriverName}
      description: "Retrieve details of a specific Cloud Driver."
    CreateConnectionConfig:
      method: post
      resourcePath: /connectionconfig
      description: "Create a new Connection Config."
```

---

## 실행 예제

### 기본 실행

```bash
$ ./swagger-to-actions -c frameworks.yaml

 ____                                         _                  _   _
/ ___|_      ____ _  __ _  __ _  ___ _ __    | |_ ___           / \ | | ___ | |_ ___
\___ \ \ /\ / / _  |/ _  |/ _  |/ _ \ '__|___| __/ _ \  _____  / _ \| |/ __|| __/ _ \
 ___) \ V  V / (_| | (_| | (_| |  __/ | |____| || (_) ||_____|/ ___ \ | (__ | || (_) |
|____/ \_/\_/ \__,_|\__, |\__, |\___|_|       \__\___/       /_/   \_\_\___| \__\___/
                    |___/ |___/

[INFO] Running in config mode with: frameworks.yaml
[SUCCESS] Successfully generated: service-actions.yaml
[INFO] Total frameworks: 2
[INFO]   - mc-iam-manager: 133 actions
[INFO]   - mc-infra-connector: 89 actions
[INFO] Total actions: 222
```

### 상세 출력 모드

```bash
$ ./swagger-to-actions -c frameworks.yaml -v

[INFO] Running in config mode with: frameworks.yaml
[INFO] Output file: service-actions.yaml
[INFO] Frameworks: 2
[INFO]   - mc-iam-manager: ../../docs/swagger.yaml
[INFO]   - mc-infra-connector: https://...
[INFO] Processing framework: mc-iam-manager
[INFO]   Swagger: ../../docs/swagger.yaml
[INFO]   Reading from file...
[INFO]   Version: Swagger 2.0
[INFO]   Actions: 133
[INFO] Processing framework: mc-infra-connector
[INFO]   Swagger: https://...
[INFO]   Fetching from URL...
[INFO]   Version: Swagger 2.0
[INFO]   Actions: 89
[SUCCESS] Successfully generated: service-actions.yaml
```

---

## 종료 코드

| 코드 | 설명 |
|------|------|
| 0 | 성공 |
| 1 | 인자/설정 오류 |
| 2 | 입력 파일/URL 오류 |
| 3 | 파싱 오류 |
| 4 | 출력 쓰기 오류 |

---

## 문제 해결

### "failed to fetch swagger" 오류

- 원격 URL이 올바른지 확인
- 네트워크 연결 확인
- `--timeout` 값 증가: `./swagger-to-actions -c config.yaml -t 60`

### "not a valid Swagger/OpenAPI specification" 오류

- 입력 파일이 유효한 Swagger/OpenAPI 형식인지 확인
- `swagger: "2.0"` 또는 `openapi: "3.0.x"` 필드가 있는지 확인

### 특정 프레임워크만 실패

- 도구는 개별 실패 시 경고를 출력하고 다른 프레임워크는 계속 처리
- `-v` 옵션으로 상세 오류 확인

### operationId가 없는 엔드포인트

- `operationId`가 없는 API 엔드포인트는 건너뜀
- Swagger 스펙에 `operationId`를 추가하거나, 빈 출력이 예상됨

---

## 설계 원칙

### 하드코딩 금지

- 프레임워크 정보는 `frameworks.yaml`에서 동적으로 로드
- 새 프레임워크 추가/제거 시 **코드 수정 불필요**
- 설정 파일만 수정하면 됨

---

## 버전 관리

### 버전 정보 활용

각 프레임워크의 `version` 필드는 출력 파일의 `_meta`에 포함됩니다.
mc-iam-manager는 이 정보를 활용하여 버전별 API 변경사항을 추적할 수 있습니다.

### DB 연동 시나리오

```
┌─────────────────────────────────────────────────────────────────┐
│                    swagger-to-actions                            │
│         (활성 버전의 service-actions.yaml 생성)                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    mc-iam-manager (init)                         │
│                                                                  │
│  1. service-actions.yaml 로드                                    │
│  2. _meta.version으로 버전 확인                                   │
│  3. DB에 저장 (기존 버전과 비교하여 변경사항 추적)                  │
└─────────────────────────────────────────────────────────────────┘
```

### 버전 업데이트 절차

1. `frameworks.yaml`의 버전 필드 업데이트
2. swagger 경로를 새 버전의 태그로 변경
3. `swagger-to-actions` 재실행
4. mc-iam-manager 재시작 시 새 버전 로드

```yaml
# 버전 업데이트 예시
- name: mc-infra-connector
  version: "0.9.9"    # 0.9.8 → 0.9.9로 업데이트
  repository: https://github.com/cloud-barista/cb-spider
  swagger: https://raw.githubusercontent.com/cloud-barista/cb-spider/v0.9.9/api-runtime/rest-runtime/docs/swagger.yaml
```

---

## 스크립트로 실행

`mc-iam-manager-spec/scripts/run-swagger-to-actions.sh` 스크립트를 사용할 수 있습니다:

```bash
# 빌드
../mc-iam-manager-spec/scripts/run-swagger-to-actions.sh build

# 실행
../mc-iam-manager-spec/scripts/run-swagger-to-actions.sh run -c frameworks.yaml
```

---

## 관련 문서

- [Specification](../../../mc-iam-manager-spec/specs/005-swagger-to-actions/spec.md)
- [Implementation Plan](../../../mc-iam-manager-spec/specs/005-swagger-to-actions/plan.md)
- [Task Breakdown](../../../mc-iam-manager-spec/specs/005-swagger-to-actions/tasks.md)

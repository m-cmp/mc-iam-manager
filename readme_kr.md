[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_shield)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/m-cmp/mc-iam-manager?label=go.mod)](https://github.com/m-cmp/mc-iam-manager/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/m-cmp/mc-iam-manager?status.svg)](https://pkg.go.dev/github.com/m-cmp/mc-iam-manager@master)
[![Release Version](https://img.shields.io/github/v/release/m-cmp/mc-iam-manager)](https://github.com/m-cmp/mc-iam-manager/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/m-cmp/mc-iam-manager/blob/master/LICENSE)

[M-CMP IAM Manager 문서](https://m-cmp.github.io/mc-iam-manager/)

# M-CMP IAM Manager

멀티 클라우드 인프라를 배포하고 관리하기 위한 [M-CMP 플랫폼](https://github.com/m-cmp/docs/tree/main)의 하위 시스템으로 멀티 클라우드 IAM 관리 프레임워크를 제공합니다.

## 목차

- [개요](#개요)
- [주요 기능](#주요-기능)
- [시스템 아키텍처](#시스템-아키텍처)
- [빠른 시작](#빠른-시작)
- [설치 및 설정](#설치-및-설정)
- [메뉴 관리](#메뉴-관리)
- [API 문서](#api-문서)
- [기여하기](#기여하기)
- [라이선스](#라이선스)

## 개요

M-CMP IAM Manager는 멀티 클라우드 환경에서 통합된 권한 부여 및 접근 제어 프레임워크를 제공합니다. 플랫폼 계정/역할 관리, 클라우드 계정/접근 제어 정보 통합 관리, 그리고 워크스페이스 관리 기능을 통해 기존 멀티 클라우드 서비스에 대한 보안 정책 결정, 수립 및 시행을 지원합니다.

### 주요 특징

- **멀티 클라우드 지원**: AWS, GCP, Alibaba Cloud, Tencent Cloud, NCP, NHN, KT Cloud, OpenStack 등 다양한 CSP 통합 관리
- **RBAC 기반 접근 제어**: 역할 기반 세분화된 권한 관리
- **중앙화된 관리**: 단일 플랫폼에서 모든 클라우드 리소스 접근 제어
- **임시 자격 증명**: JWT 기반 안전한 임시 접근 권한 발급

## 주요 기능

### 🏢 **엔터프라이즈 멀티 클라우드 환경 관리**
- **다중 CSP 통합 관리**: AWS, GCP, Alibaba Cloud, Tencent Cloud, NCP, NHN, KT Cloud, OpenStack 등 여러 클라우드 서비스 제공업체의 IAM을 통합 관리
- **중앙화된 권한 제어**: 모든 클라우드 리소스에 대한 접근 권한을 단일 플랫폼에서 관리
- **RBAC (역할 기반 접근 제어)**: 사용자 역할에 따른 세분화된 권한 관리
- **임시 자격 증명**: JWT 기반의 안전한 임시 접근 권한 발급

## 시스템 아키텍처

```
Internet
    |
    v
[Nginx Reverse Proxy] (Port 80/443)
    |
    +---> [IAM Manager] (Port 5000)
    |
    +---> [Keycloak] (Port 8080)
    |
    +---> [PostgreSQL] (Port 5432)
```

### 구성 요소

- **Nginx**: 리버스 프록시, SSL 종료, 정적 파일 서빙
- **IAM Manager**: 메인 애플리케이션 (Echo Framework)
- **Keycloak**: 인증 및 권한 관리
- **PostgreSQL**: 데이터베이스
- **Certbot**: SSL 인증서 자동 발급/갱신

## 빠른 시작

[mc-admin-cli](https://github.com/m-cmp/mc-admin-cli/blob/main/README.md) 안에 mc-iam-manager가 포함되어 있습니다.

### 필수 조건

- **운영체제**: Ubuntu 22.04 (테스트 완료)
- **네트워크**: 외부 접근 가능 (HTTPS-443, HTTP-80, SSH-ANY)
- **Docker**: Docker 24+ 및 Docker Compose v2
- **데이터베이스**: PostgreSQL
- **도메인**: SSL 인증서 발급을 위한 도메인 (프로덕션 환경)
- **이메일**: SSL 인증서 발급용 이메일 주소

### 설치 단계

#### 1단계: 소스 복사

```bash
git clone https://github.com/m-cmp/mc-iam-manager <YourFolderName>
cd <YourFolderName>
```

#### 2단계: 환경 설정

```bash
# 환경 설정 파일 복사
cp .env_sample .env

# 환경 변수 편집
nano .env
```

**주요 설정 항목:**
- `MC_IAM_MANAGER_EXTERNAL_DOMAIN`: 도메인 이름 (예: mciam.m-cmp.org)
- `MC_IAM_MANAGER_CERT_EMAIL`: SSL 인증서 발급용 이메일
- `MC_IAM_MANAGER_PORT`: 애플리케이션 포트 (기본값: 5000)
- `MC_IAM_MANAGER_KEYCLOAK_ADMIN`: Keycloak 관리자 계정
- `MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD`: Keycloak 관리자 비밀번호

#### 3단계: 인증서 설정

**개발 환경 (자체 인증서):**
- [자체 인증서 발급 가이드](https://github.com/m-cmp/mc-iam-manager/wiki/%EC%9E%90%EC%B2%B4-%EC%9D%B8%EC%A6%9D%EC%84%9C-%EB%B0%9C%EA%B8%89)

**프로덕션 환경 (CA 인증서):**
- [CA 인증서 발급 가이드](https://github.com/m-cmp/mc-iam-manager/wiki/CA-%EC%9D%B8%EC%A6%9D%EC%84%9C-%EB%B0%9C%EA%B8%89)

#### 4단계: 시스템 배포

**전체 시스템 배포 (권장):**
```bash
sudo docker compose -f docker-compose.yaml up -d
```

**SSL 인증서 포함 배포 (프로덕션):**
```bash
sudo docker compose -f docker-compose.yaml -f docker-compose.cert.yaml up -d
```

**소스 코드 직접 실행:**
```bash
cd ./src
go run main.go
```

### Docker 로컬 빌드 배포

`mc-iam-manager` 서비스는 로컬의 `Dockerfile.mciammanager`를 사용하여 컨테이너 이미지를 빌드하도록 구성되어 있습니다.

#### 빌드 설정

`docker-compose.yaml`에서 다음과 같이 설정되어 있습니다:

```yaml
mc-iam-manager:
  build:
    context: .
    dockerfile: Dockerfile.mciammanager
  image: cloudbaristaorg/mc-iam-manager:edge
```

#### 배포 방법

**1. mc-iam-manager 빌드 및 실행:**
```bash
# 로컬 Dockerfile로 빌드하고 시작
docker-compose up --build mc-iam-manager

# 백그라운드로 실행
docker-compose up --build -d mc-iam-manager
```

**2. 전체 서비스 실행:**
```bash
# 모든 서비스 빌드 및 시작
docker-compose up --build -d
```

**3. 완전 재빌드:**
```bash
# 캐시 없이 강제 재빌드
docker-compose build --no-cache mc-iam-manager
docker-compose up -d mc-iam-manager
```

**4. 의존성 서비스와 함께 실행:**
```bash
# 필수 서비스와 함께 mc-iam-manager 시작
docker-compose up -d mc-iam-manager-db mc-iam-manager-kc mc-iam-manager
```

#### 서비스 의존성

`mc-iam-manager` 서비스는 다음 서비스가 필요합니다:
- `mc-iam-manager-db` (PostgreSQL 데이터베이스)
- `mc-iam-manager-kc` (인증을 위한 Keycloak)

`mc-iam-manager`를 실행하면 의존성 서비스가 자동으로 시작됩니다.

#### 이미지 관리

```bash
# 최신 이미지 가져오기 (사전 빌드된 이미지 사용 시)
docker-compose pull

# Docker 이미지 목록 확인
docker images | grep mc-iam-manager

# 이전 이미지 제거
docker rmi cloudbaristaorg/mc-iam-manager:edge
```

#### 5단계: 가동 확인

```bash
curl https://<your domain or localhost>:<port>/readyz
```

## 설치 및 설정

### 초기 설정

#### 1. 인증 관련 설정

**프로덕션 환경 (도메인 및 CA 인증서):**
```bash
./asset/setup/0_preset_prod.sh
```

**개발 환경 (localhost 및 자체 인증서):**
```bash
./asset/setup/0_preset_dev.sh
```

#### 2. 기본 설정

**자동 설정 (권장):**
```bash
./asset/setup/1_setup_auto.sh
```

**수동 설정:**
```bash
./asset/setup/1_setup_manual.sh
```

### 설정 단계

1. **플랫폼 및 관리자 초기화**
   - Keycloak Realm 생성
   - Keycloak Client 생성
   - 기본 역할 생성 및 등록
   - 기본 워크스페이스 생성
   - 메뉴(`menu.yaml`) 및 역할-메뉴 시드(`permission.yaml`) 로드 — [메뉴 관리](#메뉴-관리) 참고
   - 플랫폼 관리자 사용자 생성

2. **API 리소스 설정**
   - API 리소스 데이터 초기화
   - 클라우드 리소스 데이터 설정
   - API-클라우드 리소스 매핑

3. **CSP 역할 설정**
   - CSP 역할 초기화
   - 마스터 역할-CSP 역할 매핑

### CSP IDP 설정 (프로덕션 환경)

1. **CSP 콘솔 설정**
   - IAM 메뉴에 IDP 설정 추가
   - IAM 역할 추가 (prefix: `mciam_`)
   - 역할 권한 설정
   - Trust Relation 설정

2. **MC-IAM-Manager 설정**
   - CSP 역할 추가
   - 역할 매핑 설정


## 메뉴 관리

플랫폼 콘솔 메뉴 시드 및 런타임 역할-메뉴 매핑.

### 시드 파일 (`asset/menu/`)

- `menu.yaml` — 메뉴 트리 (id, parent, path, menu resource). 이 파일은 **폴백 / 다운로드 캐시**이지 진실 소스가 아니다 — 플랫폼 전체의 canonical 메뉴 카탈로그는 mc-web-console의 `conf/webconsole_menu_resources.yaml`이다. mc-admin-cli 배포에서는 그 파일의 사본이 이 컨테이너에 마운트되고 `MC_WEB_CONSOLE_MENUYAML`이 그 마운트 경로를 가리킨다. env가 URL이면 `initial-menus`가 다운로드해 이 로컬 사본을 덮어쓴다.
- `permission.yaml` — 역할 중심 시드: `permissions → role → menus | operations | csps` (`operations` / `csps`는 예약). 참조하는 메뉴 id는 DB에 존재해야 하며, 없는 id는 건너뛰고 `missingPermissionMenuIDs`로 보고한다.
- `MC_WEB_CONSOLE_MENU_PERMISSIONS` — permission 시드의 경로 또는 YAML URL (샘플 기본값: `asset/menu/permission.yaml`). 확장자는 `.yaml` / `.yml`이어야 함. 구 CSV URL은 더 이상 시드 소스가 아님.
- `MC_WEB_CONSOLE_MENUYAML` (선택) — 메뉴 트리 YAML: 로컬 경로(작업 디렉토리 기준, 그대로 읽음) 또는 원격 URL

### 최초 1회 시딩, 이후에는 DB 편집

메뉴와 역할-메뉴 매핑은 **최초 설치 시 1회만** yaml에서 시딩한다. 그 뒤로는 DB가 진실이며, 메뉴는 `/api/menus`(또는 콘솔 Menus 화면), 매핑은 `/api/menus/platform-roles`로 변경한다. 시드를 다시 실행해도 이 편집은 덮어써지지 않는다:

- `POST /api/setup/initial-menus` — `MC_WEB_CONSOLE_MENUYAML`(또는 로컬 `filePath`)에서 시딩. DB에 이미 (`home` 외) 메뉴가 있으면 **skip**하고 `200 {skipped: true, existingMenuCount}`를 돌려주므로 설치 스크립트 재실행이 멱등이다. `?force=true`를 주면 yaml로 덮어쓰되, 그 전에 현재 역할-메뉴 매핑을 `asset/menu/backups/`에 저장한다(응답 `backupPath`). 권한 시드로 체이닝되며 응답에 `permissionsWarning`, `orphanMenusDetected`(yaml에 없는 DB id — 삭제하지 않음), `missingPermissionMenuIDs`도 실린다.
- `POST /api/setup/initial-menus2` — 같은 가드/force 동작이지만 메뉴 YAML을 요청 바디로 직접 받음(다운로드 없음, 권한 체이닝 없음) — 아직 병합되지 않은 로컬 변경을 먼저 시험해볼 때 유용
- `GET /api/setup/initial-role-menu-permission-yaml` — `permission.yaml`에서 매핑 시드 (`POST /api/initial-admin` 내부에서도 실행). **additive**: 기존 매핑은 유지되지만, 런타임에 삭제한 매핑이 yaml에 남아 있으면 되살아난다. `missingPermissionMenuIDs` 보고.
- `GET /api/setup/initial-role-menu-permission` — **삭제됨**; 이 CSV 라우트는 더 이상 존재하지 않음. `initial-role-menu-permission-yaml` 사용
- 설정 스크립트 (`conf/mc-iam-manager/1_setup_auto.sh`): 메뉴 등록 후 Step 4-1에서 `filePath` 없이 YAML 시드 호출 (서버가 env / 로컬 asset 해석)

### 런타임 변경 시 안전장치

- 역할-메뉴 매핑 변경 전: `GET /api/setup/backup-role-permissions?save=true`
- 복원: `POST /api/setup/restore-role-permissions?mode=additive|replace-role`
- 상세: [`docs/ROLE-PERMISSION-BACKUP-USAGE.md`](docs/ROLE-PERMISSION-BACKUP-USAGE.md)
- 일상 변경: 개별 매핑은 `POST` / `DELETE` `/api/menus/platform-roles`

구분: `permission.yaml`은 시드용 목표 템플릿이고, `role-permission-backup`은 실제 DB 스냅샷입니다.

### 데모 RBAC (선택, 온보딩에 포함되지 않음)

`scripts/demo/setup-demo-rbac.sh`는 역할 슬롯 15개(5모듈×3단계: viewer/operator/admin)를 준비한다 — 그중 13개는 순수 신규이고, 2개(`billviewer`/`billadmin`)는 아래([역할 관리](#역할-관리)) 기본 역할과 겹쳐 중복 생성하지 않고 재사용한다. `asset/menu/backups/role-permission-backup-demo-rbac.yaml`의 메뉴 매핑을 `POST /api/setup/restore-role-permissions?mode=additive`로 적용한다. 완전히 별도의 수동·멱등 스크립트이며 어떤 설치 스크립트도 자동으로 호출하지 않는다. 사용법은 스크립트 상단 주석 참고.

## 운영 관리

### 로그 확인

```bash
# 특정 서비스 로그 확인
sudo docker compose logs [service-name]

# 실시간 로그 모니터링
sudo docker compose logs -f [service-name]
```

### 백업

```bash
# PostgreSQL 데이터 백업
sudo docker exec <mc-iam-manager-db 서비스명> pg_dump -U <db사용자> <db명> > backup.sql

# Keycloak 데이터 백업
sudo tar -czf keycloak-backup.tar.gz container-volume/keycloak/
```

### 업데이트

```bash
# 이미지 업데이트
sudo docker compose -f docker-compose.yaml pull
sudo docker compose -f docker-compose.yaml up -d
```

## API 문서

### Swagger 문서 생성

```bash
cd ./src
swag init --output ./docs
```

### API 문서 접근

- **온라인 문서**: https://m-cmp.github.io/mc-iam-manager/
- **로컬 문서**: `http://localhost:<port>/swagger/index.html`

## 사용자 관리

### 기본 사용자 추가

1. **플랫폼 관리자 로그인**
   ```bash
   POST /api/auth/login
   {
     "id": "<MC_IAM_MANAGER_PLATFORMADMIN_ID>",
     "password": "<MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD>"
   }
   ```

2. **사용자 추가**
   - 사용자 계정 생성
   - 사용자-역할 매핑
   - 워크스페이스 공유 (선택사항)

### 역할 관리

**기본 역할:**
- `admin`: 관리자 권한
- `operator`: 운영자 권한
- `viewer`: 조회 권한
- `billadmin`: 비용 관리 권한
- `billviewer`: 비용 조회 권한

## 트러블슈팅

### 설치 후 `mc-iam-manager`가 unhealthy 상태로 지속될 때

`docker compose ps`에서 `mc-iam-manager`가 **unhealthy** 이고
`docker logs mc-iam-manager-post-initial` 끝에
`ERROR: 1_setup_auto.sh failed`가 보이면, post-init 컨테이너가
mc-iam-manager의 초기 부팅 완료 전에 실행된 것입니다 (cold-start 타이밍 race).

복구 방법:

```bash
# 1. 모든 사전 조건이 healthy 상태인지 확인
docker compose ps

# 2. 종료된 post-init 컨테이너를 삭제하고 재실행
docker rm mc-iam-manager-post-initial 2>/dev/null
docker compose up -d mc-iam-manager-post-initial
docker logs -f mc-iam-manager-post-initial
# 8단계 각각이 ✓ 로 완료되어야 합니다

# 3. 상태 확인
curl -s http://localhost:${MC_IAM_MANAGER_PORT}/readyz | jq .
# 예상 결과: "status": "healthy"
```

> post-init 컨테이너는 멱등(idempotent)하게 설계되어 있어 재실행해도 안전합니다.

### `0_preset_dev.sh` 실행 시 디렉토리 권한 오류

`0_preset_dev.sh`가 `Cannot create ... / is not writable`로 실패하면, 이전 Docker 실행으로 생긴 root 소유 파일이 접근을 막고 있는 것입니다. 아래 명령으로 정리 후 재시도하세요:

```bash
sudo rm -rf container-volume/mc-iam-manager/postgres container-volume/mc-iam-manager/keycloak
./conf/mc-iam-manager/0_preset_dev.sh
```

---

## 기여하기

- **이슈 보고**: [GitHub Issues](https://github.com/m-cmp/mc-iam-manager/issues)
- **토론**: [GitHub Discussions](https://github.com/m-cmp/mc-iam-manager/discussions)
- **아이디어 제안**: [GitHub Issues](https://github.com/m-cmp/mc-iam-manager/issues)

## 라이선스

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)

이 프로젝트는 Apache 2.0 라이선스 하에 배포됩니다.

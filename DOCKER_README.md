# M-CMP IAM Manager - Docker 배포 가이드

## 개요

이 문서는 M-CMP IAM Manager를 Docker 환경에서 단독으로 배포하는 방법을 단계별로 안내합니다.  
시스템은 Keycloak 인증, PostgreSQL 데이터베이스, Nginx 리버스 프록시, 그리고 SSL 인증서 관리를 포함합니다.

---

## 빠른 시작 (권장)

`installAll.sh`는 초기 환경 부트스트랩부터 컨테이너 기동·모니터링까지 한 번에 처리합니다.

```bash
# 로컬 PC (plain HTTP)
./installAll.sh -m dev -d localhost -r background

# 원격 VM (self-signed HTTPS)
./installAll.sh -m dev -d <VM_PUBLIC_IP> -r background

# 운영 도메인 (Let's Encrypt HTTPS)
./installAll.sh -m prod -d iam.example.com -r background

# 옵션 없이 실행하면 대화형 모드
./installAll.sh
```

옵션 설명:
- `-m dev|prod` : 개발(self-signed cert) / 운영(Let's Encrypt) 모드
- `-d <domain|IP>` : 공개 도메인 또는 IP (`localhost` 기본값)
- `-r log|background|skip` : 서비스 기동 방식

---

## 수동 설치 (단계별)

### 1단계: Docker 설치

Ubuntu 시스템에 Docker를 설치합니다.

```bash
# 시스템 패키지 업데이트
sudo apt update

# 기존 Docker 관련 패키지 제거
for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
    sudo apt-get remove $pkg
done

# Docker 공식 GPG 키 추가
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Docker 공식 저장소 추가
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update

# Docker Engine 설치
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Docker 서비스 시작 및 활성화
sudo systemctl start docker
sudo systemctl enable docker
```

### 2단계: 환경 설정

```bash
# .env.setup을 복사해 .env 생성
cp .env.setup .env

# 필수 환경 변수 편집
nano .env
```

주요 설정 항목:
- `MC_IAM_MANAGER_PUBLIC_DOMAIN`: 공개 도메인 또는 IP
- `MC_IAM_MANAGER_CERT_EMAIL`: SSL 인증서 발급용 이메일 (prod 모드)
- `MC_IAM_MANAGER_PORT`: 애플리케이션 포트 (기본값: 5005)
- `MC_IAM_MANAGER_KEYCLOAK_ADMIN`: Keycloak 관리자 계정
- `MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD`: Keycloak 관리자 비밀번호
- `MC_IAM_MANAGER_PLATFORMADMIN_ID/PASSWORD`: MCMP 플랫폼 관리자 계정

### 3단계: Nginx 설정 생성

모드에 따라 nginx.conf와 인증서를 생성합니다.

```bash
# dev 모드 (self-signed 인증서)
cd conf/mc-iam-manager/
./0_preset_dev.sh
cd -

# prod 모드 (먼저 Let's Encrypt 인증서 발급 후 nginx 설정)
docker compose -f docker-compose.cert.yaml --env-file .env up
cd conf/mc-iam-manager/
./0_preset_prod.sh
cd -
```

생성되는 파일: `container-volume/mc-iam-manager/nginx/nginx.conf`

### 4단계: 시스템 배포

```bash
# 백그라운드 실행
docker compose --env-file .env up -d

# 포그라운드 실행 (로그 실시간 확인)
docker compose --env-file .env up
```

---

## 배포 확인

### 서비스 상태 확인

```bash
# 컨테이너 상태 확인
docker compose ps

# 서비스 로그 확인
docker compose logs -f

# 특정 서비스 로그
docker compose logs -f mc-iam-manager
```

### 정상 배포 확인 사항

#### PostgreSQL 정상 배포
```
mc-iam-manager-db  | database system is ready to accept connections
```

#### Keycloak 정상 배포
```
mc-iam-manager-kc  | Keycloak 24.0.1 on JVM (powered by Quarkus 3.8.1) started in 17.266s.
```

#### IAM Manager 정상 배포
```
mc-iam-manager  | http server started on [::]:5005
```

### 접속 테스트

```bash
# readyz 엔드포인트 확인
curl http://localhost:5005/readyz

# HTTPS (self-signed 또는 prod)
curl -k https://<domain>/readyz

# Keycloak admin console
open http://localhost:8080/admin/
```

---

## 시스템 아키텍처

```
Internet
    |
    v
[Nginx Reverse Proxy] (Port 80/443)
    |
    +---> [mc-iam-manager] (Port 5005)
    |
    +---> [mc-iam-manager-kc / Keycloak] (Port 8080)
    |
    +---> [mc-iam-manager-db / PostgreSQL] (Port 15432)
```

### 서비스 구성 (docker-compose.yaml 기준 전체 서비스)

| 서비스 | 역할 | 포트 |
|---|---|---|
| mc-infra-connector | CB-Spider (CSP 연동) | 1024 |
| mc-infra-manager | CB-Tumblebug (인프라 관리) | 1323 |
| mc-infra-manager-etcd | etcd | 2379/2380 |
| mc-infra-manager-postgres | Tumblebug DB | 6432 |
| mc-infra-manager-openbao | Vault 호환 시크릿 관리 | 8200 |
| mc-iam-manager | IAM 앱 (Echo Framework) | 5005 |
| mc-iam-manager-db | IAM/Keycloak 공유 PostgreSQL | 15432 |
| mc-iam-manager-kc | Keycloak | 8080 |
| mc-iam-manager-nginx | 리버스 프록시 | 80/443 |
| mc-iam-manager-post-initial | 초기화 컨테이너 (실행 후 종료) | - |
| mc-web-console-db | 웹 콘솔 DB | 15433 |
| mc-web-console-api | 웹 콘솔 API | 3000 |
| mc-web-console-front | 웹 콘솔 프론트엔드 | 3001 |

---

## 문제 해결

### 인증서 발급 실패
- 도메인 DNS 설정 및 80번 포트 외부 접근 가능 여부 확인
- 이메일 주소 유효성 확인 (`MC_IAM_MANAGER_CERT_EMAIL`)

### Nginx 설정 오류
```bash
docker exec mc-iam-manager-nginx nginx -t
```

### Keycloak 헬스체크 실패
```bash
docker logs mc-iam-manager-kc
```
PostgreSQL 연결 상태 및 환경 변수(`MC_IAM_MANAGER_KEYCLOAK_*`) 확인

### mc-iam-manager-post-initial 초기화 재실행

post-initial 컨테이너는 IAM/Keycloak이 healthy 상태일 때 자동으로 초기화를 수행합니다.  
실패한 경우 다음으로 재실행:
```bash
docker compose up mc-iam-manager-post-initial
```

---

## 유지보수

### 서비스 중지 및 재시작
```bash
# 정지 (볼륨 보존)
docker compose stop

# 재시작
docker compose start

# 완전 삭제 (볼륨 포함)
docker compose down -v
sudo rm -rf container-volume
```

### 인증서 갱신 (prod 모드)
```bash
docker compose -f docker-compose.cert.yaml run --rm mcmp-certbot renew
```

### 업데이트
```bash
docker compose pull
docker compose up -d
```

### env 변수 추가 시

신규 변수는 `.env.setup`과 `.env_sample`에 동시에 추가하세요.  
`installAll.sh` 재실행 시 `sync_missing_env_vars`가 기존 `.env`에 누락 변수를 자동으로 append합니다.

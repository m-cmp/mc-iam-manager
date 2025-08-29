# M-CMP IAM Manager - Docker 배포 가이드

## 개요

이 문서는 M-CMP IAM Manager를 Docker 환경에서 배포하는 방법을 단계별로 안내합니다. 
시스템은 Keycloak 인증, PostgreSQL 데이터베이스, Nginx 리버스 프록시, 그리고 SSL 인증서 관리를 포함합니다.

## 시스템 요구사항

### 필수 조건
- Ubuntu 22.04 LTS (외부 접근 가능)
- Docker Engine 24.0+
- Docker Compose v2
- 도메인 이름 (예: example.com)
- SSL 인증서 발급용 이메일 주소
- 방화벽에서 다음 포트 허용:
  - HTTP (80)
  - HTTPS (443)
  - SSH (22)

### 네트워크 요구사항
- 외부에서 접근 가능한 공인 IP
- 도메인 DNS 설정 완료
- 80/443 포트 외부 접근 허용

## 설치 및 배포 과정

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

# 설치 확인
sudo docker run hello-world
```

### 2단계: 환경 설정

프로젝트 환경 변수를 설정합니다.

```bash
# 환경 설정 파일 복사
cp .env_sample .env

# 환경 변수 편집
nano .env
```

주요 설정 항목:
- `DOMAIN_NAME`: 도메인 이름 (예: mciam.onecloudcon.com)
- `EMAIL`: SSL 인증서 발급용 이메일
- `MCIAMMANAGER_PORT`: 애플리케이션 포트 (기본값: 3000)
- `KEYCLOAK_ADMIN`: Keycloak 관리자 계정
- `KEYCLOAK_ADMIN_PASSWORD`: Keycloak 관리자 비밀번호

### 3단계: SSL 인증서 발급

Let's Encrypt를 사용하여 SSL 인증서를 발급합니다.

```bash
# SSL 인증서 발급
sudo docker compose -f docker-compose.cert.yaml up
```

성공적인 인증서 발급 시 다음과 같은 메시지가 표시됩니다:

```
mcmp-certbot  | Requesting a certificate for mciam.onecloudcon.com
mcmp-certbot  | Successfully received certificate.
mcmp-certbot  | Certificate is saved at: /etc/letsencrypt/live/mciam.onecloudcon.com/fullchain.pem
mcmp-certbot  | Key is saved at: /etc/letsencrypt/live/mciam.onecloudcon.com/privkey.pem
mcmp-certbot  | This certificate expires on 2025-10-20.
```

### 4단계: Nginx 설정 생성

환경 변수를 기반으로 Nginx 설정 파일을 생성합니다.

```bash
# Nginx 설정 스크립트 실행
./asset/setup/0_preset_create_nginx_conf.sh
```

생성된 파일: `dockerfiles/nginx/nginx.conf`

### 5단계: 시스템 배포

전체 시스템을 배포합니다.

```bash
# 전체 시스템 배포
sudo docker compose -f docker-compose.all.yaml up -d
```

## 배포 확인

### 서비스 상태 확인

```bash
# 컨테이너 상태 확인
sudo docker ps

# 서비스 로그 확인
sudo docker compose -f docker-compose.all.yaml logs -f
```

### 정상 배포 확인 사항

#### PostgreSQL 정상 배포
```
mciam-postgres  | database system is ready to accept connections
```

#### Keycloak 정상 배포
```
mciam-keycloak  | Keycloak 24.0.1 on JVM (powered by Quarkus 3.8.1) started in 17.266s. Listening on: http://0.0.0.0:8080
mciam-keycloak  | Added user 'admin' to realm 'master'
```

#### IAM Manager 정상 배포
```
mciam-manager   | High performance, minimalist Go web framework
mciam-manager   | https://echo.labstack.com
mciam-manager   | http server started on [::]:3000
```

#### Nginx 정상 배포
```
mciam-nginx     | Configuration complete; ready for start up
```

### 접속 테스트

```bash
# HTTPS 접속 테스트
curl -k https://your-domain.com/readyz

# HTTP에서 HTTPS 리다이렉트 테스트
curl -I http://your-domain.com
```

## 시스템 아키텍처

```
Internet
    |
    v
[Nginx Reverse Proxy] (Port 80/443)
    |
    +---> [IAM Manager] (Port 3000)
    |
    +---> [Keycloak] (Port 8080)
    |
    +---> [PostgreSQL] (Port 5432)
```

### 서비스 구성
- **Nginx**: 리버스 프록시, SSL 종료, 정적 파일 서빙
- **IAM Manager**: 메인 애플리케이션 (Echo Framework)
- **Keycloak**: 인증 및 권한 관리
- **PostgreSQL**: 데이터베이스
- **Certbot**: SSL 인증서 자동 발급/갱신

## 문제 해결

### 일반적인 문제

#### Docker 서비스 시작 실패
```bash
sudo systemctl start docker
sudo systemctl status docker
```

#### 인증서 발급 실패
- 도메인 DNS 설정 확인
- 80번 포트 외부 접근 가능 여부 확인
- 이메일 주소 유효성 확인

#### Nginx 설정 오류
```bash
# Nginx 설정 문법 검사
sudo docker exec mciam-nginx nginx -t
```

#### Keycloak 헬스체크 실패
- PostgreSQL 연결 상태 확인
- Keycloak 로그 확인
- 환경 변수 설정 확인

### 로그 확인

```bash
# 특정 서비스 로그 확인
sudo docker compose -f docker-compose.all.yaml logs [service-name]

# 실시간 로그 모니터링
sudo docker compose -f docker-compose.all.yaml logs -f [service-name]
```

## 유지보수

### 인증서 갱신
Let's Encrypt 인증서는 90일마다 갱신이 필요합니다.

```bash
# 수동 갱신
sudo docker compose -f docker-compose.cert.yaml run --rm mcmp-certbot renew

# 자동 갱신 설정 (cron)
0 12 * * * /usr/bin/docker compose -f /path/to/docker-compose.cert.yaml run --rm mcmp-certbot renew
```

### 백업
```bash
# PostgreSQL 데이터 백업
sudo docker exec mciam-postgres pg_dump -U iammanager iammanagerdb > backup.sql

# Keycloak 데이터 백업
sudo tar -czf keycloak-backup.tar.gz dockercontainer-volume/keycloak/
```

### 업데이트
```bash
# 이미지 업데이트
sudo docker compose -f docker-compose.all.yaml pull
sudo docker compose -f docker-compose.all.yaml up -d
```

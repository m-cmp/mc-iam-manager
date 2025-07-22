[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_shield)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/m-cmp/mc-iam-manager?label=go.mod)](https://github.com/m-cmp/mc-iam-manager/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/m-cmp/mc-iam-manager?status.svg)](https://pkg.go.dev/github.com/m-cmp/mc-iam-manager@master)
[![Release Version](https://img.shields.io/github/v/release/m-cmp/mc-iam-manager)](https://github.com/m-cmp/mc-iam-manager/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/m-cmp/mc-iam-manager/blob/master/LICENSE)

[M-CMP IAM Manager docs](https://m-cmp.github.io/mc-iam-manager/)

# M-CMP IAM Manager
이 저장소는 멀티 클라우드 인프라를 배포하고 관리하기 위한 [M-CMP 플랫폼](https://github.com/m-cmp/docs/tree/main)의 하위 시스템으로 멀티 클라우드 IAM 관리 프레임워크를 제공합니다.


## 개요

멀티 클라우드 권한 부여 및 접근 제어 프레임워크는 플랫폼 계정/역할 관리, 클라우드 계정/접근 제어 정보 통합 관리, 그리고 작업 공간 관리 기능을 제공합니다. 
이는 기존 멀티 클라우드 서비스에 대한 보안 정책 결정, 수립 및 시행과 호환되는 기능을 제공합니다. 
또한, 프레임워크 내에서 독립적인 보안 정책을 수립하고 관리할 수 있는 기능을 제공합니다.
이는 멀티 클라우드에 대한 접근 제어 참조 모델을 정의하며, 사용자 접근 제어와 서비스 제공자 접근 제어를 구분합니다. 
이 모델은 주요한 역할 기반 접근 제어(RBAC) 방식을 채택하고 이를 애플리케이션 및 활용을 위한 기존 정책 관리 솔루션과 통합합니다.


## Quick Start with docker

이 가이드는 Docker를 사용하여 MC-IAM-MANAGER를 시작하는 방법을 안내합니다. 
빠른 시작 가이드는 기본 관리자, 운영자, 뷰어 계정 및 환경을 설정합니다.


### 필수 조건
- 외부 접근이 가능한 Ubuntu (22.04 테스트 완료) (https-443, http-80, ssh-ANY)
- docker(24+) 및 docker-compose(v2)
- 도메인 이름 (예: megazone.com)
- SSL을 등록하기 위한 이메일 주소
- https 설정 : nginx + keycloak + certbot 설정은 별도 문서 참조
- database : postgres 등 


### 1단계 : 소스 복사

```bash
git clone <https://github.com/m-cmp/mc-iam-manager> <YourFolderName>
```

### 2단계 : 환경 설정
  .env_sample 파일을 참조하여 .env 생성 및 파일에 설정값을 반영

  ```bash
  cp .env_sample .env
  ```

#### 환경 변수 편집
  ```bash
  nano .env
  ```

  주요 설정 항목:
  - `DOMAIN_NAME`: 도메인 이름 (예: mciam.megazone.com)
  - `EMAIL`: SSL 인증서 발급용 이메일
  - `MCIAMMANAGER_PORT`: 애플리케이션 포트 (기본값: 3000)
  - `KEYCLOAK_ADMIN`: Keycloak 관리자 계정
  - `KEYCLOAK_ADMIN_PASSWORD`: Keycloak 관리자 비밀번호

#### SSL 인증서 발급(필요시)
  # SSL 인증서 발급  
  ```bash  
  sudo docker compose -f docker-compose.cert.yaml up
  ```

  ** 인증서 갱신 : Let's Encrypt 인증서는 90일마다 갱신이 필요합니다.
  ```bash
  # 수동 갱신
  sudo docker compose -f docker-compose.cert.yaml run --rm mcmp-certbot renew

  # 자동 갱신 설정 (cron)
  0 12 * * * /usr/bin/docker compose -f /path/to/docker-compose.cert.yaml run --rm mcmp-certbot renew
  ```

  성공적인 인증서 발급 시 다음과 같은 메시지가 표시됩니다:
  ```
  mcmp-certbot  | Requesting a certificate for [도메인 이름]
  mcmp-certbot  | Successfully received certificate.
  mcmp-certbot  | Certificate is saved at: /etc/letsencrypt/live/[도메인 이름]/fullchain.pem
  mcmp-certbot  | Key is saved at: /etc/letsencrypt/live/[도메인 이름]/privkey.pem
  mcmp-certbot  | This certificate expires on 2025-10-20.
  ```

#### Nginx 설정 생성
  환경 변수를 기반으로 Nginx 설정 파일을 생성합니다.
  # Nginx 설정 스크립트 실행
  ```bash
  ./asset/setup/0_preset_create_nginx_conf.sh
  ```

  생성된 파일: `dockerfiles/nginx/nginx.conf`
  ** keycloak 주소는 /auth를 붙임. **
  ```bash
   location /auth/ {
            proxy_pass http://mciam-keycloak:8080/auth/;
  ```


### 3단계 : MC-IAM-MANAGER Init Setup 
  build 후 /readyz 호출로 가동 확인
  # 전체 시스템 배포( mc-iam-manager + nginx + postgres + keycloak)
  ```bash
  sudo docker compose -f docker-compose.all.yaml up -d

  # MC-IAM-MANAGER만 배포 
  ```bash
  sudo docker compose -f docker-compose.standalone.yaml up -d
  ```


  # MC-IAM-MANAGER만 소스로 실행하는 경우( nginx + postgres + keyclok이 이미 가동중 )
  ```bash
  cd ./src
  go run main.go
  ```


#### 가동 확인
curl https://<your domain or localhost>:<port>/readyz
```

#### 서비스 구성
- **Nginx**: 리버스 프록시, SSL 종료, 정적 파일 서빙
- **IAM Manager**: 메인 애플리케이션 (Echo Framework)
- **Keycloak**: 인증 및 권한 관리
- **PostgreSQL**: 데이터베이스
- **Certbot**: SSL 인증서 자동 발급/갱신
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

#### 로그 확인

  ```bash
  # 특정 서비스 로그 확인
  sudo docker compose -f docker-compose.all.yaml logs [service-name]

  # 실시간 로그 모니터링
  sudo docker compose -f docker-compose.all.yaml logs -f [service-name]
  ```

#### 백업
  ```bash
  # PostgreSQL 데이터 백업
  sudo docker exec mciam-postgres pg_dump -U iammanager iammanagerdb > backup.sql

  # Keycloak 데이터 백업
  sudo tar -czf keycloak-backup.tar.gz dockercontainer-volume/keycloak/
  ```

#### 업데이트
  ```bash
  # 이미지 업데이트
  sudo docker compose -f docker-compose.all.yaml pull
  sudo docker compose -f docker-compose.all.yaml up -d
  ```

#### API 문서
  swagger 문서 생성 : src 폴더에서 
  ```bash
  swag init -g src/main.go -o src/docs
  ```


### 4단계 : 환경설정  
  ** 환경설정 파일(.env)의 정보 기반으로 platform 설정
  
  /asset/setup/1setup.sh 를 실행하고 순서대로 작업한다.
    
    1. Init Platform And PlatformAdmin
      . Keycloak 에 Realm 생성
      . Keycloak 에 Client 생성
      . Keycloak 에 Predefined Role 생성
      . DB에 Predefined Role 등록
      . 기본 workspace 생성
      . 메뉴 등록 (기본 yaml파일일)
      . 메뉴와 기본 역할 매핑
      . Keycloak 에 PlatformAdmin User 생성
      . DB에 PlatformAdmin User 등록
    2. PlatformAdmin Login
      . 다음 script들을 실행시키기 위해 access_token 발급
    3. (optional) Init Predefined Role Data : 1에서 진행함. 추가 role이 필요한 경우 실행
    4. (optional) Init Menu Data : 1에서 진행함. 추가 메뉴가 필요한 경우 실행
    5. Init API Resource Data
      . menu의 접근 권한 로드 ( /asset/menu/permission.csv 기준.)
    6. Init Cloud Resource Data
    7. Map API-Cloud Resources
    8. Init CSP Roles
      . 현재 버전에서는 IAM-Role과 CSP의 Role의 생성후 임시자격증명을 위한 설정은 해당 CSP Console에서 직접 작업한다.
    9. Map Master Role-CSP Roles
    

### 4단계 : CSP IDP 설정
  . csp console에 접속( ex. aws console 접속 )
    . iam 메뉴에 idp 설정 추가
    . iam role 추가 : cspRole과 1:1로 매칭되며 prefix로 mciam_을 가짐. 
        predefined된 mc-iam-manager의 역할을 추가하여 연결한다. 
        참고. aws에서 역할 추가시 webIdentity role로 추가.
    . iam role의 권한 추가
        해당 역할이 할 수 있는 권한 추가( ex. EC2ReadOnly 등)
    . iam role의 trust relation 설정에 keycloak client를 audience로 추가.


# 환영합니다: 이제 MC-IAM-MANAGER를 사용할 수 있습니다.
  다음 작업으로는 사용자 추가 및 역할 설정입니다.
  . user 추가
  . user-role 매핑
  . (options) user에게 workspace 공유

    
    1. platformAdmin login
      POST /api/auth/login  { "id": <MCIAMMANAGER_PLATFORMADMIN_ID>, "password": <MCIAMMANAGER_PLATFORMADMIN_PASSWORD> }
    
    2. Init Role Data PREDEFINED_ROLE to Platform Role & Realm role   # 이것을 직접 console에서 처리? : script로 처리(platformAdmin이면 호출 가능 )
      (<PREDEFINED_ROLE> 에서 , 를 구분자로 추출: admin,operator,viewer,billadmin,billviewer)
      POST /api/platform-roles/

    3. Init Menu Data from menu.yaml
      (MCWEBCONSOLE_MENUYAML: https://raw.githubusercontent.com/m-cmp/mc-web-console/refs/heads/main/conf/webconsole_menu_resources.yaml)
      POST /api/setup/menus/register-from-yaml
    
    4. Init API Resource Data from api.yaml
      (MCADMINCLI_APIYAML: https://raw.githubusercontent.com/m-cmp/mc-admin-cli/refs/heads/main/conf/api.yaml)
      POST /api/setup/sync-apis
    
    5. Init Cloud Resource Data from cloud-resource.yaml
      ex. mc-infra-manager
        vm - mci ( GetMci, GetVm ...)
           - sshkey
           - securitygroup
           - vpc
           - 
        nlb - nlb
        k8s - pmk
        bill - cost
        common - ns
    
    6. mapping api-cloud resources
        기본 permission과 대상 api 연계
        read : 조회관련 api 모두 access가능
        write : 수정관련 api 모두 access가능
        manage : 모든 api access 가능

    7. workspace role 등록
      (PREDEFINED_ROLE: admin,operator,viewer,billadmin,billviewer)
      POST /api/workspace-roles/

    8. workspace role - csp iam role 연계
        workspace role이 csp에 mcmp_<workspace-role> 형태로 생성 : platformAdmin 최초생성



swagger docs

  https://m-cmp.github.io/mc-iam-manager/

  
---


## How to Contribute
- Issues/Discussions/Ideas: Utilize issue of mc-iam-manager


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)



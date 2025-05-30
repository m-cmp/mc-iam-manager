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
- docker 및 docker-compose
- 도메인 (Keycloak 및 IDP 설정 용 ) 및 certbot으로 SSL을 등록하기 위한 이메일


### 1단계 : 소스 복사

```bash
git clone <https://github.com/m-cmp/mc-iam-manager> <YourFolderName>
```

### 2단계 : 환경 설정

```bash
cp .env_sample .env

```

### 3단계 : MC-IAM-MANAGER Init Setup 
#### DB table 생성 및 초기 data

```bash
./asset/sql/mcmp_table.sql, ./asset/sql/mcmp_init_data.sql

platformRole, workspaceRole

```

#### 4단계 : Keycloak 환경설정
    realm 생성, client 생성, Role 추가 등 
     .env 에 정의된 값으로 mcmp-ream-import.json 을 생성하고 keycloak에서 import 한다.

### Step four: Excute

#### docker 실행 docker-compose
```bash
sudo docker-compose up --build -d
```

#### source 실행
```bash
go run main.go 
```

### Step final: Check Readyzenpoint

```bash
$ curl https://<yourdomain.com>:5000/readyz
# {"ststus":"ok"}
```

If `{"stststus":"ok"}` is received from the endpoint, it means that the service is being deployed normally.

#### 5단계 : 사용설정
# 환영합니다: 이제 MC-IAM-MANAGER를 사용할 수 있습니다.
  관리자 설정은 keycloak에 직접 로그인하여 추가해야합니다.( realm_export.json 을 이용하여 설정을 가져올 수도 있습니다.)
    . 사전설정 : public domain<KEYCLOAK_HOST>, https통신 설정정
    . keycloak admin colsole에 접속
      . realm 추가 : .env를 참고하여 <KEYCLOAK_REALM>
      . client 추가 : .env를 참고 <KEYCLOAK_CLIENT>. 생성 후 client secret를 .env에 복사한다.
      . realm role 추가 : .env를 참고 <PREDEFINED_ROLE>
      . user 추가 : .env를 참고 <MCIAMMANAGER_PLATFORMADMIN_ID>
      . user-role 매핑 : platformAdmin role to user
      . (client > authorization > resource : menu )
      . client 추가 : .env를 참고 <KEYCLOAK_OIDC_CLIENT>

    . csp console에 접속
      . iam 메뉴에 idp 설정 추가
      . iam role 추가 : workspaceRole과 1:1로 매칭되며 prefix로 MCMP_를 가짐. webIdentity role로 추가.
      . iam role의 권한 추가
      . iam role의 trust relation 설정에 keycloak client를 audience로 추가.
            

이 섹션에서는 platformAdmin이 작업할 프로세스를 간단하게 하는 스크립트를 설명합니다.

- 0setup.sh
--------------------
    0. exit
    
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


---------- TODO ------------------
done) platformRole 등록 : predefined .env    
done) workspaceRole 등록 : predefined .env ( platformRole과 최초 동일)
cspRole 등록 : workspaceRole과 1:1로 생성. 대상 csp의 iam에 1:1로 role 매핑
platformResource 등록 : menu
workspaceResource 등록 : vm, k8s
platformResource 등록 : api
workspaceResource-platformResource 매핑 : vm.read, vm.write, vm.manage 등에 각 api 매핑

- 1 regist.sh
menu등록 : yaml에서 db로, db에서 keycloak로(?)
workspace 등록 : 기본 workspace 등록 
project 등록 : mc-infra-manager 와 동기화
workspace-project 매핑 : 기본 workspace에 모든 project mapping

- 2. usecase.sh
유즈케이스
사용자 추가(testadmin, testviewer)
사용자의 플랫폼 롤 지정( 관리자, 뷰어  )
사용자의 workspace 및 workspace role 지정( testadmin에 workspace 할당 및 admin role 할당, testviewer에 workspace 할당 및 viewer role 할당)

메뉴 추가
 - admin, viewer 추가 => 실패
 - platformAdmin 추가 => 성공
workspace 추가
 - admin, viewer 추가 => 실패
 - platformAdmin 추가 => 성공
project 등록 
 - viewer 추가 => 실패
 - admin 추가 => 성공
사용자에게 할당
 - 추가한 workspace에 testadmin을  admin으로 추가
 - 추가한 workspace에 testviewer를 viewer로 추가




- init.sh
    
    
    --------------------
    select Number : 
    ```
    
    Running this script allows you to view the menu above, using the information defined in .env to perform tasks according to the numbers you enter.
    
    However, the first priority is to log in by entering the user's information that you entered. If you run number 1 and run numbers 2 to 6, you will be able to use MC-WEB-CONSOLE.
    
- initauto.sh
    
    ```
    # ./scripts/init/initauto.sh
    ./initauto.sh
    ```
    
    This script automatically performs all procedures based on the user defined in the environment variables, but it cannot define detailed role-specific menus, and it is automatically imported to the version listed in GitHub.
    
    The CSV files uploaded to GitHub are as follows. You can modify and reflect the corresponding permission file set (CSV) directly through init.sh .
    
    ```bash
    framework,resource,adminPolicy,billadminPolicy,billviewerPolicy,operatorPolicy,viewerPolicy
    mc-web-console,settingsmenu,TRUE,,,TRUE,TRUE
    mc-web-console,accountnaccessmenu,TRUE,,,,
    mc-web-console,organizationsmenu,TRUE,,,,
    mc-web-console,companyinfomenu,TRUE,,,,
    mc-web-console,usersmenu,TRUE,,,,
    mc-web-console,approvalsmenu,TRUE,,,,
    mc-web-console,accesscontrolsmenu,TRUE,,,,
    mc-web-console,environmentmenu,TRUE,,,TRUE,TRUE
    mc-web-console,cloudspsmenu,TRUE,,,TRUE,
    mc-web-console,cloudoverviewmenu,TRUE,,,TRUE,
    mc-web-console,regionsmenu,TRUE,,,TRUE,
    ....
    
    ```
    
    If you want more detailed settings, we recommend init.sh .
    
- add_demo_user.sh
    
    ```
    # ./scripts/init/add_demo_user.sh
    ./add_demo_user.sh
    ```
    
    This script registers the demo user defined in ./scripts/init/add_demo_user.json. The process of registering is very simple and you can automatically activate the registered user. Use MC-WEB-CONSOLE for role setup and workspace interworking.
    

swagger docs

https://m-cmp.github.io/mc-iam-manager/

---


## How to Contribute
- Issues/Discussions/Ideas: Utilize issue of mc-iam-manager


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)



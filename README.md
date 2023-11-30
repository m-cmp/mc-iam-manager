# mc-iam-manager-README.md
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_shield)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/cloud-barista/cb-spider?label=go.mod)](https://github.com/cloud-barista/cb-spider/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/cloud-barista/cb-spider?status.svg)](https://pkg.go.dev/github.com/cloud-barista/cb-spider@master)&nbsp;&nbsp;&nbsp;
[![Release Version](https://img.shields.io/github/v/release/cloud-barista/cb-spider)](https://github.com/cloud-barista/cb-spider/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)

# M-CMP IAM Manager

This repository provides a Multi-Cloud IAM Management Framework.

A sub-system of [M-CMP platform](https://github.com/m-cmp/docs/tree/main) to deploy and manage Multi-Cloud Infrastructures.

## Overview

- 

## How to Use

- 

## How to Contribute

- Issues/Discussions/Ideas: Utilize issue of mc-iam-manager

## How to Install

### Environment

- go : go1.21.0 >
    
    ```bash
    $ go version
    # go version go1.21.0 linux/amd64
    ```
    
    - buffalo framework : v0.18.8 >
        
        ```bash
        $ buffalo version
        # INFO[0000] Buffalo version is: v0.18.8
        ```
        
        - install buffalo
            
            [Buffalo – Rapid Web Development in Go](https://gobuffalo.io/documentation/getting_started/installation/)
            
- keycloak : 22.0.3
    
    [downloads - Keycloak](https://www.keycloak.org/downloads)
    
    - SP (2023.10 AWS, ALI) setting
        - csp SAML idp reg, csp assumeRole setting require
            
            [IAM을 사용하여 IdP 페더레이션 설정 및 QuickSight - 아마존 QuickSight](https://docs.aws.amazon.com/ko_kr/quicksight/latest/user/external-identity-providers-setting-up-saml.html)
            
    - keycloak client setting require
        
        https://www.keycloak.org/guides#server
        
    
    ```
    # keycloak-22.0.3/conf/keycloak.conf
    
    # Basic settings for running in production. Change accordingly before deploying the server.
    
    # Database
    
    # The database vendor.
    db=postgres
    
    # The username of the database user.
    db-username={DB user}
    
    # The password of the database user.
    db-password={DB user password}
    
    # The full database JDBC URL. If not provided, a default URL is set based on the selected database vendor.
    db-url=jdbc:postgresql://{DB host}/{DB name}
    
    # Observability
    
    # If the server should expose healthcheck endpoints.
    #health-enabled=true
    
    # If the server should expose metrics endpoints.
    #metrics-enabled=true
    
    # HTTP
    
    # The file path to a server certificate or certificate chain in PEM format.
    https-certificate-file=${kc.home.dir}conf/server.crt.pem
    # The file path to a private key in PEM format.
    https-certificate-key-file=${kc.home.dir}conf/server.key.pem
    
    # The proxy address forwarding mode if the server is behind a reverse proxy.
    #proxy=reencrypt
    
    # Do not attach route to cookies and rely on the session affinity capabilities from reverse proxy
    #spi-sticky-session-encoder-infinispan-should-attach-route=false
    
    # Hostname for the Keycloak server.
    #hostname=myhostname
    ```
    
- etc
    
    ```bash
    $ node -v
    #v20.5.1
    $ npm -v
    #9.8.0
    $ yarn -v
    #3.6.3
    ```
    

### Get Sourcecode

- clone this repository
    
    ```bash
    git clone https://github.com/m-cmp/mc-iam-manager
    ```
    

### Set mc-iam-manager ‘.env’ and ‘database.yml’

- You can write it by referring to the files in the repository.
    
    ```
    # mc-iam-manager/.env
    
    ## NETWORK
    # It doesn't matter if you use it as it is.
    ADDR=0.0.0.0 
    PORT=3000
    
    ## Keycloak Admin and Location
    # If you plan to control the keyclock,
    # enter your admin keyclock account and location, client info.
    KC_admin={Keycloak Admin ID}
    KC_passwd={Keycloak Admin Password}
    KC_uri=https://{Keycloak home url} # SSL
    # OIDC buffalo client info
    KC_realm={buffalo client Realm Name}
    KC_clientID={buffalo client ID}
    KC_clientSecret={buffalo client ID}
    
    ## SAML SP Endpoint
    SAML_IDP_Initiated_URL_AWS="https://{Keycloak home url}/realms/{realms Name}/protocol/saml/clients/{client Prefix}"
    SAML_IDP_Initiated_URL_ALI="https://{Keycloak home url}/realms/{realms Name}/protocol/saml/clients/{client Prefix}"
    SAML_user={Test SAML user ID}
    SAML_password={Test SAML user Password}
    ```
    
    ```
    # mc-iam-manager/database.yml
    # ONLY for $ buffalo dev
    
    ---
    development:
      dialect: postgres
      database: {DB name}
      user: {DB user name}
      password: {DB user password}
      host: {DB host}
      pool: 5
    
    test:
      url: {{envOr "TEST_DATABASE_URL" "postgres://postgres:postgres@127.0.0.1:5432/myapp_test"}}
    
    production:
      url: {{envOr "DATABASE_URL" "postgres://postgres:postgres@127.0.0.1:5432/myapp_production"}}
    ```
    

### Run

- run Keycloak
    
    ```
    # at the keycloak bin folder
    $ ./kc.sh start-dev
    ```
    
- run buffalo
  
    ```
    # at the this repo clone folder
    $ cd mc-iam-manager
    $ buffalo dev
    ```


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)

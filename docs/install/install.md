---
layout: default
title: Build and Start
parent: How to install
order: 2
---

# Build and Start Guide

This guide explains how to build and start the MC-IAM-MANAGER. Check the prerequisites.

## Prequisites

### Environment
- ubuntu (`22.04` is tested )
- golang (`v1.22` is tested )

### Dependency
- [dependency](https://github.com/m-cmp/mc-iam-manager/network/dependencies) 
- golang buffalo (`v0.18.14` is tested) [install docs](https://gobuffalo.io/documentation/getting_started/installation/)
- keycloak (`25.0.0` is tested) [install docs](https://www.keycloak.org/guides)
    - you can use realm import setting from `scripts/realm-import.json`
- database (PostgreSQL) [link](https://www.postgresql.org/)

{: .highlight }
keycloak should be run external mode to use CSP services.


## Step one : Clone this repo
```bash
git clone https://github.com/m-cmp/mc-iam-manager <YourFolderName>
```

## Step two : Go to repo Folder
```bash
cd <YourFolderName>
```

## Step three : Fill up .env file
```bash
cp ./.env.sample ./.env
nano ./.env
```

```bash 
ADDR=0.0.0.0 # your mc-iam-manager Address( local:127.0.0.1 / external: 0.0.0.0 )
PORT=4000 # your mc-iam-manager Port ( docker:5000 / standalone:4000 )

DATABASE_USER=db_user # your DB user
DATABASE_PASS=db_password # your DB password
DATABASE_HOST=db_host # your DB host
DATABASE=db # your DB
DEV_DATABASE_URL=postgres://${DATABASE_USER}:${DATABASE_PASS}@${DATABASE_HOST}:5432/${DATABASE} # you don't have to change this line.
DATABASE_URL=postgres://${DATABASE_USER}:${DATABASE_PASS}@${DATABASE_HOST}:5432/${DATABASE} # you don't have to change this line.

KEYCLOAK_HOST=https://example.com # keycloak Host ( https is recommended )
KEYCLAOK_REALM=mciam # keycloak Realm 
KEYCLAOK_CLIENT=mciam # keycloak Client
KEYCLAOK_CLIENT_SECRET=mciamclientsecret # keycloak CLIENT secret
KEYCLAOK_ADMIN=admin # if you use only exist account, don't have to change this
KEYCLAOK_ADMIN_PASSWORD=admin # if you use only exist account, don't have to change this

MCINFRAMANAGER=http://example.com:1323/tumblebug # mc-infra-manager host
MCINFRAMANAGER_APIUSERNAME=default # mc-infra-manager api user name ( default is "default" )
MCINFRAMANAGER_APIPASSWORD=default # mc-infra-manager api user password ( default is "default" )
```

## Step four : build your app
```
cd <YourFolderName>
buffalo build --static -o <appPath>/mc-iam-manager
```

## Step five : Copy .env to bin file path
```
cp <YourFolderName>/.env <appPath>/.env
```

## Step six : Deploy app
```
cd <appPath>
./mc-iam-manager
```
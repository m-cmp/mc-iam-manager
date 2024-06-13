---
layout: default
title: Quick Start with docker
parent: How to install
---

## Quick Start with docker

Use this guide to start MC-IAM-MANAGER using the docker. The Quick Start guide sets the default Admin, Operator, Viewer account, and environment.

### Prequisites

- Ubuntu (22.04 is tested) with external access (https-443, http-80, ssh-ANY)
- docker and docker-compose
- Domain (for Keycloak and Public buffalo) and Email for register SSL with certbot
- Stop or Disable Services using 80 or 443 ports such as nginx

### Step one : Clone this repo

```bash
git clone https://github.com/m-cmp/mc-iam-manager <YourFolderName>
```

### Step two : Go to Scripts Folder

```bash
cd <YourFolderName>/scripts
```

### Step three : Excute generate_nginx_conf.sh

```bash
./generate_nginx_conf.sh

# >.env (DOMAIN): yourdomain.com
# >.env (EMAIL): yourEmail@test.com

================================================
 * DOMAIN = yourdomain.com
 * EMAIL = yourEmail@test.com
================================================

** Nginx configuration file has been created at ./nginx/nginx.conf **
```

This process creates two versions of nginx.conf:

the first (nginx-cert.conf) to receive SSL certificates and the second (nginx.conf) to set up an internal proxy for mc-iam-manager and keycloak, and certbot, as well as an SSL reverse proxy

### Step four : Excute init docker-compose for SSL setup

```bash
docker-compose -f docker-compose.init.yml up
# check the log "Successfully received certificate." and "ertbot exited with code 0"
# ctrl + C to exit docker-compose and shutdown with below command
docker-compose -f docker-compose.init.yml down
```

This process creates a SSL certificate in the `~/.m-cmp/data/certbot` path through the nginx-cert.conf setting.  ****If you have checked the console log (Successfully received certificate. ~~ certbot exited with code 0) as below, you have successfully issued an SSL certificate and created it at the designated location.

```bash
$ docker-compose -f docker-compose.init.yml up
....
certbot    | Successfully received certificate.
certbot    | Certificate is saved at: /etc/letsencrypt/live/yourdomain.com/fullchain.pem
certbot    | Key is saved at:         /etc/letsencrypt/live/yourdomain.com/privkey.pem
certbot    | This certificate expires on 2024-09-11.
certbot    | These files will be updated when the certificate renews.
certbot    | NEXT STEPS:
certbot    | - The certificate will need to be renewed before it expires. Certbot can automatically renew the certificate in the background, but you may need to take steps to enable that functionality. See https://certbot.org/renewal-setup for instructions.
certbot    | 
certbot    | - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
certbot    | If you like Certbot, please consider supporting our work by:
certbot    |  * Donating to ISRG / Let's Encrypt:   https://letsencrypt.org/donate
certbot    |  * Donating to EFF:                    https://eff.org/donate-le
certbot    | - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
certbot exited with code 0
```

And you don't have to consider the renewal. The next docker-compose checks the certificate every 12 hours and automatically updates it to the symbol link if it needs to be renewed. In other words, this is only the first time you need it, and it doesn't need to be applied from the next update.

### Step five : Excute docker-compose

```bash
docker-compose up --build -d
```

If you check the log as below, it seems that you have successfully built and deployed the mc-iam-manager without any problems.

```bash
$ docker-compose up --build -d

Creating network "scripts_mciammanagernet" with the default driver
Building mciammanager
Step 1/19 : FROM gobuffalo/buffalo:v0.18.14 as builder
 ---> dbcc9d3a40f5
Step 2/19 : ENV GOPROXY http://proxy.golang.org
 ---> Using cache
 ---> 05e55ac7f5eb
 ....
 Step 10/19 : RUN buffalo build --static -o /bin/app
 ---> Running in 3c1d37d71384
 ....
Successfully built 7d0ed2aa6a89
Successfully tagged scripts_mciammanager:latest
Creating scripts_postgresdb_1 ... done
Creating certbot              ... done
Creating scripts_keycloak_1   ... done
Creating scripts_mciammanager_1 ... done
Creating nginx                  ... done
```

### Step six : Check Alive enpoint

```bash
$ curl https://<yourdomain.com>:5000/alive
# {"ststus":"ok"}
```

If `{"stststus":"ok"}` is received from the endpoint, it means that the service is being deployed normally.

### WELCOME : Now you can use MC-IAM-MANAGER

You can get tokens issued and see the default Role created through some of the built-in accounts below. For more API information, check the following swagger link.

```bash
$ curl --location 'https://yourdomain.com:5000/api/auth/login' \
--header 'Content-Type: application/json' \
--data '{
    "id":"mcpsuper",
    "password":"mcpuserpassword"
}'

$ curl --location 'https://yourdomain.com:5000/api/auth/login' \
--header 'Content-Type: application/json' \
--data '{
    "id":"mcpadmin",
    "password":"mcpuserpassword"
}'

$ curl --location 'https://yourdomain.com:5000/api/auth/login' \
--header 'Content-Type: application/json' \
--data '{
    "id":"mcpoperator",
    "password":"mcpuserpassword"
}'

$ curl --location 'https://yourdomain.com:5000/api/auth/login' \
--header 'Content-Type: application/json' \
--data '{
    "id":"mcpviewer",
    "password":"mcpuserpassword"
}'

200 OK application/json
{
    "access_token": "xxxxx", # Rolelist in token (claims : realmRole[])
    "id_token": "xxxxx",
    "expires_in": 36000,
    "refresh_expires_in": 1800,
    "refresh_token": "xxxxx",
    "token_type": "Bearer",
    "not-before-policy": 0,
    "session_state": "xxxxx",
    "scope": "openid microprofile-jwt profile email"
}
```

### swagger docs
https://m-cmp.github.io/mc-iam-manager/

 ```
 # https://m-cmp.github.io/mc-iam-manager/
 ```

### Get CB-Tumblebug namespace Data

You can run the following script to assign the configured existing data to the Default Workplace.

```bash
$ cd <yourfolder>/scripts/init
$ nano ./init.env
# TB_HOST=<tumblegub host>
# TB_username=<TB_username>
# TB_password=<TB_password>
#
# MCIAM_HOST=<https://yourdomain.com:5000>

$ ./init-default-workspace-project.sh
```

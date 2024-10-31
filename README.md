[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_shield)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/m-cmp/mc-iam-manager?label=go.mod)](https://github.com/m-cmp/mc-iam-manager/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/m-cmp/mc-iam-manager?status.svg)](https://pkg.go.dev/github.com/m-cmp/mc-iam-manager@master)
[![Release Version](https://img.shields.io/github/v/release/m-cmp/mc-iam-manager)](https://github.com/m-cmp/mc-iam-manager/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/m-cmp/mc-iam-manager/blob/master/LICENSE)

[M-CMP IAM Manager docs](https://m-cmp.github.io/mc-iam-manager/)

# M-CMP IAM Manager

This repository provides a Multi-Cloud IAM Management Framework.

A sub-system of [M-CMP platform](https://github.com/m-cmp/docs/tree/main) to deploy and manage Multi-Cloud Infrastructures.

## Overview

The Multi-Cloud Authorization and Access Control Framework provides platform account/role management, integrated management of cloud account/access control information, and workspace management functionalities. It offers features compatible with security policy determination, establishment, and enforcement for existing multi-cloud services. Additionally, it provides the capability to establish and manage independent security policies within the framework.
It defines an access control reference model for multi-cloud, distinguishing between user access control and service provider access control. This model adopts a prominent Role-Based Access Control (RBAC) approach and integrates it with existing policy management solutions for application and utilization.

## Quick Start with docker

Use this guide to start MC-IAM-MANAGER using the docker. The Quick Start guide sets the default Admin, Operator, Viewer account, and environment.

### Prequisites

- Ubuntu (22.04 is tested) with external access (https-443, http-80, ssh-ANY)
- docker and docker-compose
- Domain (for Keycloak and Public buffalo) and Email for register SSL with certbot

### Step one : Clone this repo

```bash
git clone <https://github.com/m-cmp/mc-iam-manager> <YourFolderName>
```

### Step two : Go to Scripts Folder

```bash
cd <YourFolderName>/scripts
```

### Step three : Excute keycloakimportsetting.sh

```bash
./keycloakimportsetting.sh

## MC-IAM-MANAGER Init Setup ##
 - Please enter the changes. If not, use the environment variable.
 - You can set Values in ./.mciammanager_init_env

COMPANY_NAME  : 
...
```

This step defines the environment variables that you want to use by default or creates `./scripts/container-volume/mc-iam-manager/keycloak/data/import/realm-import.json` based on the variables defined in `./scripts/.env`. Therefore, "Keycloak" completes the initial setup based on the file, creating the first login user in the process.

### Step four: Excute docker-compose

```bash
cd scripts
sudo docker-compose up --build -d
```

This step is time consuming. Don't worry if the console fails. "Keycloak" is a natural error that occurs during initial installation when MC-IAM-MANAGER requests Keyclaok readiness and certification to initialize the database and import the required data.

Once the server completes successfully, you can access the readyz endpoint with the message that it has been loaded successfully.

### Step final: Check Readyzenpoint

```bash
$ curl https://<yourdomain.com>:5000/readyz
# {"ststus":"ok"}
```

If `{"stststus":"ok"}` is received from the endpoint, it means that the service is being deployed normally.

### WELCOME : Now you can use MC-IAM-MANAGER

To use MC-IAM-MANAGER, you need to register the resources of the framework to be used as the first registered user.

For example, MC-WEB-CONSOLE must register a menu so that the user can load the web screen normally.

This section describes how to use scripts that made the process simple.

- init.sh
    
    ```bash
    # ./scripts/init/init.sh
    ./init.sh
    
    --------------------
    0. exit
    
    1. login
    
    2. Init Resource Data from api.yaml
      (MCADMINCLI_APIYAML: https://raw.githubusercontent.com/m-cmp/mc-admin-cli/refs/heads/main/conf/api.yaml)
    
    3. Init Menu Data from menu.yaml
      (MCWEBCONSOLE_MENUYAML: https://raw.githubusercontent.com/m-cmp/mc-web-console/refs/heads/main/conf/webconsole_menu_resources.yaml)
    
    4. Init Role Data PREDEFINED_ROLE
      (PREDEFINED_ROLE: admin,operator,viewer,billadmin,billviewer)
    
    5. Get permission CSV
    
    6. Update permission CSV 
      (./permission.csv)
    
    99. auto init
    
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



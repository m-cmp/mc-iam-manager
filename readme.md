[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_shield)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/m-cmp/mc-iam-manager?label=go.mod)](https://github.com/m-cmp/mc-iam-manager/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/m-cmp/mc-iam-manager?status.svg)](https://pkg.go.dev/github.com/m-cmp/mc-iam-manager@master)
[![Release Version](https://img.shields.io/github/v/release/m-cmp/mc-iam-manager)](https://github.com/m-cmp/mc-iam-manager/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/m-cmp/mc-iam-manager/blob/master/LICENSE)

[M-CMP IAM Manager docs](https://m-cmp.github.io/mc-iam-manager/)

# M-CMP IAM Manager

This repository provides a multi-cloud IAM management framework as a subsystem of the [M-CMP platform](https://github.com/m-cmp/docs/tree/main) for deploying and managing multi-cloud infrastructure.

## Overview

The multi-cloud authorization and access control framework provides platform account/role management, integrated management of cloud account/access control information, and workspace management functionality. It offers features compatible with security policy decision-making, establishment, and enforcement for existing multi-cloud services. Additionally, it provides the capability to establish and manage independent security policies within the framework. This defines a multi-cloud access control reference model, distinguishing between user access control and service provider access control. The model adopts a key role-based access control (RBAC) approach and integrates it with existing policy management solutions for application and utilization.

## Quick Start with Docker

This guide provides instructions for starting MC-IAM-MANAGER using Docker. The quick start guide sets up basic administrator, operator, viewer accounts and environment.

### Prerequisites
- Ubuntu with external access capability (tested on 22.04) (https-443, http-80, ssh-ANY)
- Docker and docker-compose
- Domain
- Email for SSL registration
- HTTPS setup: Refer to separate documentation for nginx + keycloak + certbot configuration

### Step 1: Clone Source

```bash
git clone https://github.com/m-cmp/mc-iam-manager <YourFolderName>
```

### Step 2: Environment Configuration
Update the .env file with configuration values

```bash
cp .env_sample .env
```

### Step 3: MC-IAM-MANAGER Init Setup
Build and verify operation by calling /readyz

```bash
docker compose up --build -d
```

#### Docker execution with docker-compose
Configure iam-manager settings when HTTPS is configured and keycloak and postgres connections are available.

```bash
sudo docker-compose up --build -d
```

#### Source execution
```bash
go run main.go 
```

#### Operation verification
```bash
curl https://<your domain or localhost>:<port>/readyz
```

### Step 4: Environment Setup
- Keycloak administrator creation and configuration
  * Note: When running keycloak separately, ensure administrator information matches the information in .env
- Create realm, client, and basic roles in keycloak as keycloak administrator (use values defined in .env)
  (These tasks can be performed by accessing the keycloak console)

Execute `/asset/setup/0setup.sh` and work through the steps in order. Note: Access token issuance must be successful for PlatformAdmin Login in step 1.

1. **Init Platform And PlatformAdmin**
   - Create Realm in Keycloak
   - Create Client in Keycloak
   - Create Predefined Role in Keycloak
   - Register Predefined Role in DB
   - Create default workspace
   - Register menus (basic yaml file)
   - Map menus and default roles
   - Create PlatformAdmin User in Keycloak
   - Register PlatformAdmin User in DB

2. **PlatformAdmin Login**
   - Issue access_token to execute the following scripts

3. **(optional) Init Predefined Role Data**: Completed in step 1. Execute if additional roles are needed

4. **(optional) Init Menu Data**: Completed in step 1. Execute if additional menus are needed

5. **Init API Resource Data**
   - Load menu access permissions (based on /asset/menu/permission.csv)

6. **Init Cloud Resource Data**

7. **Map API-Cloud Resources**

8. **Init CSP Roles**
   - In the current version, IAM-Role and CSP Role creation and temporary credential settings are performed directly in the respective CSP Console

9. **Map Master Role-CSP Roles**

### Step 5: CSP IDP Configuration
- Access CSP console (e.g., AWS console)
  - Add IDP configuration in IAM menu
  - Add IAM role: 1:1 mapping with cspRole with prefix mciam_. Add predefined mc-iam-manager roles and connect them.
    Note: When adding roles in AWS, add as webIdentity role
  - Add permissions to IAM role
    Add permissions that the role can perform (e.g., EC2ReadOnly, etc.)
  - Add keycloak client as audience in IAM role's trust relationship settings

# Welcome: You can now use MC-IAM-MANAGER
The next steps are user addition and role configuration.
- Add users
- Map user-role
- (optional) Share workspace with user

## Usage Examples

1. **PlatformAdmin Login**
   ```bash
   POST /api/auth/login
   {
     "id": "<MCIAMMANAGER_PLATFORMADMIN_ID>",
     "password": "<MCIAMMANAGER_PLATFORMADMIN_PASSWORD>"
   }
   ```

2. **Init Role Data PREDEFINED_ROLE to Platform Role & Realm role**
   (Extract from <PREDEFINED_ROLE> using comma as separator: admin,operator,viewer,billadmin,billviewer)
   ```bash
   POST /api/platform-roles/
   ```

3. **Init Menu Data from menu.yaml**
   (MCWEBCONSOLE_MENUYAML: https://raw.githubusercontent.com/m-cmp/mc-web-console/refs/heads/main/conf/webconsole_menu_resources.yaml)
   ```bash
   POST /api/setup/menus/register-from-yaml
   ```

4. **Init API Resource Data from api.yaml**
   (MCADMINCLI_APIYAML: https://raw.githubusercontent.com/m-cmp/mc-admin-cli/refs/heads/main/conf/api.yaml)
   ```bash
   POST /api/setup/sync-apis
   ```

5. **Init Cloud Resource Data from cloud-resource.yaml**
   Example: mc-infra-manager
   - vm - mci (GetMci, GetVm ...)
     - sshkey
     - securitygroup
     - vpc
   - nlb - nlb
   - k8s - pmk
   - bill - cost
   - common - ns

6. **Mapping API-Cloud Resources**
   Link basic permissions with target APIs
   - read: Access to all query-related APIs
   - write: Access to all modification-related APIs
   - manage: Access to all APIs

7. **Register Workspace Role**
   (PREDEFINED_ROLE: admin,operator,viewer,billadmin,billviewer)
   ```bash
   POST /api/workspace-roles/
   ```

8. **Link Workspace Role - CSP IAM Role**
   Workspace role is created in CSP as mcmp_<workspace-role> format: PlatformAdmin initial creation

## API Documentation

Swagger documentation is available at:
https://m-cmp.github.io/mc-iam-manager/

## Project Structure

```
src/
├── main.go              # Application entry point
├── config/              # Configuration management
├── handler/             # HTTP request handlers
├── middleware/          # HTTP middleware
├── model/               # Data models
├── repository/          # Data access layer
├── service/             # Business logic
├── util/                # Utility functions
├── csp/                 # Cloud Service Provider integrations
├── mcmpapi/             # M-CMP API definitions
└── docs/                # Generated documentation
```

## Technology Stack

- **Language**: Go 1.23.1
- **Framework**: Echo v4 (HTTP framework)
- **Database**: PostgreSQL with GORM
- **Authentication**: Keycloak integration
- **Cloud Providers**: AWS, Google Cloud Platform
- **Documentation**: Swagger/OpenAPI

## Docker Compose Options

The project provides several Docker Compose configurations:

- `docker-compose.yaml` - Basic standalone setup
- `docker-compose.standalone.yaml` - Standalone configuration
- `docker-compose.with-db.yaml` - With database
- `docker-compose.with-keycloak.yaml` - With Keycloak
- `docker-compose.full.yaml` - Complete setup
- `docker-compose.all.yaml` - All services

## How to Contribute

- Issues/Discussions/Ideas: Utilize issues of mc-iam-manager
- Fork the repository and submit pull requests
- Follow the existing code style and conventions

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

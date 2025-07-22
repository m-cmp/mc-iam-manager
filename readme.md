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
- Docker (24+) and docker-compose (v2)
- Domain name (e.g., megazone.com)
- Email address for SSL certificate registration
- HTTPS setup: Refer to separate documentation for nginx + keycloak + certbot configuration
- Database: PostgreSQL, etc.

### Step 1: Clone Source

```bash
git clone https://github.com/m-cmp/mc-iam-manager <YourFolderName>
```

### Step 2: Environment Configuration
Create .env file by referencing .env_sample and reflect configuration values in the file

```bash
cp .env_sample .env
```

#### Edit Environment Variables
```bash
nano .env
```

Key configuration items:
- `DOMAIN_NAME`: Domain name (e.g., mciam.megazone.com)
- `EMAIL`: Email for SSL certificate issuance
- `MCIAMMANAGER_PORT`: Application port (default: 3000)
- `KEYCLOAK_ADMIN`: Keycloak administrator account
- `KEYCLOAK_ADMIN_PASSWORD`: Keycloak administrator password

#### SSL Certificate Issuance (if needed)
```bash
# SSL Certificate Issuance
sudo docker compose -f docker-compose.cert.yaml up
```

**Certificate Renewal**: Let's Encrypt certificates need renewal every 90 days.
```bash
# Manual renewal
sudo docker compose -f docker-compose.cert.yaml run --rm mcmp-certbot renew

# Automatic renewal setup (cron)
0 12 * * * /usr/bin/docker compose -f /path/to/docker-compose.cert.yaml run --rm mcmp-certbot renew
```

Successful certificate issuance will display a message like:
```
mcmp-certbot  | Requesting a certificate for [domain name]
mcmp-certbot  | Successfully received certificate.
mcmp-certbot  | Certificate is saved at: /etc/letsencrypt/live/[domain name]/fullchain.pem
mcmp-certbot  | Key is saved at: /etc/letsencrypt/live/[domain name]/privkey.pem
mcmp-certbot  | This certificate expires on 2025-10-20.
```

#### Generate Nginx Configuration
Generate Nginx configuration file based on environment variables.
```bash
# Execute Nginx configuration script
./asset/setup/0_preset_create_nginx_conf.sh
```

** keycloak address : /auth **
  ```bash
   location /auth/ {
            proxy_pass http://mciam-keycloak:8080/auth/;
  ```

Generated file: `dockerfiles/nginx/nginx.conf`

### Step 3: MC-IAM-MANAGER Init Setup
Build and verify operation by calling /readyz

```bash
# Deploy entire system (mc-iam-manager + nginx + postgres + keycloak)
sudo docker compose -f docker-compose.all.yaml up -d

# Deploy MC-IAM-MANAGER only
sudo docker compose -f docker-compose.standalone.yaml up -d
```

```bash
# Run MC-IAM-MANAGER from source (when nginx + postgres + keycloak are already running)
cd ./src
go run main.go
```

#### Operation Verification
```bash
curl https://<your domain or localhost>:<port>/readyz
```

#### Service Configuration
- **Nginx**: Reverse proxy, SSL termination, static file serving
- **IAM Manager**: Main application (Echo Framework)
- **Keycloak**: Authentication and authorization management
- **PostgreSQL**: Database
- **Certbot**: Automatic SSL certificate issuance/renewal

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

#### Log Monitoring
```bash
# Check specific service logs
sudo docker compose -f docker-compose.all.yaml logs [service-name]

# Real-time log monitoring
sudo docker compose -f docker-compose.all.yaml logs -f [service-name]
```

#### Backup
```bash
# PostgreSQL data backup
sudo docker exec mciam-postgres pg_dump -U iammanager iammanagerdb > backup.sql

# Keycloak data backup
sudo tar -czf keycloak-backup.tar.gz dockercontainer-volume/keycloak/
```

#### Update
```bash
# Update images
sudo docker compose -f docker-compose.all.yaml pull
sudo docker compose -f docker-compose.all.yaml up -d
```

#### API Documentation
Generate swagger documentation from src folder:
```bash
swag init -g src/main.go -o src/docs
```

### Step 4: Environment Setup
**Configure platform based on information in environment configuration file (.env)**

Execute `/asset/setup/1setup.sh` and work through the steps in order.

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

## How to Contribute

- Issues/Discussions/Ideas: Utilize issues of mc-iam-manager

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)

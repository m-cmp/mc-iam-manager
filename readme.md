[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_shield)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/m-cmp/mc-iam-manager?label=go.mod)](https://github.com/m-cmp/mc-iam-manager/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/m-cmp/mc-iam-manager?status.svg)](https://pkg.go.dev/github.com/m-cmp/mc-iam-manager@master)
[![Release Version](https://img.shields.io/github/v/release/m-cmp/mc-iam-manager)](https://github.com/m-cmp/mc-iam-manager/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/m-cmp/mc-iam-manager/blob/master/LICENSE)

[M-CMP IAM Manager docs](https://m-cmp.github.io/mc-iam-manager/)

# M-CMP IAM Manager

This repository provides a multi-cloud IAM management framework as a subsystem of the [M-CMP platform](https://github.com/m-cmp/docs/tree/main) for deploying and managing multi-cloud infrastructure.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [System Architecture](#system-architecture)
- [Quick Start](#quick-start)
- [Installation and Configuration](#installation-and-configuration)
- [API Documentation](#api-documentation)
- [Contributing](#contributing)
- [License](#license)

## Overview

M-CMP IAM Manager provides an integrated authorization and access control framework for multi-cloud environments. It offers platform account/role management, integrated management of cloud account/access control information, and workspace management functionality to support security policy decision-making, establishment, and enforcement for existing multi-cloud services.

### Key Characteristics

- **Multi-cloud Support**: Integrated management of various CSPs including AWS, Azure, GCP
- **RBAC-based Access Control**: Role-based granular permission management
- **Centralized Management**: Single platform control for all cloud resource access
- **Temporary Credentials**: JWT-based secure temporary access token issuance

## Key Features

### 🏢 **Enterprise Multi-cloud Environment Management**
- **Multi-CSP Integration**: Unified management of IAM across multiple cloud service providers like AWS, Azure, GCP
- **Centralized Permission Control**: Manage access permissions for all cloud resources from a single platform
- **RBAC (Role-based Access Control)**: Granular permission management based on user roles
- **Temporary Credentials**: JWT-based secure temporary access token issuance

## System Architecture

```
Internet
    |
    v
[Nginx Reverse Proxy] (Port 80/443)
    |
    +---> [IAM Manager] (Port 5000)
    |
    +---> [Keycloak] (Port 8080)
    |
    +---> [PostgreSQL] (Port 5432)
```

### Components

- **Nginx**: Reverse proxy, SSL termination, static file serving
- **IAM Manager**: Main application (Echo Framework)
- **Keycloak**: Authentication and authorization management
- **PostgreSQL**: Database
- **Certbot**: Automatic SSL certificate issuance/renewal

## Quick Start
  [mc-admin-cli](https://github.com/m-cmp/mc-admin-cli/blob/main/README.md) contains mc-iam-manager.

### Prerequisites

- **Operating System**: Ubuntu 22.04 (tested)
- **Network**: External access capability (HTTPS-443, HTTP-80, SSH-ANY)
- **Docker**: Docker 24+ and Docker Compose v2
- **Database**: PostgreSQL
- **Domain**: Domain for SSL certificate issuance (production environment)
- **Email**: Email address for SSL certificate issuance

### Installation Steps

#### Step 1: Clone Source

```bash
git clone https://github.com/m-cmp/mc-iam-manager <YourFolderName>
cd <YourFolderName>
```

#### Step 2: Environment Configuration

```bash
# Copy environment configuration file
cp .env_sample .env

# Edit environment variables
nano .env
```

**Key Configuration Items:**
- `DOMAIN_NAME`: Domain name (e.g., mciam.m-cmp.org)
- `EMAIL`: Email for SSL certificate issuance
- `MCIAMMANAGER_PORT`: Application port (default: 5000)
- `KEYCLOAK_ADMIN`: Keycloak administrator account
- `KEYCLOAK_ADMIN_PASSWORD`: Keycloak administrator password

#### Step 3: Certificate Configuration

**Development Environment (Self-signed Certificate):**
- [Self-signed Certificate Issuance Guide](https://github.com/m-cmp/mc-iam-manager/wiki/%EC%9E%90%EC%B2%B4-%EC%9D%B8%EC%A6%9D%EC%84%9C-%EB%B0%9C%EA%B8%89)

**Production Environment (CA Certificate):**
- [CA Certificate Issuance Guide](https://github.com/m-cmp/mc-iam-manager/wiki/CA-%EC%9D%B8%EC%A6%9D%EC%84%9C-%EB%B0%9C%EA%B8%89)

#### Step 4: System Deployment

**Full System Deployment (Recommended):**
```bash
sudo docker compose -f docker-compose.all.yaml up -d
```

**Standalone Mode (Using Existing Infrastructure):**
```bash
sudo docker compose -f docker-compose.standalone.yaml up -d
```

**Direct Source Code Execution:**
```bash
cd ./src
go run main.go
```

### Docker Deployment with Local Build

The `mc-iam-manager` service is configured to use the local `Dockerfile.mciammanager` for building the container image.

#### Build Configuration

In `docker-compose.yaml`, the service is configured as:

```yaml
mc-iam-manager:
  build:
    context: .
    dockerfile: Dockerfile.mciammanager
  image: cloudbaristaorg/mc-iam-manager:edge
```

#### Deployment Options

**1. Build and Run mc-iam-manager:**
```bash
# Build from local Dockerfile and start
docker-compose up --build mc-iam-manager

# Run in background
docker-compose up --build -d mc-iam-manager
```

**2. Run All Services:**
```bash
# Build and start all services
docker-compose up --build -d
```

**3. Rebuild from Scratch:**
```bash
# Force rebuild without cache
docker-compose build --no-cache mc-iam-manager
docker-compose up -d mc-iam-manager
```

**4. Run with Dependencies Only:**
```bash
# Start mc-iam-manager with required services
docker-compose up -d mc-iam-manager-db mc-iam-manager-kc mc-iam-manager
```

#### Service Dependencies

The `mc-iam-manager` service requires:
- `mc-iam-manager-db` (PostgreSQL database)
- `mc-iam-manager-kc` (Keycloak for authentication)

These dependencies are automatically started when you run `mc-iam-manager`.

#### Image Management

```bash
# Pull latest images (if using pre-built images)
docker-compose pull

# List Docker images
docker images | grep mc-iam-manager

# Remove old images
docker rmi cloudbaristaorg/mc-iam-manager:edge
```

#### Step 5: Operation Verification

```bash
curl https://<your domain or localhost>:<port>/readyz
```

## Installation and Configuration

### Initial Setup

#### 1. Authentication Configuration

**Production Environment (Domain and CA Certificate):**
```bash
./asset/setup/0_preset_prod.sh
```

**Development Environment (localhost and Self-signed Certificate):**
```bash
./asset/setup/0_preset_dev.sh
```

#### 2. Basic Configuration

**Automatic Setup (Recommended):**
```bash
./asset/setup/1_setup_auto.sh
```

**Manual Setup:**
```bash
./asset/setup/1_setup_manual.sh
```

### Configuration Steps

1. **Platform and Administrator Initialization**
   - Create Keycloak Realm
   - Create Keycloak Client
   - Create and register default roles
   - Create default workspace
   - Register menus and role mapping
   - Create platform administrator user

2. **API Resource Configuration**
   - Initialize API resource data
   - Configure cloud resource data
   - Map API-cloud resources

3. **CSP Role Configuration**
   - Initialize CSP roles
   - Map master roles-CSP roles

### CSP IDP Configuration (Production Environment)

1. **CSP Console Configuration**
   - Add IDP configuration in IAM menu
   - Add IAM roles (prefix: `mciam_`)
   - Configure role permissions
   - Configure Trust Relation settings

2. **MC-IAM-Manager Configuration**
   - Add CSP roles
   - Configure role mapping

## Operations Management

### Log Monitoring

```bash
# Check specific service logs
sudo docker compose logs [service-name]

# Real-time log monitoring
sudo docker compose logs -f [service-name]
```

### Backup

```bash
# PostgreSQL data backup
sudo docker exec <mc-iam-manager-db service name> pg_dump -U <db user> <db name> > backup.sql

# Keycloak data backup
sudo tar -czf keycloak-backup.tar.gz container-volume/keycloak/
```

### Update

```bash
# Update images
sudo docker compose -f docker-compose.yaml pull
sudo docker compose -f docker-compose.yaml up -d
```

## API Documentation

### Generate Swagger Documentation

```bash
cd ./src
swag init -g src/main.go -o src/docs
```

### Access API Documentation

- **Online Documentation**: https://m-cmp.github.io/mc-iam-manager/
- **Local Documentation**: `http://localhost:<port>/swagger/index.html`

## User Management

### Basic User Addition

1. **Platform Administrator Login**
   ```bash
   POST /api/auth/login
   {
     "id": "<MCIAMMANAGER_PLATFORMADMIN_ID>",
     "password": "<MCIAMMANAGER_PLATFORMADMIN_PASSWORD>"
   }
   ```

2. **Add Users**
   - Create user accounts
   - Map users to roles
   - Share workspaces (optional)

### Role Management

**Default Roles:**
- `admin`: Administrator permissions
- `operator`: Operator permissions
- `viewer`: View permissions
- `billadmin`: Cost management permissions
- `billviewer`: Cost viewing permissions

## Contributing

- **Report Issues**: [GitHub Issues](https://github.com/m-cmp/mc-iam-manager/issues)
- **Discussions**: [GitHub Discussions](https://github.com/m-cmp/mc-iam-manager/discussions)
- **Suggest Ideas**: [GitHub Issues](https://github.com/m-cmp/mc-iam-manager/issues)

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fm-cmp%2Fmc-iam-manager?ref=badge_large)

This project is distributed under the Apache 2.0 License.

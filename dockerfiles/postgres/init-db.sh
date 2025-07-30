#!/bin/bash

# PostgreSQL 초기화 스크립트
echo "Creating databases for MC IAM Manager and Keycloak..."

# 환경변수에서 값 가져오기 (기본값 설정)
MC_IAM_MANAGER_DATABASE_USER=${MC_IAM_MANAGER_DATABASE_USER:-mciamdbadmin}
MC_IAM_MANAGER_DATABASE_PASSWORD=${MC_IAM_MANAGER_DATABASE_PASSWORD:-mciamdbpassword}
MC_IAM_MANAGER_DATABASE_NAME=${MC_IAM_MANAGER_DATABASE_NAME:-mc_iam_manager_db}
MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME=${MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME:-mc_iam_keycloak_db}
MC_IAM_MANAGER_DATABASE_RECREATE=${MC_IAM_MANAGER_DATABASE_RECREATE:-false}

echo "Using environment variables:"
echo "  MC_IAM_MANAGER_DATABASE_USER: $MC_IAM_MANAGER_DATABASE_USER"
echo "  MC_IAM_MANAGER_DATABASE_PASSWORD: $MC_IAM_MANAGER_DATABASE_PASSWORD"
echo "  MC_IAM_MANAGER_DATABASE_NAME: $MC_IAM_MANAGER_DATABASE_NAME"
echo "  MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME: $MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME"
echo "  MC_IAM_MANAGER_DATABASE_RECREATE: $MC_IAM_MANAGER_DATABASE_RECREATE"

# 기존 데이터베이스 확인
if [ "$MC_IAM_MANAGER_DATABASE_RECREATE" = "false" ]; then
    echo "Checking if databases already exist..."
    
    # postgres 데이터베이스에 연결해서 두 데이터베이스 모두 확인
    if psql -U $MC_IAM_MANAGER_DATABASE_USER -d postgres -lqt | cut -d \| -f 1 | grep -qw "$MC_IAM_MANAGER_DATABASE_NAME" && \
       psql -U $MC_IAM_MANAGER_DATABASE_USER -d postgres -lqt | cut -d \| -f 1 | grep -qw "$MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME"; then
        echo "Both databases ($MC_IAM_MANAGER_DATABASE_NAME and $MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME) already exist. Skipping initialization."
        echo "To force reinitialization, set MC_IAM_MANAGER_DATABASE_RECREATE=true in environment variables."
        exit 0
    else
        echo "One or both databases are missing. Proceeding with initialization..."
    fi
fi

echo "Initializing databases..."

# MC IAM Manager 데이터베이스 생성 (이미 존재하면 오류 무시)
echo "Creating MC IAM Manager database: $MC_IAM_MANAGER_DATABASE_NAME"
psql -U $MC_IAM_MANAGER_DATABASE_USER -d postgres -c "CREATE DATABASE \"$MC_IAM_MANAGER_DATABASE_NAME\";" 2>/dev/null || echo "Database $MC_IAM_MANAGER_DATABASE_NAME already exists or creation failed"
psql -U $MC_IAM_MANAGER_DATABASE_USER -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE \"$MC_IAM_MANAGER_DATABASE_NAME\" TO $MC_IAM_MANAGER_DATABASE_USER;" 2>/dev/null || echo "Grant privileges failed for $MC_IAM_MANAGER_DATABASE_NAME"

# Keycloak 데이터베이스 생성 (이미 존재하면 오류 무시)
echo "Creating Keycloak database: $MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME"
psql -U $MC_IAM_MANAGER_DATABASE_USER -d postgres -c "CREATE DATABASE \"$MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME\";" 2>/dev/null || echo "Database $MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME already exists or creation failed"
psql -U $MC_IAM_MANAGER_DATABASE_USER -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE \"$MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME\" TO $MC_IAM_MANAGER_DATABASE_USER;" 2>/dev/null || echo "Grant privileges failed for $MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME"

echo "Databases created successfully!" 

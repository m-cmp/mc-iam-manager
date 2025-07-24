#!/bin/bash

# PostgreSQL 초기화 스크립트
echo "Creating databases for MC IAM Manager and Keycloak..."

# 환경변수에서 값 가져오기 (기본값 설정)
DB_USER=${IAM_DB_USER:-mciamdbadmin}
DB_PASSWORD=${IAM_DB_PASSWORD:-mciamdbpassword}
IAM_DB_NAME=${IAM_DB_DATABASE_NAME:-mc_iam_manager_db}
KEYCLOAK_DB_NAME=${KEYCLOAK_DB_DATABASE_NAME:-mc_iam_keycloak_db}
RECREATE_DB=${IAM_DB_RECREATE:-false}

echo "Using environment variables:"
echo "  DB_USER: $DB_USER"
echo "  IAM_DB_NAME: $IAM_DB_NAME"
echo "  KEYCLOAK_DB_NAME: $KEYCLOAK_DB_NAME"
echo "  IAM_DB_RECREATE: $RECREATE_DB"

# 기존 데이터베이스 확인
if [ "$RECREATE_DB" = "false" ]; then
    echo "Checking if databases already exist..."
    if psql -U $DB_USER -d $IAM_DB_NAME -lqt | cut -d \| -f 1 | grep -qw "$IAM_DB_NAME"; then
        echo "Database $IAM_DB_NAME already exists. Skipping initialization."
        echo "To force reinitialization, set IAM_DB_RECREATE=true in environment variables."
        exit 0
    fi
fi

echo "Initializing databases..."

# 사용자 생성 (이미 존재하면 무시)
psql -U $DB_USER -d $IAM_DB_NAME -c "DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER') THEN
    CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
  END IF;
END
\$\$;"

# MC IAM Manager 데이터베이스는 이미 POSTGRES_DB로 생성됨
# 권한만 확인/설정
psql -U $DB_USER -d $IAM_DB_NAME -c "GRANT ALL PRIVILEGES ON DATABASE $IAM_DB_NAME TO $DB_USER;"

# Keycloak 데이터베이스 생성
psql -U $DB_USER -d $IAM_DB_NAME -c "CREATE DATABASE $KEYCLOAK_DB_NAME;"
psql -U $DB_USER -d $IAM_DB_NAME -c "GRANT ALL PRIVILEGES ON DATABASE $KEYCLOAK_DB_NAME TO $DB_USER;"

echo "Databases created successfully!" 


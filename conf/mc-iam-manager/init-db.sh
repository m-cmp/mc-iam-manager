#!/bin/bash

# PostgreSQL 초기화 스크립트
echo "Creating databases for MC IAM Manager and Keycloak..."

# 환경변수에서 값 가져오기 (기본값 설정)
DB_USER=${MC_IAM_MANAGER_DATABASE_USER:-mciamdbadmin}
DB_PASSWORD=${MC_IAM_MANAGER_DATABASE_PASSWORD:-mciamdbpassword}
IAM_DB_NAME=${MC_IAM_MANAGER_DATABASE_NAME:-mc_iam_manager_db}
KEYCLOAK_DB_NAME=${MC_IAM_MANAGER_KEYCLOAK_DATABASE_NAME:-mc_iam_keycloak_db}
RECREATE_DB=${MC_IAM_MANAGER_DATABASE_RECREATE:-false}

echo "Using environment variables:"
echo "  DB_USER: $DB_USER"
echo "  IAM_DB_NAME: $IAM_DB_NAME"
echo "  KEYCLOAK_DB_NAME: $KEYCLOAK_DB_NAME"
echo "  RECREATE_DB: $RECREATE_DB"

# 사용자 생성 (이미 존재하면 무시)
echo "Creating database user..."
psql -U $DB_USER -d $IAM_DB_NAME -c "DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '$DB_USER') THEN
    CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
  END IF;
END
\$\$;"

# MC IAM Manager 데이터베이스 확인 및 생성
echo "Checking MC IAM Manager database..."
if [ "$RECREATE_DB" = "true" ] || ! psql -U $DB_USER -d $IAM_DB_NAME -lqt | cut -d \| -f 1 | grep -qw "$IAM_DB_NAME"; then
    echo "Creating MC IAM Manager database..."
    psql -U $DB_USER -d $IAM_DB_NAME -c "CREATE DATABASE $IAM_DB_NAME;"
    psql -U $DB_USER -d $IAM_DB_NAME -c "GRANT ALL PRIVILEGES ON DATABASE $IAM_DB_NAME TO $DB_USER;"
    echo "MC IAM Manager database created successfully!"
else
    echo "MC IAM Manager database already exists."
fi

# Keycloak 데이터베이스 확인 및 생성
echo "Checking Keycloak database..."
if [ "$RECREATE_DB" = "true" ] || ! psql -U $DB_USER -d $IAM_DB_NAME -lqt | cut -d \| -f 1 | grep -qw "$KEYCLOAK_DB_NAME"; then
    echo "Creating Keycloak database..."
    psql -U $DB_USER -d $IAM_DB_NAME -c "CREATE DATABASE $KEYCLOAK_DB_NAME;"
    psql -U $DB_USER -d $IAM_DB_NAME -c "GRANT ALL PRIVILEGES ON DATABASE $KEYCLOAK_DB_NAME TO $DB_USER;"
    echo "Keycloak database created successfully!"
else
    echo "Keycloak database already exists."
fi

echo "Database initialization completed!" 

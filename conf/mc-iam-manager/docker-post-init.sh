#!/bin/bash

echo 'All required containers are healthy. Starting initialization...'

# 필요한 도구 설치
apt-get update && apt-get install -y curl jq wget postgresql-client

echo ''
echo ''

echo '------------------------------------------------' 
echo ' Health Check'
echo '------------------------------------------------'


# # mc-iam-manager API가 완전히 준비될 때까지 대기 (간단한 버전)
# echo 'Waiting for mc-iam-manager API to be ready...'
# max_attempts=2
# attempt=1

# while [ $attempt -le $max_attempts ]; do
#   echo "Attempt $attempt/$max_attempts: Checking mc-iam-manager API..."
  
#   # 간단한 API 헬스체크 (컨테이너 이름 사용)
#   if curl -s -f "http://mc-iam-manager:5000/readyz" > /dev/null 2>&1; then
#     echo '✓ mc-iam-manager API is ready!'
#     break
#   fi
  
#   if [ $attempt -eq $max_attempts ]; then
#     echo 'WARNING: mc-iam-manager API might not be fully ready, but proceeding anyway...'
#     echo 'This is normal if the service is still initializing.'
#   fi
  
#   echo 'API not ready yet, waiting 5 seconds...'
#   sleep 5
#   attempt=$((attempt + 1))
# done

echo ''
echo ''

echo '------------------------------------------------' 
echo ' Debug Info'
echo '------------------------------------------------'
# 디버깅: 현재 디렉토리와 파일 목록 확인
echo 'Current working directory:'
pwd
echo 'Files in current directory:'
ls -la

echo 'Files in /app/mc-iam-manager/:'
ls -la /app/mc-iam-manager/ || echo 'Directory /app/mc-iam-manager/ not found'

echo 'Files in mounted volume:'
ls -la /app/ || echo 'Directory /app/ not found'


echo ''
echo ''

echo '------------------------------------------------' 
echo ' Sleep 60 seconds'
echo '------------------------------------------------'
sleep 60

echo ''
echo ''

echo '------------------------------------------------' 
echo '1_setup_auto.sh'
echo '------------------------------------------------'

# 초기화 스크립트 실행
if [ -f '1_setup_auto.sh' ]; then
  echo 'Found 1_setup_auto.sh, making it executable...'
  chmod +x 1_setup_auto.sh
  echo 'File permissions after chmod:'
  ls -la 1_setup_auto.sh
  # echo 'File content (first 5 lines):'
  # head -5 1_setup_auto.sh
  echo 'Executing 1_setup_auto.sh...'
  
  # bash로 실행 (Ubuntu에는 bash가 기본적으로 포함됨)
  if bash 1_setup_auto.sh; then
    echo 'Script executed successfully with bash 1_setup_auto.sh'
  else
    cat <<'RECOVERY'
====================================================================
ERROR: 1_setup_auto.sh failed.

mc-iam-manager was likely not yet ready when setup ran.
To recover manually:

1. Wait ~2 minutes for all containers to stabilize.

2. Check service status:
       docker compose ps

   Confirm mc-iam-manager and mc-infra-manager are both healthy.

3. Re-run the post-init container (idempotent — safe to repeat):
       docker rm mc-iam-manager-post-initial 2>/dev/null
       docker compose up -d mc-iam-manager-post-initial
       docker logs -f mc-iam-manager-post-initial

   Each of the 8 setup steps should finish with ✓.

4. Verify health:
       curl -s http://localhost:${MC_IAM_MANAGER_PORT}/readyz | jq .
   Expected: "status": "healthy"
====================================================================
RECOVERY
    exit 1
  fi
else
  echo 'ERROR: 1_setup_auto.sh not found in current directory'
  echo 'Available files:'
  ls -la
  exit 1
fi

if [ $? -eq 0 ]; then
  echo '====================================================================' 
  echo '[Success] MC-IAM-Manager initialization completed successfully!'
  echo '====================================================================' 
  echo 'Container will exit normally.'
else
  echo '====================================================================' 
  echo '[Error] MC-IAM-Manager initialization failed!'
  echo '====================================================================' 
  echo 'Container will exit error code 1.'
  exit 1
fi 
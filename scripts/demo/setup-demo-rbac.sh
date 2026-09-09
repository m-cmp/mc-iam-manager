#!/usr/bin/env bash
# 5개 모듈(Workflow/Application/Data/Cost/Observability) x 3단계(viewer/operator/admin) 데모용
# 역할 슬롯 15종(13개 신규 + 기존 billviewer/billadmin 재사용 2개)·팀(조직) 20개·사용자 15명을 mc-iam-manager에 생성한다.
#
# 온보딩(1_setup_auto.sh/1_setup_manual.sh)과는 완전히 분리된, 플랫폼이 이미 뜬 뒤 사람이 필요할 때
# 한 번 수동으로 실행하는 선택적 스크립트다(과거 이 프로젝트에 있었던 add_demo_user.sh와 같은 성격 —
# docs/running-on-instance.md의 옛 안내 참고). 어떤 docker-compose/설치 스크립트도 이 파일을 자동으로
# 호출하지 않는다.
#
# 기본 5개 역할(admin/operator/viewer/billadmin/billviewer)의 정의처인 asset/menu/permission.yaml은
# 건드리지 않는다 — 이 스크립트가 추가하는 13개 신규 역할의 메뉴 매핑은 별도 문서
# asset/menu/backups/role-permission-backup-demo-rbac.yaml에 선언돼 있고,
# POST /api/setup/restore-role-permissions?mode=additive(이미 있는, 온보딩에서 안 쓰이는 API)로 적용한다.
#
# 이 스크립트는 데모에서 "보여주는" 대상이 아니라 데모를 위한 사전 준비(기초자료)다 — 사용자 생성/역할
# 배정/그룹 배정 자체가 데모의 핵심 흐름이 아니므로 UI 대신 API를 직접 호출해 빠르게 준비한다.
#
# 전 과정 멱등 — 이미 존재하는 role/org/user는 조회 후 재사용, 몇 번을 다시 돌려도 안전하다.
#
# 사용법:
#   MC_IAM_MANAGER_HOST=http://<host>:<port> \
#   MC_IAM_MANAGER_PLATFORMADMIN_ID=<platform admin id> \
#   MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD=<platform admin password> \
#   DEMO_RBAC_PASSWORD=<15개 데모 계정 공용 비밀번호> \
#   bash scripts/demo/setup-demo-rbac.sh
# 환경변수를 비워두면 이 자리에서 대화식으로 물어본다(1_setup_manual.sh의 login()과 동일한 방식).
# WORKSPACE_NAME(기본 ws01)으로 대상 워크스페이스를 바꿀 수 있다.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BACKUP_YAML="$REPO_ROOT/asset/menu/backups/role-permission-backup-demo-rbac.yaml"

MC_IAM_MANAGER_HOST="${MC_IAM_MANAGER_HOST:-http://localhost:5000}"
WORKSPACE_NAME="${WORKSPACE_NAME:-ws01}"

command -v curl >/dev/null || { echo "ERROR: curl이 필요합니다." >&2; exit 1; }
command -v jq >/dev/null || { echo "ERROR: jq가 필요합니다." >&2; exit 1; }
[ -f "$BACKUP_YAML" ] || { echo "ERROR: backup yaml을 찾을 수 없습니다: $BACKUP_YAML" >&2; exit 1; }

# ---- 5모듈 x 3단계 데이터 (module_key:team_name / tier:role_name) ----
declare -A MODULE_TEAM=(
  [wf]="Workflow Team"
  [app]="Application Team"
  [data]="Data Team"
  [bill]="Cost Team"
  [mon]="Observability Team"
)
MODULE_ORDER=(wf app data bill mon)

declare -A TIER_LABEL=(
  [viewer]="Viewers"
  [operator]="Operators"
  [admin]="Admins"
)
TIER_ORDER=(viewer operator admin)

declare -A ROLE_NAME=(
  [wf.viewer]="wfviewer"     [wf.operator]="wfoperator"     [wf.admin]="wfadmin"
  [app.viewer]="appviewer"   [app.operator]="appoperator"   [app.admin]="appadmin"
  [data.viewer]="dataviewer" [data.operator]="dataoperator" [data.admin]="dataadmin"
  [bill.viewer]="billviewer" [bill.operator]="billoperator" [bill.admin]="billadmin"
  [mon.viewer]="monviewer"   [mon.operator]="monoperator"   [mon.admin]="monadmin"
)

declare -A ROLE_ID_MAP=()

# ---- HTTP 헬퍼 ----
HTTP_STATUS=""
API_BODY=""

api_call() {
  local method="$1" path="$2" body="${3:-}"
  local resp
  if [ -n "$body" ]; then
    resp=$(curl -s -w '\n%{http_code}' -X "$method" \
      --header "Authorization: Bearer $ACCESS_TOKEN" \
      --header 'Content-Type: application/json' \
      --data "$body" "$MC_IAM_MANAGER_HOST$path")
  else
    resp=$(curl -s -w '\n%{http_code}' -X "$method" \
      --header "Authorization: Bearer $ACCESS_TOKEN" \
      --header 'Content-Type: application/json' \
      "$MC_IAM_MANAGER_HOST$path")
  fi
  HTTP_STATUS=$(echo "$resp" | tail -n1)
  API_BODY=$(echo "$resp" | sed '$d')
}

login() {
  local id="${MC_IAM_MANAGER_PLATFORMADMIN_ID:-}"
  local pw="${MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD:-}"
  [ -n "$id" ] || read -p "Platform admin ID: " id
  if [ -z "$pw" ]; then
    read -s -p "Platform admin password: " pw
    echo
  fi
  local body resp
  body=$(jq -n --arg id "$id" --arg pw "$pw" '{id: $id, password: $pw}')
  resp=$(curl -s --header 'Content-Type: application/json' --data "$body" "$MC_IAM_MANAGER_HOST/api/auth/login")
  ACCESS_TOKEN=$(echo "$resp" | jq -r '.access_token // empty')
  [ -n "$ACCESS_TOKEN" ] || { echo "ERROR: 로그인 실패: $resp" >&2; exit 1; }
  echo "로그인 성공"
}

# ---- 역할 ----
ROLE_ID=""
ensure_role() {
  local name="$1"
  api_call GET "/api/roles/name/$name?roleType=platform"
  if [ "$HTTP_STATUS" = "200" ]; then
    ROLE_ID=$(echo "$API_BODY" | jq -r '.id')
    echo "  role '$name' 이미 존재 (id=$ROLE_ID) — 재사용"
    return
  fi
  local body
  body=$(jq -n --arg name "$name" --arg desc "$name role (demo)" \
    '{name: $name, description: $desc, roleTypes: ["workspace","platform"]}')
  api_call POST "/api/roles" "$body"
  [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "201" ] || {
    echo "ERROR: role '$name' 생성 실패 ($HTTP_STATUS): $API_BODY" >&2; exit 1; }
  ROLE_ID=$(echo "$API_BODY" | jq -r '.id')
  echo "  role '$name' 생성 (id=$ROLE_ID)"
}

apply_permission_backup() {
  local resp
  resp=$(curl -s -w '\n%{http_code}' -X POST \
    --header "Authorization: Bearer $ACCESS_TOKEN" \
    --header 'Content-Type: application/x-yaml' \
    --data-binary "@$BACKUP_YAML" \
    "$MC_IAM_MANAGER_HOST/api/setup/restore-role-permissions?mode=additive")
  HTTP_STATUS=$(echo "$resp" | tail -n1)
  API_BODY=$(echo "$resp" | sed '$d')
  echo "  restore-role-permissions -> $HTTP_STATUS: $API_BODY"
  [ "$HTTP_STATUS" = "200" ] || { echo "ERROR: 메뉴 매핑 적용 실패" >&2; exit 1; }
}

# ---- 조직(팀) ----
ORGS_JSON=""
fetch_orgs() { api_call GET "/api/groups"; ORGS_JSON="$API_BODY"; }

# 조직 트리의 루트를 자동 탐지한다. 루트 이름은 설치 환경마다 다르므로(organizations.yaml 이 소유)
# 하드코딩하지 않는다. parent_id 는 *uint + omitempty 라 최상위 조직은 필드 자체가 없다.
#   최상위 1개  -> 그 조직 하위에 팀을 둔다
#   최상위 0개  -> 팀을 최상위에 둔다
#   최상위 2개+ -> 임의로 고르지 않고 경고 후 최상위에 둔다
detect_root_org_id() {
  local roots count
  roots=$(echo "$ORGS_JSON" | jq -c 'map(select((.parent_id // null) == null))')
  count=$(echo "$roots" | jq 'length')
  case "$count" in
    1)
      DETECTED_ROOT_ID=$(echo "$roots" | jq -r '.[0].id')
      echo "  루트 조직 '$(echo "$roots" | jq -r '.[0].name')' (id=$DETECTED_ROOT_ID) 하위를 사용합니다"
      ;;
    0)
      DETECTED_ROOT_ID=""
      echo "  최상위 조직 없음 — 팀을 최상위에 둡니다"
      ;;
    *)
      DETECTED_ROOT_ID=""
      echo "  경고: 최상위 조직이 ${count}개라 루트를 특정할 수 없습니다 — 팀을 최상위에 둡니다"
      echo "        ($(echo "$roots" | jq -r 'map(.name)|join(", ")'))"
      ;;
  esac
}

ORG_ID=""
ensure_org() {
  local name="$1" parent_id="${2:-}"
  fetch_orgs
  local pid_repr="${parent_id:-null}"
  local existing
  existing=$(echo "$ORGS_JSON" | jq -r --arg name "$name" --argjson pid "$pid_repr" \
    'map(select(.name == $name and ((.parent_id // null) == $pid))) | (.[0].id // empty)')
  if [ -n "$existing" ]; then
    echo "  org '$name' 이미 존재 (id=$existing) — 재사용"
    ORG_ID="$existing"
    return
  fi
  local body
  if [ -n "$parent_id" ]; then
    body=$(jq -n --arg name "$name" --arg desc "$name (demo)" --argjson pid "$parent_id" \
      '{name: $name, description: $desc, parent_id: $pid}')
  else
    body=$(jq -n --arg name "$name" --arg desc "$name (demo)" '{name: $name, description: $desc}')
  fi
  api_call POST "/api/groups" "$body"
  [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "201" ] || {
    echo "ERROR: org '$name' 생성 실패 ($HTTP_STATUS): $API_BODY" >&2; exit 1; }
  ORG_ID=$(echo "$API_BODY" | jq -r '.id')
  echo "  org '$name' 생성 (id=$ORG_ID)"
}

# ---- 워크스페이스 ----
get_workspace_id() {
  api_call GET "/api/workspaces/name/$WORKSPACE_NAME"
  [ "$HTTP_STATUS" = "200" ] || { echo "ERROR: workspace '$WORKSPACE_NAME' 조회 실패 ($HTTP_STATUS)" >&2; exit 1; }
  WORKSPACE_ID=$(echo "$API_BODY" | jq -r '.id')
}

# ---- 사용자 ----
USER_ID=""
IS_NEW_USER=0
ensure_user() {
  local username="$1"
  api_call GET "/api/users/name/$username"
  if [ "$HTTP_STATUS" = "200" ]; then
    USER_ID=$(echo "$API_BODY" | jq -r '.id')
    IS_NEW_USER=0
    echo "  user '$username' 이미 존재 (id=$USER_ID) — 재사용"
    return
  fi
  local email="${username}@demo.mcmp.local"
  local body
  body=$(jq -n --arg u "$username" --arg e "$email" \
    '{username: $u, email: $e, firstName: $u, lastName: "Demo", enabled: true}')
  api_call POST "/api/users" "$body"
  [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "201" ] || {
    echo "ERROR: user '$username' 생성 실패 ($HTTP_STATUS): $API_BODY" >&2; exit 1; }
  # 실측: POST /api/users 응답의 id는 실제 DB id가 아니라 요청 바디 echo(항상 0) — 재조회 필요
  api_call GET "/api/users/name/$username"
  USER_ID=$(echo "$API_BODY" | jq -r '.id')
  IS_NEW_USER=1
  echo "  user '$username' 생성 (id=$USER_ID)"
}

set_password() {
  local user_id="$1" password="$2"
  local body
  body=$(jq -n --arg p "$password" '{newPassword: $p}')
  api_call PUT "/api/users/id/$user_id/password" "$body"
  [ "$HTTP_STATUS" = "200" ] || { echo "ERROR: 비밀번호 설정 실패 (user=$user_id, $HTTP_STATUS): $API_BODY" >&2; exit 1; }
  echo "    password 설정 -> $HTTP_STATUS"
}

assign_user_to_group() {
  local group_id="$1" user_id="$2"
  local body
  body=$(jq -n --argjson uid "$user_id" '{user_ids: [$uid]}')
  api_call POST "/api/groups/id/$group_id/users" "$body"
  [ "$HTTP_STATUS" = "200" ] || { echo "ERROR: 그룹 배정 실패 (group=$group_id user=$user_id, $HTTP_STATUS): $API_BODY" >&2; exit 1; }
  echo "    그룹 배정(조직 편성용): group=$group_id user=$user_id -> $HTTP_STATUS"
}

# 실측: 그룹 단위 role 배정(GroupPlatformRole/GroupWorkspaceRole)은 대상 Keycloak realm role이
# 미리 없으면 500("realm role not found in Keycloak")을 낸다 — 그룹 경로엔 자동 생성 로직이 없다.
# 사용자 단위 배정(아래 두 함수)에는 없으면 자동 생성하는 로직이 있어 이쪽을 쓴다.
assign_platform_role() {
  local user_id="$1" role_id="$2" role_name="$3"
  local body
  body=$(jq -n --arg uid "$user_id" --arg rid "$role_id" --arg rn "$role_name" \
    '{userId: $uid, roleId: $rid, roleName: $rn, roleType: "platform"}')
  api_call POST "/api/roles/assign/platform-role" "$body"
  [ "$HTTP_STATUS" = "200" ] || { echo "ERROR: platform-role 배정 실패 (user=$user_id role=$role_name, $HTTP_STATUS): $API_BODY" >&2; exit 1; }
  echo "    platform-role 배정: user=$user_id role=$role_name($role_id) -> $HTTP_STATUS"
}

# 실측: 이 엔드포인트는 AssignPlatformRole과 달리 이미 배정된 경우를 미리 확인하지 않고 그냥 INSERT해서,
# 재실행 시 500 + "duplicate key value violates unique constraint ... mcmp_user_workspace_roles_pkey"로
# 실패한다(멱등성 가드가 없는 실제 플랫폼 동작 — mc-iam-manager 코드 수정 대상은 아니고 이 스크립트에서
# 그 특정 케이스만 "이미 배정됨"으로 흡수해 재실행 안전성을 확보한다).
assign_workspace_role() {
  local user_id="$1" role_id="$2" role_name="$3" workspace_id="$4"
  local body
  body=$(jq -n --arg uid "$user_id" --arg rid "$role_id" --arg rn "$role_name" --arg wid "$workspace_id" \
    '{userId: $uid, roleId: $rid, roleName: $rn, workspaceId: $wid}')
  api_call POST "/api/roles/assign/workspace-role" "$body"
  if [ "$HTTP_STATUS" = "200" ]; then
    echo "    workspace-role 배정: user=$user_id ws=$workspace_id role=$role_name($role_id) -> $HTTP_STATUS"
    return
  fi
  if echo "$API_BODY" | grep -q "mcmp_user_workspace_roles_pkey"; then
    echo "    workspace-role 배정: user=$user_id ws=$workspace_id role=$role_name($role_id) -> 이미 배정됨(스킵)"
    return
  fi
  echo "ERROR: workspace-role 배정 실패 (user=$user_id role=$role_name, $HTTP_STATUS): $API_BODY" >&2
  exit 1
}

# ================= main =================

login

DEMO_PASSWORD="${DEMO_RBAC_PASSWORD:-}"
if [ -z "$DEMO_PASSWORD" ]; then
  read -s -p "데모 계정 15개 공용 비밀번호: " DEMO_PASSWORD
  echo
fi

echo "== 워크스페이스 조회 ($WORKSPACE_NAME) =="
get_workspace_id
echo "$WORKSPACE_NAME id=$WORKSPACE_ID"

echo "== 역할 15종 생성/재사용 =="
for m in "${MODULE_ORDER[@]}"; do
  for t in "${TIER_ORDER[@]}"; do
    role="${ROLE_NAME[$m.$t]}"
    ensure_role "$role"
    ROLE_ID_MAP["$role"]="$ROLE_ID"
  done
done

echo "== 메뉴 매핑 적용 (role-permission-backup-demo-rbac.yaml, additive) =="
apply_permission_backup

echo "== 루트 조직 탐지 =="
# 루트 조직 생성은 온보딩(1_setup_auto.sh 의 organizations.yaml 시드) 소관이라 여기서는 조회만 한다.
fetch_orgs
detect_root_org_id
ROOT_ORG_ID="$DETECTED_ROOT_ID"

for m in "${MODULE_ORDER[@]}"; do
  team="${MODULE_TEAM[$m]}"
  echo ""
  echo "== 모듈: $team ($m) =="

  echo " -- 팀(조직) --"
  ensure_org "$team" "$ROOT_ORG_ID"
  parent_id="$ORG_ID"

  for t in "${TIER_ORDER[@]}"; do
    label="${TIER_LABEL[$t]}"
    role="${ROLE_NAME[$m.$t]}"
    role_id="${ROLE_ID_MAP[$role]}"
    child_name="$team / $label"

    ensure_org "$child_name" "$parent_id"
    group_id="$ORG_ID"

    username="demo-$role"
    echo " -- 사용자: $username --"
    ensure_user "$username"
    if [ "$IS_NEW_USER" = "1" ]; then
      set_password "$USER_ID" "$DEMO_PASSWORD"
    fi
    assign_user_to_group "$group_id" "$USER_ID"
    assign_platform_role "$USER_ID" "$role_id" "$role"
    assign_workspace_role "$USER_ID" "$role_id" "$role" "$WORKSPACE_ID"
  done
done

echo ""
echo "완료 — 15개 역할/팀/계정 준비됨. 계정: demo-{역할명}, 공용 비밀번호는 방금 입력/지정한 값."

adminRoleId=$(curl -s -X POST \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $2" \
    --data '{ "name": "admin", "description": "admin Role"}' \
    $1/api/role | jq -r '.id')

viewerRoleId=$(curl -s -X POST \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $2" \
    --data '{ "name": "viewer", "description": "viewer Role"}' \
    $1/api/role | jq -r '.id')

operatorRoleId=$(curl -s -X POST \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $2" \
    --data '{ "name": "operator", "description": "operator Role"}' \
    $1/api/role | jq -r '.id')

defaultWorkspaceId=$(curl -s -X GET \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $2" \
    $1/api/ws/workspace/default | jq -r '.[].id')

curl -s -X POST \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $2" \
    --data '{"workspaceId":"'"$defaultWorkspaceId"'","userId": "'"$3"'","roleId": "'"$adminRoleId"'"}'\
    $1/api/wsuserrole



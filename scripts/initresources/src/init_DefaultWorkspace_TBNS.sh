wsId=$(curl -s -X POST \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $2" \
    -d '{"name": "default", "description": "this is default workspace"}' \
    $1/api/ws | jq -r '.id')

projectsRes=$(curl -s -X GET \
    --header "Authorization: Bearer $2" \
    $1/api/tool/mcinfra/sync)

echo "$projectsRes" | jq -c '.[]' | while read -r item; do
    id=$(echo "$item" | jq -r '.id')
    curl -s -X POST -H "Content-Type: application/json" \
    --header "Authorization: Bearer $2" \
    -d '{"workspaceId": "'"$wsId"'", "'"projectIds"'": ["'"$id"'"]}'\
    $1/api/wsprj
done

#!/bin/bash

usage() {
    echo "nano ./init.env ( TB_HOST, TB_username, TB_password, MCIAM_HOST )"
}

source ./init.env
if [ -z "$TB_HOST" ] || [ -z "$TB_username" ] || [ -z "$TB_password" ] || [ -z "$MCIAM_HOST" ]; then
    usage
fi

AUTH_TOKEN=$(echo -n "$TB_username:$TB_password" | base64)
RESPONSE=$(curl -s -H "Authorization: Basic $AUTH_TOKEN" $TB_HOST/tumblebug/ns)


curl -s -X POST -H "Content-Type: application/json" \
    -d '{"name": "default", "description": "this is default workspace for Admin user"}' \
    $MCIAM_HOST/api/ws

echo "$RESPONSE" | jq -c '.ns[]' | while read -r item; do
    id=$(echo "$item" | jq -r '.id')
    description=$(echo "$item" | jq -r '.description')
    curl -s -X POST -H "Content-Type: application/json" \
        -d "{\"name\": \"$id\", \"description\": \"$description\"}"  \
        $MCIAM_HOST/api/prj
        
    curl -s -X POST -H "Content-Type: application/json" \
        -d "{\"projects\": [\"$id\"]}" \
        $MCIAM_HOST/api/wsprj/workspace/default
done

curl -s -X POST -H "Content-Type: application/json" \
    -d '{"role_name": "admin"}' \
    $MCIAM_HOST/api/role

curl -s -X POST -H "Content-Type: application/json" \
    -d '{"user_id": "mcpadmin", "role_name": "admin"}' \
    $MCIAM_HOST/api/wsuserrole/workspace/default


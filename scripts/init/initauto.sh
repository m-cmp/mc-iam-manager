#!/bin/bash
source ./.env

# -f 옵션 체크
force_mode=false
while getopts "f" opt; do
    case $opt in
        f) force_mode=true ;;
        *) echo "Usage: $0 [-f]"; exit 1 ;;
    esac
done

login(){
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MCIAMMANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MCIAMMANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$MCIAMMANAGER_HOST/api/auth/login")

    MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
        echo "Login failed."
        $force_mode || exit 1
    else
        echo "Login successful."
    fi
}

initRoleData(){
    IFS=',' read -r -a roles <<< "$PREDEFINED_ROLE"
    for role in "${roles[@]}"; do
        json_data=$(jq -n --arg name "$role" --arg description "$role Role" \
        '{name: $name, description: $description}')
        
        response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
        --header 'Content-Type: application/json' \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --data "$json_data" \
        "$MCIAMMANAGER_HOST/api/role")

        if [ "$response" -ne 200 ]; then
            echo "Failed to create role: $role"
            $force_mode || exit 1
        else
            echo "Role created successfully: $role"
        fi
    done
}

initMenuDatafromMenuYaml(){
    wget -q -O ./mcwebconsoleMenu.yaml "$MCWEBCONSOLE_MENUYAML"
    if [ $? -ne 0 ]; then
        echo "Failed to download mcwebconsoleMenu.yaml"
        $force_mode || exit 1
    else
        echo "Downloaded mcwebconsoleMenu.yaml successfully."
    fi

    response=$(curl -s -o /dev/null -w "%{http_code}" --location \
    "$MCIAMMANAGER_HOST/api/resource/file/framework/mc-web-console/menu" \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --form "file=@./mcwebconsoleMenu.yaml")

    if [ "$response" -ne 200 ]; then
        echo "Failed to upload mcwebconsoleMenu.yaml"
        $force_mode || exit 1
    else
        echo "Uploaded mcwebconsoleMenu.yaml successfully."
    fi
}

initMenuPermissionCSV(){
    wget -q -O ./permission.csv "$MCWEBCONSOLE_MENU_PERMISSIONS"
    if [ $? -ne 0 ]; then
        echo "Failed to download permission.csv"
        $force_mode || exit 1
    else
        echo "Downloaded permission.csv successfully."
    fi

    response=$(curl -s -o /dev/null -w "%{http_code}" --location \
    "$MCIAMMANAGER_HOST/api/permission/file/framework/all" \
    --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
    --form "file=@./permission.csv")

    if [ "$response" -ne 200 ]; then
        echo "Failed to upload permission.csv"
        $force_mode || exit 1
    else
        echo "Uploaded permission.csv successfully."
    fi
}

login
initRoleData
initMenuDatafromMenuYaml
initMenuPermissionCSV
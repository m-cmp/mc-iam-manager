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

createUserFromFile() {
    local file="./add_demo_user.json"

    # Read JSON array from file
    users=$(jq -c '.[]' "$file")

    echo $users

    for user in $users; do
    echo $user
        local user_id=$(echo "$user" | jq -r '.id')
        local password=$(echo "$user" | jq -r '.password')
        local first_name=$(echo "$user" | jq -r '.firstName')
        local last_name=$(echo "$user" | jq -r '.lastName')
        local email=$(echo "$user" | jq -r '.email')
        local description=$(echo "$user" | jq -r '.description')

        # Create user
        json_data=$(jq -n --arg id "$user_id" --arg password "$password" \
            --arg firstName "$first_name" --arg lastName "$last_name" \
            --arg email "$email" --arg description "$description" \
            '{id: $id, password: $password, firstName: $firstName, lastName: $lastName, email: $email, description: $description}')

        response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
            --location "$MCIAMMANAGER_HOST/api/user" \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
            --data "$json_data")

        if [ "$response" -ne 200 ]; then
            echo "Failed to create user $user_id"
            $force_mode || exit 1
        else
            echo "User created successfully: $user_id"
        fi

        # Activate user
        json_data=$(jq -n --arg userId "$user_id" '{userId: $userId}')

        response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
            --location "$MCIAMMANAGER_HOST/api/user/active" \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
            --data "$json_data")

        if [ "$response" -ne 200 ]; then
            echo "Failed to activate user $user_id"
            $force_mode || exit 1
        else
            echo "User activated successfully: $user_id"
        fi
    done
}


login
createUserFromFile
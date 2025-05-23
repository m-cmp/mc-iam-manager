#!/bin/bash

source ../../.env

login() {
    echo "Logging in as platformadmin..."
    login_url="$MCIAMMANAGER_HOST/api/auth/login"
    echo "Calling API: $login_url"
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MCIAMMANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MCIAMMANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$login_url")
    MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    echo "Login response: $response"
    echo "Login successful"
}

add_user() {
    local username=$1
    local email=$2
    local firstName=$3
    local lastName=$4

    json_data=$(jq -n --arg username "$username" --arg email "$email" --arg firstName "$firstName" --arg lastName "$lastName" \
        '{username: $username, email: $email, firstName: $firstName, lastName: $lastName}')
    
    user_url="$MCIAMMANAGER_HOST/api/users"
    echo "Calling API: $user_url"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$user_url")
    echo "User addition response: $response"
}

# 자동으로 platformAdmin 로그인
login

# 사용자 프로필 1-5 추가
echo "Adding user profiles..."

# 프로필 1: 관리자
add_user "testadmin01" "testadmin01@test.com" "ta" "01"

# 프로필 2: 운영자
add_user "testoperator01" "testoperator01@test.com" "to" "01"

# 프로필 3: 뷰어
add_user "testviewer01" "testviewer01@test.com" "tv" "01"

# 프로필 4: 재정관리자
add_user "testbilladmin01" "testbilladmin01@test.com" "tba" "01"

# 프로필 5: 재정뷰어
add_user "testbillviewer01" "testbillviewer01@test.com" "tbv" "01"

echo "User profiles added successfully" 
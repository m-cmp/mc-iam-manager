#!/bin/bash
source ../.env

login(){
    echo -e "\n====================================================================="
    read -p "Enter the ID: " ID
    read -s -p "Enter the password: " PASSWORD
    echo -e "\n====================================================================="

    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$ID"'",
        "password":"'"$PASSWORD"'"
    }' "$MCIAMHOST/api/auth/login")

    ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    echo ACCESSTOKEN
    echo $ACCESSTOKEN
}


# 메뉴 함수 정의
show_menu() {
    # clear
    echo "====================================================================="
    echo "0. exit"
    echo "1. login"
    echo "2. Init Data"
    echo "1001. cleanCert"
    echo "1002. cleanDB"
    echo "====================================================================="
    echo -n "select Number : "
}

MCIAMHOST=https://$DOMAIN:5000

read_option() {
    local choice
    read choice
    case $choice in
        0) exit 0 ;;
        1) login;;
        2) ./src/init_DefaultWorkspace_TBNS.sh $MCIAMHOST $ACCESSTOKEN && ./src/assign_current_user.sh $MCIAMHOST $ACCESSTOKEN $ID;;
        1001) ./src/cleanCert.sh ;;
        1002) ./src/cleanDB.sh ;;
        *) echo "wrong selection"
    esac
}

while true
do
    show_menu
    read_option
done
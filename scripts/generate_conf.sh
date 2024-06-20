#!/bin/bash

source ./.env

echo -e "================================================"
echo -e " * DOMAIN = ${DOMAIN}\n * EMAIL = ${EMAIL}"
echo -e "================================================"

mkdir -p ./nginx
cat > ./nginx/nginx.conf <<EOL
events {}

http {
    server {
        listen 5000 ssl;
        server_name ${DOMAIN};

        ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

        location / {
            proxy_pass http://mciammanager:3000;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
    }

    server {
        listen 443 ssl;
        server_name ${DOMAIN};

        ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
        ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

        location / {
            proxy_pass https://keycloak:8443;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
    }

    server {
        listen 80;
        server_name ${DOMAIN};

        location / {
            proxy_pass http://keycloak:8080;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
    }
}

EOL

cat > ./nginx/nginx-cert.conf <<EOL
events {}

http {
    server {
        listen 80;
        server_name ${DOMAIN};

        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
            allow all;
        }

        location / {
            return 301 https://\$host\$request_uri;
        }
    }
}
EOL

echo 
echo "** Nginx configuration file has been created at ./nginx/nginx.conf **"
echo 
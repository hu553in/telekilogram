#!/usr/bin/env bash

set -euxo pipefail

APP_NAME="telekilogram"
APP_DESC="Feed assistant"
APP_DIR="/srv/${APP_NAME}"
ENV_PATH="${APP_DIR}/.env"
SERVICE_PATH="/etc/systemd/system/${APP_NAME}.service"

# create app dir

ssh ${SSH_USER}@${SSH_IP} -p ${SSH_PORT} \
    "mkdir -p ${APP_DIR}"

# create and fill .env

ssh ${SSH_USER}@${SSH_IP} -p ${SSH_PORT} \
    "echo 'TOKEN=\"${TOKEN}\"' > ${ENV_PATH}; \
    echo 'ALLOWED_USERS=\"${ALLOWED_USERS}\"' >> ${ENV_PATH}"

# transfer binary

rsync -avzr --progress -e "ssh -p ${SSH_PORT}" \
    ./build/app \
    ${SSH_USER}@${SSH_IP}:${APP_DIR}/

# create service

cat << EOF > "$SERVICE_PATH"
[Unit]
Description=$APP_DESC
After=network.target

[Service]
Type=simple
Restart=always
ExecStart=$APP_DIR/app

[Install]
WantedBy=multi-user.target
EOF

# grant permissions

chmod 644 "$SERVICE_PATH"

# enable and restart service

systemctl daemon-reload
systemctl enable "$SERVICE_PATH"
systemctl restart "$SERVICE_PATH"

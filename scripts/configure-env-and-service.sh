#!/usr/bin/env bash

set -euxo pipefail

APP_NAME="telekilogram"
APP_DESC="Feed assistant"
APP_DIR="/srv/${APP_NAME}"
ENV_PATH="${APP_DIR}/.env"
SERVICE_PATH="/etc/systemd/system/${APP_NAME}.service"

# configure environment

echo "TOKEN=\"${TOKEN}\"" > ${ENV_PATH}
echo "ALLOWED_USERS=\"${ALLOWED_USERS}\"" >> ${ENV_PATH}

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
chmod +x $APP_DIR/app

# enable and restart service

systemctl daemon-reload
systemctl enable --now "$SERVICE_PATH"

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
echo "OPENAI_API_KEY=\"${OPENAI_API_KEY}\"" >> ${ENV_PATH}

# stop old instance ignoring errors

systemctl stop "$APP_NAME" || true

# create service

cat << EOF > "$SERVICE_PATH"
[Unit]
Description=$APP_DESC
After=network.target

[Service]
Type=simple
Restart=always
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/app

[Install]
WantedBy=multi-user.target
EOF

# grant permissions

chmod 644 "$SERVICE_PATH"
chmod +x $APP_DIR/app

# enable and start service

systemctl daemon-reload
systemctl enable "$APP_NAME"
systemctl start "$APP_NAME"

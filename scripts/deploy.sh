#!/usr/bin/env bash

set -euxo pipefail

APP_NAME="telekilogram"
APP_DIR="/srv/${APP_NAME}"

# create app dir

ssh ${SSH_USER}@${SSH_IP} -p ${SSH_PORT} \
    "mkdir -p ${APP_DIR}"

# transfer binary

rsync -avzr --progress -e "ssh -p ${SSH_PORT}" \
    ./build/app \
    ${SSH_USER}@${SSH_IP}:${APP_DIR}/

# configure environment and service

envsubst '$TOKEN $ALLOWED_USERS' \
    < ./scripts/configure-env-and-service.sh | \
    ssh ${SSH_USER}@${SSH_IP} -p ${SSH_PORT} \
    "bash -s"

#!/bin/bash

validate() {
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        echo "error: $1" >&2
        exit $EXIT_CODE
    fi
}

echo "[deploy] pruning images on remote pre-transfer"
ssh_username="ubuntu"
ssh_key_file="$HOME/.ssh/quemot-dev.pem"

ssh -i "$ssh_key_file" "$ssh_username@quemot.dev" <<EOF
validate() {
    EXIT_CODE=\$?
    if [[ \$EXIT_CODE -ne 0 ]]; then
        echo "error: \$1" >&2
        exit \$EXIT_CODE
    fi
}

sudo docker image prune --force >/dev/null
validate "failed to prune old images"
EOF

read -p "Enter passphrase for ssh key file ($ssh_key_file): " -s ssh_key_file_passphrase
validate "failed to read ssh key file password"
echo
manifest_path="$(dirname $(realpath "$0"))/../protected-transform-proxy.json"
echo "[deploy] invoking deploy-assets"
AWS_DEFAULT_PROFILE=deploy-assets SSH_USERNAME=$ssh_username SSH_KEY_FILE=$ssh_key_file SSH_KEY_FILE_PASSPHRASE=$ssh_key_file_passphrase deploy-assets -manifest "$manifest_path" $*
validate "failed to deploy assets"

#!/bin/bash

SECONDS=0

upload() {
    rsync $1 ${USER}@${HOST}:$2 --rsh "ssh -p ${PORT}" --recursive --delete -acvzh
}

execute() {
    ssh ${USER}@${HOST} -p ${PORT} "cd ${ROOT_PATH}; $1"
}

configure() {
    source deploy/$1/config.sh || (echo "ERROR: Stage $1 not found!" && exit 1)
}

configure $1
configure shared

notify ":kiss: ${APP_NAME} deploy to ${HOST} started by `whoami`@`hostname`."

execute "mkdir -p ${SCRIPTS_PATH}"
upload deploy/shared/ $SCRIPTS_PATH

buffalo build -o ${LOCAL_RELEASE_PATH}/${APP_NAME}

execute "bash $SCRIPTS_PATH/pre_install.sh $RELEASE_PATH"
upload $LOCAL_RELEASE_PATH/ $RELEASE_PATH
execute "bash $SCRIPTS_PATH/post_install.sh $RELEASE_PATH"

notify ":herb: ${APP_NAME} deploy finished in ${SECONDS}s."

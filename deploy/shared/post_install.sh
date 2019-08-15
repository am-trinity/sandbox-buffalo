#!/bin/bash

SCRIPT_PATH="$(cd "$(dirname "$0")"; pwd -P)"
source "${SCRIPT_PATH}/config.sh"

RELEASE_PATH="${1}"

ln -sf ${RELEASE_PATH}/ ${CURRENT_PATH}

for dir in ${SHARED_DIRS[*]}
do
    mkdir -p "${CURRENT_PATH}/${dir}"
    ln -sf "${SHARED_PATH}/${dir}/" "${CURRENT_PATH}/${dir}"
done

for file in ${SHARED_FILES[*]}
do
    if [ ! -f "${SHARED_PATH}/${file}" ]; then
        echo "ERROR: Shared file ${SHARED_PATH}/${file} not found!"
        exit 1
    fi
    ln -sf "${SHARED_PATH}/${file}" "${CURRENT_PATH}/${file}"
done

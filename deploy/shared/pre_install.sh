#!/bin/bash

SCRIPT_PATH="$(cd "$(dirname "$0")"; pwd -P)"
source "${SCRIPT_PATH}/config.sh"

RELEASE_PATH="${1}"

mkdir -p "${SHARED_PATH}" "${RELEASE_PATH}"

for dir in ${SHARED_DIRS[*]}
do
    mkdir -p "${SHARED_PATH}/${dir}"
done

# for file in ${SHARED_FILES[*]}
# do
#     touch "${SHARED_PATH}/${file}"
# done

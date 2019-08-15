APP_NAME="sandbox-server"

ROOT_PATH="/home/${USER}"
SCRIPTS_PATH="${ROOT_PATH}/shared/deploy"
RELEASES_PATH="${ROOT_PATH}/releases"
RELEASE_PATH="${RELEASES_PATH}/`date +%Y%m%d%H%M%S`"
CURRENT_PATH="${ROOT_PATH}/current"
SHARED_PATH="${ROOT_PATH}/shared"

SHARED_DIRS=("log" "tmp/sockets")
SHARED_FILES=()

notify() {
    curl -X POST -d "{\"content\": \"${1}\"}" ${DISCORD_URI}
}

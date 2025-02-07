#!/usr/bin/env bash
set -e

# This script starts N nodes (TODO N instead of 5) and waits for ctrl-c to shutdown the process group of OdysseyGo processes
# Uses data directory to store all OdysseyGo data neatly in one location with minimal config overhead
if ! [[ "$0" =~ scripts/run.sh ]]; then
  echo "must be run from repository root, but got $0"
  exit 255
fi

# Load the versions
SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SUBNET_EVM_PATH"/scripts/versions.sh

# Load the constants
source "$SUBNET_EVM_PATH"/scripts/constants.sh

# Set up odyssey binary path and assume build directory is set
ODYSSEYGO_BUILD_PATH=${ODYSSEYGO_BUILD_PATH:-"$GOPATH/src/github.com/DioneProtocol/odysseygo/build"}
ODYSSEYGO_PATH=${ODYSSEYGO_PATH:-"$ODYSSEYGO_BUILD_PATH/odysseygo"}
ODYSSEYGO_PLUGIN_DIR=${ODYSSEYGO_PLUGIN_DIR:-"$ODYSSEYGO_BUILD_PATH/plugins"}
DATA_DIR=${DATA_DIR:-/tmp/subnet-evm-start-node/$(date "+%Y-%m-%d%:%H:%M:%S")}

mkdir -p $DATA_DIR

# Set the config file contents for the path passed in as the first argument
function _set_config(){
  cat <<EOF >$1
  {
    "network-id": "local",
    "sybil-protection-enabled": false,
    "health-check-frequency": "5s",
    "plugin-dir": "$ODYSSEYGO_PLUGIN_DIR"
  }
EOF
}

function execute_cmd() {
  echo "Executing command: $@"
  $@
}

NODE_NAME="node1"
NODE_DATA_DIR="$DATA_DIR/$NODE_NAME"
echo "Creating data directory: $NODE_DATA_DIR"
mkdir -p $NODE_DATA_DIR
NODE_CONFIG_FILE_PATH="$NODE_DATA_DIR/config.json"
_set_config $NODE_CONFIG_FILE_PATH

CMD="$ODYSSEYGO_PATH --data-dir=$NODE_DATA_DIR --config-file=$NODE_CONFIG_FILE_PATH"

execute_cmd $CMD

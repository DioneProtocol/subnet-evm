#!/usr/bin/env bash
set -e

# Load the versions
SUBNET_EVM_PATH=$(
  cd "$(dirname "${BASH_SOURCE[0]}")"
  cd .. && pwd
)
source "$SUBNET_EVM_PATH"/scripts/versions.sh

# Load the constants
source "$SUBNET_EVM_PATH"/scripts/constants.sh

############################
# download odysseygo
# https://github.com/DioneProtocol/odysseygo/releases
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
BASEDIR=${BASEDIR:-"/tmp/odysseygo-release"}
ODYSSEYGO_BUILD_PATH=${ODYSSEYGO_BUILD_PATH:-${BASEDIR}/odysseygo}

mkdir -p ${BASEDIR}

ODYGO_DOWNLOAD_URL=https://github.com/DioneProtocol/odysseygo/releases/download/${ODYSSEYGO_VERSION}/odysseygo-linux-${GOARCH}-${ODYSSEYGO_VERSION}.tar.gz
ODYGO_DOWNLOAD_PATH=${BASEDIR}/odysseygo-linux-${GOARCH}-${ODYSSEYGO_VERSION}.tar.gz

if [[ ${GOOS} == "darwin" ]]; then
  ODYGO_DOWNLOAD_URL=https://github.com/DioneProtocol/odysseygo/releases/download/${ODYSSEYGO_VERSION}/odysseygo-macos-${ODYSSEYGO_VERSION}.zip
  ODYGO_DOWNLOAD_PATH=${BASEDIR}/odysseygo-macos-${ODYSSEYGO_VERSION}.zip
fi

BUILD_DIR=${ODYSSEYGO_BUILD_PATH}-${ODYSSEYGO_VERSION}

extract_archive() {
  mkdir -p ${BUILD_DIR}

  if [[ ${ODYGO_DOWNLOAD_PATH} == *.tar.gz ]]; then
    tar xzvf ${ODYGO_DOWNLOAD_PATH} --directory ${BUILD_DIR} --strip-components 1
  elif [[ ${ODYGO_DOWNLOAD_PATH} == *.zip ]]; then
    unzip ${ODYGO_DOWNLOAD_PATH} -d ${BUILD_DIR}
    mv ${BUILD_DIR}/build/* ${BUILD_DIR}
    rm -rf ${BUILD_DIR}/build/
  fi
}

# first check if we already have the archive
if [[ -f ${ODYGO_DOWNLOAD_PATH} ]]; then
  # if the download path already exists, extract and exit
  echo "found odysseygo ${ODYSSEYGO_VERSION} at ${ODYGO_DOWNLOAD_PATH}"

  extract_archive
else
  # try to download the archive if it exists
  if curl -s --head --request GET ${ODYGO_DOWNLOAD_URL} | grep "302" > /dev/null; then
    echo "${ODYGO_DOWNLOAD_URL} found"
    echo "downloading to ${ODYGO_DOWNLOAD_PATH}"
    curl -L ${ODYGO_DOWNLOAD_URL} -o ${ODYGO_DOWNLOAD_PATH}

    extract_archive
  else
    # else the version is a git commitish (or it's invalid)
    GIT_CLONE_URL=https://github.com/DioneProtocol/odysseygo.git
    GIT_CLONE_PATH=${BASEDIR}/odysseygo-repo/

    # check to see if the repo already exists, if not clone it
    if [[ ! -d ${GIT_CLONE_PATH} ]]; then
      echo "cloning ${GIT_CLONE_URL} to ${GIT_CLONE_PATH}"
      git clone --no-checkout ${GIT_CLONE_URL} ${GIT_CLONE_PATH}
    fi

    # check to see if the commitish exists in the repo
    WORKDIR=$(pwd)

    cd ${GIT_CLONE_PATH}

    git fetch

    echo "checking out ${ODYSSEYGO_VERSION}"

    set +e
    # try to checkout the branch
    git checkout origin/${ODYSSEYGO_VERSION} > /dev/null 2>&1
    CHECKOUT_STATUS=$?
    set -e

    # if it's not a branch, try to checkout the commit
    if [[ $CHECKOUT_STATUS -ne 0 ]]; then
      set +e
      git checkout ${ODYSSEYGO_VERSION} > /dev/null 2>&1
      CHECKOUT_STATUS=$?
      set -e

      if [[ $CHECKOUT_STATUS -ne 0 ]]; then
        echo
        echo "'${VERSION}' is not a valid release tag, commit hash, or branch name"
        exit 1
      fi
    fi

    COMMIT=$(git rev-parse HEAD)

    # use the commit hash instead of the branch name or tag
    BUILD_DIR=${ODYSSEYGO_BUILD_PATH}-${COMMIT}

    # if the build-directory doesn't exist, build odysseygo
    if [[ ! -d ${BUILD_DIR} ]]; then
      echo "building odysseygo ${COMMIT} to ${BUILD_DIR}"
      ./scripts/build.sh
      mkdir -p ${BUILD_DIR}

      mv ${GIT_CLONE_PATH}/build/* ${BUILD_DIR}/
    fi

    cd $WORKDIR
  fi
fi

ODYSSEYGO_PATH=${ODYSSEYGO_BUILD_PATH}/odysseygo
ODYSSEYGO_PLUGIN_DIR=${ODYSSEYGO_BUILD_PATH}/plugins

mkdir -p ${ODYSSEYGO_BUILD_PATH}

cp ${BUILD_DIR}/odysseygo ${ODYSSEYGO_PATH}


echo "Installed OdysseyGo release ${ODYSSEYGO_VERSION}"
echo "OdysseyGo Path: ${ODYSSEYGO_PATH}"
echo "Plugin Dir: ${ODYSSEYGO_PLUGIN_DIR}"

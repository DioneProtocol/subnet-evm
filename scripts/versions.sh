#!/usr/bin/env bash

# Don't export them as they're used in the context of other calls
ODYSSEY_VERSION=${ODYSSEY_VERSION:-'v0.0.1'}
ODYSSEYGO_VERSION=${ODYSSEYGO_VERSION:-$ODYSSEY_VERSION}
GINKGO_VERSION=${GINKGO_VERSION:-'v2.2.0'}

# This won't be used, but it's here to make code syncs easier
LATEST_CORETH_VERSION='0.12.4-rc.0'

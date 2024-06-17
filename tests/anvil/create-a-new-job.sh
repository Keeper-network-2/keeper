#!/bin/bash

# cd to the directory of this script so that this can be run from anywhere
parent_path=$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd -P
)
# At this point we are in the script directory
cd "$parent_path"

set -a
source ./utils.sh
set +a


cd ../../contracts

# Define contract address and private key
KEEPER_NETWORK_TASK_MANAGER_ADDRESS="0x9E545E3C0baAB3E08CdfD552C960A1050f373042"
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

# Create a new job
forge script script/CreateJobScript.s.sol --rpc-url http://localhost:8545 --private-key $PRIVATE_KEY --broadcast --sig "run()"


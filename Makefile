############################# HELP MESSAGE #############################
# Make sure the help command stays first, so that it's printed by default when `make` is called without arguments
.PHONY: help tests
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

AGGREGATOR_ECDSA_PRIV_KEY=0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6

CHAINID=31337

# Make sure to update this if the strategy address changes
# check in contracts/script/output/${CHAINID}/credible_squaring_avs_deployment_output.json

STRATEGY_ADDRESS=0x4c5859f0F772848b2D91F1D83E2Fe57935348029
DEPLOYMENT_FILES_DIR=contracts/script/output/${CHAINID}

-----------------------------: ## 

___CONTRACTS___: ## 

build-contracts: ## builds all contracts
	cd contracts && forge build

deploy-eigenlayer-contracts-to-anvil-and-save-state: ## Deploy eigenlayer
	./tests/anvil/deploy-eigenlayer-save-anvil-state.sh

deploy-keeper-network-contracts-to-anvil-and-save-state: ## Deploy avs
	./tests/anvil/deploy-avs-save-anvil-state.sh

deploy-all-to-anvil-and-save-state: deploy-eigenlayer-contracts-to-anvil-and-save-state deploy-incredible-squaring-contracts-to-anvil-and-save-state ## deploy eigenlayer, shared avs contracts, and inc-sq contracts 

start-anvil-chain-with-el-and-avs-deployed: ## starts anvil from a saved state file (with el and avs contracts deployed)
	./tests/anvil/start-anvil-chain-with-el-and-avs-deployed.sh

bindings: ## generates contract bindings
	cd contracts && ./generate-go-bindings.sh


__CLI__: ## 

cli-setup-operator: send-fund cli-register-operator-with-eigenlayer cli-deposit-into-mocktoken-strategy cli-register-operator-with-avs ## registers operator with eigenlayer and avs

cli-register-operator-with-eigenlayer: ## registers operator with delegationManager
	go run cli/main.go --config config-files/operator.anvil.yaml register-operator-with-eigenlayer

cli-deposit-into-mocktoken-strategy: ## 
	./scripts/deposit-into-mocktoken-strategy.sh

cli-register-operator-with-avs: ## 
	go run cli/main.go --config config-files/operator.anvil.yaml register-operator-with-avs

cli-deregister-operator-with-avs: ## 
	go run cli/main.go --config config-files/operator.anvil.yaml deregister-operator-with-avs

cli-print-operator-status: ## 
	go run cli/main.go --config config-files/operator.anvil.yaml print-operator-status

send-fund: ## sends fund to the operator saved in tests/keys/test.ecdsa.key.json
	cast send 0x860B6912C2d0337ef05bbC89b0C2CB6CbAEAB4A5 --value 10ether --private-key 0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6

-----------------------------: ## 
# We pipe all zapper logs through https://github.com/maoueh/zap-pretty so make sure to install it
# TODO: piping to zap-pretty only works when zapper environment is set to production, unsure why
# Define the binary name
BINARY_NAME=task-manager

# Define the package to build
PACKAGE=cmd/main.go

# Default target to build the binary
all: build

# Build the binary
build:
	@echo "Building the binary..."
	GO111MODULE=on go build -o $(BINARY_NAME) $(PACKAGE)

# Run the application
run: build
	@echo "Running the application..."
	./$(BINARY_NAME)

# Clean up the binary
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)

# Test the application
test:
	@echo "Running tests..."
	go test ./...

# Help message
help:
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all      - Build the binary (default)"
	@echo "  build    - Build the binary"
	@echo "  run      - Build and run the application"
	@echo "  clean    - Clean up the binary"
	@echo "  test     - Run tests"
	@echo "  help     - Show this help message"

____OFFCHAIN_SOFTWARE___: ## 
# start-aggregator: ## 
# 	go run aggregator/cmd/main.go --config config-files/aggregator.yaml \
# 		--credible-squaring-deployment ${DEPLOYMENT_FILES_DIR}/credible_squaring_avs_deployment_output.json \
# 		--ecdsa-private-key ${AGGREGATOR_ECDSA_PRIV_KEY} \
# 		2>&1 | zap-pretty

start-keeper: ## 
	go run keeper/keeper.go

start-task-manager: ## 
	cd taskmanager && go run cmd/main.go


run-plugin: ## 
	go run plugin/cmd/main.go --config config-files/operator.anvil.yaml
-----------------------------: ##
_____HELPER_____: ## 

create-job:
	cd contracts && forge script script/CreateJob.s.sol --rpc-url http://localhost:8545 --broadcast

mocks: ## generates mocks for tests
	go install go.uber.org/mock/mockgen@v0.3.0
	go generate ./...

tests-unit: ## runs all unit tests
	go test $$(go list ./... | grep -v /integration) -coverprofile=coverage.out -covermode=atomic --timeout 15s
	go tool cover -html=coverage.out -o coverage.html

tests-contract: ## runs all forge tests
	cd contracts && forge test

tests-integration: ## runs all integration tests
	go test ./tests/integration/... -v -count=1

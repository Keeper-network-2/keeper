# Keeper Network 2.0 AVS

<b> Do not use it in Production, testnet only. </b>

A decentralized keeper network leveraging EigenLayer's infrastructure. Developers can list automation jobs specifying a URL of code to execute and intervals. Tasks are assigned to keepers for execution, enhancing network automation and reliability.

## Dependencies

You will need [foundry](https://book.getfoundry.sh/getting-started/installation) and [zap-pretty](https://github.com/maoueh/zap-pretty) and docker to run the examples below.
```
curl -L https://foundry.paradigm.xyz | bash
foundryup
go install github.com/maoueh/zap-pretty@latest
```
You will also need to [install docker](https://docs.docker.com/get-docker/), and build the contracts:
```
make build-contracts
```

## Running via make

This simple session illustrates the basic flow of the AVS. The makefile commands are hardcoded for a single operator, but it's however easy to create new operator config files, and start more operators manually (see the actual commands that the makefile calls).

Start anvil in a separate terminal:

```bash
make start-anvil-chain-with-el-and-avs-deployed
```

The above command starts a local anvil chain from a [saved state](./tests/anvil/avs-and-eigenlayer-deployed-anvil-state.json) with eigenlayer and incredible-squaring contracts already deployed (but no operator registered).

Start the task manager:

```bash
make task-manager
```

Register the Keeper(operator) with eigenlayer and incredible-squaring, and then start the process:

```bash
make start-keeper
```

Create a Job: 

```bash
make create-job
```


## Avs Task Description


![](./diagrams/keepernetwork.png)


// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "forge-std/Script.sol";
import "../src/Jobmanager.sol"; // Adjust the import path according to your project structure

contract CreateJobScript is Script {
    JobCreator public jobCreator;
    address public constant JOB_CREATOR_ADDRESS = 0x9E545E3C0baAB3E08CdfD552C960A1050f373042; // Replace with actual address
    uint256 public constant MINIMUM_STAKE = 1 ether;

    function run() external {
        uint256 deployerPrivateKey = 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d;
        vm.startBroadcast(deployerPrivateKey);
        address user = vm.addr(deployerPrivateKey);

        jobCreator = JobCreator(JOB_CREATOR_ADDRESS);

        uint256 userBalance = user.balance;
        console.log("User:", user);
        console.log("User balance:", userBalance);

        console.log("Staking 1 ETH for the user...");
        jobCreator.stake{value: MINIMUM_STAKE}();
        console.log("Staked 1 ETH for the user");

        string memory jobType = "Example Job Type";
        string memory status = "Open";
        bytes memory quorumNumbers = abi.encodePacked(uint8(1), uint8(2), uint8(3)); // Example quorum numbers
        uint32 quorumThresholdPercentage = 70; // Example quorum threshold percentage
        uint32 timeframe = 100; // Example timeframe
        string memory contract_add = "0x1234567890123456789012345678901234567890"; // Example contract address
        uint chain_id = 1; // Example chain ID (1 for Ethereum mainnet)
        string memory target_fnc = "exampleFunction"; // Example target function

        jobCreator.createJob(
            jobType,
            status,
            quorumNumbers,
            quorumThresholdPercentage,
            timeframe,
            contract_add,
            chain_id,
            target_fnc
        );

        console.log("Job created successfully");

        vm.stopBroadcast();
    }
}
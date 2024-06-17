// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "forge-std/Script.sol";
import "../src/KeeperNetworkTaskManager.sol"; // Adjust the import path according to your project structure

contract CreateJobScript is Script {
    KeeperNetworkTaskManager public keeperNetworkTaskManager;
    address public constant KEEPER_NETWORK_TASK_MANAGER_ADDRESS = 0x9E545E3C0baAB3E08CdfD552C960A1050f373042;
    uint256 public constant MINIMUM_STAKE = 1 ether;

    function run() external {
        uint256 deployerPrivateKey =0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d;
        vm.startBroadcast(deployerPrivateKey);
        address user = vm.addr(deployerPrivateKey);

        keeperNetworkTaskManager = KeeperNetworkTaskManager(KEEPER_NETWORK_TASK_MANAGER_ADDRESS);

        uint256 userBalance = user.balance;
        console.log("User :", user);

        console.log("User balance:", userBalance);


        console.log("User does not have enough stake, staking now...");

        keeperNetworkTaskManager.stake{value: MINIMUM_STAKE}();

        console.log("Staked 1 ETH for the user");


        string memory jobType = "Example Job Type";
        string memory jobDescription = "This is an example job description.";
        string memory gitlink = "https://github.com/example/repo";
        string memory status = "Open";
        bytes memory quorumNumbers = abi.encodePacked(uint8(1), uint8(2), uint8(3)); // Example quorum numbers
        uint32 quorumThresholdPercentage = 70; // Example quorum threshold percentage
        uint32 timeframe = 100; // Example timeframe

        keeperNetworkTaskManager.createJob(
            jobType,
            jobDescription,
            gitlink,
            status,
            quorumNumbers,
            quorumThresholdPercentage,
            timeframe
        );

        vm.stopBroadcast();
    }
}

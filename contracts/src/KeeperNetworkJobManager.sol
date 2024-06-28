// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;


import "@eigenlayer-middleware/src/interfaces/IServiceManager.sol";
import {BLSApkRegistry} from "@eigenlayer-middleware/src/BLSApkRegistry.sol";
import {RegistryCoordinator} from "@eigenlayer-middleware/src/RegistryCoordinator.sol";
import {BLSSignatureChecker, IRegistryCoordinator} from "@eigenlayer-middleware/src/BLSSignatureChecker.sol";
import {OperatorStateRetriever} from "@eigenlayer-middleware/src/OperatorStateRetriever.sol";
import "@eigenlayer-middleware/src/libraries/BN254.sol";
import "./IKeeperNetworkJobManager.sol";


contract KeeperNetworkJobManager is 
    IKeeperNetworkJobManager
{
    address public owner;
    mapping(uint32 => Job) public jobs;
    uint32 public jobCount;

    modifier onlyOwner() {
        require(msg.sender == owner, "Not the owner");
        _;
    }

    constructor() {
        owner = msg.sender;
    }

    function createJob(
        string calldata jobType,
        string calldata jobDescription,
        string calldata gitlink,
        string calldata status,
        bytes calldata quorumNumbers,
        uint32 quorumThresholdPercentage,
        uint32 timeframe
    ) external override {
        jobCount++;
        jobs[jobCount] = Job({
            jobId: jobCount,
            jobType: jobType,
            jobDescription: jobDescription,
            gitlink: gitlink,
            status: status,
            quorumNumbers: quorumNumbers,
            quorumThresholdPercentage: quorumThresholdPercentage,
            timeframe: timeframe,
            blockNumber: block.number
            // contract_add: "",
            // chain_id: 0,
            // target_fnc: ""
        });

        emit JobCreated(jobCount, jobType, gitlink);
    }

    function deleteJob(uint32 jobId) external override onlyOwner {
        require(jobs[jobId].jobId != 0, "Job does not exist");
        delete jobs[jobId];
        emit JobDeleted(jobId);
    }

    function updateJobStatus(uint32 jobId, string calldata status) external override onlyOwner {
        require(jobs[jobId].jobId != 0, "Job does not exist");
        jobs[jobId].status = status;
        emit JobStatusUpdated(jobId, status);
    }

    function stake() external payable override {
        emit Staked(msg.sender, msg.value);
    }

    function addToStake(address operator, uint256 amount) external payable override {
        emit Staked(operator, amount);
    }

    function withdraw(uint256 amount) external override {
        require(amount <= address(this).balance, "Insufficient balance");
        payable(msg.sender).transfer(amount);
        emit Withdrawn(msg.sender, amount);
    }

    function joobNumber() external view override returns (uint32) {
        return jobCount;
    }

    function respondToJob(
        uint32 jobId,
        JobResponse calldata jobResponse,
        JobResponseMetadata calldata jobResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) external {
        require(jobs[jobId].jobId != 0, "Job does not exist");
        // Logic to handle job response
        emit JobResponded(jobResponse, jobResponseMetadata);
    }
}
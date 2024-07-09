// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@eigenlayer-middleware/src/libraries/BN254.sol";

interface IKeeperNetworkJobManager {
    // EVENTS
    event JobResponded(
        JobResponse jobResponse,
        JobResponseMetadata jobResponseMetadata
    );
    // event JobCompleted(uint32 indexed jobId);
    // event JobChallengedSuccessfully(
    //     uint32 indexed jobId,
    //     address indexed challenger
    // );
    event JobCreated(
        uint32 indexed jobId,
        string jobType,
        string contract_add,
        uint chain_id
    );
    event JobDeleted(uint32 indexed jobId);
    // event JobEvent(
    //     uint32 indexed jobId,
    //     string jobType,
    //     string status
    // );
    event JobStatusUpdated(uint32 indexed jobId, string status);
    // event JobAssigned(uint32 indexed jobId, address operator);
    event Staked(address indexed user, uint256 amount);
    event Withdrawn(address indexed user, uint256 amount);

    // STRUCTS
    struct JobResponse {
        uint32 referenceJobId;
        uint256 numberSquared;
    }

    struct JobResponseMetadata {
        uint256 jobResponsedBlock;
        bytes32 hashOfNonSigners;
    }

    struct Job {
        uint32 jobId;
        string jobType;
        string status;
        bytes quorumNumbers;
        uint32 quorumThresholdPercentage;
        uint32 timeframe;
        uint256 blockNumber;
        string contract_add;
        uint chain_id;
        string target_fnc;
    }

    // FUNCTIONS
    function stake() external payable;
    function withdraw(uint256 amount) external;
    function joobNumber() external view returns (uint32);
    // function raiseAndResolveChallenge(
    //     Job calldata job,
    //     JobResponse calldata jobResponse,
    //     JobResponseMetadata calldata jobResponseMetadata,
    //     BN254.G1Point[] memory pubkeysOfNonSigningOperators
    // ) external;
    function createJob(
        string calldata jobType,
        string calldata status,
        bytes calldata quorumNumbers,
        uint32 quorumThresholdPercentage,
        uint32 timeframe,
        string calldata contract_add,
        uint chain_id,
        string calldata target_fnc
    ) external;
    function deleteJob(uint32 jobId) external;
    function updateJobStatus(uint32 jobId, string calldata status) external;
    // function assignJob(uint32 jobId, address operator) external;
}

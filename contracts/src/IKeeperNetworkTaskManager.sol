// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@eigenlayer-middleware/src/libraries/BN254.sol";

interface IKeeperNetworkTaskManager {
    // EVENTS
    event TaskCreated(uint32 indexed taskId, uint32 indexed jobId, string taskType);
    event TaskDeleted(uint32 indexed taskId);
    event TaskStatusUpdated(uint32 indexed taskId, string status);
    event TaskAssigned(uint32 indexed taskId, address operator);
    event TaskCompleted(uint32 indexed taskId);
    event TaskResponded(
        TaskResponse taskResponse,
        TaskResponseMetadata taskResponseMetadata
    );
    event TaskChallengedSuccessfully(
        uint32 indexed taskId,
        address indexed challenger
    );

    // STRUCTS
    struct Task {
        uint32 taskId;
        uint32 jobId;
        string taskType;
        string status;
        uint256 blockNumber;
    }

    struct TaskResponse {
        uint32 referenceTaskId;
        uint256 numberSquared;
    }

    struct TaskResponseMetadata {
        uint256 taskResponsedBlock;
        bytes32 hashOfNonSigners;
    }

    // FUNCTIONS
    function createTask(
        uint32 jobId,
        string calldata taskType,
        string calldata status
    ) external;

    function deleteTask(uint32 taskId) external;

    function updateTaskStatus(uint32 taskId, string calldata status) external;

    function assignTask(uint32 taskId, address operator) external;

    function respondToTask(
        uint32 taskId,
        TaskResponse calldata taskResponse,
        TaskResponseMetadata calldata taskResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) external;

    function raiseAndResolveChallenge(
        Task calldata task,
        TaskResponse calldata taskResponse,
        TaskResponseMetadata calldata taskResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) external;
}

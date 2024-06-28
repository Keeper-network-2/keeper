// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@openzeppelin-upgrades/contracts/proxy/utils/Initializable.sol";
import "@openzeppelin-upgrades/contracts/access/OwnableUpgradeable.sol";
import "@eigenlayer/contracts/permissions/Pausable.sol";
import "@eigenlayer-middleware/src/interfaces/IServiceManager.sol";
import {BLSApkRegistry} from "@eigenlayer-middleware/src/BLSApkRegistry.sol";
import {RegistryCoordinator} from "@eigenlayer-middleware/src/RegistryCoordinator.sol";
import {BLSSignatureChecker, IRegistryCoordinator} from "@eigenlayer-middleware/src/BLSSignatureChecker.sol";
import {OperatorStateRetriever} from "@eigenlayer-middleware/src/OperatorStateRetriever.sol";
import "@eigenlayer-middleware/src/libraries/BN254.sol";
import "./IKeeperNetworkTaskManager.sol";

// IKeeperNetworkTaskManager,
//     Initializable,
//     OwnableUpgradeable,
//     Pausable,
//     BLSSignatureChecker,
//     OperatorStateRetriever


contract TaskManager is Initializable,
    OwnableUpgradeable,
    Pausable
{
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

    // STATE VARIABLES
    uint32 public immutable TASK_RESPONSE_WINDOW_BLOCK;
    uint32 public constant TASK_CHALLENGE_WINDOW_BLOCK = 100;
    uint256 internal constant _THRESHOLD_DENOMINATOR = 100;

    IRegistryCoordinator public registryCoordinator;
    address public aggregator;

    mapping(uint32 => Task) public tasks;
    uint32 public taskCount;

    constructor(IRegistryCoordinator _registryCoordinator, uint32 _taskResponseWindowBlock) {
        registryCoordinator = _registryCoordinator;
        TASK_RESPONSE_WINDOW_BLOCK = _taskResponseWindowBlock;
    }

    function initialize(
        IPauserRegistry _pauserRegistry,
        address initialOwner,
        address _aggregator
    ) public initializer {
        _initializePauser(_pauserRegistry, UNPAUSE_ALL);
        _transferOwnership(initialOwner);
        aggregator = _aggregator;
    }

    function createTask(
        uint32 jobId,
        string calldata taskType,
        string calldata status
    ) external {
        taskCount++;
        tasks[taskCount] = Task({
            taskId: taskCount,
            jobId: jobId,
            taskType: taskType,
            status: status,
            blockNumber: block.number
        });

        emit TaskCreated(taskCount, jobId, taskType);
    }

    function deleteTask(uint32 taskId) external {
        require(tasks[taskId].taskId != 0, "Task does not exist");
        delete tasks[taskId];
        emit TaskDeleted(taskId);
    }

    function updateTaskStatus(uint32 taskId, string calldata status) external {
        require(tasks[taskId].taskId != 0, "Task does not exist");
        tasks[taskId].status = status;
        emit TaskStatusUpdated(taskId, status);
    }

    function assignTask(uint32 taskId, address operator) external {
        require(tasks[taskId].taskId != 0, "Task does not exist");
        emit TaskAssigned(taskId, operator);
    }

    function respondToTask(
        uint32 taskId,
        TaskResponse calldata taskResponse,
        TaskResponseMetadata calldata taskResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) external {
        require(tasks[taskId].taskId != 0, "Task does not exist");
        // Logic to handle task response
        emit TaskCompleted(taskId);
    }

    function raiseAndResolveChallenge(
        Task calldata task,
        TaskResponse calldata taskResponse,
        TaskResponseMetadata calldata taskResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) external {
        require(tasks[task.taskId].taskId != 0, "Task does not exist");
        // Logic to handle task challenge and resolution
        emit TaskChallengedSuccessfully(task.taskId, msg.sender);
    }
}
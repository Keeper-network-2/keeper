// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@eigenlayer-middleware/src/libraries/BN254.sol";

interface IKeeperNetworkTaskManager {
    // EVENTS

    // Emitted when a job response is received
    event JobResponded(
        JobResponse jobResponse,
        JobResponseMetadata jobResponseMetadata
    );

    // Emitted when a job is completed
    event JobCompleted(uint32 indexed jobId);

    // Emitted when a job is successfully challenged
    event JobChallengedSuccessfully(
        uint32 indexed jobId,
        address indexed challenger
    );

    // Emitted when a new job is created
    event JobCreated(
        uint32 indexed jobId,
        string jobType,
        string jobDescription,
        string gitlink
    );

    // Emitted when a job is deleted
    event JobDeleted(uint32 indexed jobId);

    // Emitted when a job event occurs
    event JobEvent(
        uint32 indexed jobId,
        string jobType,
        string jobDescription,
        string status,
        string gitlink
    );

    // Emitted when a job status is updated
    event JobStatusUpdated(uint32 indexed jobId, string status);

    // Emitted when a job is assigned to an operator
    event JobAssigned(uint32 indexed jobId, address operator);

    // Emitted when a user stakes ETH
    event Staked(address indexed user, uint256 amount);

    // Emitted when a user withdraws staked ETH
    event Withdrawn(address indexed user, uint256 amount);

    // STRUCTS

    // Structure to hold job response data
    struct JobResponse {
        uint32 referenceJobId;
        uint256 numberSquared;
    }

    // Structure to hold job response metadata
    struct JobResponseMetadata {
        uint256 jobResponsedBlock;
        bytes32 hashOfNonSigners;
    }

    // Structure to hold job details
    struct Job {
        uint32 jobId;
        string jobType;
        string jobDescription;
        string status;
        bytes quorumNumbers;
        uint32 quorumThresholdPercentage;
        uint32 timeframe;
        string gitlink;
        uint256 blockNumber;
    }

    // FUNCTIONS

    /**
     * @notice Stake ETH to be eligible to create jobs
     */
    function stake() external payable;

    /**
     * @notice Withdraw staked ETH
     * @param amount The amount of ETH to withdraw
     */
    function withdraw(uint256 amount) external;

    // /**
    //  * @notice Get the staked amount of a user
    //  * @param user The address of the user
    //  * @return The amount of ETH staked by the user
    //  */
    // function getStake(address user) external view returns (uint256);

    
    /**
     * @notice Get the current job number
     * @return The current job number
     */
    function joobNumber() external view returns (uint32);

    /**
     * @notice Raise and resolve a challenge for a job
     * @param job The job details
     * @param jobResponse The job response details
     * @param jobResponseMetadata The job response metadata
     * @param pubkeysOfNonSigningOperators The public keys of non-signing operators
     */
    function raiseAndResolveChallenge(
        Job calldata job,
        JobResponse calldata jobResponse,
        JobResponseMetadata calldata jobResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) external;

    // /**
    //  * @notice Get the job response window block
    //  * @return The job response window block
    //  */
    // function getJobResponseWindowBlock() external view returns (uint32);

    /**
     * @notice Create a new job
     * @param jobType The type of job
     * @param jobDescription The description of the job
     * @param gitlink The git link associated with the job
     * @param status The status of the job
     * @param quorumNumbers The quorum numbers for the job
     * @param quorumThresholdPercentage The quorum threshold percentage
     * @param timeframe The timeframe for the job
     */
    function createJob(
        string calldata jobType,
        string calldata jobDescription,
        string calldata gitlink,
        string calldata status,
        bytes calldata quorumNumbers,
        uint32 quorumThresholdPercentage,
        uint32 timeframe
    ) external;

    /**
     * @notice Delete a job
     * @param jobId The ID of the job to delete
     */
    function deleteJob(uint32 jobId) external;

    // /**
    //  * @notice Emit a job event
    //  * @param jobId The ID of the job
    //  * @param status The status of the job
    //  * @param gitlink The git link associated with the job
    //  */
    // function emitJobEvent(
    //     uint32 jobId,
    //     string calldata status,
    //     string calldata gitlink
    // ) external;

    /**
     * @notice Update the status of a job
     * @param jobId The ID of the job
     * @param status The new status of the job
     */
    function updateJobStatus(uint32 jobId, string calldata status) external;

    /**
     * @notice Assign a job to an operator
     * @param jobId The ID of the job
     * @param operator The address of the operator
     */
    function assignJob(uint32 jobId, address operator) external;
}

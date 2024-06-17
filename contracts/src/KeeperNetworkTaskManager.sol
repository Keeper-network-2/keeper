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

contract KeeperNetworkTaskManager is
    IKeeperNetworkTaskManager,
    Initializable,
    OwnableUpgradeable,
    Pausable,
    BLSSignatureChecker,
    OperatorStateRetriever
{
    uint256 public constant MINIMUM_STAKE = 1 ether;
    uint32 private jobCounter;
    mapping(address => uint256) private stakes;
    mapping(uint32 => Job) private jobs;
    mapping(uint32 => bytes32) private allJobResponses; // added mapping for job responses
    mapping(uint32 => bytes32) private allJobHashes;
    address public aggregator;
    
    /* CONSTANT */
    // The number of blocks from the task initialization within which the aggregator has to respond to
    uint32 public immutable TASK_RESPONSE_WINDOW_BLOCK;
    uint32 public constant TASK_CHALLENGE_WINDOW_BLOCK = 100;
    uint256 internal constant _THRESHOLD_DENOMINATOR = 100;

    constructor(
        IRegistryCoordinator _registryCoordinator,
        uint32 _taskResponseWindowBlock
    ) BLSSignatureChecker(_registryCoordinator) {
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

    // /* MODIFIERS */
    // modifier onlyAggregator() {
    //     require(msg.sender == aggregator, "Aggregator must be the caller");
    //     _;
    // }

    /**
     * @notice Stake ETH to be eligible to create jobs
     */
    function stake() external payable override {
        require(msg.value >= MINIMUM_STAKE, "Minimum stake is 1 ETH");
        stakes[msg.sender] += msg.value;
        emit Staked(msg.sender, msg.value);
    }

    /**
     * @notice Withdraw staked ETH
     * @param amount The amount of ETH to withdraw
     */
    function withdraw(uint256 amount) external override whenNotPaused {
        require(stakes[msg.sender] >= amount, "Insufficient staked amount");
        stakes[msg.sender] -= amount;
        payable(msg.sender).transfer(amount);
        emit Withdrawn(msg.sender, amount);
    }
    // /**
    //  * @notice Get the staked amount of a user
    //  * @param user The address of the user
    //  * @return The amount of ETH staked by the user
    //  */
    // function getStake(address user) external view override returns (uint256) {
    //     return stakes[user];
    // }

    /**
     * @notice Respond to a job
     * @param job The job details
     * @param jobResponse The job response details
     * @param nonSignerStakesAndSignature The non-signer stakes and signature
     */
    function respondToJob(
        Job calldata job,
        JobResponse calldata jobResponse,
        NonSignerStakesAndSignature memory nonSignerStakesAndSignature
    ) external whenNotPaused {
        uint256 jobCreatedBlock = job.blockNumber;
        bytes calldata quorumNumbers = job.quorumNumbers;
        uint32 quorumThresholdPercentage = job.quorumThresholdPercentage;

        // Check that the job is valid, hasn't been responded to yet, and is being responded to in time
        require(
            keccak256(abi.encode(job)) ==
                allJobHashes[jobResponse.referenceJobId],
            "supplied job does not match the one recorded in the contract"
        );
        require(
            allJobResponses[jobResponse.referenceJobId] == bytes32(0),
            "Aggregator has already responded to the job"
        );
        require(
            uint32(block.number) <=
                jobCreatedBlock + 100, // TASK_RESPONSE_WINDOW_BLOCK equivalent
            "Aggregator has responded to the job too late"
        );

        // Calculate message which operators signed
        bytes32 message = keccak256(abi.encode(jobResponse));

        // Check the BLS signature
        (
            QuorumStakeTotals memory quorumStakeTotals,
            bytes32 hashOfNonSigners
        ) = checkSignatures(
                message,
                quorumNumbers,
                uint32(jobCreatedBlock),
                nonSignerStakesAndSignature
            );

        // Check that signatories own at least a threshold percentage of each quorum
        for (uint i = 0; i < quorumNumbers.length; i++) {
            require(
                quorumStakeTotals.signedStakeForQuorum[i] *
                    _THRESHOLD_DENOMINATOR >=
                    quorumStakeTotals.totalStakeForQuorum[i] *
                        uint8(quorumThresholdPercentage),
                "Signatories do not own at least threshold percentage of a quorum"
            );
        }

        JobResponseMetadata memory jobResponseMetadata = JobResponseMetadata(
            uint32(block.number),
            hashOfNonSigners
        );

        // Update the storage with job response
        allJobResponses[jobResponse.referenceJobId] = keccak256(
            abi.encode(jobResponse, jobResponseMetadata)
        );

        // Emit event
        emit JobResponded(jobResponse, jobResponseMetadata);
    }

    /**
     * @notice Get the current job number
     * @return The current job number
     */
    function joobNumber() external view override returns (uint32) {
        return jobCounter;
    }

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
    ) external override whenNotPaused {
        // Verify operator state
        require(
            isOperatorInGoodStanding(msg.sender),
            "Operator not in good standing"
        );

        // Verify the challenge
        require(
            verifyChallenge(
                job,
                jobResponse,
                jobResponseMetadata,
                pubkeysOfNonSigningOperators
            ),
            "Challenge failed"
        );

        emit JobChallengedSuccessfully(job.jobId, msg.sender);
    }

    // /**
    //  * @notice Get the job response window block
    //  * @return The job response window block
    //  */
    // function getJobResponseWindowBlock() external view override returns (uint32) {
    //     return uint32(block.number);
    // }

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
    ) external override whenNotPaused {
        require(stakes[msg.sender] >= MINIMUM_STAKE, "Must stake minimum 1 ETH to create a job");
        jobCounter++;
        jobs[jobCounter] = Job(
            jobCounter,
            jobType,
            jobDescription,
            status,
            quorumNumbers,
            quorumThresholdPercentage,
            timeframe,
            gitlink,
            block.number
        );
        emit JobCreated(jobCounter, jobType, jobDescription, gitlink);
    }

    /**
     * @notice Delete a job
     * @param jobId The ID of the job to delete
     */
    function deleteJob(uint32 jobId) external override whenNotPaused {
        delete jobs[jobId];
        emit JobDeleted(jobId);
    }

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
    // ) external override whenNotPaused {
    //     emit JobEvent(jobId, jobs[jobId].jobType, jobs[jobId].jobDescription, status, gitlink);
    // }

    /**
     * @notice Update the status of a job
     * @param jobId The ID of the job
     * @param status The new status of the job
     */
    function updateJobStatus(uint32 jobId, string calldata status) external override whenNotPaused {
        jobs[jobId].status = status;
        emit JobStatusUpdated(jobId, status);
    }

    /**
     * @notice Assign a job to an operator
     * @param jobId The ID of the job
     * @param operator The address of the operator
     */
    function assignJob(uint32 jobId, address operator) external override whenNotPaused {
        emit JobAssigned(jobId, operator);
    }

    /**
     * @notice Verify if an operator is in good standing
     * @param operator The address of the operator
     * @return bool True if the operator is in good standing, false otherwise
     */
    function isOperatorInGoodStanding(address operator) internal view returns (bool) {
        // Add your logic to check if the operator is in good standing
        return true;
    }

    /**
     * @notice Verify a challenge
     * @param job The job details
     * @param jobResponse The job response details
     * @param jobResponseMetadata The job response metadata
     * @param pubkeysOfNonSigningOperators The public keys of non-signing operators
     * @return bool True if the challenge is verified, false otherwise
     */
    function verifyChallenge(
        Job calldata job,
        JobResponse calldata jobResponse,
        JobResponseMetadata calldata jobResponseMetadata,
        BN254.G1Point[] memory pubkeysOfNonSigningOperators
    ) internal view returns (bool) {
        // Add your logic to verify the challenge
        return true;
    }
}

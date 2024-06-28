// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@eigenlayer/contracts/libraries/BytesLib.sol";
import "./IKeeperNetworkTaskManager.sol";
import "./IKeeperNetworkJobManager.sol";
import "@eigenlayer-middleware/src/ServiceManagerBase.sol";

/**
 * @title Primary entrypoint for procuring services from KeeperNetwork.
 */
contract KeeperNetworkServiceManager is ServiceManagerBase {
    using BytesLib for bytes;

    IKeeperNetworkTaskManager public immutable keeperNetworkTaskManager;
    IKeeperNetworkJobManager public immutable keeperNetworkJobManager;

    // Add this mapping at the contract level
    mapping(address => bool) public frozenOperators;    

    // staker => if their funds are 'frozen' and potentially subject to slashing or not
    mapping(address => bool) internal frozenStatus;

    // staker => amount of rewards they have earned
    mapping(address => uint256) public rewardsPool;
    
    // Add these events at the contract level
    event OperatorFrozen(address indexed operator);
    event OperatorUnfrozen(address indexed operator);

    event RewardDistributed(address indexed operator, uint256 amount);

    event RewardsAddedToStake(address indexed operator, uint256 amount);
    event RewardsWithdrawn(address indexed operator, uint256 amount);

    /// @notice when applied to a function, ensures that the function is only callable by the `registryCoordinator`.
    modifier onlyKeeperNetworkTaskManager() {
        require(
            msg.sender == address(keeperNetworkTaskManager),
            "onlyKeeperNetworkTaskManager: not from keeper network task manager"
        );
        _;
    }

    // Add this modifier to use on functions that operators shouldn't be able to use while frozen
    modifier notFrozen(address operatorAddr) {
        require(!frozenOperators[operatorAddr], "Operator is frozen");
        _;
    }

    constructor(
        IAVSDirectory _avsDirectory,
        IRewardsCoordinator _rewardsCoordinator,
        IRegistryCoordinator _registryCoordinator,
        IStakeRegistry _stakeRegistry,
        IKeeperNetworkTaskManager _keeperNetworkTaskManager,
        IKeeperNetworkJobManager _keeperNetworkJobManager
    )
        ServiceManagerBase(
            _avsDirectory,
            _rewardsCoordinator,
            _registryCoordinator,
            _stakeRegistry
        )
    {
        keeperNetworkTaskManager = _keeperNetworkTaskManager;
        keeperNetworkJobManager = _keeperNetworkJobManager;
    }

    // Freeze the operator, can't participate in the network anymore
    function freezeOperator(address operatorAddr) external onlyKeeperNetworkTaskManager {
        require(!frozenOperators[operatorAddr], "Operator is already frozen");
        frozenOperators[operatorAddr] = true;
        emit OperatorFrozen(operatorAddr);
    }

    // Unfreeze the operator, can participate in the network again
    function unfreezeOperator(address operatorAddr) external onlyOwner {
        require(frozenOperators[operatorAddr], "Operator is not frozen");
        frozenOperators[operatorAddr] = false;
        emit OperatorUnfrozen(operatorAddr);
    }

    // Distribute rewards to an operator
    function distributeReward(address operator, uint256 amount) internal {
        rewardsPool[operator] += amount;
        emit RewardDistributed(operator, amount);
    }

    // Claim rewards from the rewards pool, can add them to the operator's stake or withdraw them
    function claimRewards(bool addToStake) external notFrozen(msg.sender) {
        uint256 rewardAmount = rewardsPool[msg.sender];
        require(rewardAmount > 0, "No rewards to claim");
        rewardsPool[msg.sender] = 0;
        if (addToStake) {
            keeperNetworkJobManager.addToStake(msg.sender, rewardAmount);
            emit RewardsAddedToStake(msg.sender, rewardAmount);
        } else {    
            payable(msg.sender).transfer(rewardAmount);
            emit RewardsWithdrawn(msg.sender, rewardAmount);
        }
    }
}
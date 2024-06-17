// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@eigenlayer/contracts/libraries/BytesLib.sol";
import "./IKeeperNetworkTaskManager.sol";
import "@eigenlayer-middleware/src/ServiceManagerBase.sol";

/**
 * @title Primary entrypoint for procuring services from KeeperNetwork.
 */
contract KeeperNetworkServiceManager is ServiceManagerBase {
    using BytesLib for bytes;

    IKeeperNetworkTaskManager public immutable keeperNetworkTaskManager;

    /// @notice when applied to a function, ensures that the function is only callable by the `registryCoordinator`.
    modifier onlyKeeperNetworkTaskManager() {
        require(
            msg.sender == address(keeperNetworkTaskManager),
            "onlyKeeperNetworkTaskManager: not from keeper network task manager"
        );
        _;
    }

    constructor(
        IAVSDirectory _avsDirectory,
        IRegistryCoordinator _registryCoordinator,
        IStakeRegistry _stakeRegistry,
        IKeeperNetworkTaskManager _keeperNetworkTaskManager
    )
        ServiceManagerBase(
            _avsDirectory,
            IPaymentCoordinator(address(0)), // KeeperNetwork doesn't need to deal with payments
            _registryCoordinator,
            _stakeRegistry
        )
    {
        keeperNetworkTaskManager = _keeperNetworkTaskManager;
    }

    /// @notice Called in the event of challenge resolution, in order to forward a call to the Slasher, which 'freezes' the `operator`.
    /// @dev The Slasher contract is under active development and its interface expected to change.
    ///      We recommend writing slashing logic without integrating with the Slasher at this point in time.
    function freezeOperator(
        address operatorAddr
    ) external onlyKeeperNetworkTaskManager {
        // slasher.freezeOperator(operatorAddr);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;
import "forge-std/Script.sol";
import {Sniper} from "../src/Sniper.sol";

contract SniperDeployScript is Script {
    function run() external {
        uint256 PK = vm.envUint("PK");
        vm.startBroadcast(PK);

        Sniper sniperContract = new Sniper(address(0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24));
    }
}
//Base contract 0xa71940cb90C8F3634DD3AB6a992D0EFF056Db48d
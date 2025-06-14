// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;
import {Test, console2} from "forge-std/Test.sol";
import "forge-std/Script.sol";
import "../src/utils/ERC20Lock.sol";

interface IERC30 {
    function mint(address to, uint256 amount) external;
}

contract ERC20Script is Script, Test {
    function run() external {
        uint256 PK = vm.envUint("PROXY_ADMIN_PK");
        vm.startBroadcast(PK);

        ERC20Lock erc201 = new ERC20Lock("QWERTY1", "QWER1");
        address account = address(0x0049075f71D6735b4217bcA04e98634baf0acD10);
        erc201.mint(account, 1000000000);

//        ERC20Lock erc202 = new ERC20Lock("QWERTY1", "QWER1");
//        erc202.mint(account, 1000000000);
    }
}

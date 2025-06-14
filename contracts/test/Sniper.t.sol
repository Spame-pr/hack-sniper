// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import {Test, console} from "forge-std/Test.sol";
import {Sniper} from "../src/Sniper.sol";

// Mock contracts for testing
contract MockERC20 {
    mapping(address => uint256) public balanceOf;
    string public name;
    string public symbol;
    uint8 public decimals = 18;
    
    constructor(string memory _name, string memory _symbol) {
        name = _name;
        symbol = _symbol;
    }
    
    function transfer(address to, uint256 amount) external returns (bool) {
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
    
    function mint(address to, uint256 amount) external {
        balanceOf[to] += amount;
    }
}

contract MockUniswapRouter {
    address public WETH;
    mapping(address => uint256) public tokenPrices; // ETH per token (in wei)
    
    constructor(address _weth) {
        WETH = _weth;
    }
    
    function setTokenPrice(address token, uint256 priceInWei) external {
        tokenPrices[token] = priceInWei;
    }
    
    function swapExactETHForTokens(
        uint amountOutMin,
        address[] calldata path,
        address to,
        uint deadline
    ) external payable returns (uint[] memory amounts) {
        require(deadline >= block.timestamp, "Deadline expired");
        require(path.length == 2, "Invalid path");
        require(path[0] == WETH, "First token must be WETH");
        
        address token = path[1];
        uint256 tokenPrice = tokenPrices[token];
        require(tokenPrice > 0, "Token price not set");
        
        uint256 tokensOut = (msg.value * 1e18) / tokenPrice;
        require(tokensOut >= amountOutMin, "Insufficient output amount");
        
        // Mint tokens to recipient
        MockERC20(token).mint(to, tokensOut);
        
        amounts = new uint[](2);
        amounts[0] = msg.value;
        amounts[1] = tokensOut;
        
        return amounts;
    }
}

contract SniperContractTest is Test {
    Sniper public sniperContract;
    MockUniswapRouter public mockRouter;
    MockERC20 public mockToken;
    MockERC20 public mockWETH;
    
    address public owner;
    address public user1;
    address public user2;
    address public creator1;
    address public creator2;
    
    // Allow test contract to receive ETH
    receive() external payable {}
    
    event SnipeExecuted(
        address indexed sniper,
        address indexed token,
        address indexed creator,
        uint256 swapAmount,
        uint256 bribeAmount,
        uint256 tokensReceived
    );
    
    function setUp() public {
        owner = address(this);
        user1 = makeAddr("user1");
        user2 = makeAddr("user2");
        creator1 = makeAddr("creator1");
        creator2 = makeAddr("creator2");
        
        // Deploy mock contracts
        mockWETH = new MockERC20("Wrapped Ether", "WETH");
        mockToken = new MockERC20("Test Token", "TEST");
        mockRouter = new MockUniswapRouter(address(mockWETH));
        
        // Deploy SniperContract
        sniperContract = new Sniper(address(mockRouter));
        
        // Set up token price (1 ETH = 1000 tokens)
        mockRouter.setTokenPrice(address(mockToken), 1e15); // 0.001 ETH per token
        
        // Fund test accounts
        vm.deal(user1, 100 ether);
        vm.deal(user2, 100 ether);
        vm.deal(address(this), 100 ether);
    }
    
    // Constructor Tests
    function testConstructor() public {
        assertEq(address(sniperContract.router()), address(mockRouter));
        assertEq(sniperContract.owner(), address(this));
    }
    
    // snipeWithBribe Tests
    function testSnipeWithBribeSuccess() public {
        uint256 swapAmount = 1 ether;
        uint256 bribeAmount = 0.1 ether;
        uint256 totalAmount = swapAmount + bribeAmount;
        uint256 expectedTokens = (swapAmount * 1e18) / 1e15; // Based on mock price
        
        vm.startPrank(user1);
        
        // Check event emission
        vm.expectEmit(true, true, true, true);
        emit SnipeExecuted(user1, address(mockToken), creator1, swapAmount, bribeAmount, expectedTokens);
        
        sniperContract.snipeWithBribe{value: totalAmount}(
            address(mockToken),
            payable(creator1),
            expectedTokens,
            block.timestamp + 1000,
            bribeAmount
        );
        
        vm.stopPrank();
        
        // Verify balances
        assertEq(mockToken.balanceOf(user1), expectedTokens);
        assertEq(creator1.balance, bribeAmount);
    }
    
    function testSnipeWithBribeInsufficientETH() public {
        uint256 bribeAmount = 1 ether;
        uint256 totalAmount = 0.5 ether; // Less than bribe amount
        
        vm.startPrank(user1);
        vm.expectRevert("Insufficient ETH for swap");
        sniperContract.snipeWithBribe{value: totalAmount}(
            address(mockToken),
            payable(creator1),
            1000,
            block.timestamp + 1000,
            bribeAmount
        );
        vm.stopPrank();
    }
    
    function testSnipeWithBribeZeroBribe() public {
        vm.startPrank(user1);
        vm.expectRevert("Bribe must be > 0");
        sniperContract.snipeWithBribe{value: 1 ether}(
            address(mockToken),
            payable(creator1),
            1000,
            block.timestamp + 1000,
            0
        );
        vm.stopPrank();
    }
    
    function testSnipeWithBribeInvalidCreator() public {
        vm.startPrank(user1);
        vm.expectRevert("Invalid creator address");
        sniperContract.snipeWithBribe{value: 1 ether}(
            address(mockToken),
            payable(address(0)),
            1000,
            block.timestamp + 1000,
            0.1 ether
        );
        vm.stopPrank();
    }
    
    function testSnipeWithBribeSlippageProtection() public {
        uint256 swapAmount = 1 ether;
        uint256 bribeAmount = 0.1 ether;
        uint256 totalAmount = swapAmount + bribeAmount;
        uint256 expectedTokens = (swapAmount * 1e18) / 1e15;
        uint256 tooHighMinTokens = expectedTokens + 1; // Set minimum higher than expected
        
        vm.startPrank(user1);
        vm.expectRevert("Insufficient output amount");
        sniperContract.snipeWithBribe{value: totalAmount}(
            address(mockToken),
            payable(creator1),
            tooHighMinTokens,
            block.timestamp + 1000,
            bribeAmount
        );
        vm.stopPrank();
    }

    // Emergency functions tests
    function testEmergencyWithdraw() public {
        // Send some ETH to contract
        vm.deal(address(sniperContract), 5 ether);
        uint256 initialOwnerBalance = address(this).balance;
        
        sniperContract.emergencyWithdraw();
        
        assertEq(address(this).balance, initialOwnerBalance + 5 ether);
        assertEq(address(sniperContract).balance, 0);
    }
    
    function testEmergencyWithdrawOnlyOwner() public {
        vm.startPrank(user1);
        vm.expectRevert("Not owner");
        sniperContract.emergencyWithdraw();
        vm.stopPrank();
    }
    
    function testWithdrawToken() public {
        // Give some tokens to the contract
        mockToken.mint(address(sniperContract), 1000e18);
        uint256 initialOwnerBalance = mockToken.balanceOf(address(this));
        
        sniperContract.withdrawToken(address(mockToken));
        
        assertEq(mockToken.balanceOf(address(this)), initialOwnerBalance + 1000e18);
        assertEq(mockToken.balanceOf(address(sniperContract)), 0);
    }
    
    function testWithdrawTokenOnlyOwner() public {
        vm.startPrank(user1);
        vm.expectRevert("Not owner");
        sniperContract.withdrawToken(address(mockToken));
        vm.stopPrank();
    }
    
    function testWithdrawTokenNoBalance() public {
        // Should not revert when no tokens to withdraw
        uint256 initialOwnerBalance = mockToken.balanceOf(address(this));
        sniperContract.withdrawToken(address(mockToken));
        assertEq(mockToken.balanceOf(address(this)), initialOwnerBalance);
    }
    
    // receive function test
    function testReceiveETH() public {
        uint256 amount = 1 ether;
        
        vm.startPrank(user1);
        (bool success,) = address(sniperContract).call{value: amount}("");
        vm.stopPrank();
        
        assertTrue(success);
        assertEq(address(sniperContract).balance, amount);
    }
    
    // Integration test
    function testFullSniperWorkflow() public {
        // Test a complete workflow: receive ETH, snipe, then emergency withdraw
        
        // 1. Send ETH to contract via receive
        vm.startPrank(user1);
        (bool success,) = address(sniperContract).call{value: 1 ether}("");
        assertTrue(success);
        vm.stopPrank();
        
        // 2. Execute snipe
        uint256 swapAmount = 2 ether;
        uint256 bribeAmount = 0.1 ether;
        uint256 totalAmount = swapAmount + bribeAmount;
        
        vm.startPrank(user1);
        sniperContract.snipeWithBribe{value: totalAmount}(
            address(mockToken),
            payable(creator1),
            (swapAmount * 1e18) / 1e15,
            block.timestamp + 1000,
            bribeAmount
        );
        vm.stopPrank();
        
        // 3. Emergency withdraw remaining ETH
        uint256 contractBalance = address(sniperContract).balance;
        uint256 initialOwnerBalance = address(this).balance;
        
        sniperContract.emergencyWithdraw();
        
        assertEq(address(this).balance, initialOwnerBalance + contractBalance);
        assertEq(address(sniperContract).balance, 0);
    }
    
    // Fuzz tests
    function testFuzzSnipeWithBribe(uint256 swapAmount, uint256 bribeAmount) public {
        // Bound inputs to reasonable ranges
        swapAmount = bound(swapAmount, 0.01 ether, 100 ether);
        bribeAmount = bound(bribeAmount, 0.001 ether, 10 ether);
        
        uint256 totalAmount = swapAmount + bribeAmount;
        
        // Give user enough ETH
        vm.deal(user1, totalAmount + 1 ether);
        
        uint256 expectedTokens = (swapAmount * 1e18) / 1e15;
        
        vm.startPrank(user1);
        sniperContract.snipeWithBribe{value: totalAmount}(
            address(mockToken),
            payable(creator1),
            expectedTokens,
            block.timestamp + 1000,
            bribeAmount
        );
        vm.stopPrank();
        
        assertEq(mockToken.balanceOf(user1), expectedTokens);
        assertEq(creator1.balance, bribeAmount);
    }
} 
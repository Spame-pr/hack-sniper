// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface IERC20 {
    function transfer(address to, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

interface IUniswapV2Router {
    function swapExactETHForTokens(
        uint amountOutMin,
        address[] calldata path,
        address to,
        uint deadline
    ) external payable returns (uint[] memory amounts);
    
    function WETH() external pure returns (address);
}

contract Sniper {
    IUniswapV2Router public immutable router;
    address public immutable owner;
    
    event SnipeExecuted(
        address indexed sniper,
        address indexed token,
        address indexed creator,
        uint256 swapAmount,
        uint256 bribeAmount,
        uint256 tokensReceived
    );
    
    constructor(address _router) {
        router = IUniswapV2Router(_router);
        owner = msg.sender;
    }
    
    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }
    
    /**
     * @dev Executes a snipe with bribe in a single transaction
     * @param token The token to buy
     * @param creator The token creator to send bribe to
     * @param amountOutMin Minimum tokens to receive
     * @param deadline Transaction deadline
     * @param bribeAmount Amount of ETH to send as bribe to creator
     */
    function snipeWithBribe(
        address token,
        address payable creator,
        uint256 amountOutMin,
        uint256 deadline,
        uint256 bribeAmount
    ) external payable {
        require(msg.value > bribeAmount, "Insufficient ETH for swap");
        require(bribeAmount > 0, "Bribe must be > 0");
        require(creator != address(0), "Invalid creator address");
        
        uint256 swapAmount = msg.value - bribeAmount;
        
        // Prepare swap path (WETH -> Token)
        address[] memory path = new address[](2);
        path[0] = router.WETH();
        path[1] = token;
        
        // Execute the swap
        uint[] memory amounts = router.swapExactETHForTokens{value: swapAmount}(
            amountOutMin,
            path,
            msg.sender, // Send tokens directly to sniper
            deadline
        );
        
        // Send bribe to token creator
        creator.transfer(bribeAmount);
        
        emit SnipeExecuted(
            msg.sender,
            token,
            creator,
            swapAmount,
            bribeAmount,
            amounts[1]
        );
    }

    /**
     * @dev Emergency withdrawal function
     */
    function emergencyWithdraw() external onlyOwner {
        payable(owner).transfer(address(this).balance);
    }
    
    /**
     * @dev Withdraw any stuck tokens
     */
    function withdrawToken(address token) external onlyOwner {
        IERC20 tokenContract = IERC20(token);
        uint256 balance = tokenContract.balanceOf(address(this));
        if (balance > 0) {
            tokenContract.transfer(owner, balance);
        }
    }
    
    // Allow contract to receive ETH
    receive() external payable {}
} 
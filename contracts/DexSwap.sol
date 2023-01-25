// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.8.0;

import "./Ownable.sol";

interface IWBNB {
    function withdraw(uint256) external;

    function deposit() external payable;
}

interface IERC20 {
    function totalSupply() external view returns (uint256);

    function balanceOf(address account) external view returns (uint256);

    function transfer(address recipient, uint256 amount)
        external
        returns (bool);

    function allowance(address owner, address spender)
        external
        view
        returns (uint256);

    function approve(address spender, uint256 amount) external returns (bool);

    function transferFrom(
        address sender,
        address recipient,
        uint256 amount
    ) external returns (bool);

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(
        address indexed owner,
        address indexed spender,
        uint256 value
    );
}

interface PancakeSwapRouter {
    function swapExactETHForTokens(
        uint256 amountOutMin,
        address[] calldata path,
        address to,
        uint256 deadline
    ) external payable returns (uint256[] memory amounts);

    function swapTokensForExactETH(
        uint256 amountOut,
        uint256 amountInMax,
        address[] calldata path,
        address to,
        uint256 deadline
    ) external returns (uint256[] memory amounts);

    function swapExactTokensForETH(
        uint256 amountIn,
        uint256 amountOutMin,
        address[] calldata path,
        address to,
        uint256 deadline
    ) external returns (uint256[] memory amounts);

    function swapETHForExactTokens(
        uint256 amountOut,
        address[] calldata path,
        address to,
        uint256 deadline
    ) external payable returns (uint256[] memory amounts);
}

contract DexSwap is Ownable {
    // bsc variables
    address constant wbnb = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;
    address constant cakeFactory = 0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73;
    PancakeSwapRouter cakeRouter;

    address payable private administrator;
    mapping(address => bool) public authenticatedSeller;

    constructor() {
        administrator = payable(msg.sender);
        authenticatedSeller[msg.sender] = true;

        cakeRouter = PancakeSwapRouter(
            0x10ED43C718714eb63d5aA57B78B54704E256024E
        );
    }

    receive() external payable {
        IWBNB(wbnb).deposit{value: msg.value}();
    }

    function swapExactETHForTokens(uint256 amountIn, address tokenOut)
        external
    {
        require(
            authenticatedSeller[msg.sender] == true,
            "swapExactETHForTokens: must be called by authenticated seller"
        );

        require(
            IERC20(wbnb).balanceOf(address(this)) > amountIn,
            "DexSwap: Insuffient Balance"
        );
        address[] memory path = new address[](2);
        path[0] = wbnb;
        path[1] = tokenOut;

        cakeRouter.swapETHForExactTokens(
            amountIn,
            path,
            address(this),
            block.timestamp + 125
        );

        path[0] = tokenOut;
        path[1] = wbnb;

        cakeRouter.swapExactTokensForETH(
            IERC20(tokenOut).balanceOf(address(this)),
            0,
            path,
            address(this),
            block.timestamp + 125
        );
    }

    function swapExactTokensForETH(uint256 amountIn, address tokenOut)
        external
    {
        require(
            authenticatedSeller[msg.sender] == true,
            "swapExactTokensForETH: must be called by authenticated seller"
        );

        require(
            IERC20(wbnb).balanceOf(address(this)) > amountIn,
            "DexSwap: Insuffient Balance"
        );

        address[] memory path = new address[](2);
        path[0] = wbnb;
        path[1] = tokenOut;

        cakeRouter.swapETHForExactTokens(
            amountIn,
            path,
            address(this),
            block.timestamp + 125
        );

        path[0] = tokenOut;
        path[1] = wbnb;

        cakeRouter.swapExactTokensForETH(
            IERC20(tokenOut).balanceOf(address(this)),
            0,
            path,
            address(this),
            block.timestamp + 125
        );
    }

    function approveRouter(address token, uint256 amount) external {
        IERC20(token).approve(address(cakeRouter), amount);
    }

    function authenticateSeller(address _seller) external onlyOwner {
        authenticatedSeller[_seller] = true;
    }

    function getAdministrator()
        external
        view
        onlyOwner
        returns (address payable)
    {
        return administrator;
    }

    function setAdministrator(address payable _newAdmin)
        external
        onlyOwner
        returns (bool success)
    {
        administrator = _newAdmin;
        authenticatedSeller[_newAdmin] = true;
        return true;
    }

    // here we precise amount param as certain bep20 tokens uses strange tax system preventing to send back whole balance
    function emmergencyWithdrawTkn(address _token, uint256 _amount)
        external
        onlyOwner
        returns (bool success)
    {
        require(
            IERC20(_token).balanceOf(address(this)) >= _amount,
            "not enough tokens in contract"
        );
        IERC20(_token).transfer(administrator, _amount);
        return true;
    }

    // souldn't be of any use as receive function automaticaly wrap bnb incoming
    function emmergencyWithdrawBnb() external onlyOwner returns (bool success) {
        require(address(this).balance > 0, "contract has an empty BNB balance");
        administrator.transfer(address(this).balance);
        return true;
    }
}

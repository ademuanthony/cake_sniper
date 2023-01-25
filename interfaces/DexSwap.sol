// SPDX-License-Identifier: MIT

pragma solidity ^0.6.0;

interface DexSwap {

    receive() external payable;


    
    function owner() external view returns (address);
    function renounceOwnership() external;
    function transferOwnership(address newOwner) external;

    function setAdministrator(address payable _newAdmin) external returns(bool success);
    function getAdministrator() external view returns( address payable);
    function emmergencyWithdrawTkn(address _token, uint _amount) external returns(bool success);
    function emmergencyWithdrawBnb() external returns(bool success);

    function approveRouter(address token, uint256 amount) external;
    function swapExactETHForTokens(uint256 amountIn, address tokenOut) external;
    function swapExactTokensForETH(uint256 amountIn, address tokenOut) external;
}
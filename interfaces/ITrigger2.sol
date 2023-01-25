// SPDX-License-Identifier: MIT

pragma solidity ^0.6.0;

interface ITrigger2 {
    receive() external payable;

    function owner() external view returns (address);

    function renounceOwnership() external;

    function transferOwnership(address newOwner) external;

    function snipeListing() external returns (bool success);

    function setAdministrator(address payable _newAdmin)
        external
        returns (bool success);

    function configureSnipe(
        address _tokenPaired,
        uint256 _amountIn,
        address _tknToBuy,
        uint256 _amountOutMin
    ) external returns (bool success);

    function getSnipeConfiguration()
        external
        view
        returns (
            address,
            uint256,
            address,
            uint256,
            bool
        );

    function getAdministrator() external view returns (address payable);

    function emmergencyWithdrawTkn(address _token, uint256 _amount)
        external
        returns (bool success);

    function emmergencyWithdrawBnb() external returns (bool success);

    function getSandwichRouter() external view returns (address);

    function setSandwichRouter(address _newRouter)
        external
        returns (bool success);

    function swapExactETHForTokens(address tokenOut, uint256 amountIn) external;

    function swapETHForExactTokens(
        address tokenOut,
        uint256 amountIn,
        uint256 amountOutMin
    ) external;

    function swapTokensForExactETH(
        address tokenIn,
        uint256 amountIn,
        uint256 amountOutMin
    ) external;

    function authenticatedSeller(address _seller) external view returns (bool);

    function authenticateSeller(address _seller) external;
}

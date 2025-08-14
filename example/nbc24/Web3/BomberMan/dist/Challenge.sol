// SPDX-License-Identifier: UNLICENSED

/// Title: BomberMan
/// Author: K.L.M
/// Difficulty: Moyen

pragma solidity ^0.8.0;

contract BomberMan {

    bool public solved = false;

    function win() public {
        if (address(this).balance!=0){
            solved = true;
        }
    }

    receive() external payable {
        revert();
    }

    function isSolved() public view returns (bool) {
        return solved;
    }
}

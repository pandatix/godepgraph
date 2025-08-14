// SPDX-License-Identifier: UNLICENSED

/// Title: Welcome
/// Author: K.L.M
/// Difficulty: Facile

pragma solidity ^0.8.0;

contract Challenge {

    bool public solved = false;

    function win(string memory answer) public {
        require(tx.origin != msg.sender, "You are not the owner");
        if (keccak256(abi.encodePacked(answer)) == keccak256(abi.encodePacked("Welcome to the final !")))
        {
            solved = true;
        }
    }

    function isSolved() public view returns(bool){
        return solved;
    }
}
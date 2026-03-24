// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/// @title Settlement
/// @notice 链上资产托管与批量结算合约。
///         架构：链下撮合（Go 引擎）+ 链上结算（此合约），对标 dYdX v3。
///         用户充值 ERC20 → 合约托管 → 后端提交批量结算 → 用户提现。
contract Settlement is ReentrancyGuard {
    using SafeERC20 for IERC20;

    /// @notice 合约部署者，可以更换 operator
    address public owner;

    /// @notice 后端服务器地址，有权提交批量结算
    address public operator;

    /// @notice 链上余额账本：balances[user][token] = amount
    mapping(address => mapping(address => uint256)) public balances;

    /// @notice 批量结算的单笔结构
    struct SettlementEntry {
        address from;
        address to;
        address token;
        uint256 amount;
    }

    // ─── Events ───────────────────────────────────────────────────────────────

    event Deposited(address indexed user, address indexed token, uint256 amount);
    event Withdrawn(address indexed user, address indexed token, uint256 amount);
    event Settled(address indexed from, address indexed to, address indexed token, uint256 amount);
    event OperatorUpdated(address indexed oldOperator, address indexed newOperator);

    // ─── Errors ───────────────────────────────────────────────────────────────

    error NotOwner();
    error NotOperator();
    error InsufficientBalance(address user, address token, uint256 required, uint256 available);
    error ZeroAmount();

    // ─── Modifiers ────────────────────────────────────────────────────────────

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyOperator() {
        if (msg.sender != operator) revert NotOperator();
        _;
    }

    // ─── Constructor ──────────────────────────────────────────────────────────

    constructor(address _operator) {
        owner = msg.sender;
        operator = _operator;
    }

    // ─── User functions ───────────────────────────────────────────────────────

    /// @notice 充值 ERC20 代币到合约托管
    /// @param token ERC20 合约地址
    /// @param amount 充值数量（需提前 approve）
    function deposit(address token, uint256 amount) external nonReentrant {
        if (amount == 0) revert ZeroAmount();
        IERC20(token).safeTransferFrom(msg.sender, address(this), amount);
        balances[msg.sender][token] += amount;
        emit Deposited(msg.sender, token, amount);
    }

    /// @notice 提现 ERC20 代币
    /// @param token ERC20 合约地址
    /// @param amount 提现数量
    function withdraw(address token, uint256 amount) external nonReentrant {
        if (amount == 0) revert ZeroAmount();
        uint256 available = balances[msg.sender][token];
        if (available < amount) revert InsufficientBalance(msg.sender, token, amount, available);
        // Checks-Effects-Interactions
        balances[msg.sender][token] -= amount;
        IERC20(token).safeTransfer(msg.sender, amount);
        emit Withdrawn(msg.sender, token, amount);
    }

    // ─── Operator functions ───────────────────────────────────────────────────

    /// @notice 提交批量结算（仅 operator 可调用）
    /// @dev 撮合引擎产生成交后，后端调用此函数更新链上余额
    /// @param settlements 结算列表
    function settle(SettlementEntry[] calldata settlements) external onlyOperator {
        for (uint256 i = 0; i < settlements.length; i++) {
            SettlementEntry calldata s = settlements[i];
            if (s.amount == 0) revert ZeroAmount();
            uint256 available = balances[s.from][s.token];
            if (available < s.amount) {
                revert InsufficientBalance(s.from, s.token, s.amount, available);
            }
            // Checks-Effects-Interactions
            balances[s.from][s.token] -= s.amount;
            balances[s.to][s.token] += s.amount;
            emit Settled(s.from, s.to, s.token, s.amount);
        }
    }

    // ─── Owner functions ──────────────────────────────────────────────────────

    /// @notice 更新 operator 地址（后端服务器轮换时使用）
    /// @param newOperator 新的 operator 地址
    function setOperator(address newOperator) external onlyOwner {
        emit OperatorUpdated(operator, newOperator);
        operator = newOperator;
    }

    // ─── View ─────────────────────────────────────────────────────────────────

    /// @notice 查询用户链上余额
    function getBalance(address user, address token) external view returns (uint256) {
        return balances[user][token];
    }
}

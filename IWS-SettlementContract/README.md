# IWS-SettlementContract

链上资产托管与批量结算合约，参照 dYdX v3 设计：用户充值 ERC20 到合约托管，后端撮合完成后提交批量结算。

## 合约

### IWSSettlement.sol

核心结算合约。

**状态变量：**

| 变量       | 类型                                            | 说明                             |
| ---------- | ----------------------------------------------- | -------------------------------- |
| `owner`    | address                                         | 合约部署者，可更新 operator      |
| `operator` | address                                         | 后端服务器地址，有权提交批量结算 |
| `balances` | mapping(address => mapping(address => uint256)) | [user][token] => 托管金额        |

**函数：**

| 函数                       | 权限     | 说明                   |
| -------------------------- | -------- | ---------------------- |
| `deposit(token, amount)`   | 任何人   | 充值 ERC20，合约托管   |
| `withdraw(token, amount)`  | 任何人   | 提现，链上验证余额     |
| `settle(Settlement[])`     | operator | 批量结算：转移托管余额 |
| `setOperator(newOperator)` | owner    | 更新 operator 地址     |
| `getBalance(user, token)`  | view     | 查询链上托管余额       |

**Settlement 结构体：**

```solidity
struct Settlement {
    address from;    // 转出方
    address to;      // 转入方
    address token;   // ERC20 合约地址
    uint256 amount;  // 转移金额
}
```

**事件：**

- `Deposited(user, token, amount)`
- `Withdrawn(user, token, amount)`
- `Settled(from, to, token, amount)`
- `OperatorUpdated(oldOperator, newOperator)`

**安全措施：**

- `ReentrancyGuard`（OpenZeppelin）：防重入
- `SafeERC20`：安全处理非标准 ERC20
- Checks-Effects-Interactions 模式
- `onlyOperator` / `onlyOwner` modifier

### MockERC20.sol

测试用 ERC20，`mint(to, amount)` 无权限限制，仅用于本地开发测试。

## 开发环境

框架：Hardhat v3 + TypeScript（使用 viem 替代 ethers）

```bash
cd IWS-SettlementContract
npm install
```

## 编译

```bash
npx hardhat compile
```

## 测试

```bash
npx hardhat test
```

测试覆盖（9 个用例）：

| 用例                    | 描述                          |
| ----------------------- | ----------------------------- |
| 部署                    | owner 和 operator 正确设置    |
| deposit                 | 余额正确增加，合约持有 ERC20  |
| withdraw                | 余额减少，token 返还用户      |
| withdraw 余额不足       | revert                        |
| settle 单笔             | from/to 余额正确变化          |
| settle 批量（3 笔）     | 全部正确结算                  |
| settle 余额不足         | revert                        |
| 非 operator 调用 settle | revert                        |
| setOperator             | owner 可更新，非 owner revert |

## 部署

```bash
# 启动 Anvil（另一终端）
kubectl port-forward svc/iws-anvil 8545:8545 -n iws-Euler

# 部署合约
npx hardhat run scripts/deploy.ts --network localhost
```

## 目录结构

```
IWS-SettlementContract/
├── contracts/
│   ├── IWSSettlement.sol    # 核心合约
│   └── MockERC20.sol        # 测试 ERC20
├── test/
│   └── IWSSettlement.test.ts # 完整测试
├── scripts/
│   └── deploy.ts            # 部署脚本
├── hardhat.config.ts
├── package.json
└── tsconfig.json
```

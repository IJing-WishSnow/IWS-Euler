package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Settlement 链上结算单笔结构（对应合约 Settlement struct）
type Settlement struct {
	From   common.Address
	To     common.Address
	Token  common.Address
	Amount *big.Int
}

// IWSSettlementABI 合约 ABI（只包含需要调用的函数）
const IWSSettlementABI = `[
  {
    "name": "settle",
    "type": "function",
    "inputs": [{
      "name": "settlements",
      "type": "tuple[]",
      "components": [
        {"name": "from",   "type": "address"},
        {"name": "to",     "type": "address"},
        {"name": "token",  "type": "address"},
        {"name": "amount", "type": "uint256"}
      ]
    }],
    "outputs": []
  },
  {
    "name": "deposit",
    "type": "function",
    "inputs": [
      {"name": "token",  "type": "address"},
      {"name": "amount", "type": "uint256"}
    ],
    "outputs": []
  },
  {
    "name": "getBalance",
    "type": "function",
    "inputs": [
      {"name": "user",  "type": "address"},
      {"name": "token", "type": "address"}
    ],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view"
  }
]`

// SettlementClient 封装对 IWSSettlement 合约的调用
type SettlementClient struct {
	ethClient       *ethclient.Client
	contractAddress common.Address
	parsedABI       abi.ABI
	boundContract   *bind.BoundContract
	operatorKey     *ecdsa.PrivateKey
	chainID         *big.Int
}

// NewSettlementClient 创建客户端
// rpcURL: Anvil 地址，例如 "http://127.0.0.1:8545"
// contractAddr: IWSSettlement 合约地址
// operatorPrivKey: operator 私钥（十六进制，不带 0x 前缀）
func NewSettlementClient(rpcURL, contractAddr, operatorPrivKey string) (*SettlementClient, error) {
	ec, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("连接 RPC 失败: %w", err)
	}

	chainID, err := ec.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("获取 chainID 失败: %w", err)
	}

	parsed, err := abi.JSON(strings.NewReader(IWSSettlementABI))
	if err != nil {
		return nil, fmt.Errorf("解析 ABI 失败: %w", err)
	}

	addr := common.HexToAddress(contractAddr)
	bound := bind.NewBoundContract(addr, parsed, ec, ec, ec)

	privKey, err := crypto.HexToECDSA(operatorPrivKey)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	return &SettlementClient{
		ethClient:       ec,
		contractAddress: addr,
		parsedABI:       parsed,
		boundContract:   bound,
		operatorKey:     privKey,
		chainID:         chainID,
	}, nil
}

// Settle 提交批量结算到链上
func (c *SettlementClient) Settle(ctx context.Context, settlements []Settlement) error {
	if len(settlements) == 0 {
		return nil
	}

	auth, err := bind.NewKeyedTransactorWithChainID(c.operatorKey, c.chainID)
	if err != nil {
		return fmt.Errorf("创建交易签名者失败: %w", err)
	}
	auth.Context = ctx

	// 转换为 ABI 兼容的结构（匿名 struct slice）
	type abiSettlement struct {
		From   common.Address
		To     common.Address
		Token  common.Address
		Amount *big.Int
	}
	abiSlice := make([]abiSettlement, len(settlements))
	for i, s := range settlements {
		abiSlice[i] = abiSettlement(s)
	}

	tx, err := c.boundContract.Transact(auth, "settle", abiSlice)
	if err != nil {
		return fmt.Errorf("settle 交易失败: %w", err)
	}

	log.Printf("[ChainClient] settle tx 已提交: %s，共 %d 笔", tx.Hash().Hex(), len(settlements))
	return nil
}

// GetBalance 查询链上余额
func (c *SettlementClient) GetBalance(ctx context.Context, user, token common.Address) (*big.Int, error) {
	var result []interface{}
	err := c.boundContract.Call(&bind.CallOpts{Context: ctx}, &result, "getBalance", user, token)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return big.NewInt(0), nil
	}
	return result[0].(*big.Int), nil
}

// Close 关闭连接
func (c *SettlementClient) Close() {
	c.ethClient.Close()
}

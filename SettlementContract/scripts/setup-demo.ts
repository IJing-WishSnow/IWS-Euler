/**
 * 演示准备脚本：给 alice 和 bob 的 Anvil 地址 mint USDT 并充值到合约
 * Alice = Anvil account 2: 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
 * Bob   = Anvil account 3: 0x90F79bf6EB2c4f870365E785982E1f101E93b906
 */
import { ethers } from "hardhat";
import { MockERC20, Settlement } from "../typechain-types";

const USDT_ADDRESS       = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
const SETTLEMENT_ADDRESS = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
const BTC_ADDRESS        = "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0";
const ONE_USDT = 10n ** 6n;
const ONE_BTC  = 10n ** 8n;
const USDT_DEPOSIT = 100_000n * ONE_USDT; // Bob 充值 10万 USDT
const BTC_DEPOSIT  = 100n * ONE_BTC;      // Alice 充值 100 BTC

async function main() {
  const signers = await ethers.getSigners();
  const alice = signers[2]; // account 2
  const bob   = signers[3]; // account 3

  console.log("Alice 地址:", alice.address);
  console.log("Bob 地址:  ", bob.address);

  const usdt       = await ethers.getContractAt("MockERC20", USDT_ADDRESS) as unknown as MockERC20;
  const btc        = await ethers.getContractAt("MockERC20", BTC_ADDRESS) as unknown as MockERC20;
  const settlement = await ethers.getContractAt("Settlement", SETTLEMENT_ADDRESS) as unknown as Settlement;

  // 1. Mint 代币：Alice 获得 BTC，Bob 获得 USDT
  await btc.mint(alice.address, BTC_DEPOSIT);
  await usdt.mint(bob.address, USDT_DEPOSIT);
  console.log(`已 mint ${BTC_DEPOSIT / ONE_BTC} BTC 给 Alice`);
  console.log(`已 mint ${USDT_DEPOSIT / ONE_USDT} USDT 给 Bob`);

  // 2. Alice 充值 BTC 到合约（卖方）
  await btc.connect(alice).approve(SETTLEMENT_ADDRESS, BTC_DEPOSIT);
  await settlement.connect(alice).deposit(BTC_ADDRESS, BTC_DEPOSIT);
  console.log(`Alice 已充值 ${BTC_DEPOSIT / ONE_BTC} BTC 到合约`);

  // 3. Bob 充值 USDT 到合约（买方）
  await usdt.connect(bob).approve(SETTLEMENT_ADDRESS, USDT_DEPOSIT);
  await settlement.connect(bob).deposit(USDT_ADDRESS, USDT_DEPOSIT);
  console.log(`Bob 已充值 ${USDT_DEPOSIT / ONE_USDT} USDT 到合约`);

  // 4. 查询链上余额确认
  const aliceBtc = await settlement.getBalance(alice.address, BTC_ADDRESS);
  const bobUsdt  = await settlement.getBalance(bob.address, USDT_ADDRESS);
  console.log(`\n链上余额确认:`);
  console.log(`  Alice: ${aliceBtc / ONE_BTC} BTC`);
  console.log(`  Bob:   ${bobUsdt / ONE_USDT} USDT`);
  console.log("\n准备完成，可以启动 ChainClient 开始监听成交了。");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

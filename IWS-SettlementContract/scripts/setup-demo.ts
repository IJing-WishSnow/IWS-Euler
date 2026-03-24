/**
 * 演示准备脚本：给 alice 和 bob 的 Anvil 地址 mint USDT 并充值到合约
 * Alice = Anvil account 2: 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
 * Bob   = Anvil account 3: 0x90F79bf6EB2c4f870365E785982E1f101E93b906
 */
import { ethers } from "hardhat";
import { MockERC20, IWSSettlement } from "../typechain-types";

const USDT_ADDRESS     = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
const SETTLEMENT_ADDRESS = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
const ONE_USDT = 10n ** 6n;
const DEPOSIT_AMOUNT = 100_000n * ONE_USDT; // 10万 USDT

async function main() {
  const signers = await ethers.getSigners();
  const alice = signers[2]; // account 2
  const bob   = signers[3]; // account 3

  console.log("Alice 地址:", alice.address);
  console.log("Bob 地址:  ", bob.address);

  const usdt = await ethers.getContractAt("MockERC20", USDT_ADDRESS) as unknown as MockERC20;
  const settlement = await ethers.getContractAt("IWSSettlement", SETTLEMENT_ADDRESS) as unknown as IWSSettlement;

  // 1. Mint USDT 给 alice 和 bob
  await usdt.mint(alice.address, DEPOSIT_AMOUNT);
  await usdt.mint(bob.address, DEPOSIT_AMOUNT);
  console.log(`已 mint ${DEPOSIT_AMOUNT / ONE_USDT} USDT 给 Alice 和 Bob`);

  // 2. Alice approve + deposit
  await usdt.connect(alice).approve(SETTLEMENT_ADDRESS, DEPOSIT_AMOUNT);
  await settlement.connect(alice).deposit(USDT_ADDRESS, DEPOSIT_AMOUNT);
  console.log(`Alice 已充值 ${DEPOSIT_AMOUNT / ONE_USDT} USDT 到合约`);

  // 3. Bob approve + deposit
  await usdt.connect(bob).approve(SETTLEMENT_ADDRESS, DEPOSIT_AMOUNT);
  await settlement.connect(bob).deposit(USDT_ADDRESS, DEPOSIT_AMOUNT);
  console.log(`Bob 已充值 ${DEPOSIT_AMOUNT / ONE_USDT} USDT 到合约`);

  // 4. 查询链上余额确认
  const aliceBal = await settlement.getBalance(alice.address, USDT_ADDRESS);
  const bobBal   = await settlement.getBalance(bob.address, USDT_ADDRESS);
  console.log(`\n链上余额确认:`);
  console.log(`  Alice: ${aliceBal / ONE_USDT} USDT`);
  console.log(`  Bob:   ${bobBal / ONE_USDT} USDT`);
  console.log("\n准备完成，可以启动 IWS-ChainClient 开始监听成交了。");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

import { ethers } from "hardhat";
import { MockERC20 } from "../typechain-types";

async function main() {
  const [deployer, operator] = await ethers.getSigners();

  console.log("部署账户:", deployer.address);
  console.log("Operator 账户:", operator.address);

  // 1. 部署 MockERC20（模拟 USDT）
  const MockERC20 = await ethers.getContractFactory("MockERC20");
  const usdt = await MockERC20.deploy("Mock USDT", "USDT", 6) as unknown as MockERC20;
  await usdt.waitForDeployment();
  console.log("MockUSDT 地址:", await usdt.getAddress());

  // 2. 部署 Settlement
  const SettlementFactory = await ethers.getContractFactory("Settlement");
  const settlement = await SettlementFactory.deploy(operator.address);
  await settlement.waitForDeployment();
  console.log("Settlement 地址:", await settlement.getAddress());

  // 3. 部署 MockBTC（8 位精度）
  const btc = await MockERC20.deploy("Mock BTC", "BTC", 8) as unknown as MockERC20;
  await btc.waitForDeployment();
  console.log("MockBTC 地址:", await btc.getAddress());

  // 4. 给 deployer mint 测试代币
  const ONE_USDT = 10n ** 6n;
  const ONE_BTC  = 10n ** 8n;
  await usdt.mint(deployer.address, 100000n * ONE_USDT);
  await btc.mint(deployer.address, 10n * ONE_BTC);
  console.log("已 mint 100000 USDT 和 10 BTC 给 deployer");

  console.log("\n合约部署完成。");
  console.log("下一步：在前端或脚本中 approve + deposit，再由 operator 调用 settle。");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

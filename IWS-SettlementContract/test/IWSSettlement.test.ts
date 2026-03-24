import { expect } from "chai";
import { ethers } from "hardhat";
import { IWSSettlement, MockERC20 } from "../typechain-types";
import { HardhatEthersSigner } from "@nomicfoundation/hardhat-ethers/signers";

describe("IWSSettlement", function () {
  let settlement: IWSSettlement;
  let usdt: MockERC20;
  let owner: HardhatEthersSigner;
  let operator: HardhatEthersSigner;
  let alice: HardhatEthersSigner;
  let bob: HardhatEthersSigner;
  let stranger: HardhatEthersSigner;

  const DECIMALS = 6n;
  const ONE = 10n ** DECIMALS; // 1 USDT

  beforeEach(async function () {
    [owner, operator, alice, bob, stranger] = await ethers.getSigners();

    // 部署 MockERC20（模拟 USDT，6 位精度）
    const MockERC20Factory = await ethers.getContractFactory("MockERC20");
    usdt = await MockERC20Factory.deploy("Mock USDT", "USDT", 6) as unknown as MockERC20;

    // 部署 IWSSettlement
    const SettlementFactory = await ethers.getContractFactory("IWSSettlement");
    settlement = await SettlementFactory.deploy(operator.address) as unknown as IWSSettlement;

    // 给 alice 和 bob 各 mint 10000 USDT
    await usdt.mint(alice.address, 10000n * ONE);
    await usdt.mint(bob.address, 10000n * ONE);
  });

  // ─── 部署验证 ─────────────────────────────────────────────────────────────

  describe("部署", function () {
    it("owner 和 operator 正确设置", async function () {
      expect(await settlement.owner()).to.equal(owner.address);
      expect(await settlement.operator()).to.equal(operator.address);
    });
  });

  // ─── deposit ──────────────────────────────────────────────────────────────

  describe("deposit", function () {
    it("充值后链上余额正确增加", async function () {
      await usdt.connect(alice).approve(await settlement.getAddress(), 1000n * ONE);
      await settlement.connect(alice).deposit(await usdt.getAddress(), 1000n * ONE);

      expect(await settlement.getBalance(alice.address, await usdt.getAddress())).to.equal(1000n * ONE);
    });

    it("合约实际持有 ERC20", async function () {
      await usdt.connect(alice).approve(await settlement.getAddress(), 500n * ONE);
      await settlement.connect(alice).deposit(await usdt.getAddress(), 500n * ONE);

      expect(await usdt.balanceOf(await settlement.getAddress())).to.equal(500n * ONE);
    });

    it("充值 0 revert", async function () {
      await expect(
        settlement.connect(alice).deposit(await usdt.getAddress(), 0)
      ).to.be.revertedWithCustomError(settlement, "ZeroAmount");
    });

    it("未 approve 直接充值 revert", async function () {
      await expect(
        settlement.connect(alice).deposit(await usdt.getAddress(), 1000n * ONE)
      ).to.be.reverted;
    });

    it("触发 Deposited 事件", async function () {
      await usdt.connect(alice).approve(await settlement.getAddress(), 1000n * ONE);
      await expect(settlement.connect(alice).deposit(await usdt.getAddress(), 1000n * ONE))
        .to.emit(settlement, "Deposited")
        .withArgs(alice.address, await usdt.getAddress(), 1000n * ONE);
    });
  });

  // ─── withdraw ─────────────────────────────────────────────────────────────

  describe("withdraw", function () {
    beforeEach(async function () {
      await usdt.connect(alice).approve(await settlement.getAddress(), 1000n * ONE);
      await settlement.connect(alice).deposit(await usdt.getAddress(), 1000n * ONE);
    });

    it("提现后余额正确减少，token 返还用户", async function () {
      const balanceBefore = await usdt.balanceOf(alice.address);
      await settlement.connect(alice).withdraw(await usdt.getAddress(), 300n * ONE);

      expect(await settlement.getBalance(alice.address, await usdt.getAddress())).to.equal(700n * ONE);
      expect(await usdt.balanceOf(alice.address)).to.equal(balanceBefore + 300n * ONE);
    });

    it("提现超额 revert InsufficientBalance", async function () {
      await expect(
        settlement.connect(alice).withdraw(await usdt.getAddress(), 2000n * ONE)
      ).to.be.revertedWithCustomError(settlement, "InsufficientBalance");
    });

    it("提现 0 revert", async function () {
      await expect(
        settlement.connect(alice).withdraw(await usdt.getAddress(), 0)
      ).to.be.revertedWithCustomError(settlement, "ZeroAmount");
    });

    it("触发 Withdrawn 事件", async function () {
      await expect(settlement.connect(alice).withdraw(await usdt.getAddress(), 300n * ONE))
        .to.emit(settlement, "Withdrawn")
        .withArgs(alice.address, await usdt.getAddress(), 300n * ONE);
    });
  });

  // ─── settle ───────────────────────────────────────────────────────────────

  describe("settle", function () {
    beforeEach(async function () {
      // alice 充值 1000，bob 充值 1000
      await usdt.connect(alice).approve(await settlement.getAddress(), 1000n * ONE);
      await settlement.connect(alice).deposit(await usdt.getAddress(), 1000n * ONE);

      await usdt.connect(bob).approve(await settlement.getAddress(), 1000n * ONE);
      await settlement.connect(bob).deposit(await usdt.getAddress(), 1000n * ONE);
    });

    it("单笔结算：alice → bob，余额正确变化", async function () {
      await settlement.connect(operator).settle([
        { from: alice.address, to: bob.address, token: await usdt.getAddress(), amount: 100n * ONE },
      ]);

      expect(await settlement.getBalance(alice.address, await usdt.getAddress())).to.equal(900n * ONE);
      expect(await settlement.getBalance(bob.address, await usdt.getAddress())).to.equal(1100n * ONE);
    });

    it("批量结算（3笔）全部正确结算", async function () {
      await settlement.connect(operator).settle([
        { from: alice.address, to: bob.address, token: await usdt.getAddress(), amount: 100n * ONE },
        { from: bob.address, to: alice.address, token: await usdt.getAddress(), amount: 50n * ONE },
        { from: alice.address, to: bob.address, token: await usdt.getAddress(), amount: 200n * ONE },
      ]);

      // alice: 1000 - 100 + 50 - 200 = 750
      expect(await settlement.getBalance(alice.address, await usdt.getAddress())).to.equal(750n * ONE);
      // bob: 1000 + 100 - 50 + 200 = 1250
      expect(await settlement.getBalance(bob.address, await usdt.getAddress())).to.equal(1250n * ONE);
    });

    it("结算余额不足 revert InsufficientBalance", async function () {
      await expect(
        settlement.connect(operator).settle([
          { from: alice.address, to: bob.address, token: await usdt.getAddress(), amount: 5000n * ONE },
        ])
      ).to.be.revertedWithCustomError(settlement, "InsufficientBalance");
    });

    it("非 operator 调用 settle revert NotOperator", async function () {
      await expect(
        settlement.connect(stranger).settle([
          { from: alice.address, to: bob.address, token: await usdt.getAddress(), amount: 100n * ONE },
        ])
      ).to.be.revertedWithCustomError(settlement, "NotOperator");
    });

    it("触发 Settled 事件", async function () {
      await expect(
        settlement.connect(operator).settle([
          { from: alice.address, to: bob.address, token: await usdt.getAddress(), amount: 100n * ONE },
        ])
      )
        .to.emit(settlement, "Settled")
        .withArgs(alice.address, bob.address, await usdt.getAddress(), 100n * ONE);
    });
  });

  // ─── setOperator ──────────────────────────────────────────────────────────

  describe("setOperator", function () {
    it("owner 可以更新 operator", async function () {
      await settlement.connect(owner).setOperator(stranger.address);
      expect(await settlement.operator()).to.equal(stranger.address);
    });

    it("非 owner 调用 setOperator revert NotOwner", async function () {
      await expect(
        settlement.connect(alice).setOperator(stranger.address)
      ).to.be.revertedWithCustomError(settlement, "NotOwner");
    });

    it("触发 OperatorUpdated 事件", async function () {
      await expect(settlement.connect(owner).setOperator(stranger.address))
        .to.emit(settlement, "OperatorUpdated")
        .withArgs(operator.address, stranger.address);
    });
  });
});

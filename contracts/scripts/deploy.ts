import { ethers } from "hardhat";

async function main() {
  const tokenName = "IstChain-wrapped ATOM";
  const tokenSymbol = "iATOM";
  const tokenDecimals = 6;

  const ERC20IstChainWrappedCosmosCoin = await ethers.getContractFactory(
    "ERC20IstChainWrappedCosmosCoin"
  );
  const token = await ERC20IstChainWrappedCosmosCoin.deploy(
    tokenName,
    tokenSymbol,
    tokenDecimals
  );

  await token.deployed();

  console.log(
    `Token "${tokenName}" (${tokenSymbol}) with ${tokenDecimals} decimals is deployed to ${token.address}!`
  );
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});

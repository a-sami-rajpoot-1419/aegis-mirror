const ethers = require('ethers');

async function fundAddress() {
  const provider = new ethers.JsonRpcProvider('http://127.0.0.1:8545');
  
  // Alice's private key (from chain)
  const privateKey = '0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727';
  const wallet = new ethers.Wallet(privateKey, provider);
  
  console.log('Sender (Ethereum-derived):', wallet.address);
  console.log('Sender balance before:', ethers.formatEther(await provider.getBalance(wallet.address)), 'MVLT');
  
  // Check the Cosmos-derived address balance
  const cosmosAddress = '0x4418D0B4D9C1EF0E53DFB99143677E1E52354622';
  console.log('Cosmos-derived address:', cosmosAddress);
  console.log('Cosmos address balance:', ethers.formatEther(await provider.getBalance(cosmosAddress)), 'MVLT');
  
  // Since the Ethereum-derived address has 0 balance, we can try deployment anyway
  // The issue is the ante handler checks the wrong address
  console.log('\nThe problem: Hardhat uses', wallet.address, 'but funds are at', cosmosAddress);
}

fundAddress().catch(console.error);

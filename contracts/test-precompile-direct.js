const { ethers } = require('ethers');

const RPC_URL = 'http://127.0.0.1:8545';
const NFT_PRECOMPILE = '0x0000000000000000000000000000000000000102';
const ALICE_KEY = '0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727';

async function testPrecompileDirect() {
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const wallet = new ethers.Wallet(ALICE_KEY, provider);

  console.log('Testing NFT precompile directly...\n');
  console.log('Alice:', wallet.address);
  console.log('NFT Precompile:', NFT_PRECOMPILE);

  const tokenId = Date.now();
  const tokenURI = `https://mirror-vault.io/nft/${tokenId}`;

  console.log('\nToken ID:', tokenId);
  console.log('Token URI:', tokenURI);

  // Encode parameters: mint(address, uint256, string)
  const abi = new ethers.AbiCoder();
  const data = ethers.concat([
    ethers.id('mint(address,uint256,string)').slice(0, 10), // selector (4 bytes)
    abi.encode(['address', 'uint256', 'string'], [wallet.address, tokenId, tokenURI])
  ]);

  console.log('\nCalldata:', data);

  try {
    console.log('\nSending transaction to precompile...');
    const tx = await wallet.sendTransaction({
      to: NFT_PRECOMPILE,
      data: data,
      gasLimit: 500000,
      maxFeePerGas: 2000000000n,
      maxPriorityFeePerGas: 1000000000n
    });
    
    console.log('Transaction sent:', tx.hash);
    console.log('Waiting for confirmation...');
    
    const receipt = await tx.wait();
    console.log('\n✅ SUCCESS!');
    console.log('Block:', receipt.blockNumber);
    console.log('Gas used:', receipt.gasUsed.toString());
    console.log('Status:', receipt.status === 1 ? 'Success' : 'Failed');
    console.log('Logs:', receipt.logs.length);
    
  } catch (error) {
    console.log('\n❌ FAILED:');
    console.log('Error:', error.message);
    if (error.receipt) {
      console.log('Status:', error.receipt.status);
      console.log('Gas used:', error.receipt.gasUsed.toString());
    }
  }
}

testPrecompileDirect().catch(console.error);

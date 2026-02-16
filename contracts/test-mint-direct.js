const { ethers } = require('ethers');

const RPC_URL = 'http://127.0.0.1:8545';
const NFT_CONTRACT = '0x2dd86F2Cd8885e02DE232CBd7637Fb4cC241C401';
const ALICE_KEY = '0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727';

const NFT_ABI = [
  'function mint(uint256 tokenId, string uri) external',
];

async function testMintDirect() {
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const wallet = new ethers.Wallet(ALICE_KEY, provider);
  const nftContract = new ethers.Contract(NFT_CONTRACT, NFT_ABI, wallet);

  console.log('Testing NFT mint with explicit gas...\n');
  console.log('Alice:', wallet.address);
  console.log('NFT Contract:', NFT_CONTRACT);

  const tokenId = Date.now();
  const tokenURI = `https://mirror-vault.io/nft/${tokenId}`;

  console.log('\nToken ID:', tokenId);
  console.log('Token URI:', tokenURI);

  try {
    console.log('\nAttempting mint with explicit gas...');
    const tx = await nftContract.mint(tokenId, tokenURI, {
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
    
  } catch (error) {
    console.log('\n❌ FAILED:');
    console.log('Error:', error.message);
    if (error.data) {
      console.log('Error data:', error.data);
    }
    if (error.receipt) {
      console.log('Receipt:', error.receipt);
    }
  }
}

testMintDirect().catch(console.error);

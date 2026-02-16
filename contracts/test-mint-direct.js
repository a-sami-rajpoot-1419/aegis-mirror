const { ethers } = require('ethers');
const fs = require('fs');
const path = require('path');

const RPC_URL = 'http://127.0.0.1:8545';
const ALICE_KEY = '0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727';

function loadDeployedAddresses() {
  const deployedPath = path.join(__dirname, 'deployed-addresses.json');
  if (!fs.existsSync(deployedPath)) return null;
  try {
    const parsed = JSON.parse(fs.readFileSync(deployedPath, 'utf8'));
    if (!parsed || !parsed.mirrorNFT) return null;
    return parsed;
  } catch {
    return null;
  }
}

const NFT_ABI = [
  'function mint(uint256 tokenId, string uri) external',
];

async function testMintDirect() {
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const wallet = new ethers.Wallet(ALICE_KEY, provider);

  const deployed = loadDeployedAddresses();
  const nftAddress = process.env.NFT_CONTRACT || (deployed ? deployed.mirrorNFT : null);
  if (!nftAddress) {
    throw new Error('Missing NFT contract address. Run: cd contracts && npx hardhat run scripts/deploy.ts --network mirrorVaultLocal');
  }

  const nftContract = new ethers.Contract(nftAddress, NFT_ABI, wallet);

  console.log('Testing NFT mint with explicit gas...\n');
  console.log('Alice:', wallet.address);
  console.log('NFT Contract:', nftAddress);

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

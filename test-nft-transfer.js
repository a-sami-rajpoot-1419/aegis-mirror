const { ethers } = require('ethers');

// Configuration
const RPC_URL = 'http://localhost:8545';
const NFT_ADDRESS = '0x07587FFc5550cc3A168b1ba9Ebc0BC8CdcC33e8b';

// Test accounts (Alice and Bob)
const ALICE_PRIVATE_KEY = '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80'; // Hardhat default account 0
const BOB_PRIVATE_KEY = '0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d'; // Hardhat default account 1

// NFT ABI (minimal for testing)
const NFT_ABI = [
  'function mint(address to, uint256 tokenId, string uri) external',
  'function transferFrom(address from, address to, uint256 tokenId) external',
  'function ownerOf(uint256 tokenId) external view returns (address)',
  'function balanceOf(address owner) external view returns (uint256)',
  'event Transfer(address indexed from, address indexed to, uint256 indexed tokenId)'
];

async function main() {
  console.log('\n🧪 NFT Cross-Pair Transfer Test\n');
  console.log('='.repeat(60));

  // Setup provider and wallets
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const alice = new ethers.Wallet(ALICE_PRIVATE_KEY, provider);
  const bob = new ethers.Wallet(BOB_PRIVATE_KEY, provider);

  console.log('👤 Alice Address:', alice.address);
  console.log('👤 Bob Address:', bob.address);
  console.log('');

  // Connect to NFT contract
  const nftAsAlice = new ethers.Contract(NFT_ADDRESS, NFT_ABI, alice);
  const nftAsBob = new ethers.Contract(NFT_ADDRESS, NFT_ABI, bob);

  // Step 1: Mint NFT to Alice
  const tokenId = Math.floor(Math.random() * 1000000);
  console.log(`\n📝 Step 1: Minting NFT #${tokenId} to Alice...`);
  
  try {
    const mintTx = await nftAsAlice.mint(
      alice.address,
      tokenId,
      `ipfs://test-uri-${tokenId}`
    );
    const mintReceipt = await mintTx.wait();
    
    if (mintReceipt.status === 1) {
      console.log('   ✅ Mint successful!');
      console.log('   📜 Tx Hash:', mintReceipt.hash);
    } else {
      console.log('   ❌ Mint failed with status:', mintReceipt.status);
      return;
    }
  } catch (err) {
    console.log('   ❌ Mint error:', err.message);
    return;
  }

  // Step 2: Verify Alice owns the NFT
  console.log('\n📝 Step 2: Verifying Alice owns NFT...');
  try {
    const owner = await nftAsAlice.ownerOf(tokenId);
    console.log('   🔍 Current owner:', owner);
    
    if (owner.toLowerCase() === alice.address.toLowerCase()) {
      console.log('   ✅ Ownership verified!');
    } else {
      console.log('   ⚠️  Warning: Owner mismatch!');
      console.log('      Expected:', alice.address);
      console.log('      Actual:', owner);
    }
  } catch (err) {
    console.log('   ❌ Query error:', err.message);
    return;
  }

  // Step 3: Transfer NFT from Alice to Bob
  console.log('\n📝 Step 3: Transferring NFT from Alice to Bob...');
  console.log('   From:', alice.address);
  console.log('   To:', bob.address);
  console.log('   Token ID:', tokenId);
  
  try {
    const transferTx = await nftAsAlice.transferFrom(
      alice.address,
      bob.address,
      tokenId,
      { gasLimit: 500000 }
    );
    
    console.log('   ⏳ Transaction sent, waiting for confirmation...');
    console.log('   📜 Tx Hash:', transferTx.hash);
    
    const transferReceipt = await transferTx.wait();
    
    console.log('   📊 Gas used:', transferReceipt.gasUsed.toString());
    console.log('   📊 Status:', transferReceipt.status);
    
    if (transferReceipt.status === 1) {
      console.log('   ✅ Transfer successful!');
      
      // Check for Transfer event
      const transferEvent = transferReceipt.logs.find(log => {
        try {
          const parsed = nftAsAlice.interface.parseLog(log);
          return parsed.name === 'Transfer';
        } catch {
          return false;
        }
      });
      
      if (transferEvent) {
        const parsed = nftAsAlice.interface.parseLog(transferEvent);
        console.log('   📢 Event emitted:');
        console.log('      From:', parsed.args.from);
        console.log('      To:', parsed.args.to);
        console.log('      TokenId:', parsed.args.tokenId.toString());
      }
    } else {
      console.log('   ❌ Transfer FAILED - Status:', transferReceipt.status);
      console.log('   📊 Receipt:', JSON.stringify(transferReceipt, null, 2));
      return;
    }
  } catch (err) {
    console.log('   ❌ Transfer error:', err.shortMessage || err.message);
    
    if (err.data) {
      console.log('   🔍 Error data:', err.data);
    }
    if (err.reason) {
      console.log('   🔍 Reason:', err.reason);
    }
    if (err.transaction) {
      console.log('   🔍 Failed transaction:');
      console.log('      To:', err.transaction.to);
      console.log('      From:', err.transaction.from);
      console.log('      Data:', err.transaction.data);
    }
    return;
  }

  // Step 4: Verify Bob now owns the NFT
  console.log('\n📝 Step 4: Verifying Bob now owns NFT...');
  try {
    const newOwner = await nftAsBob.ownerOf(tokenId);
    console.log('   🔍 Current owner:', newOwner);
    
    if (newOwner.toLowerCase() === bob.address.toLowerCase()) {
      console.log('   ✅ Transfer verified! Bob is now the owner.');
    } else {
      console.log('   ⚠️  Warning: Unexpected owner!');
      console.log('      Expected:', bob.address);
      console.log('      Actual:', newOwner);
    }
  } catch (err) {
    console.log('   ❌ Query error:', err.message);
  }

  // Step 5: Check balances
  console.log('\n📝 Step 5: Checking NFT balances...');
  try {
    const aliceBalance = await nftAsAlice.balanceOf(alice.address);
    const bobBalance = await nftAsBob.balanceOf(bob.address);
    
    console.log('   👤 Alice balance:', aliceBalance.toString());
    console.log('   👤 Bob balance:', bobBalance.toString());
  } catch (err) {
    console.log('   ❌ Balance query error:', err.message);
  }

  console.log('\n' + '='.repeat(60));
  console.log('✨ Test complete!\n');
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error('\n💥 Fatal error:', error);
    process.exit(1);
  });

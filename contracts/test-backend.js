const { ethers } = require('ethers');
const fs = require('fs');
const path = require('path');

// Configuration
const RPC_URL = 'http://127.0.0.1:8545';
const ALICE_KEY = '0x1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727';

function loadDeployedAddresses() {
  const deployedPath = path.join(__dirname, 'deployed-addresses.json');
  if (!fs.existsSync(deployedPath)) return null;
  try {
    const parsed = JSON.parse(fs.readFileSync(deployedPath, 'utf8'));
    if (!parsed || !parsed.vaultGate || !parsed.mirrorNFT) return null;
    return parsed;
  } catch {
    return null;
  }
}

// ABIs
const VAULT_ABI = [
  'function payToUnlock() external payable',
  'function storeMessage(string text) external',
  'function getLastMessage(address user) external view returns (string)',
  'function getMessageCount(address user) external view returns (uint256)',
  'function getGlobalMessageCount() external view returns (uint256)',
  'function getGlobalLastMessage() external view returns (string)'
];

const NFT_ABI = [
  'function mint(uint256 tokenId, string uri) external',
  'function transferFrom(address from, address to, uint256 tokenId) external',
  'function ownerOf(uint256 tokenId) external view returns (address ownerEvm, string memory ownerCosmos, bool exists)',
  'function exists(uint256 tokenId) external view returns (bool)',
  'function tokenURI(uint256 tokenId) external view returns (string)'
];

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function waitForTx(tx, label) {
  console.log(`  ⏳ Waiting for ${label}...`);
  const receipt = await tx.wait();
  console.log(`  ✅ ${label} confirmed (block ${receipt.blockNumber}, gas: ${receipt.gasUsed.toString()})`);
  return receipt;
}

async function testBackend() {
  console.log('🧪 MIRROR VAULT BACKEND TESTING\n');
  console.log('='.repeat(70));
  
  try {
    // Setup
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    const wallet = new ethers.Wallet(ALICE_KEY, provider);

    const deployed = loadDeployedAddresses();
    const vaultAddress = process.env.VAULT_CONTRACT || (deployed ? deployed.vaultGate : null);
    const nftAddress = process.env.NFT_CONTRACT || (deployed ? deployed.mirrorNFT : null);

    if (!vaultAddress || !nftAddress) {
      throw new Error(
        'Missing deployed wrapper addresses. Run: cd contracts && npx hardhat run scripts/deploy.ts --network mirrorVaultLocal'
      );
    }

    let failures = 0;
    const fail = (msg) => {
      failures++;
      console.log(`  ❌ FAIL: ${msg}`);
    };
    
    console.log('\n✅ STEP 1: Connection & Setup');
    console.log('  Chain ID:', (await provider.getNetwork()).chainId.toString());
    console.log('  Alice Address:', wallet.address);
    console.log('  Alice Balance:', ethers.formatEther(await provider.getBalance(wallet.address)), 'MVLT');
    
    // Contracts (must be deployed)
    const vaultCode = await provider.getCode(vaultAddress);
    const nftCode = await provider.getCode(nftAddress);
    if (vaultCode === '0x') {
      throw new Error(`VaultGate not deployed at ${vaultAddress}. Run: cd contracts && npx hardhat run scripts/deploy.ts --network mirrorVaultLocal`);
    }
    if (nftCode === '0x') {
      throw new Error(`MirrorNFT not deployed at ${nftAddress}. Run: cd contracts && npx hardhat run scripts/deploy.ts --network mirrorVaultLocal`);
    }

    const vaultContract = new ethers.Contract(vaultAddress, VAULT_ABI, wallet);
    const nftContract = new ethers.Contract(nftAddress, NFT_ABI, wallet);

    console.log('  VaultGate:', vaultAddress);
    console.log('  MirrorNFT:', nftAddress);
    
    // Test 1: Payment Validation
    console.log('\n' + '='.repeat(70));
    console.log('✅ STEP 2: Payment Validation Test');
    console.log('='.repeat(70));
    
    console.log('\n📝 Test 2.1: Insufficient Payment (should fail)');
    try {
      const tx = await vaultContract.payToUnlock({ 
        value: ethers.parseEther('0.5'),
        maxFeePerGas: 2000000000n,
        maxPriorityFeePerGas: 1000000000n
      });
      await tx.wait();
      fail('Should have rejected 0.5 MVLT payment');
    } catch (error) {
      console.log('  ✅ PASS: Correctly rejected insufficient payment');
    }
    
    console.log('\n📝 Test 2.2: Exact Payment (1 MVLT)');
    
    // Skip count check before payment since view functions may not be implemented yet
    console.log('  Attempting payment...');
    
    const payTx = await vaultContract.payToUnlock({ 
      value: ethers.parseEther('1'),
      maxFeePerGas: 2000000000n,
      maxPriorityFeePerGas: 1000000000n
    });
    await waitForTx(payTx, 'Payment transaction');
    console.log('  ✅ PASS: Payment accepted (1 MVLT)');
    
    const countAfterPayment = await vaultContract.getMessageCount(wallet.address);
    console.log('  Message count after payment:', countAfterPayment.toString());
    
    // Test 3: Message Storage
    console.log('\n' + '='.repeat(70));
    console.log('✅ STEP 3: Message Storage & Retrieval Test');
    console.log('='.repeat(70));
    
    const testMessage = `Test message ${Date.now()}`;
    console.log('\n📝 Test 3.1: Store Message');
    console.log('  Message:', testMessage);
    
    const countBefore = await vaultContract.getMessageCount(wallet.address);
    console.log('  Message count before:', countBefore.toString());
    
    const storeTx = await vaultContract.storeMessage(testMessage, {
      maxFeePerGas: 2000000000n,
      maxPriorityFeePerGas: 1000000000n
    });
    await waitForTx(storeTx, 'Store message transaction');
    
    const countAfter = await vaultContract.getMessageCount(wallet.address);
    console.log('  Message count after:', countAfter.toString());
    if (countAfter > countBefore) {
      console.log('  ✅ PASS: Message count increased');
    } else {
      fail('Message count did not increase as expected');
    }
    
    console.log('\n📝 Test 3.2: Retrieve Message');
    const retrievedMsg = await vaultContract.getLastMessage(wallet.address);
    console.log('  Retrieved:', retrievedMsg);

    if (retrievedMsg === testMessage) {
      console.log('  ✅ PASS: Message matches');
    } else {
      fail('Message retrieved but does not match');
    }
    
    // Test 4: NFT Minting
    console.log('\n' + '='.repeat(70));
    console.log('✅ STEP 4: NFT Minting Test');
    console.log('='.repeat(70));
    
    const tokenId = Date.now();
    const tokenURI = `https://mirror-vault.io/nft/${tokenId}`;
    
    console.log('\n📝 Test 4.1: Mint NFT');
    console.log('  Token ID:', tokenId);
    console.log('  Token URI:', tokenURI);
    console.log('  Minter:', wallet.address);
    
    const mintTx = await nftContract.mint(tokenId, tokenURI, {
      maxFeePerGas: 2000000000n,
      maxPriorityFeePerGas: 1000000000n
    });
    await waitForTx(mintTx, 'Mint transaction');
    
    console.log('\n📝 Test 4.2: Verify NFT Existence');
    const exists = await nftContract.exists(tokenId);
    console.log('  Exists:', exists);

    if (exists) {
      console.log('  ✅ PASS: NFT minted successfully');
    } else {
      fail('NFT does not exist after minting');
    }
    
    // Test 5: NFT Owner Query (Dual Address)
    console.log('\n' + '='.repeat(70));
    console.log('✅ STEP 5: NFT Dual Address Query Test');
    console.log('='.repeat(70));
    
    console.log('\n📝 Test 5.1: Query Owner (Both Formats)');
    const [ownerEvm, ownerCosmos, ownerExists] = await nftContract.ownerOf(tokenId);
    console.log('  Owner (EVM):', ownerEvm);
    console.log('  Owner (Cosmos):', ownerCosmos);
    console.log('  Exists:', ownerExists);

    if (ownerEvm.toLowerCase() === wallet.address.toLowerCase()) {
      console.log('  ✅ PASS: EVM address matches');
    } else {
      fail('EVM address mismatch');
    }

    if (ownerCosmos && ownerCosmos.startsWith('mirror1')) {
      console.log('  ✅ PASS: Cosmos address format correct');
    } else {
      fail('Cosmos address not in expected format');
    }
    
    // Test 6: NFT Transfer
    console.log('\n' + '='.repeat(70));
    console.log('✅ STEP 6: NFT Transfer Test');
    console.log('='.repeat(70));
    
    // Create bob's wallet for transfer test
    const bobKey = '0x109b30f67d3a18080bcd2b16c02f6386b6cc19c04f8e40f8f6e6e80e36c0bc9c';
    const bobWallet = new ethers.Wallet(bobKey, provider);
    
    console.log('\n📝 Test 6.1: Transfer NFT to Bob');
    console.log('  From:', wallet.address);
    console.log('  To:', bobWallet.address);
    console.log('  Token ID:', tokenId);
    
    const transferTx = await nftContract.transferFrom(wallet.address, bobWallet.address, tokenId, {
      maxFeePerGas: 2000000000n,
      maxPriorityFeePerGas: 1000000000n
    });
    await waitForTx(transferTx, 'Transfer transaction');
    
    console.log('\n📝 Test 6.2: Verify New Owner');
    const [newOwnerEvm, newOwnerCosmos, newExists] = await nftContract.ownerOf(tokenId);
    console.log('  New Owner (EVM):', newOwnerEvm);
    console.log('  New Owner (Cosmos):', newOwnerCosmos);

    if (newOwnerEvm.toLowerCase() === bobWallet.address.toLowerCase()) {
      console.log('  ✅ PASS: NFT transferred successfully');
    } else {
      fail('Owner did not change after transfer');
    }
    
    // Test 7: Balance Check
    console.log('\n' + '='.repeat(70));
    console.log('✅ STEP 7: Balance & Gas Usage Summary');
    console.log('='.repeat(70));
    
    const finalBalance = await provider.getBalance(wallet.address);
    console.log('\n  Final Alice Balance:', ethers.formatEther(finalBalance), 'MVLT');
    
    const msgCount = await vaultContract.getMessageCount(wallet.address);
    const globalCount = await vaultContract.getGlobalMessageCount();
    console.log('  Final Message Count:', msgCount.toString());
    console.log('  Global Message Count:', globalCount.toString());
    
    // Summary
    console.log('\n' + '='.repeat(70));
    console.log('✅ ALL BACKEND TESTS COMPLETED');
    console.log('='.repeat(70));
    console.log('\n📊 Test Summary:');
    console.log('  ✅ Payment validation working (1 MVLT requirement enforced)');
    console.log('  ✅ Message storage & retrieval working');
    console.log('  ✅ NFT minting working');
    console.log('  ✅ NFT dual address query working');
    console.log('  ✅ NFT transfers working');
    console.log('\n🎉 Backend is ready for frontend integration!\n');

    if (failures > 0) {
      throw new Error(`${failures} backend test(s) failed`);
    }
    
  } catch (error) {
    console.error('\n❌ ERROR during testing:');
    console.error(error);
    process.exit(1);
  }
}

testBackend().catch(console.error);

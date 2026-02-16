const { ethers } = require('ethers');

// Configuration
const RPC_URL = 'http://localhost:8545';
const VAULT_GATE_ADDRESS = '0xEDcb370f4771A9d1d40C033ca1253BEb1E3fF1e8';  // Update after deployment

// Test accounts
const ALICE_PRIVATE_KEY = '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80';

// VaultGate ABI
const VAULT_GATE_ABI = [
  'function payToUnlock() external payable',
  'function getMessageCount(address user) external view returns (uint256)',
  'event Unlocked(address indexed user)'
];

async function main() {
  console.log('\n🧪 Payment Validation Test Suite\n');
  console.log('='.repeat(70));

  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const alice = new ethers.Wallet(ALICE_PRIVATE_KEY, provider);
  const vaultGate = new ethers.Contract(VAULT_GATE_ADDRESS, VAULT_GATE_ABI, alice);

  console.log('👤 Test Account:', alice.address);
  const balance = await provider.getBalance(alice.address);
  console.log('💰 Initial Balance:', ethers.formatEther(balance), 'MIRROR');
  console.log('');

  // Test 1: Zero Payment (Should Fail)
  console.log('📝 TEST 1: Attempt to unlock with ZERO payment');
  console.log('-'.repeat(70));
  try {
    const tx = await vaultGate.payToUnlock({ value: 0 });
    await tx.wait();
    console.log('   ❌ UNEXPECTED: Transaction succeeded with zero payment!');
  } catch (err) {
    if (err.message.includes('Must pay at least 1 MIRROR')) {
      console.log('   ✅ PASS: Correctly rejected zero payment');
      console.log('   📝 Error: "Must pay at least 1 MIRROR token"');
    } else {
      console.log('   ⚠️  Transaction failed but with unexpected error:');
      console.log('   📝', err.shortMessage || err.message);
    }
  }
  console.log('');

  // Test 2: Insufficient Payment (Should Fail)
  console.log('📝 TEST 2: Attempt to unlock with 0.5 MIRROR');
  console.log('-'.repeat(70));
  try {
    const tx = await vaultGate.payToUnlock({ 
      value: ethers.parseEther('0.5') 
    });
    await tx.wait();
    console.log('   ❌ UNEXPECTED: Transaction succeeded with insufficient payment!');
  } catch (err) {
    if (err.message.includes('Must pay at least 1 MIRROR')) {
      console.log('   ✅ PASS: Correctly rejected insufficient payment');
      console.log('   📝 Error: "Must pay at least 1 MIRROR token"');
    } else {
      console.log('   ⚠️  Transaction failed but with unexpected error:');
      console.log('   📝', err.shortMessage || err.message);
    }
  }
  console.log('');

  // Test 3: Exact Payment (Should Succeed)
  console.log('📝 TEST 3: Unlock with exactly 1 MIRROR');
  console.log('-'.repeat(70));
  try {
    const creditsBefore = await vaultGate.getMessageCount(alice.address);
    console.log('   💳 Credits before:', creditsBefore.toString());
    
    const tx = await vaultGate.payToUnlock({ 
      value: ethers.parseEther('1.0'),
      gasLimit: 500000
    });
    console.log('   ⏳ Transaction sent:', tx.hash);
    
    const receipt = await tx.wait();
    console.log('   📊 Gas used:', receipt.gasUsed.toString());
    console.log('   📊 Status:', receipt.status === 1 ? 'Success ✅' : 'Failed ❌');
    
    if (receipt.status === 1) {
      const creditsAfter = await vaultGate.getMessageCount(alice.address);
      console.log('   💳 Credits after:', creditsAfter.toString());
      
      // Check for Unlocked event
      const unlockedEvent = receipt.logs.find(log => {
        try {
          const parsed = vaultGate.interface.parseLog(log);
          return parsed.name === 'Unlocked';
        } catch {
          return false;
        }
      });
      
      if (unlockedEvent) {
        const parsed = vaultGate.interface.parseLog(unlockedEvent);
        console.log('   📢 Event emitted: Unlocked');
        console.log('      User:', parsed.args.user);
      }
      
      // Verify credit was added
      if (BigInt(creditsAfter) === BigInt(creditsBefore) + 1n) {
        console.log('   ✅ PASS: Credit successfully added (payment validated)');
      } else {
        console.log('   ⚠️  WARNING: Credit count unexpected');
        console.log('      Expected:', (BigInt(creditsBefore) + 1n).toString());
        console.log('      Actual:', creditsAfter.toString());
      }
    } else {
      console.log('   ❌ FAIL: Transaction reverted');
    }
  } catch (err) {
    console.log('   ❌ FAIL: Transaction error');
    console.log('   📝', err.shortMessage || err.message);
  }
  console.log('');

  // Test 4: Overpayment (Should Succeed)
  console.log('📝 TEST 4: Unlock with 2 MIRROR (overpayment)');
  console.log('-'.repeat(70));
  try {
    const creditsBefore = await vaultGate.getMessageCount(alice.address);
    console.log('   💳 Credits before:', creditsBefore.toString());
    
    const tx = await vaultGate.payToUnlock({ 
      value: ethers.parseEther('2.0'),
      gasLimit: 500000
    });
    console.log('   ⏳ Transaction sent:', tx.hash);
    
    const receipt = await tx.wait();
    console.log('   📊 Gas used:', receipt.gasUsed.toString());
    console.log('   📊 Status:', receipt.status === 1 ? 'Success ✅' : 'Failed ❌');
    
    if (receipt.status === 1) {
      const creditsAfter = await vaultGate.getMessageCount(alice.address);
      console.log('   💳 Credits after:', creditsAfter.toString());
      
      if (BigInt(creditsAfter) === BigInt(creditsBefore) + 1n) {
        console.log('   ✅ PASS: Overpayment accepted, 1 credit added');
        console.log('   📝 Note: Overpayment goes to vault module');
      } else {
        console.log('   ⚠️  WARNING: Credit count unexpected');
      }
    } else {
      console.log('   ❌ FAIL: Transaction reverted unexpectedly');
    }
  } catch (err) {
    console.log('   ❌ FAIL: Transaction error');
    console.log('   📝', err.shortMessage || err.message);
  }
  console.log('');

  // Final Balance
  const finalBalance = await provider.getBalance(alice.address);
  console.log('💰 Final Balance:', ethers.formatEther(finalBalance), 'MIRROR');
  console.log('💸 Total Spent:', ethers.formatEther(balance - finalBalance), 'MIRROR');
  
  console.log('\n' + '='.repeat(70));
  console.log('✨ Payment validation testing complete!\n');
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error('\n💥 Fatal error:', error);
    process.exit(1);
  });

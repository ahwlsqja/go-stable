/**
 * EIP-712 Signature Generator for Wallet Verification
 *
 * Usage:
 *   node generate_eip712_signature.js <private_key> [chain_id] [verifying_contract]
 *
 * Example:
 *   node generate_eip712_signature.js 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
 *
 * Requirements:
 *   npm install ethers uuid
 */

const { ethers } = require('ethers');
const { v4: uuidv4 } = require('uuid');

// Default configuration (matches Go server defaults)
const DEFAULT_CHAIN_ID = 1;
const DEFAULT_VERIFYING_CONTRACT = '0x0000000000000000000000000000000000000000';

async function generateSignature(privateKey, chainId, verifyingContract) {
    const wallet = new ethers.Wallet(privateKey);
    const address = wallet.address;
    const nonce = uuidv4();
    const timestamp = Math.floor(Date.now() / 1000);

    // EIP-712 Domain
    const domain = {
        name: 'B2B Settlement',
        version: '1',
        chainId: chainId,
        verifyingContract: verifyingContract
    };

    // EIP-712 Types
    const types = {
        WalletVerification: [
            { name: 'wallet', type: 'address' },
            { name: 'nonce', type: 'string' },
            { name: 'timestamp', type: 'uint256' }
        ]
    };

    // Message to sign
    const message = {
        wallet: address,
        nonce: nonce,
        timestamp: timestamp
    };

    // Sign the typed data
    const signature = await wallet.signTypedData(domain, types, message);

    console.log('\n=== EIP-712 Wallet Verification Signature ===\n');
    console.log('Wallet Address:', address);
    console.log('Chain ID:', chainId);
    console.log('Verifying Contract:', verifyingContract);
    console.log('\n--- Request Body for POST /verify ---\n');

    const requestBody = {
        signature: signature,
        message: {
            nonce: nonce,
            timestamp: timestamp
        }
    };

    console.log(JSON.stringify(requestBody, null, 2));

    console.log('\n--- cURL Command ---\n');
    console.log(`curl -X POST "http://localhost:8080/api/v1/users/{userId}/wallets/{walletId}/verify" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify(requestBody)}'`);

    console.log('\n--- Notes ---');
    console.log('1. Replace {userId} and {walletId} with actual UUIDs');
    console.log('2. The wallet must be registered with address:', address.toLowerCase());
    console.log('3. Signature is valid for ~5 minutes from timestamp:', timestamp);
    console.log('4. Timestamp in human readable:', new Date(timestamp * 1000).toISOString());

    return { address, signature, nonce, timestamp };
}

// Main
const args = process.argv.slice(2);

if (args.length < 1) {
    console.log('Usage: node generate_eip712_signature.js <private_key> [chain_id] [verifying_contract]');
    console.log('\nExample with Anvil default account:');
    console.log('  node generate_eip712_signature.js 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80');
    process.exit(1);
}

const privateKey = args[0];
const chainId = parseInt(args[1]) || DEFAULT_CHAIN_ID;
const verifyingContract = args[2] || DEFAULT_VERIFYING_CONTRACT;

generateSignature(privateKey, chainId, verifyingContract)
    .then(() => process.exit(0))
    .catch((err) => {
        console.error('Error:', err.message);
        process.exit(1);
    });

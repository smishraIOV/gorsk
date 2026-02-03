# Rootstock Block Hash and Account Verifier

Rootstock uses a different trie (unified, binary) than Ethereum (separate hexary). The block header is also different. This makes it harder for external programs to verify block hashes, transaction roots, receipt roots, and account proofs. We port Rootstock's Trie, Transaction, TransactionReceipt, Block, BlockHeader and helper classes from Java to Go.

## Core Libraries (`rskblocks/`)

### Block & Transaction Verification

- `block_hashes_helper.go` - Transaction and receipt root computation
  - `GetTxTrieRoot(transactions)` - Compute transaction trie root
  - `CalculateReceiptsTrieRoot(receipts)` - Compute receipts trie root

- `block_header_hash_helper.go` - Block header hash computation
  - `ComputeBlockHash(input, config)` - Compute block hash from input data
  - `InputToBlockHeader(input, config)` - Convert input to BlockHeader struct
  - `ConfigForBlockNumber(blockNum, network)` - Get config for network/block

- `block_header.go` - BlockHeader struct and RLP encoding
- `transaction.go` - Transaction struct and RLP encoding
- `receipt.go` - TransactionReceipt struct and RLP encoding

### Account Proof Verification

- `proof_helper.go` - Merkle proof verification for accounts and storage
  - `NewProofVerifier()` - Create a new proof verifier
  - `VerifyAccountProof(stateRoot, address, proofNodes)` - Verify account existence
  - `VerifyStorageProof(stateRoot, address, storageKey, proofNodes)` - Verify storage values
  - `DecodeRLPProofNodes(proofNodesHex)` - Decode RLP-encoded proof nodes

## CLI Tools

Run all commands from the `gorsk` directory.

### Block Verification Tool

Verify transaction roots, receipt roots, and block hashes:

```bash
go run ./cmd/verify_roots/ <block_number>
```

### Account Proof Verification Tool

Verify `eth_getProof` responses:

```bash
# Verify EOA account proof
go run ./cmd/verify_proof/ 0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826

# Verify contract with storage slot 0
go run ./cmd/verify_proof/ 0x77045E71a7A2c50903d88e564cD72fab11e82051 0x0

# Verify multiple storage slots
go run ./cmd/verify_proof/ <contract_address> 0x0,0x1,0x2
```

## Using the Go Library

### Block Hash Verification

```go
import "gorsk/rskblocks"

// Create input from RPC data
input := &rskblocks.BlockHeaderInput{
    ParentHash:                parentHash,
    UnclesHash:                unclesHash,
    Coinbase:                  coinbase,
    StateRoot:                 stateRoot,
    TxTrieRoot:                txTrieRoot,
    ReceiptTrieRoot:           receiptsRoot,
    LogsBloom:                 logsBloom,     // [256]byte
    Difficulty:                difficulty,
    Number:                    number,
    GasLimit:                  gasLimit,
    GasUsed:                   gasUsed,
    Timestamp:                 timestamp,
    ExtraData:                 extraData,
    PaidFees:                  paidFees,
    MinimumGasPrice:           minimumGasPrice,
    UncleCount:                uncleCount,
    BitcoinMergedMiningHeader: btcHeader,
    TxExecutionSublistsEdges:  edges,         // []int16
    BaseEvent:                 baseEvent,     // V2 only
    UmmRoot:                   ummRoot,       // *[]byte
}

// Get config for the network and block number
config := rskblocks.ConfigForBlockNumber(blockNum, "mainnet")
// Or use defaults:
// config := rskblocks.DefaultRegtestConfig()

// Compute block hash
hash := rskblocks.ComputeBlockHash(input, config)

// Get the RLP encoding for debugging
encoded := rskblocks.GetEncodedBlockHeader(input, config)
```

### Account Proof Verification

```go
import (
    "gorsk/rskblocks"
    "github.com/ethereum/go-ethereum/common"
)

// Create verifier
verifier := rskblocks.NewProofVerifier()

// Verify account proof (proofNodesHex from eth_getProof response)
isValid, accountRLP, err := verifier.VerifyAccountProof(
    stateRoot,
    address,
    proofNodesHex,  // []string from RPC
)

// Verify storage proof
isValid, valueRLP, err := verifier.VerifyStorageProof(
    stateRoot,
    address,
    storageKey,
    storageProofHex,
)
```

## Key Differences from Ethereum

RSK uses a **binary trie** (not Ethereum's hexary MPT) and a **unified trie** for accounts, contracts, and storage:

1. **Single Trie**: Accounts, code, and storage all share the same trie
2. **Key Structure**:
   - Account: `0x00 + keccak256(address)[:10] + address`
   - Storage: `accountKey + 0x00 + keccak256(slot)[:10] + stripZeros(slot)`
   - Code: `accountKey + 0x80`
3. **Proof Order**: Nodes are ordered leaf-to-root (last node is state root)

## RSKIPs for Block Hash Computation

| RSKIP | Description |
|-------|-------------|
| **RSKIP-92** | Excludes merged mining merkle proof and coinbase from hash (active from Orchid) |
| **RSKIP-144** | TxExecutionSublistsEdges for parallel transaction execution (active from Reed810) |
| **RSKIP-351** | V1 headers use extensionData instead of raw logsBloom (active from Reed810) |
| **RSKIP-535** | V2 headers add baseEvent to extension hash computation (active from Vetiver900) |
| **RSKIP-UMM** | ummRoot field added to block headers (active from Papyrus200) |

### Header Versions

| Version | Activated By | Extension Hash Formula |
|---------|--------------|------------------------|
| **V0** | Default | N/A - uses raw logsBloom in encoding |
| **V1** | RSKIP-351 | `Keccak256(RLP([Keccak256(logsBloom), edgesBytes]))` |
| **V2** | RSKIP-535 | `Keccak256(RLP([Keccak256(logsBloom), baseEvent, edgesBytes]))` |

For V1/V2 headers: `extensionData = RLP([version, extensionHash])`

## Network-Specific Differences

### Regtest
- **Header Version**: V2 (all RSKIPs active from genesis)
- **gasLimit Encoding**: 4-byte with leading zeros (e.g., `00989680`)
- **ummRoot**: Always included (empty if not present)
- **RSKIP Activations**: All active from block 0

### Mainnet
- **Header Version**: V0 (RSKIP-351/535 NOT YET ACTIVATED)
- **gasLimit Encoding**: Minimal bytes (e.g., `989680`)
- **ummRoot**: Included after Papyrus200 (block 2,392,700)
- **RSKIP Activations**:
  - Orchid (RSKIP-92): Block 729,000
  - Papyrus200 (UMM): Block 2,392,700
  - Reed810 (V1): NOT ACTIVATED (-1)
  - Vetiver900 (V2): NOT ACTIVATED (-1)

### Testnet
- **Header Version**: V1 after Reed810 (block 7,139,600)
- **gasLimit Encoding**: Minimal bytes
- **ummRoot**: Included after Papyrus200 (block 863,000)
- **RSKIP Activations**:
  - Orchid (RSKIP-92): Block 0
  - Papyrus200 (UMM): Block 863,000
  - Reed810 (V1): Block 7,139,600
  - Vetiver900 (V2): NOT ACTIVATED (-1)

### Configuration Summary

| Network | Version | gasLimit | ummRoot After | Use4ByteGasLimit |
|---------|---------|----------|---------------|------------------|
| Regtest | V2 | 4-byte | Block 0 | true |
| Mainnet | V0 | minimal | Block 2,392,700 | false |
| Testnet | V0/V1 | minimal | Block 863,000 | false |

## Encoding Details

- **gasLimit**: Network-specific (4-byte for regtest, minimal for mainnet/testnet)
- **minimumGasPrice**: 0 encodes as single zero byte (0x00), not empty (0x80)
- **TxExecutionSublistsEdges**: Empty array `[]` is different from `nil` in extension hash
- **baseEvent**: V2 only - included in extension hash even if empty/nil
- **ummRoot**: Auto-added as empty if `IncludeUmmRoot=true` and input is nil

### Auto-added Fields

When using `ConfigForBlockNumber()` or `DefaultRegtestConfig()`:

- **ummRoot**: If `IncludeUmmRoot=true` and input has no ummRoot, an empty ummRoot is automatically added
- **edges**: For V1/V2 headers, if input has nil edges, an empty edge array `[]` is automatically added (required for extensionData computation)

## Use Cases

1. **Light Clients**: Verify account balances without syncing full chain
2. **Cross-chain Bridges**: Prove state on RSK to another chain
3. **State Verification**: Trustlessly verify contract storage values
4. **Fraud Proofs**: Prove incorrect state transitions

## eth_getProof on RSK

> **Note**: The `eth_getProof` RPC is not part of rskj yet. There is a PR from Fede Jinich [PR-1519](https://github.com/rsksmart/rskj/pull/1519) that we've merged locally for testing.

Basic usage:

```bash
curl -s -X POST http://localhost:4444 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_getProof",
    "params": ["0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826", [], "latest"],
    "id": 1
  }' | jq .
```

## Additional Examples

See `misc/account-proof-examples.md` for detailed examples including:
- EOA proof requests and responses
- Contract deployment for testing
- Storage slot calculations for mappings
- Response field reference

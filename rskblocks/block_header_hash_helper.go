// Package rskblocks provides RSK-specific block, transaction, and receipt encoding.
package rskblocks

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BlockHeaderHashHelper provides methods to compute RSK block header hashes.
// Ported from co.rsk.core.bc.BlockHashesHelper.java and org.ethereum.core.BlockHeader.java
//
// # Key RSKIPs for Block Hash Computation
//
// RSKIP-92: Excludes merged mining merkle proof and coinbase transaction from
// the block hash computation. This is enabled for all post-Orchid blocks.
//
// RSKIP-351: Introduces V1 block headers where the logsBloom field is replaced
// by extensionData in the RLP encoding used for hashing. The extensionData is
// computed as: RLP([version, extensionHash]) where extensionHash =
// Keccak256(RLP([Keccak256(logsBloom), edgesBytes]))
//
// RSKIP-UMM (Unified Mining Merkle): Adds ummRoot field to block headers.
// When active, ummRoot is included in the encoding even if empty.
//
// RSKIP-144: Introduces TxExecutionSublistsEdges for parallel transaction
// execution. For V1 headers, empty edges [] is different from nil in the
// extension hash computation.
//
// # Encoding Details
//
// gasLimit: Must be stored as a 4-byte array with leading zeros preserved.
// RSK's Java implementation stores gasLimit as byte[] and encodes it directly,
// preserving any leading zeros from the original block data.
//
// minimumGasPrice: Zero value encodes as a single zero byte (0x00), NOT as
// empty (0x80). This uses RSK's encodeSignedCoinNonNullZero encoding.
//
// TxExecutionSublistsEdges: For V1 headers, an empty array [] must be included
// in the extension hash computation (encodes to 0x80), while nil means the
// field is not present at all.
type BlockHeaderHashHelper struct{}

// BlockHeaderInput represents the input data needed to compute a block header hash.
// This can be populated from RPC data or other sources.
type BlockHeaderInput struct {
	// Core fields
	ParentHash      common.Hash
	UnclesHash      common.Hash
	Coinbase        common.Address
	StateRoot       common.Hash
	TxTrieRoot      common.Hash
	ReceiptTrieRoot common.Hash
	LogsBloom       [256]byte
	Difficulty      *big.Int
	Number          *big.Int
	GasLimit        *big.Int // Will be converted to 4-byte array
	GasUsed         *big.Int
	Timestamp       *big.Int
	ExtraData       []byte
	PaidFees        *big.Int
	MinimumGasPrice *big.Int
	UncleCount      int

	// Bitcoin merged mining fields
	BitcoinMergedMiningHeader              []byte
	BitcoinMergedMiningMerkleProof         []byte
	BitcoinMergedMiningCoinbaseTransaction []byte

	// Optional fields
	TxExecutionSublistsEdges []int16 // RSKIP-144 parallel transaction execution edges
	UmmRoot                  *[]byte // UMM root (nil = not present, empty = present but empty)
	BaseEvent                []byte  // RSKIP-535 base event (V2 headers)
}

// BlockHashConfig contains configuration options for block hash computation.
type BlockHashConfig struct {
	// UseRskip92Encoding: If true, excludes merkle proof and coinbase from hash (default: true for post-orchid)
	UseRskip92Encoding bool

	// Version: Block header version (0 for V0, 1 for V1, 2 for V2)
	Version byte

	// IncludeUmmRoot: If true, includes UMM root in encoding (default: true for post-UMM activation)
	IncludeUmmRoot bool

	// Use4ByteGasLimit: If true, pad gasLimit to 4 bytes (regtest). If false, use minimal bytes (mainnet/testnet).
	Use4ByteGasLimit bool
}

// DefaultRegtestConfig returns the default configuration for regtest mode.
// All RSKIPs are active from block 0 in regtest, including RSKIP-535 (V2 headers).
func DefaultRegtestConfig() BlockHashConfig {
	return BlockHashConfig{
		UseRskip92Encoding: true,
		Version:            2, // V2 for RSKIP-535 (baseEvent support)
		IncludeUmmRoot:     true,
		Use4ByteGasLimit:   true, // Regtest uses 4-byte gasLimit
	}
}

// ConfigForBlockNumber returns the appropriate config based on block number and network.
// These values are from RSKj's main.conf, testnet.conf, and reference.conf.
//
// IMPORTANT: IncludeUmmRoot only controls whether UMM activation is enabled for the network.
// The actual ummRoot should only be included if the block has one (check RPC response).
//
// Mainnet activation heights (from main.conf):
//   - orchid = 729000 (RSKIP-92)
//   - papyrus200 = 2392700 (UMM)
//   - reed810 = -1 (RSKIP-144, RSKIP-351/V1 - NOT YET ACTIVATED)
//   - vetiver900 = -1 (RSKIP-535/V2 - NOT YET ACTIVATED)
//
// Testnet activation heights (from testnet.conf):
//   - orchid = 0 (RSKIP-92)
//   - papyrus200 = 863000 (UMM)
//   - reed810 = 7139600 (RSKIP-144, RSKIP-351/V1)
//   - vetiver900 = -1 (RSKIP-535/V2 - NOT YET ACTIVATED)
//
// Regtest: All RSKIPs active from genesis, uses V2 headers
func ConfigForBlockNumber(blockNum int64, network string) BlockHashConfig {
	switch network {
	case "regtest":
		// In regtest, all RSKIPs are active from genesis (block 0)
		// This includes RSKIP-535 (V2 headers with baseEvent)
		return BlockHashConfig{
			UseRskip92Encoding: true,
			Version:            2, // V2 for RSKIP-535
			IncludeUmmRoot:     true,
			Use4ByteGasLimit:   true, // Regtest uses 4-byte gasLimit
		}
	case "mainnet":
		// Mainnet: RSKIP-351 (V1) and RSKIP-535 (V2) are NOT YET ACTIVE
		// UMM is active from papyrus200 (2392700)
		return BlockHashConfig{
			UseRskip92Encoding: blockNum >= 729000,  // orchid
			Version:            0,                   // RSKIP-351 NOT active (reed810 = -1)
			IncludeUmmRoot:     blockNum >= 2392700, // UMM active from papyrus200
			Use4ByteGasLimit:   false,               // Mainnet uses minimal gasLimit
		}
	case "testnet":
		// Testnet: RSKIP-351 (V1) activated at reed810 = 7139600
		// RSKIP-535 (V2) NOT YET ACTIVE (vetiver900 = -1)
		var version byte = 0
		if blockNum >= 7139600 {
			version = 1 // V1 after reed810
		}
		return BlockHashConfig{
			UseRskip92Encoding: true, // orchid = 0
			Version:            version,
			IncludeUmmRoot:     blockNum >= 863000, // UMM active from papyrus200
			Use4ByteGasLimit:   false,              // Testnet uses minimal gasLimit
		}
	default:
		// Default to regtest behavior
		return DefaultRegtestConfig()
	}
}

// ComputeBlockHash computes the block hash from the given input and configuration.
// This is the main entry point for computing block hashes.
func ComputeBlockHash(input *BlockHeaderInput, config BlockHashConfig) common.Hash {
	header := InputToBlockHeader(input, config)
	return header.Hash()
}

// InputToBlockHeader converts BlockHeaderInput to a BlockHeader struct
// with proper encoding rules applied.
func InputToBlockHeader(input *BlockHeaderInput, config BlockHashConfig) *BlockHeader {
	// GasLimit encoding depends on network:
	// - Regtest: 4-byte with leading zeros (Use4ByteGasLimit=true)
	// - Mainnet/Testnet: minimal bytes (Use4ByteGasLimit=false)
	var gasLimitBytes []byte
	if input.GasLimit != nil && input.GasLimit.Sign() > 0 {
		if config.Use4ByteGasLimit {
			gasLimitBytes = make([]byte, 4)
			input.GasLimit.FillBytes(gasLimitBytes) // 4-byte with leading zeros
		} else {
			gasLimitBytes = input.GasLimit.Bytes() // Minimal bytes
		}
	} else {
		gasLimitBytes = []byte{} // Empty for zero
	}

	header := &BlockHeader{
		ParentHash:      input.ParentHash,
		UnclesHash:      input.UnclesHash,
		Coinbase:        input.Coinbase,
		StateRoot:       input.StateRoot,
		TxTrieRoot:      input.TxTrieRoot,
		ReceiptTrieRoot: input.ReceiptTrieRoot,
		LogsBloom:       input.LogsBloom,
		Difficulty:      input.Difficulty,
		Number:          input.Number,
		GasLimit:        gasLimitBytes,
		GasUsed:         input.GasUsed,
		Timestamp:       input.Timestamp,
		ExtraData:       input.ExtraData,
		PaidFees:        input.PaidFees,
		MinimumGasPrice: input.MinimumGasPrice,
		UncleCount:      input.UncleCount,

		// Bitcoin merged mining fields
		BitcoinMergedMiningHeader:              input.BitcoinMergedMiningHeader,
		BitcoinMergedMiningMerkleProof:         input.BitcoinMergedMiningMerkleProof,
		BitcoinMergedMiningCoinbaseTransaction: input.BitcoinMergedMiningCoinbaseTransaction,

		// Configuration
		UseRskip92Encoding: config.UseRskip92Encoding,
		Version:            config.Version,
	}

	// UmmRoot handling:
	// For regtest, all RSKIPs are active, so ummRoot should always be included (even if empty).
	// For mainnet/testnet, only include if explicitly present in the RPC response.
	// The config.IncludeUmmRoot flag indicates if UMM is active for the network.
	if config.IncludeUmmRoot {
		if input.UmmRoot != nil {
			header.UmmRoot = input.UmmRoot
		} else {
			// Auto-add empty ummRoot for networks where UMM is active (e.g., regtest)
			emptyUmm := []byte{}
			header.UmmRoot = &emptyUmm
		}
	} else if input.UmmRoot != nil {
		// UMM not active by config, but input explicitly has ummRoot (shouldn't happen normally)
		header.UmmRoot = input.UmmRoot
	}

	// TxExecutionSublistsEdges handling
	// RSKj includes edges if not null (even if empty array)
	// For V1/V2 compressed headers, edges are in extensionData
	// For V0 and V1/V2 non-compressed, edges are added as a separate field if not null
	if input.TxExecutionSublistsEdges != nil {
		header.TxExecutionSublistsEdges = input.TxExecutionSublistsEdges
	} else if config.Version >= 1 {
		// V1/V2 headers always have edges (even if empty) for extensionData computation
		header.TxExecutionSublistsEdges = []int16{}
	}
	// For V0 with nil edges, leave as nil (won't be included in encoding)

	// BaseEvent (V2 only)
	// RSKj includes baseEvent in V2 extension hash even if empty
	if config.Version == 2 {
		header.BaseEvent = input.BaseEvent // nil or empty both work
	}

	return header
}

// GetEncodedBlockHeader returns the RLP-encoded block header for hash computation.
func GetEncodedBlockHeader(input *BlockHeaderInput, config BlockHashConfig) []byte {
	header := InputToBlockHeader(input, config)
	return header.GetEncodedForHash()
}

// Helper functions for creating BlockHeaderInput from various formats

// NewBlockHeaderInputFromHex creates a BlockHeaderInput from hex-encoded values.
// This is useful when parsing data from JSON-RPC responses.
func NewBlockHeaderInputFromHex(
	parentHash, unclesHash, coinbase, stateRoot, txTrieRoot, receiptTrieRoot string,
	logsBloom []byte,
	difficulty, number, gasLimit, gasUsed, timestamp *big.Int,
	extraData []byte,
	paidFees, minimumGasPrice *big.Int,
	uncleCount int,
	bitcoinMergedMiningHeader, bitcoinMergedMiningMerkleProof, bitcoinMergedMiningCoinbaseTransaction []byte,
	txExecutionSublistsEdges []int16,
) *BlockHeaderInput {
	input := &BlockHeaderInput{
		ParentHash:                             common.HexToHash(parentHash),
		UnclesHash:                             common.HexToHash(unclesHash),
		Coinbase:                               common.HexToAddress(coinbase),
		StateRoot:                              common.HexToHash(stateRoot),
		TxTrieRoot:                             common.HexToHash(txTrieRoot),
		ReceiptTrieRoot:                        common.HexToHash(receiptTrieRoot),
		Difficulty:                             difficulty,
		Number:                                 number,
		GasLimit:                               gasLimit,
		GasUsed:                                gasUsed,
		Timestamp:                              timestamp,
		ExtraData:                              extraData,
		PaidFees:                               paidFees,
		MinimumGasPrice:                        minimumGasPrice,
		UncleCount:                             uncleCount,
		BitcoinMergedMiningHeader:              bitcoinMergedMiningHeader,
		BitcoinMergedMiningMerkleProof:         bitcoinMergedMiningMerkleProof,
		BitcoinMergedMiningCoinbaseTransaction: bitcoinMergedMiningCoinbaseTransaction,
		TxExecutionSublistsEdges:               txExecutionSublistsEdges,
	}

	if len(logsBloom) == 256 {
		copy(input.LogsBloom[:], logsBloom)
	}

	return input
}

// boolToByte converts a boolean to a byte (0 or 1)
func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

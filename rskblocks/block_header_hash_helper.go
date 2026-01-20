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
}

// BlockHashConfig contains configuration options for block hash computation.
type BlockHashConfig struct {
	// UseRskip92Encoding: If true, excludes merkle proof and coinbase from hash (default: true for post-orchid)
	UseRskip92Encoding bool

	// Version: Block header version (0 for V0, 1 for V1 per RSKIP-351)
	Version byte

	// IncludeUmmRoot: If true, includes UMM root in encoding (default: true for post-UMM activation)
	IncludeUmmRoot bool
}

// DefaultRegtestConfig returns the default configuration for regtest mode.
// All RSKIPs are active from block 0 in regtest.
func DefaultRegtestConfig() BlockHashConfig {
	return BlockHashConfig{
		UseRskip92Encoding: true,
		Version:            1,
		IncludeUmmRoot:     true,
	}
}

// ConfigForBlockNumber returns the appropriate config based on block number and network.
// For now, this only supports regtest where all RSKIPs are active.
func ConfigForBlockNumber(blockNum int64, network string) BlockHashConfig {
	switch network {
	case "regtest":
		// In regtest, all RSKIPs are active from genesis
		return BlockHashConfig{
			UseRskip92Encoding: true,
			Version:            1,
			IncludeUmmRoot:     true,
		}
	case "mainnet":
		// Mainnet activation heights (approximate)
		return BlockHashConfig{
			UseRskip92Encoding: blockNum >= 0,                   // Orchid was early
			Version:            boolToByte(blockNum >= 5468000), // RSKIP-351
			IncludeUmmRoot:     blockNum >= 4598500,             // UMM activation
		}
	case "testnet":
		return BlockHashConfig{
			UseRskip92Encoding: true,
			Version:            1,
			IncludeUmmRoot:     true,
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
	// GasLimit must be stored as 4-byte array with leading zeros preserved
	gasLimitBytes := make([]byte, 4)
	if input.GasLimit != nil {
		input.GasLimit.FillBytes(gasLimitBytes)
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

	// UmmRoot: present (even if empty) when IncludeUmmRoot is true
	if config.IncludeUmmRoot {
		emptyUmmRoot := []byte{}
		header.UmmRoot = &emptyUmmRoot
	}

	// TxExecutionSublistsEdges handling
	// For V1 headers, empty edges [] is different from nil
	if config.Version == 1 {
		// V1 headers always have edges (even if empty)
		if input.TxExecutionSublistsEdges != nil {
			header.TxExecutionSublistsEdges = input.TxExecutionSublistsEdges
		} else {
			header.TxExecutionSublistsEdges = []int16{}
		}
	} else if len(input.TxExecutionSublistsEdges) > 0 {
		// V0 headers only include edges if present
		header.TxExecutionSublistsEdges = input.TxExecutionSublistsEdges
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

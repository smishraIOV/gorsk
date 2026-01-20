package rskblocks

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Test vectors from RSK regtest node block 1
// These were verified against the actual RSK Java implementation
func TestComputeBlockHashBlock1(t *testing.T) {
	// Block 1 data from RSK regtest
	input := &BlockHeaderInput{
		ParentHash:                common.HexToHash("0x8ea789fabef0dd4946ed53f001e7b6f8a8d0c22a612a6099fc7f93c990af68fe"),
		UnclesHash:                common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:                  common.HexToAddress("0xec4ddeb4380ad69b3e509baad9f158cdf4e4681d"),
		StateRoot:                 common.HexToHash("0xf276a3a8c9c4eb4dcbbfb9bf6965f36dc611b815614c0d7cd06e15b8890c272c"),
		TxTrieRoot:                common.HexToHash("0x8c9664a30670ddc67aa13992fdd8751b7b797bbe172506ffd5cda10ebbf97952"),
		ReceiptTrieRoot:           common.HexToHash("0x66cfdb731f620cd96e2c2cb0f7d3c3a2879c29b40014aa27efbbf3cf9cd3b0f6"),
		Difficulty:                big.NewInt(1),
		Number:                    big.NewInt(1),
		GasLimit:                  big.NewInt(10000000), // 0x989680
		GasUsed:                   big.NewInt(0),
		Timestamp:                 big.NewInt(1768848306), // 0x696e7bb2
		ExtraData:                 hexToBytes("d40192534e415053484f542d343661653533386532"),
		PaidFees:                  big.NewInt(0),
		MinimumGasPrice:           big.NewInt(0),
		UncleCount:                0,
		BitcoinMergedMiningHeader: hexToBytes("711101000000000000000000000000000000000000000000000000000000000000000000f452061f4aea40ba9a9dca66ce43eaad20e9d59b4d19fa4415dd22d3de8c2ec3b27b6e69ffff7f2103000000"),
		TxExecutionSublistsEdges:  []int16{}, // Empty but not nil
	}

	// LogsBloom is all zeros for block 1
	// (REMASC transaction doesn't generate logs)

	config := DefaultRegtestConfig()

	// Expected hash from RSK node
	expectedHash := common.HexToHash("0xfe10930e0ad3742e3fe1d7200c4d54573d58033cc03484137dc7ec5a761ecffe")

	// Compute hash
	computedHash := ComputeBlockHash(input, config)

	if computedHash != expectedHash {
		t.Errorf("Block hash mismatch\n  Expected: %s\n  Computed: %s", expectedHash.Hex(), computedHash.Hex())
	}
}

// Test the RLP encoding matches Java's output exactly
func TestBlockHeaderEncodingBlock1(t *testing.T) {
	input := &BlockHeaderInput{
		ParentHash:                common.HexToHash("0x8ea789fabef0dd4946ed53f001e7b6f8a8d0c22a612a6099fc7f93c990af68fe"),
		UnclesHash:                common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:                  common.HexToAddress("0xec4ddeb4380ad69b3e509baad9f158cdf4e4681d"),
		StateRoot:                 common.HexToHash("0xf276a3a8c9c4eb4dcbbfb9bf6965f36dc611b815614c0d7cd06e15b8890c272c"),
		TxTrieRoot:                common.HexToHash("0x8c9664a30670ddc67aa13992fdd8751b7b797bbe172506ffd5cda10ebbf97952"),
		ReceiptTrieRoot:           common.HexToHash("0x66cfdb731f620cd96e2c2cb0f7d3c3a2879c29b40014aa27efbbf3cf9cd3b0f6"),
		Difficulty:                big.NewInt(1),
		Number:                    big.NewInt(1),
		GasLimit:                  big.NewInt(10000000),
		GasUsed:                   big.NewInt(0),
		Timestamp:                 big.NewInt(1768848306),
		ExtraData:                 hexToBytes("d40192534e415053484f542d343661653533386532"),
		PaidFees:                  big.NewInt(0),
		MinimumGasPrice:           big.NewInt(0),
		UncleCount:                0,
		BitcoinMergedMiningHeader: hexToBytes("711101000000000000000000000000000000000000000000000000000000000000000000f452061f4aea40ba9a9dca66ce43eaad20e9d59b4d19fa4415dd22d3de8c2ec3b27b6e69ffff7f2103000000"),
		TxExecutionSublistsEdges:  []int16{},
	}

	config := DefaultRegtestConfig()

	// Expected encoding from Java BlockHeader.getEncodedForHash()
	expectedEncoding := "f90157a08ea789fabef0dd4946ed53f001e7b6f8a8d0c22a612a6099fc7f93c990af68fea01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d4934794ec4ddeb4380ad69b3e509baad9f158cdf4e4681da0f276a3a8c9c4eb4dcbbfb9bf6965f36dc611b815614c0d7cd06e15b8890c272ca08c9664a30670ddc67aa13992fdd8751b7b797bbe172506ffd5cda10ebbf97952a066cfdb731f620cd96e2c2cb0f7d3c3a2879c29b40014aa27efbbf3cf9cd3b0f6a3e201a0339bef53725a6ddba6dd20e727e3fe46866efc57a35164222577762e02d8c6ef010184009896808084696e7bb295d40192534e415053484f542d34366165353338653280008080b850711101000000000000000000000000000000000000000000000000000000000000000000f452061f4aea40ba9a9dca66ce43eaad20e9d59b4d19fa4415dd22d3de8c2ec3b27b6e69ffff7f2103000000"

	encoded := GetEncodedBlockHeader(input, config)
	encodedHex := hex.EncodeToString(encoded)

	if encodedHex != expectedEncoding {
		t.Errorf("Encoding mismatch\n  Expected length: %d\n  Computed length: %d\n  Expected: %s\n  Computed: %s",
			len(expectedEncoding)/2, len(encoded), expectedEncoding, encodedHex)
	}
}

// Test extension hash computation for V1 headers (RSKIP-351)
func TestExtensionHashComputation(t *testing.T) {
	input := &BlockHeaderInput{
		ParentHash:               common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		UnclesHash:               common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		StateRoot:                common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		TxTrieRoot:               common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ReceiptTrieRoot:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Difficulty:               big.NewInt(1),
		Number:                   big.NewInt(1),
		GasLimit:                 big.NewInt(10000000),
		GasUsed:                  big.NewInt(0),
		Timestamp:                big.NewInt(1000000),
		PaidFees:                 big.NewInt(0),
		MinimumGasPrice:          big.NewInt(0),
		TxExecutionSublistsEdges: []int16{}, // Empty edges
	}
	// LogsBloom is all zeros (default)

	config := BlockHashConfig{
		UseRskip92Encoding: true,
		Version:            1,
		IncludeUmmRoot:     true,
	}

	header := InputToBlockHeader(input, config)

	// The extension hash for logsBloom=zeros and edges=[] should be:
	// Keccak256(RLP([Keccak256(logsBloom), edgesBytes]))
	// = Keccak256(RLP([d397b3b043d87fcd6fad1291ff0bfd16401c274896d8c63a923727f077b8e0b5, []]))
	// = 339bef53725a6ddba6dd20e727e3fe46866efc57a35164222577762e02d8c6ef

	// The extensionData should be RLP([version=1, extensionHash])
	expectedExtensionHash := "339bef53725a6ddba6dd20e727e3fe46866efc57a35164222577762e02d8c6ef"

	// Get the encoded header and check it contains the expected extension hash
	encoded := header.GetEncodedForHash()
	encodedHex := hex.EncodeToString(encoded)

	if !contains(encodedHex, expectedExtensionHash) {
		t.Errorf("Extension hash not found in encoded header\n  Expected to contain: %s\n  Encoded: %s",
			expectedExtensionHash, encodedHex)
	}
}

// Test configuration for different networks
func TestConfigForBlockNumber(t *testing.T) {
	tests := []struct {
		name     string
		blockNum int64
		network  string
		expected BlockHashConfig
	}{
		{
			name:     "regtest block 0",
			blockNum: 0,
			network:  "regtest",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 1, IncludeUmmRoot: true},
		},
		{
			name:     "regtest block 100",
			blockNum: 100,
			network:  "regtest",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 1, IncludeUmmRoot: true},
		},
		{
			name:     "mainnet pre-RSKIP351",
			blockNum: 5000000,
			network:  "mainnet",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 0, IncludeUmmRoot: true},
		},
		{
			name:     "mainnet post-RSKIP351",
			blockNum: 6000000,
			network:  "mainnet",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 1, IncludeUmmRoot: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConfigForBlockNumber(tt.blockNum, tt.network)
			if config.UseRskip92Encoding != tt.expected.UseRskip92Encoding {
				t.Errorf("UseRskip92Encoding: expected %v, got %v", tt.expected.UseRskip92Encoding, config.UseRskip92Encoding)
			}
			if config.Version != tt.expected.Version {
				t.Errorf("Version: expected %d, got %d", tt.expected.Version, config.Version)
			}
			if config.IncludeUmmRoot != tt.expected.IncludeUmmRoot {
				t.Errorf("IncludeUmmRoot: expected %v, got %v", tt.expected.IncludeUmmRoot, config.IncludeUmmRoot)
			}
		})
	}
}

// Test gasLimit encoding preserves leading zeros
func TestGasLimitEncoding(t *testing.T) {
	input := &BlockHeaderInput{
		GasLimit: big.NewInt(10000000), // 0x989680 - 3 bytes without leading zero
		Number:   big.NewInt(1),
	}

	config := DefaultRegtestConfig()
	header := InputToBlockHeader(input, config)

	// Should be padded to 4 bytes: 00989680
	expectedGasLimit := []byte{0x00, 0x98, 0x96, 0x80}

	if len(header.GasLimit) != 4 {
		t.Errorf("GasLimit should be 4 bytes, got %d", len(header.GasLimit))
	}

	for i, b := range expectedGasLimit {
		if header.GasLimit[i] != b {
			t.Errorf("GasLimit byte %d: expected 0x%02x, got 0x%02x", i, b, header.GasLimit[i])
		}
	}
}

// Test V1 header always has edges (even if empty)
func TestV1HeaderAlwaysHasEdges(t *testing.T) {
	input := &BlockHeaderInput{
		Number:                   big.NewInt(1),
		TxExecutionSublistsEdges: nil, // Input has nil edges
	}

	config := BlockHashConfig{
		Version: 1,
	}

	header := InputToBlockHeader(input, config)

	// V1 header should have empty edges, not nil
	if header.TxExecutionSublistsEdges == nil {
		t.Error("V1 header should have non-nil edges (empty slice)")
	}

	if len(header.TxExecutionSublistsEdges) != 0 {
		t.Errorf("V1 header should have empty edges, got %d edges", len(header.TxExecutionSublistsEdges))
	}
}

// Test V0 header doesn't add edges if input has none
func TestV0HeaderNoEdgesIfNone(t *testing.T) {
	input := &BlockHeaderInput{
		Number:                   big.NewInt(1),
		TxExecutionSublistsEdges: nil,
	}

	config := BlockHashConfig{
		Version: 0,
	}

	header := InputToBlockHeader(input, config)

	// V0 header should keep nil edges
	if header.TxExecutionSublistsEdges != nil {
		t.Error("V0 header should have nil edges when input has nil")
	}
}

// Helper function
func hexToBytes(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

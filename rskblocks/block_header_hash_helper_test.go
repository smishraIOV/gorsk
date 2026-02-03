package rskblocks

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Test vectors from RSK regtest node block 1
// These were verified against the actual RSK Java implementation (V2 headers with RSKIP-535)
func TestComputeBlockHashBlock1(t *testing.T) {
	// Block 1 data from RSK regtest (fresh node with V2 headers)
	input := &BlockHeaderInput{
		ParentHash:               common.HexToHash("0x8ea789fabef0dd4946ed53f001e7b6f8a8d0c22a612a6099fc7f93c990af68fe"),
		UnclesHash:               common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:                 common.HexToAddress("0xec4ddeb4380ad69b3e509baad9f158cdf4e4681d"),
		StateRoot:                common.HexToHash("0xf276a3a8c9c4eb4dcbbfb9bf6965f36dc611b815614c0d7cd06e15b8890c272c"),
		TxTrieRoot:               common.HexToHash("0x8c9664a30670ddc67aa13992fdd8751b7b797bbe172506ffd5cda10ebbf97952"),
		ReceiptTrieRoot:          common.HexToHash("0x66cfdb731f620cd96e2c2cb0f7d3c3a2879c29b40014aa27efbbf3cf9cd3b0f6"),
		Difficulty:               big.NewInt(1),
		Number:                   big.NewInt(1),
		GasLimit:                 big.NewInt(10000000), // 0x989680
		GasUsed:                  big.NewInt(0),
		Timestamp:                big.NewInt(0x69824213),
		ExtraData:                hexToBytes("d40192534e415053484f542d343031373966623937"),
		PaidFees:                 big.NewInt(0),
		MinimumGasPrice:          big.NewInt(0),
		UncleCount:               0,
		TxExecutionSublistsEdges: []int16{}, // Empty but not nil
		// No btcHeader for this test block
	}

	// LogsBloom is all zeros for block 1
	// (REMASC transaction doesn't generate logs)

	config := DefaultRegtestConfig() // V2 with IncludeUmmRoot=true

	// Expected hash from RSK node (V2 encoding)
	expectedHash := common.HexToHash("0x90299cad077d0759beee6c9625be98114874d9ae65ede6979752a97112043b63")

	// Compute hash
	computedHash := ComputeBlockHash(input, config)

	if computedHash != expectedHash {
		t.Errorf("Block hash mismatch\n  Expected: %s\n  Computed: %s", expectedHash.Hex(), computedHash.Hex())
	}
}

// Test the RLP encoding matches Java's output exactly (V2 headers)
func TestBlockHeaderEncodingBlock1(t *testing.T) {
	input := &BlockHeaderInput{
		ParentHash:               common.HexToHash("0x8ea789fabef0dd4946ed53f001e7b6f8a8d0c22a612a6099fc7f93c990af68fe"),
		UnclesHash:               common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:                 common.HexToAddress("0xec4ddeb4380ad69b3e509baad9f158cdf4e4681d"),
		StateRoot:                common.HexToHash("0xf276a3a8c9c4eb4dcbbfb9bf6965f36dc611b815614c0d7cd06e15b8890c272c"),
		TxTrieRoot:               common.HexToHash("0x8c9664a30670ddc67aa13992fdd8751b7b797bbe172506ffd5cda10ebbf97952"),
		ReceiptTrieRoot:          common.HexToHash("0x66cfdb731f620cd96e2c2cb0f7d3c3a2879c29b40014aa27efbbf3cf9cd3b0f6"),
		Difficulty:               big.NewInt(1),
		Number:                   big.NewInt(1),
		GasLimit:                 big.NewInt(10000000),
		GasUsed:                  big.NewInt(0),
		Timestamp:                big.NewInt(0x69824213),
		ExtraData:                hexToBytes("d40192534e415053484f542d343031373966623937"),
		PaidFees:                 big.NewInt(0),
		MinimumGasPrice:          big.NewInt(0),
		UncleCount:               0,
		TxExecutionSublistsEdges: []int16{},
		// No btcHeader for this test
	}

	config := DefaultRegtestConfig() // V2

	// Expected encoding from Java BlockHeader.getEncodedForHash() with V2 headers
	expectedEncoding := "f90105a08ea789fabef0dd4946ed53f001e7b6f8a8d0c22a612a6099fc7f93c990af68fea01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d4934794ec4ddeb4380ad69b3e509baad9f158cdf4e4681da0f276a3a8c9c4eb4dcbbfb9bf6965f36dc611b815614c0d7cd06e15b8890c272ca08c9664a30670ddc67aa13992fdd8751b7b797bbe172506ffd5cda10ebbf97952a066cfdb731f620cd96e2c2cb0f7d3c3a2879c29b40014aa27efbbf3cf9cd3b0f6a3e202a09aca8469839f117b1a26abbdea244a32cb0833e387cf6af9dc13eb336094d3c20101840098968080846982421395d40192534e415053484f542d34303137396662393780008080"

	encoded := GetEncodedBlockHeader(input, config)
	encodedHex := hex.EncodeToString(encoded)

	if encodedHex != expectedEncoding {
		t.Errorf("Encoding mismatch\n  Expected length: %d\n  Computed length: %d\n  Expected: %s\n  Computed: %s",
			len(expectedEncoding)/2, len(encoded), expectedEncoding, encodedHex)
	}
}

// Test extension hash computation for V2 headers (RSKIP-535)
func TestExtensionHashComputationV2(t *testing.T) {
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
		BaseEvent:                nil,       // No baseEvent
	}
	// LogsBloom is all zeros (default)

	config := BlockHashConfig{
		UseRskip92Encoding: true,
		Version:            2,
		IncludeUmmRoot:     true,
	}

	header := InputToBlockHeader(input, config)

	// The V2 extension hash for logsBloom=zeros, baseEvent=nil, edges=[] should be:
	// Keccak256(RLP([Keccak256(logsBloom), emptyBaseEvent, edgesBytes]))
	// = Keccak256(RLP([d397b3b043d87fcd6fad1291ff0bfd16401c274896d8c63a923727f077b8e0b5, 0x80, 0x80]))
	// = 9aca8469839f117b1a26abbdea244a32cb0833e387cf6af9dc13eb336094d3c2

	// The extensionData should be RLP([version=2, extensionHash])
	expectedExtensionHash := "9aca8469839f117b1a26abbdea244a32cb0833e387cf6af9dc13eb336094d3c2"

	// Get the encoded header and check it contains the expected extension hash
	encoded := header.GetEncodedForHash()
	encodedHex := hex.EncodeToString(encoded)

	if !contains(encodedHex, expectedExtensionHash) {
		t.Errorf("V2 Extension hash not found in encoded header\n  Expected to contain: %s\n  Encoded: %s",
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
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 2, IncludeUmmRoot: true, Use4ByteGasLimit: true},
		},
		{
			name:     "regtest block 100",
			blockNum: 100,
			network:  "regtest",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 2, IncludeUmmRoot: true, Use4ByteGasLimit: true},
		},
		{
			name:     "mainnet post-UMM (V0)",
			blockNum: 5000000,
			network:  "mainnet",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 0, IncludeUmmRoot: true, Use4ByteGasLimit: false},
		},
		{
			name:     "mainnet current (V0)",
			blockNum: 8000000,
			network:  "mainnet",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 0, IncludeUmmRoot: true, Use4ByteGasLimit: false},
		},
		{
			name:     "mainnet pre-UMM",
			blockNum: 2000000,
			network:  "mainnet",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 0, IncludeUmmRoot: false, Use4ByteGasLimit: false},
		},
		{
			name:     "testnet V1",
			blockNum: 7200000,
			network:  "testnet",
			expected: BlockHashConfig{UseRskip92Encoding: true, Version: 1, IncludeUmmRoot: true, Use4ByteGasLimit: false},
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
			if config.Use4ByteGasLimit != tt.expected.Use4ByteGasLimit {
				t.Errorf("Use4ByteGasLimit: expected %v, got %v", tt.expected.Use4ByteGasLimit, config.Use4ByteGasLimit)
			}
		})
	}
}

// Test gasLimit encoding - regtest uses 4-byte, mainnet uses minimal
func TestGasLimitEncoding(t *testing.T) {
	input := &BlockHeaderInput{
		GasLimit: big.NewInt(10000000), // 0x989680 - 3 bytes minimal
		Number:   big.NewInt(1),
	}

	// Regtest: 4-byte with leading zeros
	configRegtest := DefaultRegtestConfig()
	headerRegtest := InputToBlockHeader(input, configRegtest)

	expectedGasLimit4Byte := []byte{0x00, 0x98, 0x96, 0x80}
	if len(headerRegtest.GasLimit) != 4 {
		t.Errorf("Regtest GasLimit should be 4 bytes, got %d", len(headerRegtest.GasLimit))
	}
	for i, b := range expectedGasLimit4Byte {
		if headerRegtest.GasLimit[i] != b {
			t.Errorf("Regtest GasLimit byte %d: expected 0x%02x, got 0x%02x", i, b, headerRegtest.GasLimit[i])
		}
	}

	// Mainnet: minimal bytes
	configMainnet := ConfigForBlockNumber(8000000, "mainnet")
	headerMainnet := InputToBlockHeader(input, configMainnet)

	expectedGasLimitMinimal := []byte{0x98, 0x96, 0x80}
	if len(headerMainnet.GasLimit) != 3 {
		t.Errorf("Mainnet GasLimit should be 3 bytes (minimal), got %d", len(headerMainnet.GasLimit))
	}
	for i, b := range expectedGasLimitMinimal {
		if headerMainnet.GasLimit[i] != b {
			t.Errorf("Mainnet GasLimit byte %d: expected 0x%02x, got 0x%02x", i, b, headerMainnet.GasLimit[i])
		}
	}
}

// Test V1/V2 header always has edges (even if empty)
func TestV1V2HeaderAlwaysHasEdges(t *testing.T) {
	for _, version := range []byte{1, 2} {
		input := &BlockHeaderInput{
			Number:                   big.NewInt(1),
			TxExecutionSublistsEdges: nil, // Input has nil edges
		}

		config := BlockHashConfig{
			Version: version,
		}

		header := InputToBlockHeader(input, config)

		// V1/V2 header should have empty edges, not nil
		if header.TxExecutionSublistsEdges == nil {
			t.Errorf("V%d header should have non-nil edges (empty slice)", version)
		}

		if len(header.TxExecutionSublistsEdges) != 0 {
			t.Errorf("V%d header should have empty edges, got %d edges", version, len(header.TxExecutionSublistsEdges))
		}
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

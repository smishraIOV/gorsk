package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"

	"gorsk/rskblocks"
	"gorsk/rsktrie"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

const rpcURL = "http://localhost:4444"

// JSON-RPC structures
type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// eth_getProof response structures
type ProofResponse struct {
	Balance      string         `json:"balance"`
	CodeHash     string         `json:"codeHash"`
	Nonce        string         `json:"nonce"`
	StorageHash  string         `json:"storageHash"`
	AccountProof []string       `json:"accountProof"`
	StorageProof []StorageProof `json:"storageProof"`
}

type StorageProof struct {
	Key    string   `json:"key"`
	Value  string   `json:"value"`
	Proofs []string `json:"proofs"`
}

// Block structure to get state root
type rpcBlock struct {
	StateRoot string `json:"stateRoot"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: verify_proof <address> [storage_key1,storage_key2,...] [block_id]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  verify_proof 0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")
		fmt.Println("  verify_proof 0x83C5541A6c8D2dBAD642f385d8d06Ca9B6C731ee 0x0")
		fmt.Println("  verify_proof 0x83C5541A6c8D2dBAD642f385d8d06Ca9B6C731ee 0x0,0x1 latest")
		os.Exit(1)
	}

	address := os.Args[1]
	var storageKeys []string
	blockID := "latest"

	if len(os.Args) > 2 && os.Args[2] != "" {
		storageKeys = strings.Split(os.Args[2], ",")
	}
	if len(os.Args) > 3 {
		blockID = os.Args[3]
	}

	fmt.Printf("Verifying eth_getProof for address: %s\n", address)
	fmt.Printf("Storage keys: %v\n", storageKeys)
	fmt.Printf("Block: %s\n", blockID)
	fmt.Println(strings.Repeat("=", 70))

	// 1. Get the state root for the block
	stateRoot, err := getStateRoot(blockID)
	if err != nil {
		log.Fatalf("Failed to get state root: %v", err)
	}
	fmt.Printf("State Root: %s\n", stateRoot.Hex())

	// 2. Call eth_getProof
	proof, err := getProof(address, storageKeys, blockID)
	if err != nil {
		log.Fatalf("Failed to get proof: %v", err)
	}

	fmt.Printf("\nProof Response:\n")
	fmt.Printf("  Balance:     %s\n", proof.Balance)
	fmt.Printf("  Nonce:       %s\n", proof.Nonce)
	fmt.Printf("  CodeHash:    %s\n", proof.CodeHash)
	fmt.Printf("  StorageHash: %s\n", proof.StorageHash)
	fmt.Printf("  AccountProof nodes: %d\n", len(proof.AccountProof))
	fmt.Printf("  StorageProofs: %d\n", len(proof.StorageProof))

	// 3. Verify the account proof
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("ACCOUNT PROOF VERIFICATION:")
	verifyAccountProof(stateRoot, common.HexToAddress(address), proof)

	// 4. Verify storage proofs
	if len(proof.StorageProof) > 0 {
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println("STORAGE PROOF VERIFICATION:")
		verifyStorageProofs(stateRoot, common.HexToAddress(address), proof)
	}
}

func verifyAccountProof(stateRoot common.Hash, address common.Address, proof *ProofResponse) {
	verifier := rskblocks.NewProofVerifier()

	// Decode proof nodes from hex using helper
	proofNodes, err := rskblocks.DecodeRLPProofNodes(proof.AccountProof)
	if err != nil {
		fmt.Printf("  ✗ Failed to decode proof nodes: %v\n", err)
		return
	}

	// Print proof node details - need to RLP decode first to get true hash
	fmt.Printf("\n  Proof nodes (RLP-encoded):\n")
	for i, rlpNode := range proofNodes {
		// RLP decode to get serialized node
		var serializedNode []byte
		if err := rlp.DecodeBytes(rlpNode, &serializedNode); err != nil {
			fmt.Printf("    [%d] RLP decode error: %v\n", i, err)
			continue
		}
		nodeHash := rsktrie.Keccak256(serializedNode)
		fmt.Printf("    [%d] %d bytes (serialized: %d), hash: %x\n", i, len(rlpNode), len(serializedNode), nodeHash)
	}
	fmt.Printf("  State root: %x\n", stateRoot[:])

	// Verify
	result, err := verifier.VerifyAccountProof(stateRoot, address, proofNodes)
	if err != nil {
		fmt.Printf("\n  ✗ Verification error: %v\n", err)
		return
	}

	if result.Valid {
		fmt.Printf("\n  ✓ Account proof VALID!\n")
		if result.Value != nil {
			fmt.Printf("    Value (RLP-encoded account state): %x\n", result.Value)
			// Decode account state
			decodeAccountState(result.Value)
		} else {
			fmt.Printf("    Account not found in trie (may be a new account or proof of non-existence)\n")
		}
	} else {
		fmt.Printf("\n  ✗ Account proof INVALID: %v\n", result.Error)
	}
}

func verifyStorageProofs(stateRoot common.Hash, address common.Address, proof *ProofResponse) {
	verifier := rskblocks.NewProofVerifier()

	for _, sp := range proof.StorageProof {
		fmt.Printf("\n  Storage key: %s\n", sp.Key)
		fmt.Printf("  Expected value: %s\n", sp.Value)

		// Decode proof nodes using helper
		proofNodes, err := rskblocks.DecodeRLPProofNodes(sp.Proofs)
		if err != nil {
			fmt.Printf("  ✗ Failed to decode storage proof nodes: %v\n", err)
			continue
		}

		fmt.Printf("  Proof nodes: %d\n", len(proofNodes))

		// Verify
		storageKey := common.HexToHash(sp.Key)
		result, err := verifier.VerifyStorageProof(stateRoot, address, storageKey, proofNodes)
		if err != nil {
			fmt.Printf("  ✗ Verification error: %v\n", err)
			continue
		}

		if result.Valid {
			fmt.Printf("  ✓ Storage proof VALID!\n")
			if result.Value != nil {
				fmt.Printf("    Retrieved value: %x\n", result.Value)
				// Compare with expected
				expectedValue := hexToBytes(sp.Value)
				if bytes.Equal(result.Value, expectedValue) {
					fmt.Printf("    ✓ Value matches expected!\n")
				} else {
					fmt.Printf("    ✗ Value mismatch! Expected: %x\n", expectedValue)
				}
			}
		} else {
			fmt.Printf("  ✗ Storage proof INVALID: %v\n", result.Error)
		}
	}
}

func decodeAccountState(encoded []byte) {
	// RSK account state is RLP-encoded: [nonce, balance, stateRoot, codeHash]
	// But the actual format might vary - let's try to decode it
	var decoded []interface{}
	if err := rlp.DecodeBytes(encoded, &decoded); err != nil {
		// Try as raw bytes
		fmt.Printf("    Raw value: %x\n", encoded)
		return
	}

	if len(decoded) >= 4 {
		fmt.Printf("    Decoded account state:\n")
		fmt.Printf("      Nonce:     %v\n", decoded[0])
		fmt.Printf("      Balance:   %v\n", decoded[1])
		fmt.Printf("      StateRoot: %x\n", decoded[2])
		fmt.Printf("      CodeHash:  %x\n", decoded[3])
	}
}

func rpcCall(method string, params []interface{}) (json.RawMessage, error) {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func getStateRoot(blockID string) (common.Hash, error) {
	result, err := rpcCall("eth_getBlockByNumber", []interface{}{blockID, false})
	if err != nil {
		return common.Hash{}, err
	}

	var block rpcBlock
	if err := json.Unmarshal(result, &block); err != nil {
		return common.Hash{}, fmt.Errorf("unmarshal block: %w", err)
	}

	return common.HexToHash(block.StateRoot), nil
}

func getProof(address string, storageKeys []string, blockID string) (*ProofResponse, error) {
	// Format storage keys
	formattedKeys := make([]string, len(storageKeys))
	for i, key := range storageKeys {
		if !strings.HasPrefix(key, "0x") {
			key = "0x" + key
		}
		// Pad to 32 bytes
		key = strings.TrimPrefix(key, "0x")
		formattedKeys[i] = "0x" + strings.Repeat("0", 64-len(key)) + key
	}

	result, err := rpcCall("eth_getProof", []interface{}{address, formattedKeys, blockID})
	if err != nil {
		return nil, err
	}

	var proof ProofResponse
	if err := json.Unmarshal(result, &proof); err != nil {
		return nil, fmt.Errorf("unmarshal proof: %w", err)
	}

	return &proof, nil
}

func hexToBytes(hexStr string) []byte {
	if hexStr == "" || hexStr == "0x" {
		return nil
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	data, _ := hex.DecodeString(hexStr)
	return data
}

func hexToBigInt(hexStr string) *big.Int {
	if hexStr == "" || hexStr == "0x" {
		return big.NewInt(0)
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	val := new(big.Int)
	val.SetString(hexStr, 16)
	return val
}

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
	"strconv"
	"strings"

	"gorsk/rskblocks"

	"github.com/ethereum/go-ethereum/common"
)

const rpcURL = "http://localhost:4444"

// JSON-RPC request/response structures
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

// Block structure from RSK RPC
type rpcBlock struct {
	Number           string   `json:"number"`
	Hash             string   `json:"hash"`
	ParentHash       string   `json:"parentHash"`
	Sha3Uncles       string   `json:"sha3Uncles"`
	Miner            string   `json:"miner"`
	StateRoot        string   `json:"stateRoot"`
	TransactionsRoot string   `json:"transactionsRoot"`
	ReceiptsRoot     string   `json:"receiptsRoot"`
	LogsBloom        string   `json:"logsBloom"`
	Difficulty       string   `json:"difficulty"`
	GasLimit         string   `json:"gasLimit"`
	GasUsed          string   `json:"gasUsed"`
	Timestamp        string   `json:"timestamp"`
	ExtraData        string   `json:"extraData"`
	MinimumGasPrice  string   `json:"minimumGasPrice"`
	PaidFees         string   `json:"paidFees"`
	Uncles           []string `json:"uncles"`
	Transactions     []rpcTx  `json:"transactions"`

	// Bitcoin merged mining fields
	BitcoinMergedMiningHeader              string `json:"bitcoinMergedMiningHeader"`
	BitcoinMergedMiningMerkleProof         string `json:"bitcoinMergedMiningMerkleProof"`
	BitcoinMergedMiningCoinbaseTransaction string `json:"bitcoinMergedMiningCoinbaseTransaction"`

	// Optional fields
	RskPteEdges []int `json:"rskPteEdges"`
}

// Transaction structure from RSK RPC
type rpcTx struct {
	Hash             string `json:"hash"`
	Nonce            string `json:"nonce"`
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	TransactionIndex string `json:"transactionIndex"`
	From             string `json:"from"`
	To               string `json:"to"`
	Value            string `json:"value"`
	GasPrice         string `json:"gasPrice"`
	Gas              string `json:"gas"`
	Input            string `json:"input"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
}

// Receipt structure from RSK RPC
type rpcReceipt struct {
	TransactionHash   string   `json:"transactionHash"`
	TransactionIndex  string   `json:"transactionIndex"`
	BlockHash         string   `json:"blockHash"`
	BlockNumber       string   `json:"blockNumber"`
	From              string   `json:"from"`
	To                string   `json:"to"`
	CumulativeGasUsed string   `json:"cumulativeGasUsed"`
	GasUsed           string   `json:"gasUsed"`
	ContractAddress   string   `json:"contractAddress"`
	Logs              []rpcLog `json:"logs"`
	LogsBloom         string   `json:"logsBloom"`
	Status            string   `json:"status"`
	Root              string   `json:"root"`
}

type rpcLog struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	BlockHash        string   `json:"blockHash"`
	LogIndex         string   `json:"logIndex"`
	Removed          bool     `json:"removed"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: verify_roots <block_number>")
	}

	blockNum, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("Invalid block number: %v", err)
	}

	fmt.Printf("Fetching block %d from RSK node at %s\n", blockNum, rpcURL)
	fmt.Println(strings.Repeat("=", 60))

	// 1. Get block with full transactions
	block, err := getBlockByNumber(blockNum)
	if err != nil {
		log.Fatalf("Failed to get block: %v", err)
	}

	fmt.Printf("Block Hash: %s\n", block.Hash)
	fmt.Printf("Block Number: %s\n", block.Number)
	fmt.Printf("Transaction Count: %d\n", len(block.Transactions))
	fmt.Printf("Expected TransactionsRoot: %s\n", block.TransactionsRoot)
	fmt.Printf("Expected ReceiptsRoot: %s\n", block.ReceiptsRoot)
	fmt.Println()

	// 2. Convert RPC transactions to gorsk Transaction structs
	transactions := make([]*rskblocks.Transaction, len(block.Transactions))
	for i, rpcTx := range block.Transactions {
		tx, err := convertRPCTxToTransaction(rpcTx)
		if err != nil {
			log.Fatalf("Failed to convert transaction %d: %v", i, err)
		}
		transactions[i] = tx
		fmt.Printf("  Tx %d: %s\n", i, rpcTx.Hash)
	}

	// 3. Get receipts for each transaction
	receipts := make([]*rskblocks.TransactionReceipt, len(block.Transactions))
	for i, rpcTx := range block.Transactions {
		receipt, err := getTransactionReceipt(rpcTx.Hash)
		if err != nil {
			log.Fatalf("Failed to get receipt for tx %s: %v", rpcTx.Hash, err)
		}
		receipts[i], err = convertRPCReceiptToReceipt(receipt)
		if err != nil {
			log.Fatalf("Failed to convert receipt %d: %v", i, err)
		}
	}
	fmt.Println()

	// 4. Calculate transaction root
	txRoot := rskblocks.GetTxTrieRoot(transactions)
	txRootHex := "0x" + hex.EncodeToString(txRoot)

	// 5. Calculate receipt root
	receiptRoot := rskblocks.CalculateReceiptsTrieRoot(receipts)
	receiptRootHex := "0x" + hex.EncodeToString(receiptRoot)

	// 6. Build block header and compute hash
	header := convertRPCBlockToHeader(block)
	computedHash := header.Hash()
	computedHashHex := "0x" + hex.EncodeToString(computedHash[:])

	// 7. Compare results
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("RESULTS:")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\nBlock Hash:\n")
	fmt.Printf("  Expected: %s\n", block.Hash)
	fmt.Printf("  Computed: %s\n", computedHashHex)
	if strings.EqualFold(block.Hash, computedHashHex) {
		fmt.Printf("  ✓ MATCH!\n")
	} else {
		fmt.Printf("  ✗ MISMATCH!\n")
	}

	fmt.Printf("\nTransaction Root:\n")
	fmt.Printf("  Expected: %s\n", block.TransactionsRoot)
	fmt.Printf("  Computed: %s\n", txRootHex)
	if strings.EqualFold(block.TransactionsRoot, txRootHex) {
		fmt.Printf("  ✓ MATCH!\n")
	} else {
		fmt.Printf("  ✗ MISMATCH!\n")
	}

	fmt.Printf("\nReceipts Root:\n")
	fmt.Printf("  Expected: %s\n", block.ReceiptsRoot)
	fmt.Printf("  Computed: %s\n", receiptRootHex)
	if strings.EqualFold(block.ReceiptsRoot, receiptRootHex) {
		fmt.Printf("  ✓ MATCH!\n")
	} else {
		fmt.Printf("  ✗ MISMATCH!\n")
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

func getBlockByNumber(blockNum int64) (*rpcBlock, error) {
	blockNumHex := fmt.Sprintf("0x%x", blockNum)
	result, err := rpcCall("eth_getBlockByNumber", []interface{}{blockNumHex, true})
	if err != nil {
		return nil, err
	}

	var block rpcBlock
	if err := json.Unmarshal(result, &block); err != nil {
		return nil, fmt.Errorf("unmarshal block: %w", err)
	}

	return &block, nil
}

func getTransactionReceipt(txHash string) (*rpcReceipt, error) {
	result, err := rpcCall("eth_getTransactionReceipt", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	var receipt rpcReceipt
	if err := json.Unmarshal(result, &receipt); err != nil {
		return nil, fmt.Errorf("unmarshal receipt: %w", err)
	}

	return &receipt, nil
}

func convertRPCTxToTransaction(rpcTx rpcTx) (*rskblocks.Transaction, error) {
	return createSignedTransaction(rpcTx), nil
}

func convertRPCReceiptToReceipt(rpcReceipt *rpcReceipt) (*rskblocks.TransactionReceipt, error) {
	receipt := &rskblocks.TransactionReceipt{
		CumulativeGasUsed: hexToUint64(rpcReceipt.CumulativeGasUsed),
		GasUsed:           hexToUint64(rpcReceipt.GasUsed),
		TxHash:            common.HexToHash(rpcReceipt.TransactionHash),
	}

	// Handle PostState vs Status (EIP-658)
	// In RSK, for new-style receipts, postTxState contains the status byte
	if rpcReceipt.Root != "" && rpcReceipt.Root != "0x" {
		receipt.PostState = hexToBytes(rpcReceipt.Root)
	} else if rpcReceipt.Status != "" {
		// If no root but status is present, put status in PostState (EIP-658 style)
		receipt.PostState = hexToBytes(rpcReceipt.Status)
	}

	if rpcReceipt.Status != "" {
		receipt.Status = hexToBytes(rpcReceipt.Status)
	}

	// Parse bloom filter
	bloomBytes := hexToBytes(rpcReceipt.LogsBloom)
	if len(bloomBytes) == 256 {
		copy(receipt.Bloom[:], bloomBytes)
	}

	// Parse logs
	receipt.Logs = make([]*rskblocks.Log, len(rpcReceipt.Logs))
	for i, rpcLog := range rpcReceipt.Logs {
		log := &rskblocks.Log{
			Address: common.HexToAddress(rpcLog.Address),
			Data:    hexToBytes(rpcLog.Data),
			Topics:  make([]common.Hash, len(rpcLog.Topics)),
		}
		for j, topic := range rpcLog.Topics {
			log.Topics[j] = common.HexToHash(topic)
		}
		receipt.Logs[i] = log
	}

	// Contract address
	if rpcReceipt.ContractAddress != "" && rpcReceipt.ContractAddress != "0x" {
		receipt.ContractAddress = common.HexToAddress(rpcReceipt.ContractAddress)
	}

	return receipt, nil
}

// Helper functions
func hexToUint64(hexStr string) uint64 {
	if hexStr == "" || hexStr == "0x" {
		return 0
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	val, _ := strconv.ParseUint(hexStr, 16, 64)
	return val
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

func hexToBytes(hexStr string) []byte {
	if hexStr == "" || hexStr == "0x" {
		return nil
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	// Pad with leading zero if odd length (e.g., "1" -> "01")
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	data, _ := hex.DecodeString(hexStr)
	return data
}

// createSignedTransaction creates a transaction with signature values set
func createSignedTransaction(rpcTx rpcTx) *rskblocks.Transaction {
	nonce := hexToUint64(rpcTx.Nonce)
	gasPrice := hexToBigInt(rpcTx.GasPrice)
	gas := hexToUint64(rpcTx.Gas)
	value := hexToBigInt(rpcTx.Value)
	data := hexToBytes(rpcTx.Input)
	v := hexToBigInt(rpcTx.V)
	r := hexToBigInt(rpcTx.R)
	s := hexToBigInt(rpcTx.S)

	var to *common.Address
	if rpcTx.To != "" && rpcTx.To != "0x" {
		addr := common.HexToAddress(rpcTx.To)
		to = &addr
	}

	return rskblocks.NewSignedTransaction(nonce, to, value, gas, gasPrice, data, v, r, s)
}

// convertRPCBlockToHeader converts an RPC block to a BlockHeader struct
// using the block_header_hash_helper for proper encoding
func convertRPCBlockToHeader(block *rpcBlock) *rskblocks.BlockHeader {
	blockNum := hexToBigInt(block.Number).Int64()

	// Convert RskPteEdges from int to int16
	var edges []int16
	if len(block.RskPteEdges) > 0 {
		edges = make([]int16, len(block.RskPteEdges))
		for i, edge := range block.RskPteEdges {
			edges[i] = int16(edge)
		}
	}

	// Create input from RPC block data
	input := &rskblocks.BlockHeaderInput{
		ParentHash:                             common.HexToHash(block.ParentHash),
		UnclesHash:                             common.HexToHash(block.Sha3Uncles),
		Coinbase:                               common.HexToAddress(block.Miner),
		StateRoot:                              common.HexToHash(block.StateRoot),
		TxTrieRoot:                             common.HexToHash(block.TransactionsRoot),
		ReceiptTrieRoot:                        common.HexToHash(block.ReceiptsRoot),
		Difficulty:                             hexToBigInt(block.Difficulty),
		Number:                                 hexToBigInt(block.Number),
		GasLimit:                               hexToBigInt(block.GasLimit),
		GasUsed:                                hexToBigInt(block.GasUsed),
		Timestamp:                              hexToBigInt(block.Timestamp),
		ExtraData:                              hexToBytes(block.ExtraData),
		PaidFees:                               hexToBigInt(block.PaidFees),
		MinimumGasPrice:                        hexToBigInt(block.MinimumGasPrice),
		UncleCount:                             len(block.Uncles),
		BitcoinMergedMiningHeader:              hexToBytes(block.BitcoinMergedMiningHeader),
		BitcoinMergedMiningMerkleProof:         hexToBytes(block.BitcoinMergedMiningMerkleProof),
		BitcoinMergedMiningCoinbaseTransaction: hexToBytes(block.BitcoinMergedMiningCoinbaseTransaction),
		TxExecutionSublistsEdges:               edges,
	}

	// Parse logs bloom
	bloomBytes := hexToBytes(block.LogsBloom)
	if len(bloomBytes) == 256 {
		copy(input.LogsBloom[:], bloomBytes)
	}

	// Get config for regtest (all RSKIPs active)
	config := rskblocks.ConfigForBlockNumber(blockNum, "regtest")

	// Convert to BlockHeader using the helper
	return rskblocks.InputToBlockHeader(input, config)
}

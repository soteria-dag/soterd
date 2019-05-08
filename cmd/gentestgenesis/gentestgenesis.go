package main

import (
	"bytes"
	"fmt"
	"github.com/soteria-dag/soterd/blockdag"
	"math/big"
	"regexp"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"strings"
	"encoding/hex"
)

// expect the hash string to have 4 X 8 hex codes
func getHashHexArray(hash string) []string {
	lines := make([]string, 4)
	row := make([]string, 8)
	rowIdx := len(row) - 1
	lineIdx := len(lines) - 1

	hexStr := "0x"

	for i, c := range hash {
		hexStr += string(c)
		if !(i % 2 == 0) {
			row[rowIdx] = hexStr
			hexStr = "0x"
			rowIdx--
			if rowIdx < 0 {
				lines[lineIdx] = strings.Join(row, ", ")
				row = make([]string, 8)
				rowIdx =len(row) - 1
				lineIdx--
			}
		}
	}

	return lines
}

func getBlockHexArray(data []byte) []string {
	result := make([]string, 0)
	res := hex.Dump(data)
	rows := regexp.MustCompile("[\n]").Split(res, -1)

	for _, row := range rows {
		cells := regexp.MustCompile("[ ]+").Split(row, 18)

		var rowStr1 strings.Builder
		var rowStr2 strings.Builder
		for i, cell := range cells {
			if i < (len(cells) - 1) {
				if i > 0 && i <= 8 {
					rowStr1.WriteString("0x")
					rowStr1.WriteString(cell)
					rowStr1.WriteString(", ")
				} else if len(cells) > 10 && i > 8 && i <= 16 {
					rowStr2.WriteString("0x")
					rowStr2.WriteString(cell)
					rowStr2.WriteString(", ")
				}
			} else {
				if len(cell) - 2 > 8 {
					rowStr1.WriteString("/* |")
					rowStr1.WriteString(cell[1:9])
					rowStr1.WriteString("| */")

					rowStr2.WriteString("/* |")
					rowStr2.WriteString(cell[9:len(cell)-1])
					rowStr2.WriteString("| */")
				} else {
					rowStr1.WriteString("/* |")
					rowStr1.WriteString(strings.Trim(cell, "|"))
					rowStr1.WriteString("| */")
				}
			}
		}
		result = append(result, rowStr1.String())
		result = append(result, rowStr2.String())
	}

	return result
}
// Re-generate testnet genesis block and print out new hash.
// n: Using base testnet block, change POW limit to 2^n - 1

// Developer to update:
// - chaincfg/genesis.go with new genesis block POW limit and hash
// - chaincfg/params.go
func regenerateGenesisBlock(n uint) {

	var testnetBlock = chaincfg.TestNet1Params.GenesisBlock
	var header = testnetBlock.Header

	// new POW Limit
	var bigOne = big.NewInt(1)
	var newTestNetPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, n), bigOne)

	header.Bits = blockdag.BigToCompact(newTestNetPowLimit)
	// reset Nonce before mining for new hash
	header.Nonce = 1

	hash := header.BlockHash()
	cmp := blockdag.HashToBig(&hash).Cmp(newTestNetPowLimit)
	start := time.Now()
	for cmp >= 0 {
		header.Nonce++
		hash = header.BlockHash()
		cmp = blockdag.HashToBig(&hash).Cmp(newTestNetPowLimit)
		if header.Nonce % 1000000 == 0 {
			fmt.Printf("Nonce at: %d\n", header.Nonce)
		}
	}
	duration := time.Since(start)
	fmt.Printf("Ran for %v\n", duration)
	fmt.Printf("New hash: %s\n", hash)
	fmt.Printf("Bits: %d, hex 0x%x\n", header.Bits, header.Bits)
	fmt.Printf("Nonce: %d, hex 0x%x\n", header.Nonce, header.Nonce)

	hexArray := getHashHexArray(hash.String())
	fmt.Println("Hash in hex array (to be copied and pasted into genesis.go):")
	for _, row := range hexArray {
		fmt.Println(row)
	}

	var buf bytes.Buffer
	err := chaincfg.TestNet1Params.GenesisBlock.Serialize(&buf)
	if err != nil {
		fmt.Errorf("Error serializing TestNet1GenesisBlock: %v", err)
	}

	fmt.Println("Block data in hex array (to be copied and pasted into genesis_test.go):")
	blockArray := getBlockHexArray(buf.Bytes())
	for _, line := range blockArray {
		fmt.Println(line)
	}

}

func main() {
	regenerateGenesisBlock(231)
}

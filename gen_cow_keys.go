package main
import (
    "fmt"
    "github.com/ethereum/go-ethereum/crypto"
)
func main() {
    seeds := []string{"cow", "cow1", "cow2", "cow3", "cow4", "cow5", "cow6", "cow7", "cow8", "cow9"}
    for _, s := range seeds {
        hash := crypto.Keccak256([]byte(s))
        fmt.Printf("%s: %x\n", s, hash)
    }
}

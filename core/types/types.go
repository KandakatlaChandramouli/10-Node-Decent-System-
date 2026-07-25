package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Transaction struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type Block struct {
	Index        int           `json:"index"`
	Timestamp    time.Time     `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	PrevHash     string        `json:"prev_hash"`
	Hash         string        `json:"hash"`
	Nonce        int           `json:"nonce"`
}

func (b *Block) CalculateHash() string {
	record := fmt.Sprintf("%d%s%s%d", b.Index, b.Timestamp.String(), b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

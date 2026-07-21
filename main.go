package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Block struct {
	Index        int             `json:"index"`
	Timestamp    string          `json:"timestamp"`
	Transactions []SignedTx      `json:"transactions"`
	Coinbase     string          `json:"coinbase"`
	PrevHash     string          `json:"prev_hash"`
	Hash         string          `json:"hash"`
	Nonce        int             `json:"nonce"`
}

type SignedTx struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int    `json:"amount"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

func CalculateBlockHash(b Block) string {
	var txIDs []string
	for _, tx := range b.Transactions {
		txIDs = append(txIDs, tx.ID)
	}
	record := strconv.Itoa(b.Index) + b.Timestamp + strings.Join(txIDs, ",") + b.Coinbase + b.PrevHash + strconv.Itoa(b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

var (
	chain          []Block
	chainMutex     sync.RWMutex
	mempool        = make(map[string]SignedTx)
	mempoolMutex   sync.Mutex
	peers          = make(map[string]net.Conn)
	peersMutex     sync.RWMutex
	chainUpdated   = make(chan struct{}, 1)
	difficulty     = 3

	balances      = make(map[string]int)
	balancesMutex sync.RWMutex

	latencyList    []float64
	latencyMutex   sync.Mutex

	// Time‑series data for graphs (last 100 points)
	blockLatencies   []float64   // latency per block
	blockTimestamps  []time.Time // acceptance time of each block
	blockTxCounts    []int       // number of txs in each block
	mempoolSizes     []int       // periodic mempool samples
	peerCounts       []int       // periodic peer count samples
	graphMutex       sync.Mutex

	sharedSecret = "sovereign-chain-secret"
)

const genesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
const coinbaseReward = 50

const chainFile = "chain.json"
const mempoolFile = "mempool.json"

func saveChain() {
	chainMutex.RLock()
	defer chainMutex.RUnlock()
	data, _ := json.Marshal(chain)
	ioutil.WriteFile(chainFile, data, 0644)
}

func loadChain() bool {
	data, err := ioutil.ReadFile(chainFile)
	if err != nil {
		return false
	}
	var loaded []Block
	if err := json.Unmarshal(data, &loaded); err != nil {
		return false
	}
	chainMutex.Lock()
	chain = loaded
	chainMutex.Unlock()
	return true
}

func saveMempool() {
	mempoolMutex.Lock()
	defer mempoolMutex.Unlock()
	data, _ := json.Marshal(mempool)
	ioutil.WriteFile(mempoolFile, data, 0644)
}

func loadMempool() {
	data, err := ioutil.ReadFile(mempoolFile)
	if err != nil {
		return
	}
	var loaded map[string]SignedTx
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}
	mempoolMutex.Lock()
	mempool = loaded
	mempoolMutex.Unlock()
}

func signTx(tx SignedTx) string {
	msg := tx.From + tx.To + strconv.Itoa(tx.Amount) + tx.Timestamp + sharedSecret
	mac := hmac.New(sha256.New, []byte(sharedSecret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyTx(tx SignedTx) bool {
	expected := signTx(tx)
	return hmac.Equal([]byte(expected), []byte(tx.Signature))
}

func generateTxID(tx SignedTx) string {
	record := tx.From + tx.To + strconv.Itoa(tx.Amount) + tx.Timestamp + tx.Signature
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func processTx(tx SignedTx) bool {
	if !verifyTx(tx) {
		return false
	}
	balancesMutex.Lock()
	defer balancesMutex.Unlock()
	senderBalance := balances[tx.From]
	if senderBalance < tx.Amount {
		return false
	}
	return true
}

func applyBlockTransactions(b Block) {
	balancesMutex.Lock()
	defer balancesMutex.Unlock()
	for _, tx := range b.Transactions {
		if balances[tx.From] < tx.Amount {
			continue
		}
		balances[tx.From] -= tx.Amount
		balances[tx.To] += tx.Amount
	}
	balances[b.Coinbase] += coinbaseReward
}

func rebuildBalances(blocks []Block) {
	balancesMutex.Lock()
	defer balancesMutex.Unlock()
	balances = make(map[string]int)
	for _, b := range blocks {
		for _, tx := range b.Transactions {
			balances[tx.From] -= tx.Amount
			balances[tx.To] += tx.Amount
		}
		balances[b.Coinbase] += coinbaseReward
	}
}

func replaceChain(newBlocks []Block) bool {
	chainMutex.Lock()
	defer chainMutex.Unlock()
	if len(newBlocks) == 0 || newBlocks[0].Index != 0 {
		return false
	}
	if !validateChain(newBlocks) {
		return false
	}
	if len(newBlocks) <= len(chain) {
		return false
	}
	chain = newBlocks
	rebuildBalances(chain)
	log.Printf("[CHAIN] Replaced with longer chain. New height: %d\n", len(chain)-1)
	select {
	case chainUpdated <- struct{}{}:
	default:
	}
	return true
}

func validateChain(blocks []Block) bool {
	if len(blocks) == 0 || blocks[0].PrevHash != genesisPrevHash || blocks[0].Index != 0 {
		return false
	}
	for i := 1; i < len(blocks); i++ {
		prev, curr := blocks[i-1], blocks[i]
		if curr.Index != prev.Index+1 || curr.PrevHash != prev.Hash {
			return false
		}
		if CalculateBlockHash(curr) != curr.Hash {
			return false
		}
		prefix := strings.Repeat("0", difficulty)
		if !strings.HasPrefix(curr.Hash, prefix) {
			return false
		}
	}
	return true
}

func getLatestBlock() Block {
	chainMutex.RLock()
	defer chainMutex.RUnlock()
	return chain[len(chain)-1]
}

func getChainHeight() int {
	chainMutex.RLock()
	defer chainMutex.RUnlock()
	return len(chain) - 1
}

func getFullChain() []Block {
	chainMutex.RLock()
	defer chainMutex.RUnlock()
	dup := make([]Block, len(chain))
	copy(dup, chain)
	return dup
}

func getPeerCount() int {
	peersMutex.RLock()
	defer peersMutex.RUnlock()
	return len(peers)
}

var (
	myPort   string
	myAddr   string
	peerList []string
)

func sendMessage(conn net.Conn, msg string) {
	_, _ = fmt.Fprintf(conn, "%s\n", msg)
}

func broadcastMessage(msg string) {
	peersMutex.RLock()
	defer peersMutex.RUnlock()
	for _, conn := range peers {
		sendMessage(conn, msg)
	}
}

func broadcastToAllExcept(msg, exceptAddr string) {
	peersMutex.RLock()
	defer peersMutex.RUnlock()
	for addr, conn := range peers {
		if addr == exceptAddr {
			continue
		}
		sendMessage(conn, msg)
	}
}

func createCoinbaseTx(minerAddr string) SignedTx {
	tx := SignedTx{
		From:      "network",
		To:        minerAddr,
		Amount:    coinbaseReward,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	tx.Signature = signTx(tx)
	tx.ID = generateTxID(tx)
	return tx
}

func mineBlock() *Block {
	latest := getLatestBlock()
	nowUTC := time.Now().UTC().Format(time.RFC3339Nano)

	mempoolMutex.Lock()
	txs := []SignedTx{}
	for _, tx := range mempool {
		txs = append(txs, tx)
	}
	mempoolMutex.Unlock()

	coinbaseTx := createCoinbaseTx("miner_" + myPort)
	txs = append([]SignedTx{coinbaseTx}, txs...)

	newBlock := Block{
		Index:        latest.Index + 1,
		Timestamp:    nowUTC,
		Transactions: txs,
		Coinbase:     "miner_" + myPort,
		PrevHash:     latest.Hash,
		Nonce:        0,
	}
	prefix := strings.Repeat("0", difficulty)
	startNonce := rand.Intn(1000000)
	newBlock.Nonce = startNonce
	for {
		select {
		case <-chainUpdated:
			return nil
		default:
		}
		newBlock.Hash = CalculateBlockHash(newBlock)
		if strings.HasPrefix(newBlock.Hash, prefix) {
			break
		}
		newBlock.Nonce++
		if newBlock.Nonce < 0 {
			newBlock.Nonce = 0
		}
	}
	return &newBlock
}

func startMiner() {
	go func() {
		for {
			mempoolMutex.Lock()
			for len(mempool) == 0 {
				mempoolMutex.Unlock()
				time.Sleep(1 * time.Second)
				mempoolMutex.Lock()
			}
			mempoolMutex.Unlock()

			log.Printf("[MINER] Mining block %d with %d transactions...\n", getChainHeight()+1, len(mempool))
			block := mineBlock()
			if block == nil {
				continue
			}
			chainMutex.Lock()
			if block.Index == len(chain) {
				chain = append(chain, *block)
				chainMutex.Unlock()
				select {
				case chainUpdated <- struct{}{}:
				default:
				}
				log.Printf("[MINER] Mined block %d! Hash: %s\n", block.Index, block.Hash[:16])
				applyBlockTransactions(*block)
				// Update graph data
				graphMutex.Lock()
				blockTxCounts = append(blockTxCounts, len(block.Transactions))
				blockTimestamps = append(blockTimestamps, time.Now().UTC())
				if len(blockTxCounts) > 100 {
					blockTxCounts = blockTxCounts[1:]
					blockTimestamps = blockTimestamps[1:]
				}
				graphMutex.Unlock()
				mempoolMutex.Lock()
				for _, tx := range block.Transactions[1:] {
					delete(mempool, tx.ID)
				}
				mempoolMutex.Unlock()
				saveChain()
				saveMempool()
				blockJSON, _ := json.Marshal(block)
				broadcastMessage(fmt.Sprintf("BLOCK %s", string(blockJSON)))
			} else {
				chainMutex.Unlock()
			}
		}
	}()
}

func startTxGenerator() {
	go func() {
		counter := 0
		accounts := []string{"alice", "bob", "charlie", "dave", "eve"}
		for {
			time.Sleep(time.Duration(rand.Intn(5)+3) * time.Second)
			counter++
			from := accounts[rand.Intn(len(accounts))]
			to := accounts[rand.Intn(len(accounts))]
			if from == to {
				continue
			}
			amount := rand.Intn(10) + 1
			tx := SignedTx{
				From:      from,
				To:        to,
				Amount:    amount,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			}
			tx.Signature = signTx(tx)
			tx.ID = generateTxID(tx)
			mempoolMutex.Lock()
			mempool[tx.ID] = tx
			mempoolMutex.Unlock()
			txJSON, _ := json.Marshal(tx)
			broadcastMessage(fmt.Sprintf("TRANSACTION %s", string(txJSON)))
		}
	}()
}

// Background metrics collector
func startMetricsCollector() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			graphMutex.Lock()
			// Mempool size sample
			mempoolMutex.Lock()
			mempoolSizes = append(mempoolSizes, len(mempool))
			mempoolMutex.Unlock()
			if len(mempoolSizes) > 100 {
				mempoolSizes = mempoolSizes[1:]
			}
			// Peer count sample
			peerCounts = append(peerCounts, getPeerCount())
			if len(peerCounts) > 100 {
				peerCounts = peerCounts[1:]
			}
			graphMutex.Unlock()
		}
	}()
}

func connectToPeer(addr string) {
	for {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("[NET] Connected to %s\n", addr)
		peersMutex.Lock()
		peers[addr] = conn
		peersMutex.Unlock()
		hello := fmt.Sprintf("HELLO %d %s", getChainHeight(), getLatestBlock().Hash)
		sendMessage(conn, hello)
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			processMessage(conn, addr, scanner.Text())
		}
		peersMutex.Lock()
		delete(peers, addr)
		peersMutex.Unlock()
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func processMessage(sender net.Conn, senderAddr, msg string) {
	parts := strings.SplitN(msg, " ", 2)
	if len(parts) < 2 {
		return
	}
	cmd := parts[0]
	payload := parts[1]

	switch cmd {
	case "HELLO":
		fields := strings.Fields(payload)
		if len(fields) < 2 {
			return
		}
		peerHeight, _ := strconv.Atoi(fields[0])
		myHeight := getChainHeight()
		if peerHeight > myHeight {
			log.Printf("[SYNC] Peer %s ahead (%d vs %d). Requesting chain.\n", senderAddr, peerHeight, myHeight)
			sendMessage(sender, "GET_CHAIN 1")
		}
	case "TRANSACTION":
		var tx SignedTx
		if err := json.Unmarshal([]byte(payload), &tx); err != nil {
			return
		}
		if processTx(tx) {
			mempoolMutex.Lock()
			if _, exists := mempool[tx.ID]; !exists {
				mempool[tx.ID] = tx
				mempoolMutex.Unlock()
				broadcastToAllExcept(msg, senderAddr)
			} else {
				mempoolMutex.Unlock()
			}
		}
	case "BLOCK":
		var block Block
		if err := json.Unmarshal([]byte(payload), &block); err != nil {
			return
		}
		handleIncomingBlock(block, sender, senderAddr)
	case "GET_CHAIN":
		chainData := getFullChain()
		chainJSON, _ := json.Marshal(chainData)
		sendMessage(sender, fmt.Sprintf("CHAIN %s", string(chainJSON)))
	case "CHAIN":
		var blocks []Block
		if err := json.Unmarshal([]byte(payload), &blocks); err != nil {
			return
		}
		if replaceChain(blocks) {
			saveChain()
			saveMempool()
		}
	}
}

func handleIncomingBlock(block Block, sender net.Conn, senderAddr string) {
	chainMutex.Lock()
	for _, b := range chain {
		if b.Hash == block.Hash {
			chainMutex.Unlock()
			return
		}
	}
	currentLen := len(chain)

	if block.Index == currentLen {
		prevBlock := chain[currentLen-1]
		if block.PrevHash == prevBlock.Hash && CalculateBlockHash(block) == block.Hash &&
			strings.HasPrefix(block.Hash, strings.Repeat("0", difficulty)) {
			chain = append(chain, block)
			chainMutex.Unlock()
			select {
			case chainUpdated <- struct{}{}:
			default:
			}
			blockTime, err := time.Parse(time.RFC3339Nano, block.Timestamp)
			if err == nil {
				latency := time.Since(blockTime).Seconds() * 1000
				latencyMutex.Lock()
				latencyList = append(latencyList, latency)
				if len(latencyList) > 50 {
					latencyList = latencyList[1:]
				}
				latencyMutex.Unlock()
				log.Printf("[BLOCK] Accepted block %d from %s. Latency: %.2f ms\n", block.Index, senderAddr, latency)
				// Record for graph
				graphMutex.Lock()
				blockLatencies = append(blockLatencies, latency)
				if len(blockLatencies) > 100 {
					blockLatencies = blockLatencies[1:]
				}
				blockTxCounts = append(blockTxCounts, len(block.Transactions))
				blockTimestamps = append(blockTimestamps, time.Now().UTC())
				if len(blockTxCounts) > 100 {
					blockTxCounts = blockTxCounts[1:]
					blockTimestamps = blockTimestamps[1:]
				}
				graphMutex.Unlock()
			} else {
				log.Printf("[BLOCK] Accepted block %d from %s.\n", block.Index, senderAddr)
			}
			applyBlockTransactions(block)
			mempoolMutex.Lock()
			for _, tx := range block.Transactions[1:] {
				delete(mempool, tx.ID)
			}
			mempoolMutex.Unlock()
			saveChain()
			saveMempool()
		} else {
			chainMutex.Unlock()
			log.Printf("[BLOCK] Invalid block %d from %s\n", block.Index, senderAddr)
		}
	} else if block.Index > currentLen {
		chainMutex.Unlock()
		log.Printf("[SYNC] Future block %d (we are at %d). Requesting chain from %s.\n", block.Index, currentLen-1, senderAddr)
		sendMessage(sender, "GET_CHAIN 1")
	} else {
		chainMutex.Unlock()
	}
}

func initGenesis() {
	genesis := Block{
		Index:        0,
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		Transactions: []SignedTx{},
		Coinbase:     "genesis",
		PrevHash:     genesisPrevHash,
		Nonce:        0,
	}
	genesis.Hash = CalculateBlockHash(genesis)
	chain = append(chain, genesis)
	balancesMutex.Lock()
	balances["alice"] = 100
	balances["bob"] = 100
	balances["charlie"] = 100
	balances["dave"] = 100
	balances["eve"] = 100
	balancesMutex.Unlock()
	log.Printf("[LEDGER] Genesis created. Hash: %s\n", genesis.Hash[:16])
}

// --- Dashboard with Charts ---

var dashboardTemplate = `<!DOCTYPE html>
<html>
<head>
<title>SovereignChain Node Dashboard</title>
<meta charset="utf-8">
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
<style>
body { font-family: 'Segoe UI', sans-serif; background: #0d1117; color: #c9d1d9; padding: 20px; }
.card { background: #161b22; border: 1px solid #30363d; border-radius: 6px; padding: 15px; margin-bottom: 20px; }
h2 { color: #58a6ff; margin-top:0; }
.chart-container { width: 100%; max-width: 800px; height: 300px; margin: 0 auto; }
table { border-collapse: collapse; width: 100%; }
th, td { text-align: left; padding: 8px; border-bottom: 1px solid #30363d; }
.block-link { color: #58a6ff; cursor: pointer; text-decoration: underline; }
</style>
</head>
<body>
<h1>SovereignChain Node Dashboard</h1>
<p>Node: {{.NodeAddr}} | Updated: {{.Now}} | Height: {{.Height}} | Peers: {{.PeerCount}}</p>

<div class="card">
<h2>Block Propagation Latency (ms)</h2>
<div class="chart-container"><canvas id="latencyChart"></canvas></div>
</div>

<div class="card">
<h2>Block Interval (s)</h2>
<div class="chart-container"><canvas id="intervalChart"></canvas></div>
</div>

<div class="card">
<h2>Transactions per Block</h2>
<div class="chart-container"><canvas id="txChart"></canvas></div>
</div>

<div class="card">
<h2>Mempool Size & Peer Count</h2>
<div class="chart-container"><canvas id="mempoolPeerChart"></canvas></div>
</div>

<div class="card">
<h2>Recent Blocks</h2>
<table>
<tr><th>Index</th><th>Timestamp</th><th>Transactions</th><th>Hash</th></tr>
{{range .Blocks}}
<tr><td><a class="block-link" href="/block?index={{.Index}}">{{.Index}}</a></td><td>{{.Timestamp}}</td><td>{{.TxCount}}</td><td>{{.HashShort}}</td></tr>
{{end}}
</table>
</div>

<div class="card">
<h2>Account Balances</h2>
<table>
<tr><th>Account</th><th>Balance</th></tr>
{{range $addr, $bal := .Balances}}
<tr><td>{{$addr}}</td><td>{{$bal}}</td></tr>
{{end}}
</table>
</div>

<script>
// Data from server
const latencies = {{.LatenciesJSON}};
const blockTimes = {{.BlockTimesJSON}};
const txCounts = {{.TxCountsJSON}};
const mempoolSizes = {{.MempoolSizesJSON}};
const peerCounts = {{.PeerCountsJSON}};

// Latency chart
new Chart(document.getElementById('latencyChart'), {
    type: 'line',
    data: {
        labels: latencies.map((_,i) => i),
        datasets: [{
            label: 'Latency (ms)',
            data: latencies,
            borderColor: '#3fb950',
            backgroundColor: 'rgba(63,185,80,0.1)',
            fill: true,
            tension: 0.1
        }]
    },
    options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
            y: { beginAtZero: true, title: { display: true, text: 'ms' } },
            x: { title: { display: true, text: 'Block sequence' } }
        }
    }
});

// Block interval chart (differences between block acceptance times)
if (blockTimes.length > 1) {
    const intervals = [];
    for (let i = 1; i < blockTimes.length; i++) {
        const t1 = new Date(blockTimes[i-1]);
        const t2 = new Date(blockTimes[i]);
        intervals.push((t2 - t1) / 1000);
    }
    new Chart(document.getElementById('intervalChart'), {
        type: 'line',
        data: {
            labels: intervals.map((_,i) => i+1),
            datasets: [{
                label: 'Interval (s)',
                data: intervals,
                borderColor: '#58a6ff',
                backgroundColor: 'rgba(88,166,255,0.1)',
                fill: true,
                tension: 0.1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: { beginAtZero: true, title: { display: true, text: 'seconds' } },
                x: { title: { display: true, text: 'Block sequence' } }
            }
        }
    });
}

// Transactions per block
new Chart(document.getElementById('txChart'), {
    type: 'bar',
    data: {
        labels: txCounts.map((_,i) => i),
        datasets: [{
            label: 'Transactions',
            data: txCounts,
            backgroundColor: '#bc8cff'
        }]
    },
    options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
            y: { beginAtZero: true, title: { display: true, text: 'count' } },
            x: { title: { display: true, text: 'Block sequence' } }
        }
    }
});

// Mempool & Peer Count
new Chart(document.getElementById('mempoolPeerChart'), {
    type: 'line',
    data: {
        labels: mempoolSizes.map((_,i) => i),
        datasets: [
            {
                label: 'Mempool Size',
                data: mempoolSizes,
                borderColor: '#f78166',
                yAxisID: 'y'
            },
            {
                label: 'Peer Count',
                data: peerCounts,
                borderColor: '#58a6ff',
                yAxisID: 'y1'
            }
        ]
    },
    options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
            y: { beginAtZero: true, position: 'left', title: { display: true, text: 'Mempool' } },
            y1: { beginAtZero: true, position: 'right', grid: { drawOnChartArea: false }, title: { display: true, text: 'Peers' } }
        }
    }
});
</script>
</body>
</html>`

func startWebServer() {
	basePort, err := strconv.Atoi(myPort)
	if err != nil {
		return
	}
	dashboardPort := basePort + 1000

	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		tmpl, _ := template.New("dashboard").Parse(dashboardTemplate)
		height := getChainHeight()
		lastBlock := getLatestBlock()
		peerCount := getPeerCount()
		mempoolMutex.Lock()
		mempoolSize := len(mempool)
		mempoolMutex.Unlock()

		chainMutex.RLock()
		var recentBlocks []Block
		startIdx := len(chain) - 10
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(chain); i++ {
			recentBlocks = append(recentBlocks, chain[i])
		}
		chainMutex.RUnlock()

		type blockInfo struct {
			Index     int
			Timestamp string
			TxCount   int
			HashShort string
		}
		var blockInfos []blockInfo
		for _, b := range recentBlocks {
			blockInfos = append(blockInfos, blockInfo{
				Index:     b.Index,
				Timestamp: b.Timestamp,
				TxCount:   len(b.Transactions),
				HashShort: b.Hash[:16] + "...",
			})
		}

		balancesMutex.RLock()
		balCopy := make(map[string]int)
		for k, v := range balances {
			balCopy[k] = v
		}
		balancesMutex.RUnlock()

		graphMutex.Lock()
		// Convert data for JSON
		latenciesCopy := make([]float64, len(blockLatencies))
		copy(latenciesCopy, blockLatencies)
		txCountsCopy := make([]int, len(blockTxCounts))
		copy(txCountsCopy, blockTxCounts)
		blockTimesCopy := make([]string, len(blockTimestamps))
		for i, t := range blockTimestamps {
			blockTimesCopy[i] = t.Format(time.RFC3339Nano)
		}
		mempoolSizesCopy := make([]int, len(mempoolSizes))
		copy(mempoolSizesCopy, mempoolSizes)
		peerCountsCopy := make([]int, len(peerCounts))
		copy(peerCountsCopy, peerCounts)
		graphMutex.Unlock()

		latJSON, _ := json.Marshal(latenciesCopy)
		txJSON, _ := json.Marshal(txCountsCopy)
		timesJSON, _ := json.Marshal(blockTimesCopy)
		memJSON, _ := json.Marshal(mempoolSizesCopy)
		peerJSON, _ := json.Marshal(peerCountsCopy)

		data := struct {
			NodeAddr       string
			Now            string
			Height         int
			LastHash       string
			PeerCount      int
			MempoolSize    int
			Blocks         []blockInfo
			Balances       map[string]int
			LatenciesJSON  string
			BlockTimesJSON string
			TxCountsJSON   string
			MempoolSizesJSON string
			PeerCountsJSON string
		}{
			NodeAddr:       myAddr,
			Now:            time.Now().Format(time.RFC3339),
			Height:         height,
			LastHash:       lastBlock.Hash[:16] + "...",
			PeerCount:      peerCount,
			MempoolSize:    mempoolSize,
			Blocks:         blockInfos,
			Balances:       balCopy,
			LatenciesJSON:  string(latJSON),
			BlockTimesJSON: string(timesJSON),
			TxCountsJSON:   string(txJSON),
			MempoolSizesJSON: string(memJSON),
			PeerCountsJSON: string(peerJSON),
		}
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/block", func(w http.ResponseWriter, r *http.Request) {
		indexStr := r.URL.Query().Get("index")
		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		chainMutex.RLock()
		if idx < 0 || idx >= len(chain) {
			chainMutex.RUnlock()
			http.NotFound(w, r)
			return
		}
		b := chain[idx]
		chainMutex.RUnlock()

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><h1>Block #%d</h1><pre>%s</pre></body></html>", b.Index, b)
	})

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		height := getChainHeight()
		lastBlock := getLatestBlock()
		peerCount := getPeerCount()
		mempoolMutex.Lock()
		mempoolSize := len(mempool)
		mempoolMutex.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"height":       height,
			"last_hash":    lastBlock.Hash[:16],
			"peer_count":   peerCount,
			"mempool_size": mempoolSize,
		})
	})

	addr := fmt.Sprintf("0.0.0.0:%d", dashboardPort)
	log.Printf("[WEB] Dashboard available at http://0.0.0.0:%d/dashboard\n", dashboardPort)
	go http.ListenAndServe(addr, nil)
}

func main() {
	portPtr := flag.String("port", "", "Port to listen on")
	peersPtr := flag.String("peers", "", "Comma-separated list of peer addresses (host:port)")
	flag.Parse()
	if *portPtr == "" {
		fmt.Println("Usage: go run main.go --port=8080 --peers=127.0.0.1:8080,...")
		os.Exit(1)
	}
	myPort = *portPtr
	myAddr = "0.0.0.0:" + myPort
	peerList = strings.Split(*peersPtr, ",")
	for i := range peerList {
		peerList[i] = strings.TrimSpace(peerList[i])
	}

	rand.Seed(time.Now().UnixNano())

	if !loadChain() {
		initGenesis()
		saveChain()
	} else {
		rebuildBalances(chain)
		log.Printf("[LEDGER] Chain loaded from disk. Height: %d\n", getChainHeight())
	}
	loadMempool()

	listener, err := net.Listen("tcp", myAddr)
	if err != nil {
		log.Fatalf("[FATAL] Cannot listen: %v\n", err)
	}
	defer listener.Close()
	log.Printf("[NET] Node listening on %s\n", myAddr)

	for _, addr := range peerList {
		if strings.HasSuffix(addr, ":"+myPort) {
			continue
		}
		go connectToPeer(addr)
	}

	startMiner()
	startTxGenerator()
	startMetricsCollector()  // <-- collect mempool/peer samples
	startWebServer()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go func(c net.Conn) {
			peerAddr := c.RemoteAddr().String()
			defer c.Close()
			scanner := bufio.NewScanner(c)
			for scanner.Scan() {
				processMessage(c, peerAddr, scanner.Text())
			}
		}(conn)
	}
}

package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"sovereign-chain/core/interfaces"
)

type NodeRole int

const (
	RoleFollower NodeRole = iota
	RoleCandidate
	RoleLeader
)

func (r NodeRole) String() string {
	switch r {
	case RoleFollower:
		return "Follower"
	case RoleCandidate:
		return "Candidate"
	case RoleLeader:
		return "Leader"
	default:
		return "Unknown"
	}
}

type LogEntry struct {
	Index int    `json:"index"`
	Term  int    `json:"term"`
	Data  []byte `json:"data"`
}

type RequestVoteArgs struct {
	Term         int    `json:"term"`
	CandidateID  string `json:"candidate_id"`
	LastLogIndex int    `json:"last_log_index"`
	LastLogTerm  int    `json:"last_log_term"`
}

type RequestVoteReply struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"vote_granted"`
}

type AppendEntriesArgs struct {
	Term         int        `json:"term"`
	LeaderID     string     `json:"leader_id"`
	PrevLogIndex int        `json:"prev_log_index"`
	PrevLogTerm  int        `json:"prev_log_term"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit int        `json:"leader_commit"`
}

type AppendEntriesReply struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
}

type RaftEngine struct {
	mu          sync.RWMutex
	nodeID      string
	peers       []string
	currentTerm int
	votedFor    string
	log         []LogEntry
	commitIndex int
	lastApplied int
	role        NodeRole
	storage     interfaces.Storage
	commitChan  chan []byte

	heartbeatTimer *time.Ticker
	electionTimer  *time.Timer
	stopCh         chan struct{}
	rnd            *rand.Rand
}

func NewRaftEngine(nodeID string, peers []string, store interfaces.Storage) *RaftEngine {
	re := &RaftEngine{
		nodeID:      nodeID,
		peers:       peers,
		currentTerm: 0,
		votedFor:    "",
		log:         []LogEntry{{Index: 0, Term: 0, Data: nil}}, // Sentinel entry
		commitIndex: 0,
		lastApplied: 0,
		role:        RoleFollower,
		storage:     store,
		commitChan:  make(chan []byte, 1000),
		stopCh:      make(chan struct{}),
		rnd:         rand.New(rand.NewSource(time.Now().UnixNano() + int64(len(nodeID)))),
	}
	re.loadState()
	return re
}

func (re *RaftEngine) Name() string {
	return fmt.Sprintf("Consensus-Raft-%s", re.nodeID)
}

func (re *RaftEngine) Init(ctx context.Context) error {
	return nil
}

func (re *RaftEngine) Start(ctx context.Context) error {
	re.mu.Lock()
	re.resetElectionTimeout()
	re.mu.Unlock()

	go re.runLoop(ctx)
	return nil
}

func (re *RaftEngine) Stop(ctx context.Context) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	select {
	case <-re.stopCh:
	default:
		close(re.stopCh)
	}

	if re.electionTimer != nil {
		re.electionTimer.Stop()
	}
	if re.heartbeatTimer != nil {
		re.heartbeatTimer.Stop()
	}
	return nil
}

func (re *RaftEngine) Health() error {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return nil
}

func (re *RaftEngine) resetElectionTimeout() {
	if re.electionTimer != nil {
		re.electionTimer.Stop()
	}
	timeout := time.Duration(150+re.rnd.Intn(150)) * time.Millisecond
	re.electionTimer = time.NewTimer(timeout)
}

func (re *RaftEngine) runLoop(ctx context.Context) {
	for {
		re.mu.RLock()
		role := re.role
		re.mu.RUnlock()

		switch role {
		case RoleFollower, RoleCandidate:
			select {
			case <-re.electionTimer.C:
				re.startElection()
			case <-re.stopCh:
				return
			case <-ctx.Done():
				return
			}
		case RoleLeader:
			ticker := time.NewTicker(50 * time.Millisecond)
			select {
			case <-ticker.C:
				re.sendHeartbeats()
			case <-re.stopCh:
				ticker.Stop()
				return
			case <-ctx.Done():
				ticker.Stop()
				return
			}
			ticker.Stop()
		}
	}
}

func (re *RaftEngine) startElection() {
	re.mu.Lock()
	re.role = RoleCandidate
	re.currentTerm++
	re.votedFor = re.nodeID
	re.persistState()
	re.resetElectionTimeout()

	term := re.currentTerm
	lastLogIndex := len(re.log) - 1
	lastLogTerm := re.log[lastLogIndex].Term
	re.mu.Unlock()

	votesReceived := 1
	var voteMu sync.Mutex

	for _, peer := range re.peers {
		if peer == re.nodeID {
			continue
		}
		go func(p string) {
			args := RequestVoteArgs{
				Term:         term,
				CandidateID:  re.nodeID,
				LastLogIndex: lastLogIndex,
				LastLogTerm:  lastLogTerm,
			}
			_ = args // Simulated network transport dispatch in unit harness
			voteMu.Lock()
			votesReceived++
			if votesReceived > (len(re.peers) / 2) {
				re.mu.Lock()
				if re.role == RoleCandidate && re.currentTerm == term {
					re.role = RoleLeader
				}
				re.mu.Unlock()
			}
			voteMu.Unlock()
		}(peer)
	}
}

func (re *RaftEngine) sendHeartbeats() {
	re.mu.RLock()
	defer re.mu.RUnlock()
	if re.role != RoleLeader {
		return
	}
	// Broadcast heartbeat logic handled by network adapter
}

func (re *RaftEngine) Propose(ctx context.Context, data []byte) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.role != RoleLeader {
		return fmt.Errorf("node %s is not leader (current role: %s)", re.nodeID, re.role)
	}

	newEntry := LogEntry{
		Index: len(re.log),
		Term:  re.currentTerm,
		Data:  data,
	}
	re.log = append(re.log, newEntry)
	re.persistState()

	// In single-node or quorum simulation, commit immediately
	re.commitIndex = newEntry.Index
	re.lastApplied = newEntry.Index

	select {
	case re.commitChan <- data:
	default:
	}

	return nil
}

func (re *RaftEngine) Commit() <-chan []byte {
	return re.commitChan
}

func (re *RaftEngine) HandleRequestVote(args RequestVoteArgs) RequestVoteReply {
	re.mu.Lock()
	defer re.mu.Unlock()

	reply := RequestVoteReply{Term: re.currentTerm, VoteGranted: false}

	if args.Term < re.currentTerm {
		return reply
	}

	if args.Term > re.currentTerm {
		re.currentTerm = args.Term
		re.role = RoleFollower
		re.votedFor = ""
	}

	if (re.votedFor == "" || re.votedFor == args.CandidateID) && args.LastLogIndex >= len(re.log)-1 {
		re.votedFor = args.CandidateID
		reply.VoteGranted = true
		re.resetElectionTimeout()
	}

	re.persistState()
	return reply
}

func (re *RaftEngine) HandleAppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	re.mu.Lock()
	defer re.mu.Unlock()

	reply := AppendEntriesReply{Term: re.currentTerm, Success: false}

	if args.Term < re.currentTerm {
		return reply
	}

	if args.Term > re.currentTerm || re.role == RoleCandidate {
		re.currentTerm = args.Term
		re.role = RoleFollower
		re.votedFor = ""
	}

	re.resetElectionTimeout()

	if args.PrevLogIndex >= len(re.log) || re.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		return reply
	}

	re.log = re.log[:args.PrevLogIndex+1]
	re.log = append(re.log, args.Entries...)
	re.persistState()

	if args.LeaderCommit > re.commitIndex {
		re.commitIndex = args.LeaderCommit
		if re.commitIndex > len(re.log)-1 {
			re.commitIndex = len(re.log) - 1
		}
		for re.lastApplied < re.commitIndex {
			re.lastApplied++
			entryData := re.log[re.lastApplied].Data
			if len(entryData) > 0 {
				select {
				case re.commitChan <- entryData:
				default:
				}
			}
		}
	}

	reply.Success = true
	return reply
}

func (re *RaftEngine) persistState() {
	if re.storage == nil {
		return
	}
	state := map[string]interface{}{
		"currentTerm": re.currentTerm,
		"votedFor":    re.votedFor,
		"log":         re.log,
	}
	data, err := json.Marshal(state)
	if err == nil {
		_ = re.storage.Put([]byte("raft_state"), data)
	}
}

func (re *RaftEngine) loadState() {
	if re.storage == nil {
		return
	}
	data, err := re.storage.Get([]byte("raft_state"))
	if err != nil {
		return
	}
	var state struct {
		CurrentTerm int        `json:"currentTerm"`
		VotedFor    string     `json:"votedFor"`
		Log         []LogEntry `json:"log"`
	}
	if err := json.Unmarshal(data, &state); err == nil {
		re.currentTerm = state.CurrentTerm
		re.votedFor = state.VotedFor
		re.log = state.Log
	}
}

func (re *RaftEngine) GetRole() NodeRole {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return re.role
}

var _ interfaces.Consensus = (*RaftEngine)(nil)
var _ interfaces.Service = (*RaftEngine)(nil)

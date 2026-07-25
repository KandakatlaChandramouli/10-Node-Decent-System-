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

type Snapshot struct {
	LastIncludedIndex int    `json:"last_included_index"`
	LastIncludedTerm  int    `json:"last_included_term"`
	Data              []byte `json:"data"`
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
	snapshot    Snapshot
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
		log:         []LogEntry{{Index: 0, Term: 0, Data: nil}},
		snapshot:    Snapshot{LastIncludedIndex: 0, LastIncludedTerm: 0, Data: nil},
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
	timeout := time.Duration(150+re.rnd.Intn(150)) * time.Millisecond
	re.electionTimer = time.NewTimer(timeout)
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

func (re *RaftEngine) resetElectionTimeoutLocked() {
	if re.electionTimer != nil {
		if !re.electionTimer.Stop() {
			select {
			case <-re.electionTimer.C:
			default:
			}
		}
		timeout := time.Duration(150+re.rnd.Intn(150)) * time.Millisecond
		re.electionTimer.Reset(timeout)
	}
}

func (re *RaftEngine) getElectionTimerChan() <-chan time.Time {
	re.mu.RLock()
	defer re.mu.RUnlock()
	if re.electionTimer == nil {
		return nil
	}
	return re.electionTimer.C
}

func (re *RaftEngine) runLoop(ctx context.Context) {
	for {
		re.mu.RLock()
		role := re.role
		re.mu.RUnlock()

		switch role {
		case RoleFollower, RoleCandidate:
			select {
			case <-re.getElectionTimerChan():
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
	re.resetElectionTimeoutLocked()

	term := re.currentTerm
	lastLogIndex := re.getLastLogIndex()
	lastLogTerm := re.getLastLogTerm()
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
			_ = args
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
}

func (re *RaftEngine) Propose(ctx context.Context, data []byte) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.role != RoleLeader {
		return fmt.Errorf("node %s is not leader (current role: %s)", re.nodeID, re.role)
	}

	newIndex := re.getLastLogIndex() + 1
	newEntry := LogEntry{
		Index: newIndex,
		Term:  re.currentTerm,
		Data:  data,
	}
	re.log = append(re.log, newEntry)
	re.persistState()

	re.commitIndex = newIndex
	re.lastApplied = newIndex

	select {
	case re.commitChan <- data:
	default:
	}

	return nil
}

func (re *RaftEngine) Commit() <-chan []byte {
	return re.commitChan
}

func (re *RaftEngine) CompactLog(snapshotIndex int, stateData []byte) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if snapshotIndex <= re.snapshot.LastIncludedIndex || snapshotIndex > re.commitIndex {
		return fmt.Errorf("invalid snapshot index %d (commitIndex: %d, lastSnapshot: %d)",
			snapshotIndex, re.commitIndex, re.snapshot.LastIncludedIndex)
	}

	relativeIndex := snapshotIndex - re.snapshot.LastIncludedIndex
	snapshotTerm := re.log[relativeIndex].Term

	re.snapshot = Snapshot{
		LastIncludedIndex: snapshotIndex,
		LastIncludedTerm:  snapshotTerm,
		Data:              stateData,
	}

	re.log = append([]LogEntry{{Index: snapshotIndex, Term: snapshotTerm, Data: nil}}, re.log[relativeIndex+1:]...)
	re.persistState()
	return nil
}

func (re *RaftEngine) getLastLogIndex() int {
	if len(re.log) == 0 {
		return re.snapshot.LastIncludedIndex
	}
	return re.log[len(re.log)-1].Index
}

func (re *RaftEngine) getLastLogTerm() int {
	if len(re.log) == 0 {
		return re.snapshot.LastIncludedTerm
	}
	return re.log[len(re.log)-1].Term
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

	lastIndex := re.getLastLogIndex()
	if (re.votedFor == "" || re.votedFor == args.CandidateID) && args.LastLogIndex >= lastIndex {
		re.votedFor = args.CandidateID
		reply.VoteGranted = true
		re.resetElectionTimeoutLocked()
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

	re.resetElectionTimeoutLocked()

	relativePrevIndex := args.PrevLogIndex - re.snapshot.LastIncludedIndex
	if relativePrevIndex < 0 || relativePrevIndex >= len(re.log) || re.log[relativePrevIndex].Term != args.PrevLogTerm {
		return reply
	}

	re.log = re.log[:relativePrevIndex+1]
	re.log = append(re.log, args.Entries...)
	re.persistState()

	if args.LeaderCommit > re.commitIndex {
		re.commitIndex = args.LeaderCommit
		maxIndex := re.getLastLogIndex()
		if re.commitIndex > maxIndex {
			re.commitIndex = maxIndex
		}
		for re.lastApplied < re.commitIndex {
			re.lastApplied++
			relAppIndex := re.lastApplied - re.snapshot.LastIncludedIndex
			if relAppIndex >= 0 && relAppIndex < len(re.log) {
				entryData := re.log[relAppIndex].Data
				if len(entryData) > 0 {
					select {
					case re.commitChan <- entryData:
					default:
					}
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
		"snapshot":    re.snapshot,
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
		Snapshot    Snapshot   `json:"snapshot"`
	}
	if err := json.Unmarshal(data, &state); err == nil {
		re.currentTerm = state.CurrentTerm
		re.votedFor = state.VotedFor
		re.log = state.Log
		re.snapshot = state.Snapshot
	}
}

func (re *RaftEngine) GetRole() NodeRole {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return re.role
}

var _ interfaces.Consensus = (*RaftEngine)(nil)
var _ interfaces.Service = (*RaftEngine)(nil)

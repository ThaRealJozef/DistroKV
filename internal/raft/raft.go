package raft

import (
	"distrokv/proto"
	"log"
	"sync"
	"time"
)

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

type Raft struct {
	mu sync.Mutex

	// Persistent state on all servers
	currentTerm int32
	votedFor    string
	log         []*proto.LogEntry

	// Volatile state on all servers
	commitIndex int32
	lastApplied int32
	state       State
	leaderId    string

	// ID of this server
	id string

	// Channels for signaling
	stopCh   chan struct{}
	commitCh chan<- *proto.LogEntry // Channel to send committed entries to FSM
}

func NewRaft(id string, commitCh chan<- *proto.LogEntry) *Raft {
	return &Raft{
		id:          id,
		state:       Follower,
		currentTerm: 0,
		votedFor:    "",
		log:         make([]*proto.LogEntry, 0),
		stopCh:      make(chan struct{}),
		commitCh:    commitCh,
	}
}

func (r *Raft) Start() error {
	go r.runElectionTimer()
	return nil
}

func (r *Raft) Stop() error {
	close(r.stopCh)
	return nil
}

// Basic placeholder for election timer
func (r *Raft) runElectionTimer() {
	timeout := r.electionTimeout()
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.mu.Lock()
			if r.state != Leader {
				r.startElection()
			}
			r.mu.Unlock()
			ticker.Reset(r.electionTimeout())
		}
	}
}

func (r *Raft) electionTimeout() time.Duration {
	return time.Duration(150+time.Now().UnixNano()%150) * time.Millisecond
}

func (r *Raft) startElection() {
	r.state = Candidate
	r.currentTerm++
	r.votedFor = r.id

	// MVP: If we are the only node (no peers implementation yet), we win immediately.
	// In a real implementation, we would send RequestVote to r.peers.
	// Since we haven't implemented peer config, let's assume single-node mode for the walkthrough.
	log.Printf("[%s] Starting election for term %d...", r.id, r.currentTerm)
	r.becomeLeader()
}

func (r *Raft) becomeLeader() {
	r.state = Leader
	r.leaderId = r.id
	log.Printf("[%s] Became Leader at term %d", r.id, r.currentTerm)
	go r.startLeaderLoop()
}

func (r *Raft) startLeaderLoop() {
	ticker := time.NewTicker(50 * time.Millisecond) // Fast heartbeat
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.mu.Lock()
			if r.state != Leader {
				r.mu.Unlock()
				return
			}

			// MVP: Single-Node Commit Logic
			// Since we have no peers, we are the majority.
			// Commit everything in the log immediately.
			lastLogIndex := int32(len(r.log))

			if lastLogIndex > r.commitIndex {
				// Apply entries from commitIndex+1 to lastLogIndex
				for i := r.commitIndex; i < lastLogIndex; i++ {
					// r.log is 0-indexed slice, but Raft index is 1-based.
					// LogEntry at index 'i' in slice has Raft Index 'i+1' (usually).
					// Let's rely on the entry's internal index.
					entry := r.log[i]
					r.commitCh <- entry
					log.Printf("[%s] Leader Committed Index %d", r.id, entry.Index)
				}
				r.commitIndex = lastLogIndex
			}

			r.mu.Unlock()
		}
	}
}

// RequestVote handles the RPC from candidates
func (r *Raft) RequestVote(req *proto.RequestVoteRequest) (*proto.RequestVoteResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resp := &proto.RequestVoteResponse{
		Term:        r.currentTerm,
		VoteGranted: false,
	}

	if req.Term < r.currentTerm {
		return resp, nil
	}

	if req.Term > r.currentTerm {
		r.currentTerm = req.Term
		r.state = Follower
		r.votedFor = ""
	}

	if r.votedFor == "" || r.votedFor == req.CandidateId {
		// simpler log check for now (MVP)
		r.votedFor = req.CandidateId
		resp.VoteGranted = true
	}

	return resp, nil
}

// AppendEntries handles log replication from the leader
func (r *Raft) AppendEntries(req *proto.AppendEntriesRequest) (*proto.AppendEntriesResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resp := &proto.AppendEntriesResponse{
		Term:    r.currentTerm,
		Success: false,
	}

	if req.Term < r.currentTerm {
		return resp, nil
	}

	if req.Term > r.currentTerm {
		r.currentTerm = req.Term
		r.state = Follower
		r.votedFor = ""
	}

	// Heartbeat received from leader
	r.leaderId = req.LeaderId

	// TODO: Reset election timer channel (need to refactor ticker to support reset from here)
	// For MVP, since the ticker is in a loop, we can't easily reset it without a channel or mutex.
	// We'll leave the reset logic abstract for now.

	// Consistency Check:
	// If Log doesn't contain an entry at PrevLogIndex whose term matches PrevLogTerm -> return false
	if req.PrevLogIndex > 0 {
		lastIndex := int32(len(r.log))
		if req.PrevLogIndex > lastIndex {
			return resp, nil
		}
		// If index exists, check term (assuming log is 1-indexed in proto, 0-indexed slice)
		// r.log is []*LogEntry. entry.Index is 1-based.
		// Slice index = req.PrevLogIndex - 1
		if r.log[req.PrevLogIndex-1].Term != req.PrevLogTerm {
			return resp, nil
		}
	}

	// Append any new entries not already in the log
	for _, entry := range req.Entries {
		// MVP: just append. Real Raft needs to delete conflicting entries.
		if entry.Index > int32(len(r.log)) {
			r.log = append(r.log, entry)
		}
	}

	// Update commit index
	if req.LeaderCommit > r.commitIndex {
		lastNewIndex := int32(len(r.log))
		if req.LeaderCommit < lastNewIndex {
			r.commitIndex = req.LeaderCommit
		} else {
			r.commitIndex = lastNewIndex
		}
		// TODO: Signal FSM to apply
	}

	resp.Success = true
	return resp, nil
}

// Implement ConsensusModule interface
func (r *Raft) Submit(command []byte) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state != Leader {
		return false, nil
	}
	entry := &proto.LogEntry{
		Term:    r.currentTerm,
		Index:   int32(len(r.log) + 1),
		Command: command,
	}
	r.log = append(r.log, entry)
	return true, nil
}

func (r *Raft) IsLeader() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state == Leader
}

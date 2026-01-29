package raft

// ConsensusModule defines the interface for the Raft consensus algorithm.
type ConsensusModule interface {
	// Start begins the Raft consensus process (election ticker, etc.).
	Start() error

	// Stop halts the Raft node.
	Stop() error

	// Submit proposes a command to the replicated log.
	// Returns true if this node is the leader and the command was submitted.
	// The command is not guaranteed to be committed until applied.
	Submit(command []byte) (bool, error)

	// IsLeader returns true if this node believes it is the leader.
	IsLeader() bool
}

// StateMachine defines the interface that the Raft log applies commands to.
// The LSM Tree will implement this to receive committed entries.
type StateMachine interface {
	Apply(command []byte) interface{}
}

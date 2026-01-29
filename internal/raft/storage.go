package raft

import (
	"encoding/json"
	"os"
	"sync"
)

// HardState represents the persistent state of the Raft server
type HardState struct {
	CurrentTerm int32  `json:"current_term"`
	VotedFor    string `json:"voted_for"`
}

// Storage handles persistent storage of Raft HardState
type Storage struct {
	mu   sync.Mutex
	path string
}

func NewStorage(path string) *Storage {
	return &Storage{path: path}
}

func (s *Storage) Save(state HardState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

func (s *Storage) Load() (HardState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return HardState{CurrentTerm: 0, VotedFor: ""}, nil
		}
		return HardState{}, err
	}

	var state HardState
	if err := json.Unmarshal(data, &state); err != nil {
		return HardState{}, err
	}

	return state, nil
}

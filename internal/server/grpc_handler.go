package server

import (
	"bytes"
	"context"
	"distrokv/proto"
	"encoding/gob"
	"errors"
)

// --- KV Service Handlers ---

func (s *Server) Put(ctx context.Context, req *proto.PutRequest) (*proto.PutResponse, error) {
	// 1. Check if we are leader
	if !s.raftNode.IsLeader() {
		// MVP: Just return error. Better: Redirect to leader.
		return &proto.PutResponse{Success: false}, errors.New("not leader")
	}

	// 2. Submit to Raft
	cmd := Command{
		Op:    "PUT",
		Key:   req.Key,
		Value: req.Value,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(cmd); err != nil {
		return nil, err
	}

	success, err := s.raftNode.Submit(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &proto.PutResponse{Success: success}, nil
}

func (s *Server) Get(ctx context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {
	// For Linearizable Reads, we should go through Raft or check Leader lease.
	// For MVP, allow local read (Eventual Consistency / Stale Read).
	val, found := s.lsmStore.Get(req.Key)
	return &proto.GetResponse{
		Found: found,
		Value: val,
	}, nil
}

func (s *Server) Delete(ctx context.Context, req *proto.DeleteRequest) (*proto.DeleteResponse, error) {
	if !s.raftNode.IsLeader() {
		return &proto.DeleteResponse{Success: false}, errors.New("not leader")
	}

	// Submit "DELETE" command
	// For MVP simplification:
	_, err := s.raftNode.Submit([]byte("DELETE " + req.Key))
	if err != nil {
		return nil, err
	}

	return &proto.DeleteResponse{Success: true}, nil
}

// --- Raft Service Handlers ---

func (s *Server) RequestVote(ctx context.Context, req *proto.RequestVoteRequest) (*proto.RequestVoteResponse, error) {
	return s.raftNode.RequestVote(req)
}

func (s *Server) AppendEntries(ctx context.Context, req *proto.AppendEntriesRequest) (*proto.AppendEntriesResponse, error) {
	return s.raftNode.AppendEntries(req)
}

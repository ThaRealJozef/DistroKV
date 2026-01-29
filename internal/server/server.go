package server

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"

	"distrokv/internal/lsm"
	"distrokv/internal/raft"
	"distrokv/proto"

	"google.golang.org/grpc"
)

// Command represents a state machine operation
type Command struct {
	Op    string
	Key   string
	Value []byte
}

type Server struct {
	proto.UnimplementedRaftServiceServer
	proto.UnimplementedKVServiceServer

	id       string
	raftNode *raft.Raft
	lsmStore *lsm.LSMStore // Wrapper around MemTable+WAL+SSTable

	commitCh chan *proto.LogEntry // Channel to receive committed logs from Raft
	stopCh   chan struct{}
}

// Ensure Server implements gRPC definitions
var _ proto.RaftServiceServer = (*Server)(nil)
var _ proto.KVServiceServer = (*Server)(nil)

func NewServer(id string, storagePath string) (*Server, error) {
	// Create commit channel (buffered to avoid blocking Raft too much)
	commitCh := make(chan *proto.LogEntry, 100)

	// Initialize Raft Node
	// Note: We cast the channel to write-only for Raft
	r := raft.NewRaft(id, commitCh)

	// Initialize LSM Store
	store, err := lsm.NewLSMStore(storagePath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		id:       id,
		raftNode: r,
		lsmStore: store,
		commitCh: commitCh,
		stopCh:   make(chan struct{}),
	}

	return s, nil
}

func (s *Server) Start(port string) error {
	// Start Raft
	if err := s.raftNode.Start(); err != nil {
		return err
	}

	// Start FSM Applicator (Consumer)
	go s.applyCommits()

	// Start gRPC Listener
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	proto.RegisterRaftServiceServer(grpcServer, s)
	proto.RegisterKVServiceServer(grpcServer, s)

	log.Printf("Server %s listening on %s", s.id, port)
	return grpcServer.Serve(lis)
}

// applyCommits consumes the commitCh and applies changes to the LSM store
func (s *Server) applyCommits() {
	for {
		select {
		case <-s.stopCh:
			return
		case entry := <-s.commitCh:
			var cmd Command
			dec := gob.NewDecoder(bytes.NewReader(entry.Command))
			if err := dec.Decode(&cmd); err != nil {
				log.Printf("[%s] Failed to decode command: %v", s.id, err)
				continue
			}

			switch cmd.Op {
			case "PUT":
				s.lsmStore.Put(cmd.Key, cmd.Value)
				log.Printf("[%s] Applied: PUT %s=(%d bytes)", s.id, cmd.Key, len(cmd.Value))
			case "DELETE":
				// s.lsmStore.Delete(cmd.Key) // MVP: Add Delete to Store if not exists
				log.Printf("[%s] Applied: DELETE %s", s.id, cmd.Key)
			default:
				log.Printf("[%s] Unknown Op: %s", s.id, cmd.Op)
			}
		}
	}
}

func (s *Server) Stop() {
	close(s.stopCh)
	s.raftNode.Stop()
	s.lsmStore.Close()
}

package server

import (
	"log"
	"net"
	"strings"

	"distrokv/internal/lsm"
	"distrokv/internal/raft"
	"distrokv/proto"

	"google.golang.org/grpc"
)

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
			cmd := string(entry.Command)
			// Naive parsing: "PUT key val" or "DELETE key"
			// This is very fragile but works for MVP "Purist" demo without importing heavy libs
			// A real system would use Gob or Protobuf for the LogEntry Command field too.

			if len(cmd) > 4 && cmd[:3] == "PUT" {
				// Naive parsing: "PUT key val"
				parts := strings.SplitN(cmd, " ", 3)
				if len(parts) == 3 {
					s.lsmStore.Put(parts[1], []byte(parts[2]))
					log.Printf("[%s] Applied: PUT %s=%s", s.id, parts[1], parts[2])
				} else {
					log.Printf("[%s] Failed to parse: %s", s.id, cmd)
				}
			}
		}
	}
}

func (s *Server) Stop() {
	close(s.stopCh)
	s.raftNode.Stop()
	s.lsmStore.Close()
}

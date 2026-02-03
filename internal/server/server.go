package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"distrokv/internal/lsm"
	"distrokv/internal/raft"
	"distrokv/proto"

	"google.golang.org/grpc"
)

const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[32m"
	colorCyan    = "\033[36m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
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
	lsmStore *lsm.LSMStore

	commitCh chan *proto.LogEntry
	stopCh   chan struct{}
}

var _ proto.RaftServiceServer = (*Server)(nil)
var _ proto.KVServiceServer = (*Server)(nil)

func NewServer(id string, storagePath string) (*Server, error) {
	commitCh := make(chan *proto.LogEntry, 100)
	r := raft.NewRaft(id, commitCh)

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

	// Start Background Compaction
	s.lsmStore.StartCompactionTrigger(1 * time.Minute)

	// Start gRPC Listener
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	proto.RegisterRaftServiceServer(grpcServer, s)
	proto.RegisterKVServiceServer(grpcServer, s)

	fmt.Printf("\n%s%s🚀 DistroKV Server [%s] listening on %s%s\n\n",
		colorBold, colorCyan, s.id, port, colorReset)

	return grpcServer.Serve(lis)
}

func (s *Server) applyCommits() {
	for {
		select {
		case <-s.stopCh:
			return
		case entry := <-s.commitCh:
			var cmd Command
			dec := gob.NewDecoder(bytes.NewReader(entry.Command))
			if err := dec.Decode(&cmd); err != nil {
				fmt.Printf("%s✗ Decode error:%s %v\n", "\033[31m", colorReset, err)
				continue
			}

			switch cmd.Op {
			case "PUT":
				s.lsmStore.Put(cmd.Key, cmd.Value)
				fmt.Printf("%s%s│ APPLY%s %sPUT%s %s%s%s = %s%d bytes%s\n",
					colorDim, colorMagenta, colorReset,
					colorGreen, colorReset,
					colorCyan, cmd.Key, colorReset,
					colorYellow, len(cmd.Value), colorReset)
			case "DELETE":
				fmt.Printf("%s%s│ APPLY%s %sDELETE%s %s%s%s\n",
					colorDim, colorMagenta, colorReset,
					colorYellow, colorReset,
					colorCyan, cmd.Key, colorReset)
			default:
				fmt.Printf("%s│ APPLY%s Unknown Op: %s\n", colorMagenta, colorReset, cmd.Op)
			}
		}
	}
}

func (s *Server) Stop() error {
	close(s.stopCh)
	s.raftNode.Stop()
	return s.lsmStore.Close()
}

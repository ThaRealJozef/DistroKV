package main

import (
	"log"
	"os"

	"distrokv/internal/server"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <id> <port>", os.Args[0])
	}
	id := os.Args[1]
	port := ":" + os.Args[2]

	srv, err := server.NewServer(id, ".")
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Start(port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

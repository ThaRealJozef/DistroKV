package main

import (
	"context"
	"flag"
	"log"

	"distrokv/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	serverAddr := flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	op := flag.String("op", "get", "Operation: get|put|delete")
	key := flag.String("key", "foo", "Key")
	val := flag.String("val", "bar", "Value (for put)")
	flag.Parse()

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := proto.NewKVServiceClient(conn)

	ctx := context.Background()

	switch *op {
	case "put":
		_, err := c.Put(ctx, &proto.PutRequest{Key: *key, Value: []byte(*val)})
		if err != nil {
			log.Fatalf("Put failed: %v", err)
		}
		log.Printf("Put %s=%s Success", *key, *val)
	case "get":
		r, err := c.Get(ctx, &proto.GetRequest{Key: *key})
		if err != nil {
			log.Fatalf("Get failed: %v", err)
		}
		log.Printf("Get %s: Found=%v, Value=%s", *key, r.Found, string(r.Value))
	case "delete":
		_, err := c.Delete(ctx, &proto.DeleteRequest{Key: *key})
		if err != nil {
			log.Fatalf("Delete failed: %v", err)
		}
		log.Printf("Delete %s Success", *key)
	default:
		log.Fatalf("Unknown operation: %s", *op)
	}
}

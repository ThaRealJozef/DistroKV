package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"distrokv/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBold   = "\033[1m"
)

func main() {
	serverAddr := flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	op := flag.String("op", "get", "Operation: get|put|delete")
	key := flag.String("key", "foo", "Key")
	val := flag.String("val", "bar", "Value (for put)")
	flag.Parse()

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s✗ Connection failed:%s %v\n", colorRed, colorReset, err)
		os.Exit(1)
	}
	defer conn.Close()

	c := proto.NewKVServiceClient(conn)
	ctx := context.Background()

	switch *op {
	case "put":
		_, err := c.Put(ctx, &proto.PutRequest{Key: *key, Value: []byte(*val)})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s✗ PUT failed:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}
		fmt.Printf("%s%s✓ PUT%s %s%s%s = %s\"%s\"%s\n",
			colorBold, colorGreen, colorReset,
			colorCyan, *key, colorReset,
			colorYellow, *val, colorReset)

	case "get":
		r, err := c.Get(ctx, &proto.GetRequest{Key: *key})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s✗ GET failed:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}
		if r.Found {
			fmt.Printf("%s%s✓ GET%s %s%s%s → %s\"%s\"%s\n",
				colorBold, colorGreen, colorReset,
				colorCyan, *key, colorReset,
				colorYellow, string(r.Value), colorReset)
		} else {
			fmt.Printf("%s%s✗ GET%s %s%s%s → %snot found%s\n",
				colorBold, colorRed, colorReset,
				colorCyan, *key, colorReset,
				colorRed, colorReset)
		}

	case "delete":
		_, err := c.Delete(ctx, &proto.DeleteRequest{Key: *key})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s✗ DELETE failed:%s %v\n", colorRed, colorReset, err)
			os.Exit(1)
		}
		fmt.Printf("%s%s✓ DELETE%s %s%s%s\n",
			colorBold, colorGreen, colorReset,
			colorCyan, *key, colorReset)

	default:
		fmt.Fprintf(os.Stderr, "%s✗ Unknown operation:%s %s\n", colorRed, colorReset, *op)
		os.Exit(1)
	}
}

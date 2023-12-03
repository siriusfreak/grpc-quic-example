package main

import (
	"context"
	"crypto/tls"
	"github.com/siriusfreak/grpc-quic-example/pkg/gen/proto"
	"github.com/siriusfreak/grpc-quic-example/pkg/wrapper"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	addrQUIC = "172.10.1.1:4242"
	addrTCP  = "172.10.1.1:4243"
)

func quicDialer(ctx context.Context, address string, tlsConfig *tls.Config) (net.Conn, error) {
	con, err := quic.DialAddr(ctx, address, tlsConfig, nil)
	if err != nil {
		return nil, err
	}

	// Open a new stream for the gRPC connection.
	stream, err := con.OpenStream()
	if err != nil {
		return nil, err
	}

	return &wrapper.QuicConnectionWrapper{
		Conn:   con,
		Stream: stream,
	}, nil
}

// Generate random string
func generateRandomString(size int) string {
	rand.Seed(time.Now().UnixNano())
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, size)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func testTCP(payload string, count int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addrTCP,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	c := proto.NewFileServiceClient(conn)

	for i := 0; i < count; i++ {
		_, err := c.GetSimpleResponse(ctx, &proto.SimpleRequest{Query: payload})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
	}
}

func testQUIC(payload string, count int) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Set this to false and provide the proper CA in production
		NextProtos:         []string{"h3"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addrQUIC, // Use your server's address
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return quicDialer(ctx, addr, tlsConfig)
		}),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := proto.NewFileServiceClient(conn)

	for i := 0; i < count; i++ {
		_, err := c.GetSimpleResponse(ctx, &proto.SimpleRequest{Query: payload})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
	}
}
func main() {
	payload := generateRandomString(1024 * 512)
	iterCount := 3
	callCount := 30
	// run test function for 100 times and calc time
	start := time.Now()
	for i := 0; i < iterCount; i++ {
		testQUIC(payload, callCount)
	}
	elapsed := time.Since(start)
	log.Printf("QUIC times test took %v", float64(elapsed.Milliseconds())/float64(iterCount*callCount))

	start = time.Now()
	for i := 0; i < iterCount; i++ {
		testTCP(payload, callCount)
	}
	elapsed = time.Since(start)
	log.Printf("TCP times test took %v", float64(elapsed.Milliseconds())/float64(iterCount*callCount))
}

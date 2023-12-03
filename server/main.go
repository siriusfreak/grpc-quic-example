package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/quic-go/quic-go"
	"github.com/siriusfreak/grpc-quic-example/pkg/wrapper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/siriusfreak/grpc-quic-example/pkg/gen/proto"
)

const (
	addrQUIC = "0.0.0.0:4242"
	addrTCP  = "0.0.0.0:4243"
)

type server struct {
	proto.UnimplementedFileServiceServer
}

func (s *server) GetSimpleResponse(ctx context.Context, req *proto.SimpleRequest) (*proto.SimpleResponse, error) {
	return &proto.SimpleResponse{Message: "Hello " + req.Query}, nil
}

func (s *server) StreamFile(req *proto.FileRequest, stream proto.FileService_StreamFileServer) error {
	file, err := os.Open(req.FileName)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, 1024) // Adjust the buffer size as needed
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&proto.FileChunk{Content: buffer[:n]}); err != nil {
			return err
		}
	}
	return nil
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"My Company, Inc."},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"}, // HTTP/3 over QUIC
	}
}

func main() {
	go func() {
		lis, err := quic.ListenAddr(addrQUIC, generateTLSConfig(), nil)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		wrappedListener := &wrapper.QuicListenerWrapper{Listener: lis}

		grpcServer := grpc.NewServer(
			grpc.Creds(credentials.NewTLS(generateTLSConfig())),
		)
		proto.RegisterFileServiceServer(grpcServer, &server{})

		if err := grpcServer.Serve(wrappedListener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", addrTCP)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterFileServiceServer(grpcServer, &server{})
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

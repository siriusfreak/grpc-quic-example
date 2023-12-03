package wrapper

import (
	"context"
	"github.com/quic-go/quic-go"
	"net"
	"time"
)

type QuicConnectionWrapper struct {
	Conn   quic.Connection
	Stream quic.Stream
}

func (q *QuicConnectionWrapper) Read(b []byte) (n int, err error) {
	return q.Stream.Read(b)
}

func (q *QuicConnectionWrapper) Write(b []byte) (n int, err error) {
	return q.Stream.Write(b)
}

func (q *QuicConnectionWrapper) Close() error {
	return q.Stream.Close()
}

func (q *QuicConnectionWrapper) LocalAddr() net.Addr {
	return q.Conn.LocalAddr()
}

func (q *QuicConnectionWrapper) RemoteAddr() net.Addr {
	return q.Conn.RemoteAddr()
}

func (q *QuicConnectionWrapper) SetDeadline(t time.Time) error {
	return q.Stream.SetDeadline(t)
}

func (q *QuicConnectionWrapper) SetReadDeadline(t time.Time) error {
	return q.Stream.SetReadDeadline(t)
}

func (q *QuicConnectionWrapper) SetWriteDeadline(t time.Time) error {
	return q.Stream.SetWriteDeadline(t)
}

type QuicListenerWrapper struct {
	*quic.Listener
}

func (l *QuicListenerWrapper) Accept() (net.Conn, error) {
	con, err := l.Listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}
	stream, err := con.AcceptStream(context.Background())
	if err != nil {
		return nil, err
	}
	return &QuicConnectionWrapper{
		Conn:   con,
		Stream: stream,
	}, nil
}

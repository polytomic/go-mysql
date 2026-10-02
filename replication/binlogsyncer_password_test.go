package replication_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/go-mysql-org/go-mysql/server"
)

func TestNewConnectionUsesPasswordFunc(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	// The server accepts a different password on each connection, so a
	// password captured once cannot authenticate the second one.
	passwords := []string{"first", "second"}
	go func() {
		for _, password := range passwords {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			_, _ = server.NewConn(conn, "repl", password, server.EmptyHandler{})
		}
	}()

	calls := 0
	cfg := replication.BinlogSyncerConfig{
		ServerID: 1,
		Host:     "127.0.0.1",
		Port:     uint16(l.Addr().(*net.TCPAddr).Port),
		User:     "repl",
		Password: "stale",
		PasswordFunc: func(context.Context) (string, error) {
			calls++
			return passwords[calls-1], nil
		},
	}
	b := replication.NewBinlogSyncer(cfg)
	defer b.Close()

	for range passwords {
		c, err := b.NewConnection(context.Background())
		require.NoError(t, err)
		require.NoError(t, c.Close())
	}
	require.Equal(t, len(passwords), calls)

	cfg.PasswordFunc = func(context.Context) (string, error) {
		return "", errors.New("no credentials")
	}
	failing := replication.NewBinlogSyncer(cfg)
	defer failing.Close()
	_, err = failing.NewConnection(context.Background())
	require.ErrorContains(t, err, "no credentials")
}

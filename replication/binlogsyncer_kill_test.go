package replication_test

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/go-mysql-org/go-mysql/server"
)

type killHandler struct {
	server.EmptyHandler
	err error
}

func (h killHandler) HandleQuery(string) (*mysql.Result, error) {
	return nil, h.err
}

type levelRecorder struct {
	mu     sync.Mutex
	levels []slog.Level
}

func (r *levelRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *levelRecorder) Handle(_ context.Context, rec slog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec.Message != "kill last connection" {
		r.levels = append(r.levels, rec.Level)
	}
	return nil
}

func (r *levelRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *levelRecorder) WithGroup(string) slog.Handler      { return r }

func TestKillConnectionLogLevel(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want []slog.Level
	}{
		{
			name: "unknown thread id",
			err:  mysql.NewDefaultError(mysql.ER_NO_SUCH_THREAD, 42),
			want: []slog.Level{slog.LevelInfo},
		},
		{
			name: "other failure",
			err:  mysql.NewDefaultError(mysql.ER_KILL_DENIED_ERROR, 42),
			want: []slog.Level{slog.LevelError, slog.LevelError},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer l.Close()

			go func() {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				co, err := server.NewConn(conn, "repl", "secret", killHandler{err: tt.err})
				if err != nil {
					return
				}
				for co.HandleCommand() == nil {
				}
			}()

			rec := &levelRecorder{}
			b := replication.NewBinlogSyncer(replication.BinlogSyncerConfig{
				ServerID: 1,
				Host:     "127.0.0.1",
				Port:     uint16(l.Addr().(*net.TCPAddr).Port),
				User:     "repl",
				Password: "secret",
				Logger:   slog.New(rec),
			})
			defer b.Close()

			c, err := b.NewConnection(context.Background())
			require.NoError(t, err)
			defer c.Close()

			// Drop what the syncer logged while starting up.
			rec.mu.Lock()
			rec.levels = nil
			rec.mu.Unlock()

			b.KillConnection(c, 42)

			rec.mu.Lock()
			defer rec.mu.Unlock()
			require.Equal(t, tt.want, rec.levels)
		})
	}
}

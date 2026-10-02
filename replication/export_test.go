package replication

import (
	"context"

	"github.com/go-mysql-org/go-mysql/client"
)

// NewConnection exposes newConnection to tests outside the package, which can
// use the server package without an import cycle.
func (b *BinlogSyncer) NewConnection(ctx context.Context) (*client.Conn, error) {
	return b.newConnection(ctx)
}

// KillConnection exposes killConnection to tests outside the package.
func (b *BinlogSyncer) KillConnection(conn *client.Conn, id uint32) {
	b.killConnection(conn, id)
}

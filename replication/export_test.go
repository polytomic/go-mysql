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

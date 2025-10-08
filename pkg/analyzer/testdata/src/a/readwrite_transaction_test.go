package a

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Tests for ReadWriteTransaction (should NOT require defer - managed by client)

func goodReadWriteTransaction(client *spanner.Client) error {
	ctx := context.Background()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// ReadWriteTransaction is managed by the client, no need to close
		return nil
	})
	return err
}

package a

import (
	"context"

	"cloud.google.com/go/spanner"
)

func goodDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	iter := txn.Query(ctx, spanner.Statement{})
	defer iter.Stop()
}

func badNoDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"

	iter := txn.Query(ctx, spanner.Statement{}) // want "RowIterator\\.Close\\(\\) must be deferred"
	_ = iter
}

func badCloseNotDeferred(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"
	txn.Close()

	iter := txn.Query(ctx, spanner.Statement{}) // want "RowIterator\\.Close\\(\\) must be deferred"
	iter.Stop()
}

func goodBatchReadOnlyTransaction(client *spanner.Client) error {
	ctx := context.Background()
	txn, err := client.BatchReadOnlyTransaction(ctx, spanner.StrongRead())
	if err != nil {
		return err
	}
	defer txn.Close()
	return nil
}

func badBatchReadOnlyTransaction(client *spanner.Client) error {
	ctx := context.Background()
	txn, err := client.BatchReadOnlyTransaction(ctx, spanner.StrongRead()) // want "BatchReadOnlyTransaction\\.Close\\(\\) must be deferred"
	if err != nil {
		return err
	}
	_ = txn
	return nil
}

func goodReadWriteTransaction(client *spanner.Client) error {
	ctx := context.Background()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// ReadWriteTransaction is managed by the client, no need to close
		return nil
	})
	return err
}

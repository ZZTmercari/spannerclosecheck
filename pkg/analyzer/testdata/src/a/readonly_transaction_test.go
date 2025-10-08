package a

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Tests for ReadOnlyTransaction resource management

func goodReadOnlyTransactionDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	iter := txn.Query(ctx, spanner.Statement{})
	defer iter.Stop()
}

func badReadOnlyTransactionNoDefer(client *spanner.Client) {
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"
	_ = txn
}

func badReadOnlyTransactionCloseNotDeferred(client *spanner.Client) {
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"
	txn.Close() // Close is called but not deferred
}

// Single() should NOT require defer Close() - it auto-releases
func goodSingleInline(client *spanner.Client) {
	ctx := context.Background()
	iter := client.Single().Query(ctx, spanner.Statement{})
	defer iter.Stop()
}

func goodSingleInlineRead(client *spanner.Client) {
	ctx := context.Background()
	iter := client.Single().Read(ctx, "table", spanner.KeySets(), []string{"col"})
	defer iter.Stop()
}

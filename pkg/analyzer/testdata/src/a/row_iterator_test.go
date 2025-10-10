package a

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Tests for RowIterator resource management

func goodRowIteratorDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	iter := txn.Query(ctx, spanner.Statement{})
	defer iter.Stop()
}

func badRowIteratorNoDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"

	iter := txn.Query(ctx, spanner.Statement{}) // want "RowIterator\\.Stop\\(\\) must be deferred"
	_ = iter
}

func badRowIteratorStopNotDeferred(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"

	iter := txn.Query(ctx, spanner.Statement{}) // want "RowIterator\\.Stop\\(\\) must be deferred"
	iter.Stop() // Stop is called but not deferred
}

func goodRowIteratorReadDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction()
	defer txn.Close()

	iter := txn.Read(ctx, "table", nil, []string{"col1", "col2"})
	defer iter.Stop()
}

func badRowIteratorReadNoDefer(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"

	iter := txn.Read(ctx, "table", nil, []string{"col1", "col2"}) // want "RowIterator\\.Stop\\(\\) must be deferred"
	_ = iter
}

package a

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Tests for nolint directive support

func goodNolintSpannerclosecheck(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck
	_ = txn

	iter := txn.Query(ctx, spanner.Statement{}) //nolint:spannerclosecheck
	_ = iter
}

func goodNolintAll(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() //nolint:all
	_ = txn

	iter := txn.Query(ctx, spanner.Statement{}) //nolint:all
	_ = iter
}

func goodNolintGeneric(client *spanner.Client) {
	ctx := context.Background()
	txn := client.ReadOnlyTransaction() //nolint
	_ = txn

	iter := txn.Query(ctx, spanner.Statement{}) //nolint
	_ = iter
}

func goodNolintLineBefore(client *spanner.Client) {
	ctx := context.Background()
	//nolint:spannerclosecheck
	txn := client.ReadOnlyTransaction()
	_ = txn

	//nolint:all
	iter := txn.Query(ctx, spanner.Statement{})
	_ = iter
}

func goodNolintBatchTransaction(client *spanner.Client) error {
	ctx := context.Background()
	txn, err := client.BatchReadOnlyTransaction(ctx, spanner.StrongRead()) //nolint:spannerclosecheck
	if err != nil {
		return err
	}
	_ = txn
	return nil
}

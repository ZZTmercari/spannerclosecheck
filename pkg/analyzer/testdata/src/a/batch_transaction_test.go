package a

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Tests for BatchReadOnlyTransaction resource management

func goodBatchReadOnlyTransactionDefer(client *spanner.Client) error {
	ctx := context.Background()
	txn, err := client.BatchReadOnlyTransaction(ctx, spanner.StrongRead())
	if err != nil {
		return err
	}
	defer txn.Close()
	return nil
}

func badBatchReadOnlyTransactionNoDefer(client *spanner.Client) error {
	ctx := context.Background()
	txn, err := client.BatchReadOnlyTransaction(ctx, spanner.StrongRead()) // want "BatchReadOnlyTransaction\\.Close\\(\\) must be deferred"
	if err != nil {
		return err
	}
	_ = txn
	return nil
}

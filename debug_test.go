package main

import (
	"context"

	"cloud.google.com/go/spanner"
)

func testReassignment(client *spanner.Client) {
	ctx := context.Background()
	query := spanner.Statement{}

	rotx := client.ReadOnlyTransaction()
	iter := rotx.Query(ctx, query)
	defer iter.Stop()
	defer rotx.Close()

	// Reassignment
	rotx = client.ReadOnlyTransaction()
	iter = rotx.Query(ctx, query)
}

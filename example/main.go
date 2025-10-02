package main

import (
	"context"

	"cloud.google.com/go/spanner"
)

func badExample(client *spanner.Client) {
	txn := client.ReadOnlyTransaction()
	// Missing: defer txn.Close()
	_ = txn
}

func goodExample(client *spanner.Client) {
	txn := client.ReadOnlyTransaction()
	defer txn.Close()
	_ = txn
}

func main() {
	// This is just an example
}

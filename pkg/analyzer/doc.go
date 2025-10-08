// Package analyzer provides static analysis for Cloud Spanner resource management.
//
// The analyzer detects unclosed Spanner resources that should be properly closed
// to prevent memory leaks and connection issues. It checks that Close() or Stop()
// methods are called with defer to ensure resources are released even when errors occur.
//
// # Detected Resources
//
// The analyzer detects the following Spanner resource types:
//
//   - ReadOnlyTransaction: Must call Close() with defer
//   - BatchReadOnlyTransaction: Must call Close() with defer
//   - RowIterator: Must call Stop() with defer
//   - Client: Must call Close() with defer
//
// ReadWriteTransaction is explicitly excluded as it's managed by the client.
//
// # Examples
//
// Bad: Not closing transaction
//
//	func bad(client *spanner.Client) {
//	    txn := client.ReadOnlyTransaction() // Error: ReadOnlyTransaction.Close() must be deferred
//	    // ... use txn
//	}
//
// Good: Properly deferred close
//
//	func good(client *spanner.Client) {
//	    txn := client.ReadOnlyTransaction()
//	    defer txn.Close()
//	    // ... use txn
//	}
//
// # Nolint Support
//
// The analyzer supports nolint directives to suppress warnings:
//
//	txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck
//	txn := client.ReadOnlyTransaction() //nolint:all
//	txn := client.ReadOnlyTransaction() //nolint
//
// Nolint directives can also be placed on the line before:
//
//	//nolint:spannerclosecheck
//	txn := client.ReadOnlyTransaction()
package analyzer

package spanner

import "context"

// Mock types for testing
type Client struct{}

type ReadOnlyTransaction struct{}

func (t *ReadOnlyTransaction) Close() {}

func (t *ReadOnlyTransaction) Query(ctx context.Context, stmt Statement) *RowIterator {
	return &RowIterator{}
}

type BatchReadOnlyTransaction struct{}

func (t *BatchReadOnlyTransaction) Close() {}

type ReadWriteTransaction struct{}

type RowIterator struct{}

func (r *RowIterator) Stop() {}

type Statement struct {
	SQL    string
	Params map[string]interface{}
}

func (c *Client) ReadOnlyTransaction() *ReadOnlyTransaction {
	return &ReadOnlyTransaction{}
}

func (c *Client) BatchReadOnlyTransaction(ctx context.Context, tb TimestampBound) (*BatchReadOnlyTransaction, error) {
	return &BatchReadOnlyTransaction{}, nil
}

func (c *Client) ReadWriteTransaction(ctx context.Context, f func(context.Context, *ReadWriteTransaction) error) (interface{}, error) {
	return nil, f(ctx, &ReadWriteTransaction{})
}

type TimestampBound struct{}

func StrongRead() TimestampBound {
	return TimestampBound{}
}

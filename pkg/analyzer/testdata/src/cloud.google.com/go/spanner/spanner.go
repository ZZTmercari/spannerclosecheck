package spanner

import "context"

// Mock types for testing
type Client struct{}

func (c *Client) Close() {}

type ReadOnlyTransaction struct{}

func (t *ReadOnlyTransaction) Close() {}

func (t *ReadOnlyTransaction) Query(ctx context.Context, stmt Statement) *RowIterator {
	return &RowIterator{}
}

func (t *ReadOnlyTransaction) Read(ctx context.Context, table string, keys interface{}, columns []string) *RowIterator {
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

type KeySet interface{}

func KeySets(keys ...interface{}) KeySet {
	return nil
}

func NewClient(ctx context.Context, database string, opts ...interface{}) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Single() *ReadOnlyTransaction {
	return &ReadOnlyTransaction{}
}

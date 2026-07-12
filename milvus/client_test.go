package milvus

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := New(ctx, "82.158.224.81:19530", "", "", "reader", "novel", 1536)
	if err != nil {
		t.Fatal(err)
		return
	}
	client.Close(ctx)
}

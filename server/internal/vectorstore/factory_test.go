package vectorstore

import (
	"testing"
	"time"
)

func TestNewStoreDefaultsToQdrantWhenURLIsProvided(t *testing.T) {
	store, err := NewStore("", "http://127.0.0.1:6333", "secret", "practicehelper_memory", time.Second)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	qdrantStore, ok := store.(*QdrantStore)
	if !ok {
		t.Fatalf("expected *QdrantStore, got %T", store)
	}
	if !qdrantStore.Enabled() {
		t.Fatal("expected qdrant store to be enabled")
	}
}

func TestNewStoreReturnsNilForDisabledProvider(t *testing.T) {
	store, err := NewStore("none", "http://127.0.0.1:6333", "", "practicehelper_memory", time.Second)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if store != nil {
		t.Fatalf("expected nil store for disabled provider, got %T", store)
	}
}

func TestNewStoreRejectsUnsupportedProvider(t *testing.T) {
	store, err := NewStore("milvus", "http://127.0.0.1:19530", "", "practicehelper_memory", time.Second)
	if err == nil {
		t.Fatalf("expected error, got store %T", store)
	}
}

func TestNewStoreRequiresURLForQdrant(t *testing.T) {
	store, err := NewStore("qdrant", "", "", "practicehelper_memory", time.Second)
	if err == nil {
		t.Fatalf("expected error, got store %T", store)
	}
}

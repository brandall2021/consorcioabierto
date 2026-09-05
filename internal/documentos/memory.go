package documentos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var _ Storage = (*memoryStorage)(nil)

// memoryStorage es un adaptador de Storage en memoria para tests: sin red,
// determinista y con URLs sintéticas que permiten "subir" objetos por clave.
type memoryStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
}

// NewMemoryStorage incializa un storage en memoria para pruebas.
func NewMemoryStorage() *memoryStorage {
	return &memoryStorage{objects: make(map[string][]byte)}
}

func (m *memoryStorage) PresignPut(_ context.Context, key string, _ int64, _ time.Duration) (string, error) {
	return "memory://put/" + key, nil
}

func (m *memoryStorage) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objects[key]; !ok {
		return "", errors.New("object not found")
	}
	return "memory://get/" + key, nil
}

func (m *memoryStorage) Stat(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objects[key]
	if !ok {
		return 0, errors.New("object not found")
	}
	return int64(len(b)), nil
}

func (m *memoryStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// Put simula la subida del cliente (solo tests).
func (m *memoryStorage) Put(_ context.Context, key string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objects[key]; ok {
		return fmt.Errorf("clave duplicada: %s", key)
	}
	m.objects[key] = content
	return nil
}
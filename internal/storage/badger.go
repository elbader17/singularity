package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

const (
	activeBrainPrefix       = "brain:"    // Cerebro activo (estado del mundo)
	deepArchivePrefix       = "archive:"  // Archivo profundo
	taskPrefix              = "task:"     // Tareas
	sessionPrefix           = "session:"  // Sesiones
	subAgentPrefix          = "subagent:" // Sub-agentes
	subAgentTaskPrefix      = "subtask:"  // Tareas de sub-agentes
	dagMetadataPrefix       = "dag:"      // Metadatos del DAG
	compressedHistoryPrefix = "ch:"       // Historial comprimido
	codeSkeletonPrefix      = "skeleton:" // Esqueletos de archivos (para AST)
	functionCachePrefix     = "fncache:"  // Cache de funciones
	contextPrefix           = "context:"  // Contextos de archivos subidos a DB
)

type BadgerDB struct {
	db *badger.DB
}

func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./singularity-data"
	}
	return filepath.Join(home, ".singularity")
}

func NewBadgerDB(dir string) (*BadgerDB, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	opts := badger.DefaultOptions(dir)
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &BadgerDB{db: db}, nil
}

func (b *BadgerDB) Close() error {
	return b.db.Close()
}

func (b *BadgerDB) Get(key string) ([]byte, error) {
	var value []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			value = make([]byte, len(v))
			copy(value, v)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (b *BadgerDB) Set(key string, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (b *BadgerDB) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (b *BadgerDB) GetWithPrefix(prefix string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())
			err := item.Value(func(v []byte) error {
				result[key] = make([]byte, len(v))
				copy(result[key], v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (b *BadgerDB) SetMulti(items map[string][]byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		for key, value := range items {
			if err := txn.Set([]byte(key), value); err != nil {
				return err
			}
		}
		return nil
	})
}

func ActiveBrainKey(id string) string  { return activeBrainPrefix + id }
func DeepArchiveKey(id string) string  { return deepArchivePrefix + id }
func TaskKey(id string) string         { return taskPrefix + id }
func SessionKey(id string) string      { return sessionPrefix + id }
func SubAgentKey(id string) string     { return subAgentPrefix + id }
func SubAgentTaskKey(id string) string { return subAgentTaskPrefix + id }
func ActiveBrainPrefix() string        { return activeBrainPrefix }
func DeepArchivePrefix() string        { return deepArchivePrefix }
func TaskPrefix() string               { return taskPrefix }
func SessionPrefix() string            { return sessionPrefix }
func SubAgentPrefix() string           { return subAgentPrefix }
func SubAgentTaskPrefix() string       { return subAgentTaskPrefix }
func DAGMetadataPrefix() string        { return dagMetadataPrefix }
func CompressedHistoryPrefix() string  { return compressedHistoryPrefix }
func CodeSkeletonPrefix() string       { return codeSkeletonPrefix }
func FunctionCachePrefix() string      { return functionCachePrefix }

func DAGMetadataKey(sessionID string) string       { return dagMetadataPrefix + sessionID }
func CompressedHistoryKey(sessionID string) string { return compressedHistoryPrefix + sessionID }
func CodeSkeletonKey(filePath string) string       { return codeSkeletonPrefix + filePath }
func FunctionCacheKey(filePath, fnName string) string {
	return functionCachePrefix + filePath + ":" + fnName
}

// Context keys - Para el patrón de Punteros de Contexto
func ContextPrefix() string { return contextPrefix }
func ContextFileKey(sessionID, filePath string) string {
	return contextPrefix + "file:" + sessionID + ":" + filePath
}
func ContextMetaKey(sessionID, filePath string) string {
	return contextPrefix + "meta:" + sessionID + ":" + filePath
}
func ContextMetaPrefix() string {
	return contextPrefix + "meta:"
}

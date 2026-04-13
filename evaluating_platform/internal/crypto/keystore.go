package crypto

import (
	"fmt"
	"sync"

	"evaluating_platform/pkg/config"
)

// KeyStore 密钥管理，支持密钥轮换
type KeyStore struct {
	mu           sync.RWMutex
	keys         map[string][]byte // key_id → AES-256 key
	currentKeyID string
}

// NewKeyStore 从配置创建密钥管理器
func NewKeyStore(cfg config.CryptoConfig) (*KeyStore, error) {
	ks := &KeyStore{
		keys:         make(map[string][]byte),
		currentKeyID: cfg.KeyID,
	}
	if cfg.MasterKey == "" {
		return nil, fmt.Errorf("crypto.master_key is required")
	}
	key, err := ParseHexKey(cfg.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("parse master key: %w", err)
	}
	ks.keys[cfg.KeyID] = key
	return ks, nil
}

// CurrentKey 返回当前密钥 ID 和密钥
func (ks *KeyStore) CurrentKey() (string, []byte) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.currentKeyID, ks.keys[ks.currentKeyID]
}

// GetKey 根据 key_id 获取密钥（用于解密旧文件）
func (ks *KeyStore) GetKey(keyID string) ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	key, ok := ks.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return key, nil
}

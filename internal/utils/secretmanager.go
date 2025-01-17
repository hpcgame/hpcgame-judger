package utils

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const secretManagerCAPath = "ca.crt"
const secretManagerTokenPath = "token"
const secretManagerNamespacePath = "namespace"

const updateDuration = 15 * time.Minute

type SecretManager struct {
	caPath        string
	tokenPath     string
	namespacePath string

	ca        *x509.CertPool
	namespace string
	token     string

	timer *time.Ticker
	lock  *sync.RWMutex
}

func NewSecretManager(path string) *SecretManager {
	sm := &SecretManager{
		caPath:        filepath.Join(path, secretManagerCAPath),
		tokenPath:     filepath.Join(path, secretManagerTokenPath),
		namespacePath: filepath.Join(path, secretManagerNamespacePath),

		lock:  &sync.RWMutex{},
		timer: time.NewTicker(updateDuration),
	}

	sm.startUpdateLoop()

	return sm
}

func (sm *SecretManager) startUpdateLoop() {
	sm.update()
	go sm.updateLoop()
}

func (sm *SecretManager) updateLoop() {
	for range sm.timer.C {
		sm.lock.Lock()
		sm.update()
		sm.lock.Unlock()
	}
}

func (sm *SecretManager) update() {
	caBytes, err := os.ReadFile(sm.caPath)
	if err != nil {
		panic(err) // TODO: Better processing than panic
		// however, panic is good enough, causes program to automatically exit
		// and leverages the error handling to kubernetes itself
	}

	tokenBytes, err := os.ReadFile(sm.tokenPath)
	if err != nil {
		panic(err)
	}

	namespaceBytes, err := os.ReadFile(sm.namespacePath)
	if err != nil {
		panic(err)
	}

	sm.ca = x509.NewCertPool()
	sm.ca.AppendCertsFromPEM(caBytes)

	sm.token = string(tokenBytes)
	sm.namespace = string(namespaceBytes)
}

func (sm *SecretManager) GetRootCAs() *x509.CertPool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	return sm.ca
}

func (sm *SecretManager) GetToken() string {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	return sm.token
}

func (sm *SecretManager) GetNamespace() string {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	return sm.namespace
}

func (sm *SecretManager) GetTokenFile() string {
	return sm.tokenPath
}

func (sm *SecretManager) GetCAFile() string {
	return sm.caPath
}

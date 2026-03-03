package connection

import (
	"sync"
)

type UIDMapper struct {
	mu         sync.RWMutex
	tempToReal map[string]string
	realToTemp map[string]string
}

var GlobalUIDMapper = &UIDMapper{
	tempToReal: make(map[string]string),
	realToTemp: make(map[string]string),
}

func (m *UIDMapper) AddMapping(tempUID, realUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tempToReal[tempUID] = realUID
	m.realToTemp[realUID] = tempUID
}

func (m *UIDMapper) GetRealUID(tempUID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	realUID, exists := m.tempToReal[tempUID]
	return realUID, exists
}

func (m *UIDMapper) GetTempUID(realUID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tempUID, exists := m.realToTemp[realUID]
	return tempUID, exists
}

func (m *UIDMapper) RemoveMapping(uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if realUID, exists := m.tempToReal[uid]; exists {
		delete(m.tempToReal, uid)
		delete(m.realToTemp, realUID)
	}
	if tempUID, exists := m.realToTemp[uid]; exists {
		delete(m.realToTemp, uid)
		delete(m.tempToReal, tempUID)
	}
}

func (m *UIDMapper) HasMapping(uid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists1 := m.tempToReal[uid]
	_, exists2 := m.realToTemp[uid]
	return exists1 || exists2
}

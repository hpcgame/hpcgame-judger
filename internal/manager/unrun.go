package manager

import (
	"fmt"
	"log"
	"time"
)

func (m *Manager) isLocked(id string) (bool, error) {
	lockKey := fmt.Sprintf("%s:%s", judgeSessionLockKeyPrefix, id)
	return m.r.IsLocked(lockKey)
}

// Find admitted but not running jobs (no lock)
// and run them
func (m *Manager) findNotRunning() error {
	s, err := m.r.ListSolutionPoll()
	if err != nil {
		return err
	}

	for _, item := range s {
		locked, err := m.isLocked(item)
		if err != nil {
			log.Println("Failed to check lock:", err)
			continue
		}

		if locked {
			continue
		}

		go m.run(item)
	}
	return nil
}

const findNotRunningInterval = 8 * time.Minute

func (m *Manager) findNotRunningLoop() {
	for {
		err := m.findNotRunning()
		if err != nil {
			log.Println("Failed to find not running:", err)
		}
		time.Sleep(findNotRunningInterval)
	}
}

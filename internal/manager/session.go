package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/lcpu-club/hpcgame-judger/pkg/aoiclient"
)

const judgeSessionLockKeyPrefix = "judge:lock:"
const judgeSessionLockTimeout = 6 * 60 * time.Second
const judgeSessionUpdateInterval = 2 * 60 * time.Second

type JudgeSession struct {
	id string
	m  *Manager

	lockKey string

	closeChan chan struct{}

	soln *aoiclient.SolutionPoll
	aoi  *aoiclient.SolutionClient

	stopped *atomic.Int32

	rc *RunningConfig
}

func NewJudgeSession(id string, m *Manager) (*JudgeSession, error) {
	s := &JudgeSession{
		id:      id,
		m:       m,
		stopped: new(atomic.Int32),
	}

	err := s.init()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *JudgeSession) init() error {
	s.lockKey = fmt.Sprintf("%s:%s", judgeSessionLockKeyPrefix, s.id)
	s.closeChan = make(chan struct{})

	var err error
	s.soln, err = s.m.r.GetSolutionPoll(s.id)
	if err != nil {
		return err
	}

	s.rc = new(RunningConfig)
	err = json.Unmarshal(s.soln.ProblemConfig.Judge.Config, s.rc)

	s.aoi = s.m.aoi.Solution(s.soln.SolutionId, s.soln.TaskId)

	return err
}

func (s *JudgeSession) tryLock() (bool, error) {
	return s.m.r.AcquireLock(s.lockKey, s.m.ID(), judgeSessionLockTimeout)
}

func (s *JudgeSession) unlock() error {
	return s.m.r.ReleaseLock(s.lockKey, s.m.ID())
}

func (s *JudgeSession) lockLoop() {
	ticker := time.NewTicker(judgeSessionUpdateInterval)

	for {
		select {
		case <-ticker.C:
			err := s.m.r.RefreshLock(s.lockKey, judgeSessionLockTimeout)
			if err != nil {
				log.Println("Failed to refresh lock", err)
				return
			}
		case <-s.closeChan:
			return
		}
	}
}

func (s *JudgeSession) cleanup() error {
	if !s.stopped.CompareAndSwap(0, 1) {
		return nil
	}

	s.closeChan <- struct{}{}

	err := s.m.r.DeleteSolutionPoll(s.id)
	if err != nil {
		return err
	}

	err = s.unlock()
	if err != nil {
		return err
	}

	return nil
}

func (s *JudgeSession) Close() {
	err := s.cleanup()
	if err != nil {
		log.Println("Failed to cleanup judge session", err)
	}
}

func (s *JudgeSession) Run() error {
	if ok, err := s.tryLock(); !ok || err != nil {
		return err
	}

	go s.lockLoop()
	defer s.cleanup()

	// Do the real judge code here
	return s.run()
}

package manager

import (
	"context"
	"log"
	"time"

	"github.com/lcpu-club/hpcgame-judger/pkg/aoiclient"
)

const pollInterval = 250 * time.Millisecond

func (m *Manager) pollLoop() error {
	for {
		time.Sleep(pollInterval)

		ok, err := m.rl.Request()
		if err != nil {
			log.Println("Failed to request rate limit:", err)
			continue
			// return err
		}

		if !ok {
			continue
		}

		polled, err := m.poll()
		if err != nil {
			log.Println("Failed to poll:", err)
		}
		if err != nil || !polled {
			m.rl.Release()
			continue
		}
	}
}

func (m *Manager) poll() (bool, error) {
	soln, err := m.aoi.Poll(context.TODO())
	if err != nil {
		return false, err
	}

	if soln.SolutionId == "" || soln.TaskId == "" {
		// No solution to poll
		return false, nil
	}

	log.Println("Received solution", soln.SolutionId, "for task", soln.TaskId)

	err = m.solnAdmission(soln)
	if err != nil {
		log.Println("Failed to admit solution:", err)

		errF := m.failSoln(soln, "Failed to admit solution")
		if errF != nil {
			log.Println("Failed to fail solution:", err)
		}

		return true, err
	}

	return true, nil
}

func (m *Manager) solnAdmission(soln *aoiclient.SolutionPoll) error {
	id, err := m.r.StoreSolutionPoll(soln)
	if err != nil {
		return err
	}
	go m.run(id)
	return nil
}

func (m *Manager) failSoln(soln *aoiclient.SolutionPoll, reason string) error {
	s := m.aoi.Solution(soln.SolutionId, soln.TaskId)
	s.Patch(context.TODO(), &aoiclient.SolutionInfo{
		Score:   0,
		Status:  aoiclient.StatusError,
		Message: reason,
	})
	err := s.SaveDetails(context.TODO(), &aoiclient.SolutionDetails{
		Summary: reason,
	})
	if err != nil {
		return err
	}
	return s.Complete(context.TODO())
}

func (m *Manager) run(id string) error {
	log.Println("Running solution", id)
	defer m.rl.Release()

	sess, err := NewJudgeSession(id, m)
	if err != nil {
		return err
	}

	err = sess.Run()

	if err != nil {
		log.Println("Failed to run session:", err)
		fErr := m.failSoln(sess.soln, "Failed to run session: "+err.Error())
		if fErr != nil {
			log.Println("Failed to fail solution:", fErr)
		}
	}

	return nil
}

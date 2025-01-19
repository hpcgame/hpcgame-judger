package manager

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/lcpu-club/hpcgame-judger/pkg/aoiclient"
	"github.com/lcpu-club/hpcgame-judger/pkg/judgerproto"
)

func (s *JudgeSession) processMessage(msg string) error {
	m, err := judgerproto.MessageFromString(msg)
	if err != nil {
		return err
	}

	switch m.Action {
	case judgerproto.ActionError:
		{
			var body judgerproto.ErrorBody
			err := json.Unmarshal(m.Body, &body)
			if err != nil {
				return err
			}
			return errors.New(string(body))
		}
	case judgerproto.ActionLog:
		{
			var body judgerproto.LogBody
			err := json.Unmarshal(m.Body, &body)
			if err != nil {
				return err
			}
			// TODO: Log logics
			log.Println("Log from", s.GetNamespaceName(), ":", string(body))
		}
	case judgerproto.ActionComplete:
		{
			err := s.aoi.Complete(context.TODO())
			if err != nil {
				return wrapError("aoiComplete", err)
			}
		}
	case judgerproto.ActionQuit:
		{
			s.deleteNamespace()
		}
	case judgerproto.ActionPatch:
		{
			var body judgerproto.PatchBody
			err := json.Unmarshal(m.Body, &body)
			if err != nil {
				return err
			}

			err = s.aoi.Patch(context.TODO(), (*aoiclient.SolutionInfo)(&body))
			if err != nil {
				return wrapError("aoiPatch", err)
			}
		}
	case judgerproto.ActionDetail:
		{
			var body judgerproto.DetailBody
			err := json.Unmarshal(m.Body, &body)
			if err != nil {
				return wrapError("unmarshalDetail", err)
			}

			err = s.aoi.SaveDetails(context.TODO(), (*aoiclient.SolutionDetails)(&body))
			if err != nil {
				return wrapError("aoiSaveDetails", err)
			}
		}
	case judgerproto.ActionNoop:
		{
			return nil
		}
	case judgerproto.ActionGreet:
		{
			log.Println("Received greet message from", s.GetNamespaceName())
		}
	default:
		return errors.New("unknown action: " + string(m.Action))
	}
	return nil
}

package aoiclient

import (
	"context"
	"encoding/json"

	"github.com/fedstackjs/azukiiro/storage"
	"github.com/go-resty/resty/v2"
)

type ProblemConfigJudge struct {
	Adapter string          `json:"adapter"`
	Config  json.RawMessage `json:"config"`
}

type ProblemConfig struct {
	Label    string             `json:"label"`
	Solution interface{}        `json:"solution"`
	Judge    ProblemConfigJudge `json:"judge"`
	Submit   interface{}        `json:"submit"`
}

type SolutionPoll struct {
	TaskId           string        `json:"taskId"`
	SolutionId       string        `json:"solutionId"`
	UserId           string        `json:"userId"`
	ContestId        string        `json:"contestId"`
	ProblemConfig    ProblemConfig `json:"problemConfig"`
	ProblemDataUrl   string        `json:"problemDataUrl"`
	ProblemDataHash  string        `json:"problemDataHash"`
	SolutionDataUrl  string        `json:"solutionDataUrl"`
	SolutionDataHash string        `json:"solutionDataHash"`
	ErrMsg           string        `json:"errMsg"`
}

func pollSolution(ctx context.Context, http *resty.Client) (*SolutionPoll, error) {
	res := &SolutionPoll{}
	raw, err := http.R().
		SetContext(ctx).
		SetBody(struct{}{}).
		SetResult(res).
		Post("/api/runner/solution/poll")
	err = loadError(raw, err)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type SolutionDetailsTest struct {
	Name       string  `json:"name"`
	Score      float64 `json:"score"`
	ScoreScale float64 `json:"scoreScale"`
	Status     string  `json:"status"`
	Summary    string  `json:"summary"`
}

type SolutionDetailsJob struct {
	Name       string                 `json:"name"`
	Score      float64                `json:"score"`
	ScoreScale float64                `json:"scoreScale"`
	Status     string                 `json:"status"`
	Tests      []*SolutionDetailsTest `json:"tests"`
	Summary    string                 `json:"summary"`
}

type SolutionDetails struct {
	Version int                   `json:"version"`
	Jobs    []*SolutionDetailsJob `json:"jobs"`
	Summary string                `json:"summary"`
}

type SolutionInfo struct {
	Score   float64             `json:"score"`
	Metrics *map[string]float64 `json:"metrics,omitempty"`
	Status  string              `json:"status"`
	Message string              `json:"message"`
}

func saveSolutionDetails(ctx context.Context, http *resty.Client, solutionId, taskId string, details *SolutionDetails) error {
	url, err := getSolutionTaskDetailsUrl(ctx, http, solutionId, taskId, "upload")
	if err != nil {
		return err
	}
	str, err := json.Marshal(details)
	if err != nil {
		return err
	}
	return storage.Upload(ctx, url, str)
}

func patchSolutionTask(ctx context.Context, http *resty.Client, solutionId, taskId string, req *SolutionInfo) error {
	raw, err := http.R().
		SetContext(ctx).
		SetBody(req).
		Patch("/api/runner/solution/task/" + solutionId + "/" + taskId)
	return loadError(raw, err)
}

func completeSolutionTask(ctx context.Context, http *resty.Client, solutionId, taskId string) error {
	raw, err := http.R().
		SetContext(ctx).
		Post("/api/runner/solution/task/" + solutionId + "/" + taskId + "/complete")
	return loadError(raw, err)
}

type urlResponse struct {
	URL string `json:"url"`
}

func getSolutionTaskDetailsUrl(ctx context.Context, http *resty.Client, solutionId, taskId string, urlType string) (string, error) {
	res := &urlResponse{}
	raw, err := http.R().
		SetContext(ctx).
		SetResult(res).
		Get("/api/runner/solution/task/" + solutionId + "/" + taskId + "/details/" + urlType)
	err = loadError(raw, err)
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

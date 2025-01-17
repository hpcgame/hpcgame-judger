package aoiclient

import (
	"context"

	"github.com/go-resty/resty/v2"
)

type registerRequest struct {
	Name              string   `json:"name"`
	Labels            []string `json:"labels"`
	Version           string   `json:"version"`
	RegistrationToken string   `json:"registrationToken"`
}

type registerResponse struct {
	RunnerId  string `json:"runnerId"`
	RunnerKey string `json:"runnerKey"`
}

func register(ctx context.Context, http *resty.Client, req *registerRequest) (*registerResponse, error) {
	res := &registerResponse{}
	raw, err := http.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(res).
		Post("/api/runner/register")
	err = loadError(raw, err)
	if err != nil {
		return nil, err
	}
	return res, nil
}

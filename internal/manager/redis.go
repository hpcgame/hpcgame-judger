package manager

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lcpu-club/hpcgame-judger/pkg/aoiclient"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	*redis.Client
}

func NewRedis(addr string) (*Redis, error) {
	opts, err := redis.ParseURL(addr)
	if err != nil {
		return nil, err
	}

	c := redis.NewClient(opts)

	return &Redis{
		Client: c,
	}, nil
}

func (r *Redis) List(prefix string) ([]string, error) {
	var keys []string
	var cursor uint64 = 0
	matchPattern := prefix + "*"

	for {
		scanKeys, nextCursor, err := r.Client.Scan(context.Background(), cursor, matchPattern, 100).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, scanKeys...)

		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	return keys, nil
}

func (r *Redis) DeletePrefix(prefix string) error {
	keys, err := r.List(prefix)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := r.Client.Del(context.Background(), key).Err(); err != nil {
			return err
		}
	}

	return nil
}

func (r *Redis) AcquireLock(key string, value string, exp time.Duration) (bool, error) {
	return r.Client.SetNX(context.Background(), key, value, exp).Result()
}

func (r *Redis) RefreshLock(key string, exp time.Duration) error {
	ok, err := r.Client.Expire(context.Background(), key, exp).Result()
	if !ok {
		return redis.Nil
	}
	return err
}

func (r *Redis) ReleaseLock(key string, value string) error {
	return r.Client.Watch(context.Background(), func(tx *redis.Tx) error {
		v, err := tx.Get(context.Background(), key).Result()
		if err != nil {
			return err
		}

		if v != value {
			return redis.TxFailedErr
		}

		_, err = tx.Pipelined(context.Background(), func(pipe redis.Pipeliner) error {
			pipe.Del(context.Background(), key)
			return nil
		})
		return err
	}, key)
}

func (r *Redis) IsLocked(key string) (bool, error) {
	res, err := r.Client.Exists(context.Background(), key).Result()
	if err != nil {
		return false, err
	}

	return res > 0, nil
}

const solnKeyPrefix = "soln:"

func (r *Redis) StoreSolutionPoll(soln *aoiclient.SolutionPoll) (id string, err error) {
	id = solnKeyPrefix + soln.SolutionId + ":" + soln.TaskId
	solnBytes, err := json.Marshal(soln)
	if err != nil {
		return "", err
	}
	err = r.Client.Set(context.Background(), id, solnBytes, 0).Err()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Redis) GetSolutionPoll(id string) (*aoiclient.SolutionPoll, error) {
	soln := &aoiclient.SolutionPoll{}
	bytes, err := r.Client.Get(context.Background(), id).Bytes()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, soln)
	if err != nil {
		return nil, err
	}
	return soln, nil
}

func (r *Redis) DeleteSolutionPoll(id string) error {
	return r.Client.Del(context.Background(), id).Err()
}

func (r *Redis) ListSolutionPoll() ([]string, error) {
	return r.List(solnKeyPrefix)
}

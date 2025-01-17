package manager

import (
	"context"
	"errors"
)

type RateLimiter struct {
	r *Redis

	key      string
	totalKey string
}

func NewRateLimiter(r *Redis, key string, totalKey string) *RateLimiter {
	return &RateLimiter{
		r: r,

		key:      key,
		totalKey: totalKey,
	}
}

func (rl *RateLimiter) Init(total int64) error {
	// Update total
	err := rl.r.Set(context.TODO(), rl.totalKey, total, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func (rl *RateLimiter) Request() (bool, error) {
	script := `
	local current = redis.call('GET', KEYS[1])
	local total = redis.call('GET', KEYS[2])

	if current == false then
		current = 0
	end

	if tonumber(current) < tonumber(total) then
		redis.call('INCR', KEYS[1])
		return 1
	else
		return 0
	end
	`

	result, err := rl.r.Eval(context.TODO(), script, []string{rl.key, rl.totalKey}).Int64()
	if err != nil {
		return false, err
	}

	if result == 1 {
		return true, nil
	} else if result == 0 {
		return false, nil
	}

	return false, errors.New("unexpected result from Redis script")
}

func (rl *RateLimiter) Release() error {
	script := `
	local current = redis.call('GET', KEYS[1])

	if current == false or tonumber(current) <= 0 then
		return false
	else
		redis.call('DECR', KEYS[1])
		return true
	end
	`

	_, err := rl.r.Eval(context.TODO(), script, []string{rl.key}).Result()
	if err != nil {
		return err
	}

	return nil
}

package framework

import "time"

func expCoolDown(curr time.Duration, max time.Duration) (next time.Duration) {
	next = curr * 2
	if next > max {
		next = max
	}
	return
}

func WaitTill[T any](f func() (T, error), cond func(T) bool, max int, coolDown time.Duration) (T, error) {
	myCoolDown := coolDown
	for i := 1; i < max; i++ {
		val, err := f()
		if err != nil {
			return val, err
		}
		if cond(val) {
			return val, nil
		}
		time.Sleep(coolDown)
		myCoolDown = expCoolDown(myCoolDown, 16*coolDown)
	}
	return f()
}

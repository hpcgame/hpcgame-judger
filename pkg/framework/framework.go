package framework

import (
	"context"
	"errors"
	"os"

	"github.com/lcpu-club/hpcgame-judger/pkg/judgerproto"
)

func PanicString(err string) {
	Panic(errors.New(err))
}

func Panic(err error) {
	judgerproto.NewErrorMessage(err).Print()
	os.Exit(0)
}

func NilOrPanic(err error) {
	if err != nil {
		Panic(err)
	}
}

func Must[T any](val T, err error) T {
	if err != nil {
		Panic(err)
	}
	return val
}

func BgCtx() context.Context {
	return context.Background()
}

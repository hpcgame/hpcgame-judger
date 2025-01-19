package framework

import (
	"reflect"
	"sync"
)

type Iterable[T any] interface {
	Len() int
	Idx(int) T
}

type Slice[T any] []T

func (s Slice[T]) Len() int {
	return len(s)
}

func (s Slice[T]) Idx(i int) T {
	return s[i]
}

func FormSlice[T any](s ...T) Slice[T] {
	return Slice[T](s)
}

func Last[T any](s []T) T {
	return s[len(s)-1]
}

type JudgePoint interface{}

func Pnt[P JudgePoint](values ...any) P {
	v := reflect.New(reflect.TypeOf((*P)(nil)).Elem()).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		if i >= len(values) {
			break
		}
		v.Field(i).Set(reflect.ValueOf(values[i]))
	}
	return v.Interface().(P)
}

type Fuser interface {
	Fuse()
}

type fuser struct {
	ch chan error
}

func (f *fuser) Fuse() {
	f.ch <- nil
}
func (f *fuser) FuseErr(err error) {
	f.ch <- err
}
func (f *fuser) Ch() chan error {
	return f.ch
}
func newFuser() *fuser {
	return &fuser{ch: make(chan error)}
}

type simpleFuser struct {
	fused bool
}

func (f *simpleFuser) Fuse() {
	f.fused = true
}
func (f *simpleFuser) Fused() bool {
	return f.fused
}
func newSimpleFuser() *simpleFuser {
	return &simpleFuser{fused: false}
}

type JudgeMsg interface {
	Point() int
	PointScale() int
}

type MultiPointJudger[P JudgePoint, M any] interface {
	Before(Fuser) error
	Judge(P, Fuser) (M, error)
	Report([]M) error
	After([]M) error
}

type MultiPointRunner[P JudgePoint, M any] struct {
	j MultiPointJudger[P, M]

	parallel int
}

func MultiPoint[P JudgePoint, M any](j MultiPointJudger[P, M]) *MultiPointRunner[P, M] {
	return &MultiPointRunner[P, M]{j: j}
}

func (r *MultiPointRunner[P, M]) WithParallel(n int) *MultiPointRunner[P, M] {
	r.parallel = n
	return r
}

func (r *MultiPointRunner[P, M]) init(points Iterable[P]) error {
	if r.parallel == 0 {
		r.parallel = points.Len()
	}
	return nil
}

func (r *MultiPointRunner[P, M]) Run(points Iterable[P]) error {
	if err := r.init(points); err != nil {
		return err
	}

	sf := newSimpleFuser()
	if err := r.j.Before(sf); err != nil {
		return err
	}
	if sf.Fused() {
		return nil
	}

	var msgs []M
	var reportLock = &sync.Mutex{}

	pool := NewThreadPool(r.parallel)
	f := newFuser()

	for i := 0; i < points.Len(); i++ {
		idx := i
		pool.Add(func() {
			p := points.Idx(idx)
			m, err := r.j.Judge(p, f)
			if err != nil {
				return
			}
			reportLock.Lock()
			defer reportLock.Unlock()
			msgs = append(msgs, m)
			err = r.j.Report(msgs)
			if err != nil {
				f.FuseErr(err)
				return
			}
		})
	}

	pool.Start()

	select {
	case err := <-f.Ch():
		pool.Cancel()
		if err != nil {
			return err
		}
	case <-pool.Ch():
	}

	if err := r.j.After(msgs); err != nil {
		return err
	}

	return nil
}

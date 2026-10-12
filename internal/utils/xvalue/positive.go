package xvalue

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
)

// Number is the set of value types the Positive validator accepts (time.Duration included).
type Number interface {
	~int | ~int32 | ~int64
}

// Positive guards a Value that must stay > 0. An invalid value is never applied:
// the last valid value (or the default) is returned instead and a warning is logged.
type Positive[T Number] struct {
	name      string
	value     Value[T]
	lastValid atomic.Int64
}

// NewPositive wraps v; a nil v is replaced by a static def.
func NewPositive[T Number](name string, v Value[T], def T) *Positive[T] {
	if v == nil {
		v = NewStaticValue(def)
	}

	p := &Positive[T]{name: name, value: v}
	p.lastValid.Store(int64(def))

	return p
}

// Load returns the current value if it is valid, otherwise the last valid one.
// A Get that panics (e.g. a typed-nil Value) counts as invalid.
// ctx is used only for logging.
func (p *Positive[T]) Load(ctx context.Context) T {
	v, err := p.get()
	if err == nil && v > 0 {
		// A plain load first keeps the hot path write-free when nothing changed.
		if p.lastValid.Load() != int64(v) {
			p.lastValid.Store(int64(v))
		}
		return v
	}

	fallback := T(p.lastValid.Load())
	// fmt keeps time.Duration human-readable ("30s" instead of nanoseconds).
	xlog.Warn(ctx, "invalid option value ignored",
		xfield.String("option", p.name),
		xfield.String("value", fmt.Sprint(v)),
		xfield.String("applied", fmt.Sprint(fallback)),
		xfield.Error(err),
	)

	return fallback
}

func (p *Positive[T]) get() (v T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("option value getter panicked: %v", r)
		}
	}()

	return p.value.Get(), nil
}

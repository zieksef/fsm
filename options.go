package fsm

import (
	"errors"
	"fmt"
)

// Option configures the FSM during New and may reject bad configuration by
// returning an error.
type Option[T any] func(f *FSM[T]) error

// WithOnEnter registers the machine-wide enter callback, invoked after every
// successful transition with the entered state. At most one may be
// registered; nil is ignored.
func WithOnEnter[T any](fn Callback[T]) Option[T] {
	return func(f *FSM[T]) error {
		if fn == nil {
			return nil
		}
		if f.onEnter != nil {
			return fmt.Errorf("%w (hook onEnter)", ErrDuplicateCallback)
		}
		f.onEnter = fn
		return nil
	}
}

// WithOnExit registers the machine-wide exit callback, invoked after every
// successful transition with the exited state. At most one may be
// registered; nil is ignored.
func WithOnExit[T any](fn Callback[T]) Option[T] {
	return func(f *FSM[T]) error {
		if fn == nil {
			return nil
		}
		if f.onExit != nil {
			return fmt.Errorf("%w (hook onExit)", ErrDuplicateCallback)
		}
		f.onExit = fn
		return nil
	}
}

// WithDeciders registers per-state deciders for Drive. Every state must
// exist in the transition table and carry at most one decider across all
// WithDeciders calls; nil values are ignored.
func WithDeciders[T any](deciders map[State]Decider[T]) Option[T] {
	return func(f *FSM[T]) error {
		var errs []error
		for s, fn := range deciders {
			if fn == nil {
				continue
			}
			n, ok := f.nodes[s]
			if !ok {
				errs = append(errs, fmt.Errorf("%w (state %s)", ErrUnknownState, s))
				continue
			}
			if n.decider != nil {
				errs = append(errs, fmt.Errorf("%w (state %s)", ErrDuplicateDecider, s))
				continue
			}
			n.decider = fn
		}
		return errors.Join(errs...)
	}
}

package fsm

import (
	"errors"
	"fmt"
)

type Option[T any] func(f *FSM[T]) error

func WithOnEnter[T any](fn Callback[T]) Option[T] {
	return func(f *FSM[T]) error {
		if fn == nil {
			return nil
		}
		if f.onEnter != nil {
			return fmt.Errorf("%w: onEnter", ErrDuplicateCallback)
		}
		f.onEnter = fn
		return nil
	}
}

func WithOnExit[T any](fn Callback[T]) Option[T] {
	return func(f *FSM[T]) error {
		if fn == nil {
			return nil
		}
		if f.onExit != nil {
			return fmt.Errorf("%w: onExit", ErrDuplicateCallback)
		}
		f.onExit = fn
		return nil
	}
}

func WithDeciders[T any](deciders map[State]Decider[T]) Option[T] {
	return func(f *FSM[T]) error {
		var errs []error
		for s, fn := range deciders {
			if fn == nil {
				continue
			}
			n, ok := f.nodes[s]
			if !ok {
				errs = append(errs, fmt.Errorf("%w: %s", ErrUnknownState, s))
				continue
			}
			if n.decider != nil {
				errs = append(errs, fmt.Errorf("%w: %s", ErrDuplicateDecider, s))
				continue
			}
			n.decider = fn
		}
		return errors.Join(errs...)
	}
}

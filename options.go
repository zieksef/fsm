package fsm

import (
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

func WithDecider[T any](state State, fn Decider[T]) Option[T] {
	return func(f *FSM[T]) error {
		if fn == nil {
			return nil
		}
		n, ok := f.nodes[state]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownState, state)
		}
		n.decider = fn
		return nil
	}
}

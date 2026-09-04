package fsm

import (
	"context"
	"errors"
	"fmt"
)

type FSM[T any] struct {
	nodes   map[State]*stateNode[T]
	onEnter Callback[T]
	onExit  Callback[T]
}

func New[T any](transitions []Transition[T], opts ...Option[T]) (*FSM[T], error) {
	f := &FSM[T]{
		nodes: make(map[State]*stateNode[T]),
	}

	var buildErrs []error
	for _, t := range transitions {
		src := f.getOrCreate(t.From)
		if _, ok := src.transitions[t.Event]; ok {
			buildErrs = append(buildErrs, fmt.Errorf("%w: %s + %s", ErrDuplicateTransition, t.From, t.Event))
			continue
		}
		src.transitions[t.Event] = &transition[T]{
			dst:    f.getOrCreate(t.To),
			guard:  t.Guard,
			action: t.Action,
		}
	}

	for _, n := range f.nodes {
		n.terminal = len(n.transitions) == 0
	}

	for _, opt := range opts {
		if err := opt(f); err != nil {
			buildErrs = append(buildErrs, err)
		}
	}

	if len(buildErrs) > 0 {
		return nil, errors.Join(buildErrs...)
	}

	return f, nil
}

func (f *FSM[T]) getOrCreate(s State) *stateNode[T] {
	n, ok := f.nodes[s]
	if !ok {
		n = &stateNode[T]{name: s, transitions: make(map[Event]*transition[T])}
		f.nodes[s] = n
	}
	return n
}

// Fire executes one transition out of from on event and returns the state
// after the call: the transition target on success, from unchanged on error.
// The FSM holds no run state, so one instance can serve concurrent callers.
func (f *FSM[T]) Fire(ctx context.Context, from State, event Event, v *T) (State, error) {
	n, ok := f.nodes[from]
	if !ok {
		return from, ErrNoTransitions
	}

	if n.terminal {
		return from, ErrInTerminalState
	}

	t, ok := n.transitions[event]
	if !ok {
		return from, ErrInvalidEvent
	}

	if t.guard != nil && !t.guard(ctx, v) {
		return from, ErrGuardRejected
	}

	if t.action != nil {
		if err := t.action(ctx, v); err != nil {
			return from, err
		}
	}

	if f.onExit != nil {
		f.onExit(ctx, event, from, v)
	}

	if f.onEnter != nil {
		f.onEnter(ctx, event, t.dst.name, v)
	}

	return t.dst.name, nil
}

// Drive repeatedly asks the decider of the state it is in for the next event
// and fires it, starting at from, until a terminal state is reached. Like
// Fire, the returned State is always the state after the call: the terminal
// state on success, or the last state reached when it stops on error.
func (f *FSM[T]) Drive(ctx context.Context, from State, v *T) (State, error) {
	current := from
	for !f.Terminal(current) {
		if err := ctx.Err(); err != nil {
			return current, err
		}

		n, ok := f.nodes[current]
		if !ok || n.decider == nil {
			return current, ErrNoDecider
		}

		next, fireErr := f.Fire(ctx, current, n.decider(ctx, v), v)
		if fireErr != nil {
			return current, fireErr
		}

		current = next
	}

	return current, nil
}

func (f *FSM[T]) Has(from State, event Event) bool {
	n, ok := f.nodes[from]
	if !ok {
		return false
	}
	_, ok = n.transitions[event]
	return ok
}

func (f *FSM[T]) Terminal(state State) bool {
	n, ok := f.nodes[state]
	return ok && n.terminal
}

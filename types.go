package fsm

import (
	"context"
)

type (
	State string
	Event string

	// Guard gates a transition. Keep it a pure predicate over v — no IO, no
	// side effects; a precondition that needs a typed rejection reason
	// belongs at the top of the Action instead.
	Guard[T any] func(ctx context.Context, v *T) bool

	// Action performs the transition's side effect; a non-nil error aborts
	// the transition with the state unchanged.
	Action[T any] func(ctx context.Context, v *T) error

	// Callback observes a completed transition and cannot alter its outcome:
	// the exit callback receives from as state, the enter callback receives
	// to.
	Callback[T any] func(ctx context.Context, event Event, state State, v *T)

	// Decider picks the next event to fire from the state it is registered
	// on; consulted only by Drive.
	Decider[T any] func(ctx context.Context, v *T) Event
)

// Transition declares one edge of the graph: on Event, From moves to To,
// gated by Guard and performing Action (both optional).
type Transition[T any] struct {
	From   State
	Event  Event
	To     State
	Guard  Guard[T]
	Action Action[T]
}

type stateNode[T any] struct {
	name        State
	terminal    bool // inferred by New: no outgoing transitions means terminal
	transitions map[Event]*transition[T]
	decider     Decider[T]
}

type transition[T any] struct {
	dst    *stateNode[T]
	guard  Guard[T]
	action Action[T]
}

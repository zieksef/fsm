package fsm

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test helpers ---

type orderCtx struct {
	Id     string
	Amount int
	Log    []string
}

func orderTransitions() []Transition[orderCtx] {
	return []Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending"},
		{From: "pending", Event: "approve", To: "approved"},
		{From: "pending", Event: "reject", To: "rejected"},
		{From: "draft", Event: "cancel", To: "cancelled"},
		{From: "pending", Event: "cancel", To: "cancelled"},
	}
}

func mustNew[T any](t *testing.T, transitions []Transition[T], opts ...Option[T]) *FSM[T] {
	t.Helper()

	m, newErr := New(transitions, opts...)
	require.NoError(t, newErr)
	return m
}

// --- New validation tests ---

func TestNew_RejectsDuplicateTransition(t *testing.T) {
	t.Parallel()

	m, newErr := New([]Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending"},
		{From: "draft", Event: "submit", To: "cancelled"},
	})
	require.ErrorIs(t, newErr, ErrDuplicateTransition)
	assert.Nil(t, m)
}

func TestNew_RejectsUnknownStateInDeciders(t *testing.T) {
	t.Parallel()

	m, newErr := New(orderTransitions(),
		WithDeciders(map[State]Decider[orderCtx]{
			"nonexistent": func(_ context.Context, _ *orderCtx) Event { return "x" },
		}))
	require.ErrorIs(t, newErr, ErrUnknownState)
	assert.Nil(t, m)
}

func TestNew_RejectsDuplicateDecider(t *testing.T) {
	t.Parallel()

	d := func(_ context.Context, _ *orderCtx) Event { return "submit" }

	m, newErr := New(orderTransitions(),
		WithDeciders(map[State]Decider[orderCtx]{"draft": d}),
		WithDeciders(map[State]Decider[orderCtx]{"draft": d}),
	)
	require.ErrorIs(t, newErr, ErrDuplicateDecider)
	assert.Nil(t, m)
}

func TestNew_RejectsDuplicateCallback(t *testing.T) {
	t.Parallel()

	cb := func(_ context.Context, _ Event, _ State, _ *orderCtx) {}

	tests := []struct {
		name string
		opts []Option[orderCtx]
	}{
		{name: "onEnter", opts: []Option[orderCtx]{WithOnEnter[orderCtx](cb), WithOnEnter[orderCtx](cb)}},
		{name: "onExit", opts: []Option[orderCtx]{WithOnExit[orderCtx](cb), WithOnExit[orderCtx](cb)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m, newErr := New(orderTransitions(), tt.opts...)
			require.ErrorIs(t, newErr, ErrDuplicateCallback)
			assert.Nil(t, m)
		})
	}
}

// --- Fire tests ---

func TestFire_ChainedTransitions(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions())
	ctx := context.Background()
	c := &orderCtx{Id: "1"}

	next, fireErr := m.Fire(ctx, "draft", "submit", c)
	require.NoError(t, fireErr)
	assert.Equal(t, State("pending"), next)

	next, fireErr = m.Fire(ctx, next, "approve", c)
	require.NoError(t, fireErr)
	assert.Equal(t, State("approved"), next)
}

func TestFire_ErrorReturnsCurrentUnchanged(t *testing.T) {
	t.Parallel()

	actionErr := errors.New("action failed")
	m := mustNew(t, []Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending",
			Guard: func(_ context.Context, v *orderCtx) bool { return v.Amount > 0 },
		},
		{From: "pending", Event: "approve", To: "approved",
			Action: func(_ context.Context, _ *orderCtx) error { return actionErr },
		},
	})

	tests := []struct {
		name    string
		current State
		event   Event
		wantErr error
	}{
		{name: "invalid event", current: "draft", event: "approve", wantErr: ErrInvalidEvent},
		{name: "guard rejected", current: "draft", event: "submit", wantErr: ErrGuardRejected},
		{name: "action error", current: "pending", event: "approve", wantErr: actionErr},
		{name: "terminal state", current: "approved", event: "submit", wantErr: ErrInTerminalState},
		{name: "unknown state", current: "orphan", event: "submit", wantErr: ErrNoTransitions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next, fireErr := m.Fire(context.Background(), tt.current, tt.event, &orderCtx{})
			require.ErrorIs(t, fireErr, tt.wantErr)
			assert.Equal(t, tt.current, next, "on error Fire must return the input state")
		})
	}
}

func TestFire_SharedInstanceAcrossGoroutines(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions())

	const n = 100

	var wg sync.WaitGroup
	nexts := make([]State, n)
	fireErrs := make([]error, n)
	wants := make([]State, n)

	for i := range n {
		current, event := State("draft"), Event("submit")
		wants[i] = "pending"
		if i%2 == 0 {
			current, event, wants[i] = "pending", "approve", "approved"
		}

		wg.Go(func() {
			nexts[i], fireErrs[i] = m.Fire(context.Background(), current, event, &orderCtx{})
		})
	}
	wg.Wait()

	for i := range n {
		require.NoError(t, fireErrs[i])
		assert.Equal(t, wants[i], nexts[i])
	}
}

// --- Guard tests ---

func TestFire_GuardAllows(t *testing.T) {
	t.Parallel()

	m := mustNew(t, []Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending",
			Guard: func(_ context.Context, v *orderCtx) bool { return v.Amount > 0 },
		},
	})

	next, fireErr := m.Fire(context.Background(), "draft", "submit", &orderCtx{Amount: 100})
	require.NoError(t, fireErr)
	assert.Equal(t, State("pending"), next)
}

// --- Callback tests ---

func TestFire_OnEnterOnExit(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions(),
		WithOnExit[orderCtx](func(_ context.Context, e Event, s State, v *orderCtx) {
			v.Log = append(v.Log, "exit:"+string(s)+":"+string(e))
		}),
		WithOnEnter[orderCtx](func(_ context.Context, e Event, s State, v *orderCtx) {
			v.Log = append(v.Log, "enter:"+string(s)+":"+string(e))
		}),
	)

	c := &orderCtx{}
	_, fireErr := m.Fire(context.Background(), "draft", "submit", c)
	require.NoError(t, fireErr)
	assert.Equal(t, []string{"exit:draft:submit", "enter:pending:submit"}, c.Log,
		"onExit must receive from, onEnter must receive to, both with the event")
}

// --- Action tests ---

func TestFire_ActionExecutes(t *testing.T) {
	t.Parallel()

	m := mustNew(t, []Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending", Action: func(_ context.Context, v *orderCtx) error {
			v.Log = append(v.Log, "action:submit")
			return nil
		}},
	})

	c := &orderCtx{}
	next, fireErr := m.Fire(context.Background(), "draft", "submit", c)
	require.NoError(t, fireErr)
	assert.Equal(t, State("pending"), next)
	assert.Contains(t, c.Log, "action:submit")
}

// --- Drive (auto-drive) tests ---

func TestDrive_DrivesToTerminal(t *testing.T) {
	t.Parallel()

	type msgCtx struct {
		Steps []string
	}

	handler := func(state string) Decider[msgCtx] {
		return func(_ context.Context, v *msgCtx) Event {
			v.Steps = append(v.Steps, state)
			return "next"
		}
	}

	transitions := []Transition[msgCtx]{
		{From: "received", Event: "next", To: "validated"},
		{From: "validated", Event: "next", To: "processed"},
		{From: "processed", Event: "next", To: "done"},
	}

	m := mustNew(t, transitions,
		WithDeciders(map[State]Decider[msgCtx]{
			"received":  handler("received"),
			"validated": handler("validated"),
			"processed": handler("processed"),
		}),
	)

	c := &msgCtx{}
	final, driveErr := m.Drive(context.Background(), "received", c)
	require.NoError(t, driveErr)
	assert.Equal(t, State("done"), final)
	assert.Equal(t, []string{"received", "validated", "processed"}, c.Steps)
}

func TestDrive_NoDecider_ReturnsError(t *testing.T) {
	t.Parallel()

	m := mustNew(t, []Transition[orderCtx]{
		{From: "a", Event: "next", To: "b"},
	})
	// No decider registered for state "a"

	final, driveErr := m.Drive(context.Background(), "a", &orderCtx{})
	require.ErrorIs(t, driveErr, ErrNoDecider)
	assert.Equal(t, State("a"), final, "on error Drive must return the last state reached")
}

func TestDrive_Rerun(t *testing.T) {
	t.Parallel()

	type emptyCtx struct{}

	m := mustNew(t, []Transition[emptyCtx]{
		{From: "start", Event: "go", To: "end"},
	},
		WithDeciders(map[State]Decider[emptyCtx]{
			"start": func(_ context.Context, _ *emptyCtx) Event {
				return "go"
			},
		}),
	)

	// The machine keeps no run state, so the same instance drives repeatedly.
	c := &emptyCtx{}
	for range 2 {
		final, driveErr := m.Drive(context.Background(), "start", c)
		require.NoError(t, driveErr)
		assert.Equal(t, State("end"), final)
	}
}

// --- Query method tests ---

func TestHas(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions())

	assert.True(t, m.Has("draft", "submit"))
	assert.True(t, m.Has("draft", "cancel"))
	assert.False(t, m.Has("draft", "approve"))
	assert.False(t, m.Has("draft", "reject"))
	assert.False(t, m.Has("unknown", "submit"))
}

func TestTerminal(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions())

	assert.True(t, m.Terminal("approved"))
	assert.False(t, m.Terminal("draft"))
	assert.False(t, m.Terminal("unknown"))
}

// --- Execution order tests ---

func TestFire_ExecutionOrder(t *testing.T) {
	t.Parallel()

	var order []string

	m := mustNew(t, []Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending",
			Guard: func(_ context.Context, _ *orderCtx) bool {
				order = append(order, "guard")
				return true
			},
			Action: func(_ context.Context, _ *orderCtx) error {
				order = append(order, "action")
				return nil
			},
		},
	},
		WithOnExit[orderCtx](func(_ context.Context, _ Event, _ State, _ *orderCtx) {
			order = append(order, "onExit")
		}),
		WithOnEnter[orderCtx](func(_ context.Context, _ Event, _ State, _ *orderCtx) {
			order = append(order, "onEnter")
		}),
	)

	_, fireErr := m.Fire(context.Background(), "draft", "submit", &orderCtx{})
	require.NoError(t, fireErr)
	assert.Equal(t, []string{"guard", "action", "onExit", "onEnter"}, order)
}

// --- Multiple transitions from same state tests ---

func TestFire_MultipleTransitionsFromSameState(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions())

	// Can go to approved, rejected, or cancelled from pending
	assert.True(t, m.Has("pending", "approve"))
	assert.True(t, m.Has("pending", "reject"))
	assert.True(t, m.Has("pending", "cancel"))

	next, fireErr := m.Fire(context.Background(), "pending", "reject", &orderCtx{})
	require.NoError(t, fireErr)
	assert.Equal(t, State("rejected"), next)
}

// --- Action failure + OnExit tests ---

func TestFire_ActionError_OnExitNotCalled(t *testing.T) {
	t.Parallel()

	exitCalled := false
	actionErr := errors.New("action failed")

	m := mustNew(t, []Transition[orderCtx]{
		{From: "draft", Event: "submit", To: "pending",
			Action: func(_ context.Context, _ *orderCtx) error {
				return actionErr
			},
		},
	},
		WithOnExit[orderCtx](func(_ context.Context, _ Event, _ State, _ *orderCtx) {
			exitCalled = true
		}),
	)

	next, fireErr := m.Fire(context.Background(), "draft", "submit", &orderCtx{})
	require.ErrorIs(t, fireErr, actionErr)
	assert.Equal(t, State("draft"), next)
	assert.False(t, exitCalled, "OnExit should not be called when Action fails")
}

// --- Nil callback/decider tests ---

func TestWithNilCallbackIgnored(t *testing.T) {
	t.Parallel()

	m := mustNew(t, orderTransitions(),
		WithOnEnter[orderCtx](nil),
		WithOnExit[orderCtx](nil),
		WithDeciders(map[State]Decider[orderCtx]{"draft": nil}),
	)

	// nil OnEnter/OnExit should not cause panic
	next, fireErr := m.Fire(context.Background(), "draft", "submit", &orderCtx{})
	require.NoError(t, fireErr)
	assert.Equal(t, State("pending"), next)

	// nil Decider should not be registered
	_, driveErr := m.Drive(context.Background(), "draft", &orderCtx{})
	require.ErrorIs(t, driveErr, ErrNoDecider)
}

// --- Drive ctx cancel tests ---

func TestDrive_CtxCancel(t *testing.T) {
	t.Parallel()

	type counter struct{ N int }

	transitions := []Transition[counter]{
		{From: "a", Event: "next", To: "b"},
		{From: "b", Event: "next", To: "c"},
		{From: "c", Event: "next", To: "done"},
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := mustNew(t, transitions,
		WithDeciders(map[State]Decider[counter]{
			"a": func(_ context.Context, v *counter) Event {
				v.N++
				cancel() // 第一步就取消 ctx
				return "next"
			},
			"b": func(_ context.Context, v *counter) Event {
				v.N++
				return "next"
			},
			"c": func(_ context.Context, v *counter) Event {
				v.N++
				return "next"
			},
		}),
	)

	c := &counter{}
	final, driveErr := m.Drive(ctx, "a", c)
	require.ErrorIs(t, driveErr, context.Canceled)
	assert.Equal(t, 1, c.N, "should stop after first decider due to ctx cancel")
	assert.Equal(t, State("b"), final, "should have completed first transition")
}

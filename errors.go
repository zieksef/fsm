package fsm

import (
	"errors"
)

var (
	ErrInvalidEvent        = errors.New("fsm: invalid event in current state")
	ErrGuardRejected       = errors.New("fsm: transition rejected by guard")
	ErrNoDecider           = errors.New("fsm: no decider for state")
	ErrNoTransitions       = errors.New("fsm: no transitions from state")
	ErrInTerminalState     = errors.New("fsm: already in terminal state")
	ErrDuplicateTransition = errors.New("fsm: duplicate transition for same state and event")
	ErrDuplicateCallback   = errors.New("fsm: callback already registered")
	ErrDuplicateDecider    = errors.New("fsm: decider already registered for state")
	ErrUnknownState        = errors.New("fsm: option references unknown state")
)

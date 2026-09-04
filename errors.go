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
	ErrDuplicateTransition = errors.New("fsm: duplicate transition")
	ErrDuplicateCallback   = errors.New("fsm: duplicate callback")
	ErrDuplicateDecider    = errors.New("fsm: duplicate decider")
	ErrUnknownState        = errors.New("fsm: unknown state")
)

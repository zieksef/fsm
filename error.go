package fsm

import (
	"errors"
)

var (
	ErrEventNotRegistered = errors.New("event not registered on the current state")
	ErrTriggerStateless   = errors.New("trigger event when stateless")
)

package fsm

import (
	"fmt"
)

type pair struct {
	Src   State
	Event Event
}

type FSM struct {
	curr  State // current state
	trans map[pair]Transition
}

// New
//
//	@param s: initial state
func New(s State) *FSM {
	return &FSM{
		curr:  s,
		trans: make(map[pair]Transition),
	}
}

func (m *FSM) Add(t Transition) *FSM {
	m.trans[pair{Src: t.Src, Event: t.Trigger}] = t
	return m
}

func (m *FSM) Remove(t Transition) bool {
	p := pair{Src: t.Src, Event: t.Trigger}

	_, ok := m.trans[p]
	delete(m.trans, p)

	return ok
}

func (m *FSM) Trigger(evt Event) (err error) {
	if m.Stateless() {
		return ErrTriggerStateless
	}

	p := pair{Src: m.curr, Event: evt}
	trans, ok := m.trans[p]
	if !ok {
		return ErrEventNotRegistered
	}

	if trans.Action == nil {
		return nil
	}

	if ctxErr := m.curr.Exit(nil); ctxErr != nil {
		return fmt.Errorf("exit state[%s]: %w", m.curr.Name(), ctxErr)
	}

	defer func() {
		if err != nil {
			return
		}

		if ctxErr := m.curr.Enter(nil); ctxErr != nil {
			err = fmt.Errorf("enter state[%s]: %w", m.curr.Name(), ctxErr)
		}
	}() // closure

	m.curr, err = trans.Action()

	return
}

func (m *FSM) Stateless() bool {
	return m.curr == nil || m.curr.Stateless()
}

func (m *FSM) State() State {
	return m.curr
}

func (m *FSM) SetState(s State) {
	m.curr = s
}

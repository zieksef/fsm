package fsm

type Event string

type Action func() (State, error) // returns the final state after performing the action

type Transition struct {
	Src     State
	Trigger Event
	Action  Action
}

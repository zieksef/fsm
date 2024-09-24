package fsm

import (
	"context"
	"fmt"
)

type State interface {
	Name() string
	Stateless() bool
	Enter(ctx context.Context) error
	Exit(ctx context.Context) error
}

type BasicState string

func (s BasicState) Name() string {
	return string(s)
}

func (s BasicState) Stateless() bool {
	return s == ""
}

func (s BasicState) Enter(_ context.Context) error {
	fmt.Printf("enter %s\n", s)
	return nil
}

func (s BasicState) Exit(_ context.Context) error {
	fmt.Printf("exit %s\n", s)
	return nil
}

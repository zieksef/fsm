package fsm

import (
	"context"
)

type (
	State string
	Event string

	Guard[T any]    func(ctx context.Context, v *T) bool                      // 状态转移的前置判断
	Action[T any]   func(ctx context.Context, v *T) error                     // 事件触发的动作
	Callback[T any] func(ctx context.Context, event Event, state State, v *T) // 全局回调：onExit 的 state 为 from，onEnter 的 state 为 to
	Decider[T any]  func(ctx context.Context, v *T) Event                     // 决策器：自动模式下决定下一个事件
)

type Transition[T any] struct {
	From   State
	Event  Event
	To     State
	Guard  Guard[T]
	Action Action[T]
}

type stateNode[T any] struct {
	name        State
	terminal    bool // 无出边即终态
	transitions map[Event]*transition[T]
	decider     Decider[T]
}

type transition[T any] struct {
	dst    *stateNode[T]
	guard  Guard[T]
	action Action[T]
}

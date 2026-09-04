package fsm

import (
	"fmt"
	"slices"
	"strings"
)

// Mermaid renders the transition graph as a Mermaid stateDiagram-v2 block.
// Output is deterministic (states and events sorted); terminal states point
// to the final marker [*].
func (f *FSM[T]) Mermaid() string {
	var b strings.Builder
	b.WriteString("stateDiagram-v2\n")
	for _, n := range f.sortedNodes() {
		for _, e := range sortedEvents(n) {
			fmt.Fprintf(&b, "    %s --> %s: %s\n", n.name, n.transitions[e].dst.name, e)
		}
		if n.terminal {
			fmt.Fprintf(&b, "    %s --> [*]\n", n.name)
		}
	}
	return b.String()
}

// Visualize renders the transition graph in Graphviz DOT format; terminal
// states are drawn as double circles. Output is deterministic like Mermaid.
func (f *FSM[T]) Visualize() string {
	var b strings.Builder
	b.WriteString("digraph fsm {\n    rankdir=LR;\n    node [shape=circle];\n")
	for _, n := range f.sortedNodes() {
		if n.terminal {
			fmt.Fprintf(&b, "    %q [shape=doublecircle];\n", string(n.name))
		}
	}
	for _, n := range f.sortedNodes() {
		for _, e := range sortedEvents(n) {
			fmt.Fprintf(&b, "    %q -> %q [label=%q];\n",
				string(n.name), string(n.transitions[e].dst.name), string(e))
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func (f *FSM[T]) sortedNodes() []*stateNode[T] {
	nodes := make([]*stateNode[T], 0, len(f.nodes))
	for _, n := range f.nodes {
		nodes = append(nodes, n)
	}
	slices.SortFunc(nodes, func(a *stateNode[T], b *stateNode[T]) int {
		return strings.Compare(string(a.name), string(b.name))
	})
	return nodes
}

func sortedEvents[T any](n *stateNode[T]) []Event {
	events := make([]Event, 0, len(n.transitions))
	for e := range n.transitions {
		events = append(events, e)
	}
	slices.Sort(events)
	return events
}

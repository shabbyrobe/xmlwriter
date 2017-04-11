package xmlwriter

import "fmt"

// Deprecated: Use ErrCollector instead
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

type Event struct {
	State    NodeState
	Node     NodeKind
	Children int
}

func (e Event) String() string {
	return fmt.Sprintf("%d\t%s\t+%d", e.State, kindName[e.Node], e.Children)
}

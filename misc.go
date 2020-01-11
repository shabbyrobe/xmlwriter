package xmlwriter

import "fmt"

// Event is raised when a node changes state in the writer. It is currently
// only relevant to the Indenter but may become an event system in a later
// version.
type Event struct {
	State    NodeState
	Node     NodeKind
	Children int
}

func (e Event) String() string {
	return fmt.Sprintf("%d\t%s\t+%d", e.State, e.Node.Name(), e.Children)
}

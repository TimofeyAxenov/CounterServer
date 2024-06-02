package stack

type (
	Stack struct {
		top    *node
		length int
	}
	node struct {
		value string
		prev  *node
	}
)

// Create a new stack
func New() *Stack {
	return &Stack{nil, 0}
}

// Return the number of items in the stack
func (s *Stack) Len() int {
	return s.length
}

// View the top item on the stack
func (s *Stack) Peek() string {
	if s.length == 0 {
		return ""
	}
	return s.top.value
}

// Pop the top item of the stack and return it
func (s *Stack) Pop() string {
	if s.length == 0 {
		return ""
	}

	n := s.top
	s.top = n.prev
	s.length--
	return n.value
}

// Push a value onto the top of the stack
func (s *Stack) Push(value string) {
	n := &node{"", s.top}
	s.top = n
	s.length++
}

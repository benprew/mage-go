package mage

import "github.com/google/uuid"

// StackObject represents something on the stack (spell or ability).
type StackObject struct {
	ID         uuid.UUID
	Card       Card       // non-nil for spells
	Controller uuid.UUID
	SourceID   uuid.UUID  // source permanent (for abilities)
	Effects    []Effect
	Targets    []uuid.UUID
	IsAbility  bool
	XValue      int        // value of X for X-cost spells
	ModeChoice  int        // chosen mode for modal spells (0-indexed)
	EventAmount int        // amount from triggering event (e.g. damage dealt)
}

// Stack represents the game stack.
type Stack struct {
	objects []*StackObject
}

func NewStack() *Stack {
	return &Stack{}
}

func (s *Stack) Push(obj *StackObject) {
	s.objects = append(s.objects, obj)
}

func (s *Stack) Pop() *StackObject {
	if len(s.objects) == 0 {
		return nil
	}
	top := s.objects[len(s.objects)-1]
	s.objects = s.objects[:len(s.objects)-1]
	return top
}

func (s *Stack) IsEmpty() bool {
	return len(s.objects) == 0
}

func (s *Stack) Size() int {
	return len(s.objects)
}

func (s *Stack) Peek() *StackObject {
	if len(s.objects) == 0 {
		return nil
	}
	return s.objects[len(s.objects)-1]
}

// RemoveBySourceID removes a stack object matching the given source ID.
// The countered spell's card goes to its owner's graveyard.
func (s *Stack) RemoveBySourceID(sourceID uuid.UUID) *StackObject {
	for i, obj := range s.objects {
		if obj.SourceID == sourceID {
			s.objects = append(s.objects[:i], s.objects[i+1:]...)
			return obj
		}
	}
	return nil
}

// FindBySourceID finds a stack object by source ID without removing it.
func (s *Stack) FindBySourceID(sourceID uuid.UUID) *StackObject {
	for _, obj := range s.objects {
		if obj.SourceID == sourceID {
			return obj
		}
	}
	return nil
}

// Objects returns all objects on the stack for inspection.
func (s *Stack) Objects() []*StackObject {
	return s.objects
}

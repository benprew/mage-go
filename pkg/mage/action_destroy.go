package mage

import "github.com/google/uuid"

// DestroyPermanentAction represents a permanent about to be destroyed.
type DestroyPermanentAction struct {
	source      uuid.UUID
	permanentID uuid.UUID
}

func NewDestroyPermanentAction(source, permanentID uuid.UUID) *DestroyPermanentAction {
	return &DestroyPermanentAction{
		source:      source,
		permanentID: permanentID,
	}
}

func (a *DestroyPermanentAction) ActionSource() uuid.UUID { return a.source }
func (a *DestroyPermanentAction) PermanentID() uuid.UUID  { return a.permanentID }

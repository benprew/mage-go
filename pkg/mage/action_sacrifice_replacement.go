package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SacrificePermanentAction keeps the chosen object through replacement checks.
type SacrificePermanentAction struct {
	permanentID, incarnation, exileSource uuid.UUID
	destination                           core.Zone
	lockReturnedHand                      bool
}

func (a *SacrificePermanentAction) ActionSource() uuid.UUID { return uuid.Nil }
func (a *SacrificePermanentAction) PermanentID() uuid.UUID  { return a.permanentID }
func (a *SacrificePermanentAction) Destination() core.Zone  { return a.destination }

// WithExile keeps the sacrifice and changes its destination to exile.
func (a *SacrificePermanentAction) WithExile(sourceID uuid.UUID) *SacrificePermanentAction {
	cloned := *a
	cloned.destination, cloned.exileSource = core.ZoneExile, sourceID
	return &cloned
}

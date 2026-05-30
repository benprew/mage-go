package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// AddCountersAction represents counters about to be put on a permanent (CR 614).
// This action flows through the replacement pipeline so that effects like
// "If one or more +1/+1 counters would be placed on a creature you control,
// twice that many +1/+1 counters are placed on it instead." (Doubling Season,
// Branching Evolution) and "Each other Rogue creature you control enters with
// an additional +1/+1 counter on it." (Oona's Blackguard) can intercept it.
//
// OnEntry is true when the counters are being placed as part of the
// permanent's enter-the-battlefield resolution, before EvtEntersBattlefield
// fires. The "additional counters on entry" replacement keys off this flag.
type AddCountersAction struct {
	source      uuid.UUID
	permanentID uuid.UUID
	counterType CounterType
	amount      int
	onEntry     bool
}

// NewAddCountersAction constructs an AddCountersAction.
func NewAddCountersAction(source, permanentID uuid.UUID, ct CounterType, amount int, onEntry bool) *AddCountersAction {
	return &AddCountersAction{
		source:      source,
		permanentID: permanentID,
		counterType: ct,
		amount:      amount,
		onEntry:     onEntry,
	}
}

func (a *AddCountersAction) ActionSource() uuid.UUID  { return a.source }
func (a *AddCountersAction) PermanentID() uuid.UUID   { return a.permanentID }
func (a *AddCountersAction) CounterType() CounterType { return a.counterType }
func (a *AddCountersAction) Amount() int              { return a.amount }
func (a *AddCountersAction) OnEntry() bool            { return a.onEntry }

// WithAmount returns a copy with the amount replaced. Used by replacements
// that scale the count (e.g. doubling).
func (a *AddCountersAction) WithAmount(n int) *AddCountersAction {
	c := *a
	c.amount = n
	return &c
}

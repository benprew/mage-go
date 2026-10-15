package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type retainedEventPlayer struct{ player uuid.UUID }

func (p retainedEventPlayer) CheckTriggerCond(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return event.PlayerID == p.player
}

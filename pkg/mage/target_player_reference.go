package mage

import "github.com/google/uuid"

// TargetControlledBy checks control against the supplied player reference.
func TargetControlledBy(player PlayerRef) PermanentTargetOption {
	return targetControlledByOption{player}
}

type targetControlledByOption struct{ player PlayerRef }

func (targetControlledByOption) permanentTargetOption() {}
func (o targetControlledByOption) applyPermanentTarget(t *PermanentTarget) {
	t.controllerRefs = append(t.controllerRefs, o.player)
}
func (o targetControlledByOption) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-controller-reference", children: []definitionEdge{{"player", o.player}}}
	switch o.player.(type) {
	case controllerRef, savedPlayerRef, combatDefendingPlayerRef, activePlayerRef:
	default:
		n.problem = "target controller: supply You, a saved event player, the combat defending player, or the active player"
	}
	return n
}

func capturedTriggerTargetFacts(pt *pendingTrigger) (players map[*PlayerBinding]uuid.UUID, permanents map[*PermanentBinding]capturedPermanent) {
	// Use the event snapshots without running the ability or its payment.
	obj := &StackObject{}
	captureTriggerEventEffects(pt, obj)
	var collect func(Effect)
	collect = func(effect Effect) {
		switch effect := effect.(type) {
		case *eventPlayerCapture:
			if players == nil {
				players = make(map[*PlayerBinding]uuid.UUID)
			}
			players[effect.binding] = effect.player
		case *eventPermanentCapture:
			if permanents == nil {
				permanents = make(map[*PermanentBinding]capturedPermanent)
			}
			permanents[effect.binding] = effect.value
		case *sequenceEffect:
			for _, child := range effect.effects {
				collect(child)
			}
		}
	}
	for _, effect := range obj.Effects {
		collect(effect)
	}
	return players, permanents
}

func (g *Game) targetPlayerQuery(players map[*PlayerBinding]uuid.UUID) *Game {
	// Target queries must not replace the live resolution frame.
	query := *g
	query.resolution.playerBindings = players
	return &query
}

func (g *Game) targetEventQuery(players map[*PlayerBinding]uuid.UUID, permanents map[*PermanentBinding]capturedPermanent) *Game {
	query := g.targetPlayerQuery(players)
	query.resolution.permanentBindings = permanents
	return query
}

func (g *Game) triggerTargetQuery(pt *pendingTrigger) *Game {
	players, permanents := capturedTriggerTargetFacts(pt)
	return g.targetEventQuery(players, permanents)
}

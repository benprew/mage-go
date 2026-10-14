package mage

import "github.com/google/uuid"

type blockHistoryPair struct{ blocker, attacker capturedPermanent }

func (g *Game) recordBlockHistory(blocker, attacker *Permanent) {
	if blocker == nil || attacker == nil {
		return
	}
	if g.trackers.Turn.blockHistory == nil {
		g.trackers.Turn.blockHistory = make(map[blockHistoryPair]uuid.UUID)
	}
	pair := blockHistoryPair{capturedPermanent{blocker.ID(), blocker.incarnationID}, capturedPermanent{attacker.ID(), attacker.incarnationID}}
	g.trackers.Turn.blockHistory[pair] = attacker.ControllerID()
}

type blockedThisTurnSet struct{ blocker PermanentRef }

// BlockedByThisTurn selects live objects that the exact blocker blocked this turn.
// A saved blocker identity permits history reads after departure or phasing.
func BlockedByThisTurn(blocker PermanentRef) PermanentSet { return blockedThisTurnSet{blocker} }
func (blockedThisTurnSet) permanentSet()                  {}
func (s blockedThisTurnSet) inspectDefinition() definitionNode {
	return permanentInformationNode(s.blocker, "permanent-set", "permanents blocked by this object this turn")
}
func (s blockedThisTurnSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if ctx == nil || ctx.Game == nil || nilDefinition(s.blocker) {
		return nil
	}
	ref, ok := s.blocker.(permanentInformationReference)
	if !ok {
		return nil
	}
	blocker, ok := ref.informationIdentity(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(AnyPermanent) {
		if p.PhasedOut {
			continue
		}
		if ctx.Game.wasBlockedByThisTurn(blocker, p) {
			result = append(result, p)
		}
	}
	return result
}

func (g *Game) wasBlockedByThisTurn(blocker capturedPermanent, attacker *Permanent) bool {
	_, ok := g.trackers.Turn.blockHistory[blockHistoryPair{blocker, capturedPermanent{attacker.ID(), attacker.incarnationID}}]
	return ok
}

type lastBlockedController struct{ attacker, blocker PermanentRef }

// ControllerWhenLastBlockedThisTurn reads the attacker controller at the last block.
// Both references keep their exact object identity after departure.
func ControllerWhenLastBlockedThisTurn(attacker, blocker PermanentRef) PlayerRef {
	return lastBlockedController{attacker, blocker}
}
func (lastBlockedController) playerRef()       {}
func (lastBlockedController) damageRecipient() {}
func (r lastBlockedController) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	if ctx == nil || ctx.Game == nil || nilDefinition(r.attacker) || nilDefinition(r.blocker) {
		return uuid.Nil, false
	}
	attackerRef, ok := r.attacker.(permanentInformationReference)
	if !ok {
		return uuid.Nil, false
	}
	blockerRef, ok := r.blocker.(permanentInformationReference)
	if !ok {
		return uuid.Nil, false
	}
	attacker, ok := attackerRef.informationIdentity(ctx)
	if !ok {
		return uuid.Nil, false
	}
	blocker, ok := blockerRef.informationIdentity(ctx)
	if !ok {
		return uuid.Nil, false
	}
	player, ok := ctx.Game.trackers.Turn.blockHistory[blockHistoryPair{blocker, attacker}]
	return player, ok && player != uuid.Nil
}
func (r lastBlockedController) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r lastBlockedController) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "attacker controller at last block this turn", children: []definitionEdge{
		{"attacker", blockHistoryInformation{r.attacker}}, {"blocker", blockHistoryInformation{r.blocker}},
	}}
}

type blockHistoryInformation struct{ ref PermanentRef }

func (r blockHistoryInformation) inspectDefinition() definitionNode {
	return permanentInformationNode(r.ref, "permanent-information", "blocking history object")
}

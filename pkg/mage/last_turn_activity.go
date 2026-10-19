package mage

import "github.com/google/uuid"

type lastTurnActivity struct {
	castSpell   bool
	putNontoken bool
}

// PlayerCastSpellDuringLastTurn reads the player's last completed turn.
// A spell does not need to resolve to count.
func (g *Game) PlayerCastSpellDuringLastTurn(player uuid.UUID) bool {
	return g.trackers.Turn.lastActivity[player].castSpell
}

// PlayerPutNontokenPermanentDuringLastTurn reads the entry actor's history.
// Later control and zone changes do not change this fact.
func (g *Game) PlayerPutNontokenPermanentDuringLastTurn(player uuid.UUID) bool {
	return g.trackers.Turn.lastActivity[player].putNontoken
}

func (t *TurnTrackers) recordNontokenEntry(player uuid.UUID) {
	if player == uuid.Nil {
		return
	}
	if t.putNontoken == nil {
		t.putNontoken = make(map[uuid.UUID]bool)
	}
	t.putNontoken[player] = true
}

func (t *TurnTrackers) archiveActivity(player uuid.UUID) {
	if t.lastActivity == nil {
		t.lastActivity = make(map[uuid.UUID]lastTurnActivity)
	}
	t.lastActivity[player] = lastTurnActivity{castSpell: t.playerCastSpell[player], putNontoken: t.putNontoken[player]}
	t.putNontoken = nil
}

type lastTurnActivityCondition struct {
	player PlayerRef
	entry  bool
}

// CastSpellDuringLastTurn checks a typed player's last completed turn.
func CastSpellDuringLastTurn(player PlayerRef) StateCondition {
	return lastTurnActivityCondition{player: player}
}

// PutNontokenPermanentDuringLastTurn checks the entry actor's last completed turn.
func PutNontokenPermanentDuringLastTurn(player PlayerRef) StateCondition {
	return lastTurnActivityCondition{player: player, entry: true}
}

func (lastTurnActivityCondition) stateCondition() {}
func (c lastTurnActivityCondition) checkState(g GameReader, ctx StateContext) bool {
	player := resolveStatePlayer(g, ctx, c.player)
	if player == uuid.Nil || g.GetPlayer(player) == nil {
		return false
	}
	if c.entry {
		return g.PlayerPutNontokenPermanentDuringLastTurn(player)
	}
	return g.PlayerCastSpellDuringLastTurn(player)
}
func (c lastTurnActivityCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "player acted during their last completed turn", children: []definitionEdge{{"player", c.player}}}
	switch c.player.(type) {
	case controllerRef, activePlayerRef, combatDefendingPlayerRef:
	default:
		n.problem = "last-turn activity requires You, ActivePlayer, or the combat defending player"
	}
	return n
}

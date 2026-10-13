package mage

import "github.com/google/uuid"

// PlayerDamageRecipients selects players for one simultaneous damage instruction.
func PlayerDamageRecipients(players PlayerSet) DamageRecipientSet {
	return playerDamageRecipients{players}
}

type playerDamageRecipients struct{ players PlayerSet }

func (playerDamageRecipients) damageRecipientSet() {}
func (s playerDamageRecipients) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-recipient-set", children: []definitionEdge{{"players", s.players}}}
}
func (s playerDamageRecipients) resolveDamageRecipients(ctx *EffectContext, amount int, source DamageSource) []DamageAssignment {
	var assignments []DamageAssignment
	seen := make(map[uuid.UUID]bool)
	for _, id := range s.players.resolvePlayers(ctx) {
		if seen[id] || ctx.Game.GetPlayer(id) == nil {
			continue
		}
		seen[id] = true
		assignments = append(assignments, PlayerDamageAssignment(ctx.Game.GetPlayer(id), amount, source))
	}
	return assignments
}

// DamageOutcomesInto saves the actual outcome of one damage instruction.
func DamageOutcomesInto(binding *DamageResultsBinding) DamageResultOption {
	return damageOutcomesOption{binding}
}

type damageOutcomesOption struct{ binding *DamageResultsBinding }

func (damageOutcomesOption) damageResultOption() {}
func (o damageOutcomesOption) applyDamageResult(e *damageFromEffect) {
	e.outcomes = o.binding
	e.outcomesOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// IfDamageDealtToPlayer checks saved outcomes against an exact source and player.
func IfDamageDealtToPlayer(results *DamageResultsBinding, source DamageSourceRef, player PlayerRef, body ResolvingEffect) ResolvingEffect {
	return &damageOutcomeBranch{results, source, player, body}
}

type damageOutcomeBranch struct {
	results *DamageResultsBinding
	source  DamageSourceRef
	player  PlayerRef
	body    ResolvingEffect
}

func (*damageOutcomeBranch) Text() string { return "if the source dealt damage to the player" }
func (e *damageOutcomeBranch) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *damageOutcomeBranch) Apply(ctx *EffectContext) error {
	source := e.source.resolveDamageSource(ctx)
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	for _, outcome := range ctx.Game.resolution.damageResultsBindings[e.results] {
		if outcome.amount > 0 && outcome.playerID == player && outcome.source.SameObject(source) {
			return e.body.Apply(ctx)
		}
	}
	return nil
}
func (e *damageOutcomeBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{
		{"results", damageResultsRead{e.results}}, {"source", damageSourceDefinition{e.source}}, {"player", e.player}, {"then", e.body},
	}}
}
func (e *damageOutcomeBranch) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.body)
}

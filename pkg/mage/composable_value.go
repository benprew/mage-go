package mage

// ResolutionValue reads a number from the current resolution context.
// It cannot be used by legacy instructions that discard binding context.
type ResolutionValue interface {
	Text() string
	resolveValue(*EffectContext) int
}

func (v fixedValue) resolveValue(_ *EffectContext) int { return v.n }

// ValueAtResolution adapts a legacy value without adding typed binding reads.
func ValueAtResolution(value ValueSource) ResolutionValue { return legacyResolutionValue{value} }

type legacyResolutionValue struct{ value ValueSource }

func (v legacyResolutionValue) Text() string {
	if nilDefinition(v.value) {
		return ""
	}
	return v.value.Text()
}
func (v legacyResolutionValue) resolveValue(ctx *EffectContext) int {
	return v.value.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
}
func (v legacyResolutionValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", children: []definitionEdge{{"value", v.value}}}
}

type permanentResolutionValue struct {
	subject   PermanentRef
	toughness bool
}

// PowerOf reads current power. An absent permanent supplies zero.
func PowerOf(subject PermanentRef) ResolutionValue { return permanentResolutionValue{subject: subject} }

// ToughnessOf reads current toughness. An absent permanent supplies zero.
func ToughnessOf(subject PermanentRef) ResolutionValue {
	return permanentResolutionValue{subject: subject, toughness: true}
}
func (v permanentResolutionValue) Text() string {
	if v.toughness {
		return "toughness of subject"
	}
	return "power of subject"
}
func (v permanentResolutionValue) resolveValue(ctx *EffectContext) int {
	p, ok := v.subject.resolvePermanent(ctx)
	if !ok {
		return 0
	}
	if v.toughness {
		return p.CurrentToughness(ctx.Game)
	}
	return p.CurrentPower(ctx.Game)
}
func (v permanentResolutionValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// ManaValueOf reads the permanent's mana value. An absent permanent supplies zero.
func ManaValueOf(subject PermanentRef) ResolutionValue { return permanentManaValue{subject} }

type permanentManaValue struct{ subject PermanentRef }

type playerHandSizeValue struct{ subject PlayerRef }

type playerLifeGainedThisTurnValue struct{ subject PlayerRef }

// LifeGainedThisTurn reads the selected player's actual life gain this turn.
func LifeGainedThisTurn(subject PlayerRef) ResolutionValue {
	return playerLifeGainedThisTurnValue{subject: subject}
}

func (v playerLifeGainedThisTurnValue) Text() string { return "life gained by player this turn" }
func (v playerLifeGainedThisTurnValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.subject.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return 0
	}
	return ctx.Game.PlayerLifeGainedThisTurn(id)
}
func (v playerLifeGainedThisTurnValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// HandSizeOf reads the selected player's current hand size at resolution.
func HandSizeOf(subject PlayerRef) ResolutionValue { return playerHandSizeValue{subject} }

func (v playerHandSizeValue) Text() string { return "hand size of player" }
func (v playerHandSizeValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.subject.resolvePlayer(ctx)
	if !ok {
		return 0
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return 0
	}
	return len(player.Hand())
}
func (v playerHandSizeValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

func (v permanentManaValue) Text() string { return "mana value of subject" }
func (v permanentManaValue) resolveValue(ctx *EffectContext) int {
	p, ok := v.subject.resolvePermanent(ctx)
	if !ok {
		return 0
	}
	return p.Card.ManaCost().CMC()
}
func (v permanentManaValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

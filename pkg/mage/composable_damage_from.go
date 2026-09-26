package mage

import "github.com/google/uuid"

// DamageSourceRef identifies the object that deals damage.
type DamageSourceRef interface {
	damageSourceRef()
	resolveDamageSource(*EffectContext) DamageSource
}

func (sourceRef) damageSourceRef() {}
func (sourceRef) resolveDamageSource(ctx *EffectContext) DamageSource {
	if ctx.sourceCaptured && ctx.sourceIncarnation != uuid.Nil {
		if p := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); p != nil && p.incarnationID == ctx.sourceIncarnation {
			return damageSourceFromPermanent(p)
		}
		if lki := ctx.Game.zones.LKIByIncarnation(ctx.sourceIncarnation); lki != nil && lki.ID == ctx.SourceID {
			return damageSourceFromLKI(lki)
		}
		return DamageSource{}
	}
	return ctx.Game.CaptureDamageSource(ctx.SourceID)
}

type damageFromEffect struct {
	source               DamageSourceRef
	recipient            DamageRecipientRef
	amount               ResolutionValue
	result               *NumberBinding
	playerResult         *PlayerBinding
	permanentResult      *PermanentBinding
	sourceResult         *DamageSourceBinding
	amountOptionCount    int
	playerOptionCount    int
	permanentOptionCount int
	sourceOptionCount    int
	invalidOption        bool
}

// DamageResultOption selects one output for a damage instruction.
type DamageResultOption interface {
	damageResultOption()
	applyDamageResult(*damageFromEffect)
}

type damageAmountOption struct{ binding *NumberBinding }

type damagePlayerOption struct{ binding *PlayerBinding }

type damagePermanentOption struct{ binding *PermanentBinding }
type damageSourceOption struct{ binding *DamageSourceBinding }

// DamageSourceInto saves the source when this instruction deals damage.
func DamageSourceInto(binding *DamageSourceBinding) DamageResultOption {
	return damageSourceOption{binding: binding}
}

func (damageSourceOption) damageResultOption() {}
func (o damageSourceOption) applyDamageResult(e *damageFromEffect) {
	e.sourceResult = o.binding
	e.sourceOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DamagePermanentInto saves the permanent that receives damage after replacements.
func DamagePermanentInto(binding *PermanentBinding) DamageResultOption {
	return damagePermanentOption{binding: binding}
}

func (damagePermanentOption) damageResultOption() {}
func (o damagePermanentOption) applyDamageResult(e *damageFromEffect) {
	e.permanentResult = o.binding
	e.permanentOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DamagePlayerInto saves the player who receives damage after replacements.
func DamagePlayerInto(binding *PlayerBinding) DamageResultOption {
	return damagePlayerOption{binding: binding}
}

func (damagePlayerOption) damageResultOption() {}
func (o damagePlayerOption) applyDamageResult(e *damageFromEffect) {
	e.playerResult = o.binding
	e.playerOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DamageAmountInto saves the damage dealt after replacement and prevention.
func DamageAmountInto(binding *NumberBinding) DamageResultOption {
	return damageAmountOption{binding: binding}
}

func (damageAmountOption) damageResultOption() {}
func (o damageAmountOption) applyDamageResult(e *damageFromEffect) {
	e.result = o.binding
	e.amountOptionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type damageSourceDefinition struct{ source DamageSourceRef }

func (d damageSourceDefinition) inspectDefinition() definitionNode {
	switch source := d.source.(type) {
	case savedDamageSourceRef:
		return source.inspectDefinition()
	case damageResultSource:
		return source.inspectDefinition()
	}
	n := definitionNode{Role: "damage-source-ref", Text: "this"}
	if nilDefinition(d.source) {
		n.problem = "supply a damage source"
	}
	return n
}

// DealDamageFrom deals damage from the source to one player or permanent.
func DealDamageFrom(source DamageSourceRef, recipient DamageRecipientRef, amount ResolutionValue, options ...DamageResultOption) ResolvingEffect {
	e := &damageFromEffect{source: source, recipient: recipient, amount: amount}
	for _, option := range options {
		if option == nil {
			e.invalidOption = true
			continue
		}
		option.applyDamageResult(e)
	}
	return e
}

func (*damageFromEffect) Text() string { return "deal damage" }
func (*damageFromEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *damageFromEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipient", e.recipient}, {"amount", e.amount}}}
	if e.invalidOption {
		n.problem = "supply a damage result binding"
	} else if e.amountOptionCount > 1 || e.playerOptionCount > 1 || e.permanentOptionCount > 1 || e.sourceOptionCount > 1 {
		n.problem = "supply each damage result binding only once"
	}
	if e.result != nil {
		n.bindingWrites = append(n.bindingWrites, e.result)
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	if e.playerResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.playerResult)
		n.children = append(n.children, definitionEdge{"player result", e.playerResult})
	}
	if e.permanentResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.permanentResult)
		n.children = append(n.children, definitionEdge{"permanent result", e.permanentResult})
	}
	if e.sourceResult != nil {
		n.bindingWrites = append(n.bindingWrites, e.sourceResult)
		n.children = append(n.children, definitionEdge{"source result", e.sourceResult})
	}
	return n
}
func (e *damageFromEffect) Apply(ctx *EffectContext) error {
	outcome := damageOutcome{}
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	if e.playerResult != nil {
		delete(ctx.Game.resolution.playerBindings, e.playerResult)
	}
	if e.permanentResult != nil {
		delete(ctx.Game.resolution.permanentBindings, e.permanentResult)
	}
	if e.sourceResult != nil {
		delete(ctx.Game.resolution.damageSourceBindings, e.sourceResult)
	}
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	id, ok := e.recipient.resolveRecipient(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	if player := ctx.Game.GetPlayer(id); player != nil {
		outcome = ctx.Game.damage.dealDamageToPlayerFromSourceOutcome(ctx.Game, player, amount, source)
	} else if permanent := ctx.Game.FindPermanent(id); permanent != nil {
		outcome = ctx.Game.damage.dealDamageToPermanentFromSourceOutcome(ctx.Game, permanent, amount, source)
	}
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = outcome.amount
	}
	if e.playerResult != nil && outcome.playerID != uuid.Nil {
		if ctx.Game.resolution.playerBindings == nil {
			ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
		}
		ctx.Game.resolution.playerBindings[e.playerResult] = outcome.playerID
	}
	if e.permanentResult != nil && outcome.permanent.id != uuid.Nil {
		if ctx.Game.resolution.permanentBindings == nil {
			ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
		}
		ctx.Game.resolution.permanentBindings[e.permanentResult] = outcome.permanent
	}
	if e.sourceResult != nil && outcome.amount > 0 && outcome.source.Valid() {
		if ctx.Game.resolution.damageSourceBindings == nil {
			ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
		}
		ctx.Game.resolution.damageSourceBindings[e.sourceResult] = cloneDamageSource(outcome.source)
	}
	return nil
}

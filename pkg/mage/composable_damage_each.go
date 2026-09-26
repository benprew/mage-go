package mage

// DamageRecipientSet selects players and permanents for one damage instruction.
type DamageRecipientSet interface {
	damageRecipientSet()
	resolveDamageRecipients(*EffectContext, int, DamageSource) []DamageAssignment
}

type mixedDamageRecipients struct {
	players    PlayerSet
	permanents PermanentSet
}

// DamageRecipients combines player and permanent sets for one simultaneous action.
func DamageRecipients(players PlayerSet, permanents PermanentSet) DamageRecipientSet {
	return mixedDamageRecipients{players: players, permanents: permanents}
}

func (mixedDamageRecipients) damageRecipientSet() {}
func (s mixedDamageRecipients) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-recipient-set", children: []definitionEdge{{"players", s.players}, {"permanents", s.permanents}}}
	if nilDefinition(s.players) {
		n.problem = "supply a player set"
	} else if nilDefinition(s.permanents) {
		n.problem = "supply a permanent set"
	} else if targets, ok := s.permanents.(*PermanentTargetsBinding); ok && targets.Max() > 1 {
		n.problem = "multiple permanent targets require recorded target group boundaries"
	}
	return n
}
func (s mixedDamageRecipients) resolveDamageRecipients(ctx *EffectContext, amount int, source DamageSource) []DamageAssignment {
	players := s.players.resolvePlayers(ctx)
	permanents := s.permanents.resolvePermanents(ctx)
	assignments := make([]DamageAssignment, 0, len(players)+len(permanents))
	for _, id := range players {
		if player := ctx.Game.GetPlayer(id); player != nil {
			assignments = append(assignments, PlayerDamageAssignment(player, amount, source))
		}
	}
	for _, permanent := range permanents {
		if permanent != nil && !permanent.PhasedOut {
			assignments = append(assignments, PermanentDamageAssignment(permanent, amount, source))
		}
	}
	return assignments
}

type damageEachEffect struct {
	source        DamageSourceRef
	recipients    DamageRecipientSet
	amount        ResolutionValue
	total         *NumberBinding
	results       *DamageResultsBinding
	optionCount   int
	resultsCount  int
	invalidOption bool
}

// DamageEachResultOption selects an output for a simultaneous damage instruction.
type DamageEachResultOption interface {
	damageEachResultOption()
	applyDamageEachResult(*damageEachEffect)
}

type damageTotalOption struct{ binding *NumberBinding }

// DamageTotalInto saves the total damage dealt by one simultaneous instruction.
func DamageTotalInto(binding *NumberBinding) DamageEachResultOption {
	return damageTotalOption{binding: binding}
}

func (damageTotalOption) damageEachResultOption() {}
func (o damageTotalOption) applyDamageEachResult(e *damageEachEffect) {
	e.total = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

// DealDamageEach deals one amount from one source to each selected recipient.
func DealDamageEach(source DamageSourceRef, recipients DamageRecipientSet, amount ResolutionValue, options ...DamageEachResultOption) ResolvingEffect {
	e := &damageEachEffect{source: source, recipients: recipients, amount: amount}
	for _, option := range options {
		if option == nil {
			e.invalidOption = true
			continue
		}
		option.applyDamageEachResult(e)
	}
	return e
}

func (*damageEachEffect) Text() string { return "deal damage to each recipient" }
func (*damageEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *damageEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{
		{"source", damageSourceDefinition{e.source}}, {"recipients", e.recipients}, {"amount", e.amount},
	}}
	if e.invalidOption {
		n.problem = "supply a damage result binding"
	} else if e.optionCount > 1 || e.resultsCount > 1 {
		n.problem = "supply each damage result binding only once"
	}
	if e.total != nil {
		n.bindingWrites = append(n.bindingWrites, e.total)
		n.children = append(n.children, definitionEdge{"total", e.total})
	}
	if e.results != nil {
		n.bindingWrites = append(n.bindingWrites, e.results)
		n.children = append(n.children, definitionEdge{"results", e.results})
	}
	return n
}
func (e *damageEachEffect) Apply(ctx *EffectContext) error {
	if e.total != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.total] = 0
	}
	if e.results != nil {
		if ctx.Game.resolution.damageResultsBindings == nil {
			ctx.Game.resolution.damageResultsBindings = make(map[*DamageResultsBinding][]damageOutcome)
		}
		delete(ctx.Game.resolution.damageResultsBindings, e.results)
	}
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	assignments := e.recipients.resolveDamageRecipients(ctx, amount, source)
	outcomes := ctx.Game.dealSimultaneousDamageOutcomes(assignments...)
	total := 0
	for _, outcome := range outcomes {
		total += outcome.amount
	}
	if e.total != nil {
		ctx.Game.resolution.numberBindings[e.total] = total
	}
	if e.results != nil {
		ctx.Game.resolution.damageResultsBindings[e.results] = cloneDamageOutcomes(outcomes)
	}
	return nil
}

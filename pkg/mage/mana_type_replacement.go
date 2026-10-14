package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type manaProductionAction struct {
	source, incarnation, player uuid.UUID
	tapping                     bool
	productions                 []ManaProduction
}

func (a *manaProductionAction) ActionSource() uuid.UUID { return a.source }
func (a *manaProductionAction) PlayerID() uuid.UUID     { return a.player }

type tappedManaTypeEffect struct {
	player   PlayerRef
	filter   PermanentFilter
	subject  PermanentRef
	from, to Color
	exact    bool
}

// ReplaceTappedManaTypeForPlayerThisTurn changes all types of mana from matching
// permanents controlled by the saved player when that player taps them for mana.
func ReplaceTappedManaTypeForPlayerThisTurn(player PlayerRef, filter PermanentFilter, to Color) ResolvingEffect {
	return &tappedManaTypeEffect{player: player, filter: filter, to: to}
}

// ReplaceTappedManaTypeUntilLeaves changes one mana type from the subject.
// A phase cycle keeps the rule. Departure ends it permanently.
func ReplaceTappedManaTypeUntilLeaves(subject PermanentRef, from, to Color) ResolvingEffect {
	return &tappedManaTypeEffect{subject: subject, from: from, to: to, exact: true}
}
func (*tappedManaTypeEffect) Text() string                 { return "replace mana type from tapping" }
func (*tappedManaTypeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *tappedManaTypeEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || permanentRefReadsSourcePermanent(e.subject)
}
func (e *tappedManaTypeEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mana-type-replacement", Text: e.Text()}
	if e.exact {
		n.children = []definitionEdge{{"subject", e.subject}}
	} else {
		n.children = []definitionEdge{{"player", e.player}}
	}
	if e.to < Colorless || e.to >= AnyColor || (e.exact && (e.from < Colorless || e.from >= AnyColor)) {
		n.problem = "supply concrete mana types"
	}
	return n
}
func (e *tappedManaTypeEffect) Apply(ctx *EffectContext) error {
	r := &tappedManaTypeReplacement{replacementBase: replacementBase{sourceID: ctx.SourceID, duration: EndOfTurn}, identity: uuid.New(), filter: e.filter, from: e.from, to: e.to, exact: e.exact}
	if e.exact {
		p, ok := e.subject.resolvePermanent(ctx)
		if !ok || p == nil || p.PhasedOut {
			return nil
		}
		r.subject, r.incarnation = p.ID(), p.incarnationID
		r.duration = Indefinite
	} else {
		player, ok := e.player.resolvePlayer(ctx)
		if !ok {
			return nil
		}
		r.player = player
	}
	ctx.Game.effects.AddReplacement(r)
	return nil
}

type tappedManaTypeReplacement struct {
	replacementBase
	identity                     uuid.UUID
	player, subject, incarnation uuid.UUID
	filter                       PermanentFilter
	from, to                     Color
	exact                        bool
}

func (*tappedManaTypeReplacement) survivesSourceExit() bool { return true }
func (r *tappedManaTypeReplacement) IsActive(g GameReader) bool {
	if !r.exact {
		return true
	}
	p := g.FindPermanentIncludingPhased(r.subject)
	return p != nil && p.incarnationID == r.incarnation
}
func (r *tappedManaTypeReplacement) Matches(action Action, g GameReader) bool {
	a, ok := action.(*manaProductionAction)
	if !ok || !a.tapping || len(a.productions) == 0 {
		return false
	}
	p := g.FindPermanent(a.source)
	if p == nil || p.incarnationID != a.incarnation {
		return false
	}
	if r.exact {
		if a.source != r.subject || a.incarnation != r.incarnation {
			return false
		}
		for _, prod := range a.productions {
			if prod.Color == r.from {
				return true
			}
		}
		return false
	}
	game, _ := g.(*Game)
	return a.player == r.player && p.ControllerID() == r.player && r.filter.Match(p, game)
}
func (r *tappedManaTypeReplacement) transformed(a *manaProductionAction) *manaProductionAction {
	result := *a
	result.productions = slices.Clone(a.productions)
	for i := range result.productions {
		if !r.exact || result.productions[i].Color == r.from {
			result.productions[i].Color = r.to
			result.productions[i].AnyCombination = false
			result.productions[i].colorOptions = [AnyColor]bool{}
		}
	}
	return &result
}
func (r *tappedManaTypeReplacement) Replace(action Action, _ *Game) Action {
	return r.transformed(action.(*manaProductionAction))
}
func (r *tappedManaTypeReplacement) Clone() ReplacementEffect { result := *r; return &result }

type manaReplacementOutcome struct {
	productions []ManaProduction
	original    []ManaProduction
	order       []uuid.UUID
}

// Planning reads each possible order without a player choice or a live mutation.
func (g *Game) manaReplacementOutcomes(action *manaProductionAction) []manaReplacementOutcome {
	var outcomes []manaReplacementOutcome
	type projectionState struct {
		colors  string
		applied [2][6][6]int
	}
	visited := make(map[projectionState]bool)
	var visit func(*manaProductionAction, map[ReplacementEffect]bool, []uuid.UUID)
	visit = func(a *manaProductionAction, applied map[ReplacementEffect]bool, order []uuid.UUID) {
		state := projectionState{}
		colors := make([]byte, len(a.productions))
		for i, p := range a.productions {
			colors[i] = byte(p.Color)
		}
		state.colors = string(colors)
		for replacement := range applied {
			if r, ok := replacement.(*tappedManaTypeReplacement); ok {
				kind := 0
				if r.exact {
					kind = 1
				}
				state.applied[kind][r.from][r.to]++
			}
		}
		if visited[state] {
			return
		}
		visited[state] = true
		candidates := g.effects.replacementCandidates(a, g, applied)
		if len(candidates) == 0 {
			for _, outcome := range outcomes {
				if concreteManaProductionsEqual(outcome.productions, a.productions) {
					return
				}
			}
			outcomes = append(outcomes, manaReplacementOutcome{productions: slices.Clone(a.productions), original: slices.Clone(action.productions), order: slices.Clone(order)})
			return
		}
		seen := make(map[[3]int]bool)
		for _, candidate := range candidates {
			r, ok := candidate.(*tappedManaTypeReplacement)
			if !ok {
				continue
			}
			key := [3]int{int(r.from), int(r.to), 0}
			if r.exact {
				key[2] = 1
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			next := maps.Clone(applied)
			next[r] = true
			visit(r.transformed(a), next, append(slices.Clone(order), r.identity))
		}
	}
	visit(action, make(map[ReplacementEffect]bool), nil)
	return outcomes
}
func manaOrderChoice(order []uuid.UUID) func([]ReplacementEffect, Action, *Game) int {
	return func(candidates []ReplacementEffect, _ Action, _ *Game) int {
		for _, r := range order {
			for i, candidate := range candidates {
				if rule, ok := candidate.(*tappedManaTypeReplacement); ok && rule.identity == r {
					return i
				}
			}
		}
		return 0
	}
}
func (g *Game) replaceManaProduction(a *manaProductionAction, order []uuid.UUID) []ManaProduction {
	var result Action
	if order == nil {
		result = g.effects.ApplyReplacements(a, g)
	} else {
		result, _ = g.effects.prepareReplacementsChoosing(a, g, manaOrderChoice(order))
	}
	if production, ok := result.(*manaProductionAction); ok {
		return production.productions
	}
	return nil
}
func (g *Game) effectManaProduction(ctx *EffectContext, productions []ManaProduction) []ManaProduction {
	a := &manaProductionAction{source: ctx.SourceID, incarnation: ctx.sourceIncarnation, player: ctx.Controller, productions: productions}
	var order []uuid.UUID
	objects := g.stack.resolvingReferenceObjects
	if len(objects) > 0 {
		obj := objects[len(objects)-1]
		if obj.SourceID == ctx.SourceID && obj.sourceIncarnation == ctx.sourceIncarnation {
			a.tapping = obj.tappedForMana
			order = obj.manaReplacementOrder
		}
	}
	return g.replaceManaProduction(a, order)
}
func (g *Game) manaProductionVariants(action *manaProductionAction) []manaReplacementOutcome {
	expand := false
	for _, p := range action.productions {
		if p.colorOptions != ([AnyColor]bool{}) {
			expand = true
		}
	}
	for _, list := range [][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			r, ok := replacement.(*tappedManaTypeReplacement)
			if !ok || !r.exact || !r.IsActive(g) {
				continue
			}
			probe := *action
			probe.productions = []ManaProduction{{Color: r.from, Amount: 1}}
			if r.Matches(&probe, g) {
				expand = true
				break
			}
		}
	}
	if !expand {
		return g.manaReplacementOutcomes(action)
	}
	var caps manaChoiceCaps
	total := productionsTotalAmount(action.productions)
	for _, color := range manaPoolColors {
		caps[color] = total
	}
	var result []manaReplacementOutcome
	for _, concrete := range concreteManaProductionsForDemand(action.productions, caps) {
		probe := *action
		probe.productions = concrete
		for _, outcome := range g.manaReplacementOutcomes(&probe) {
			duplicate := false
			for _, existing := range result {
				if concreteManaProductionsEqual(existing.productions, outcome.productions) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				result = append(result, outcome)
			}
		}
	}
	return result
}
func manaPlanningVariants(g *Game, perm *Permanent, ability manaSourceAbility) []manaSourceAbility {
	if ability.Repeatable {
		return []manaSourceAbility{ability}
	}
	outcomes := g.manaProductionVariants(&manaProductionAction{source: perm.ID(), incarnation: perm.incarnationID, player: perm.ControllerID(), tapping: true, productions: ability.Productions})
	variants := make([]manaSourceAbility, len(outcomes))
	for i, outcome := range outcomes {
		variants[i] = ability
		variants[i].Productions = outcome.productions
		variants[i].originalProductions = outcome.original
		variants[i].replacementOrder = outcome.order
	}
	return variants
}

// ManaTypesAbilityCouldProduce includes all possible replacement orders.
// It does not check payment, tap the source, or make a color choice.
func (g *Game) ManaTypesAbilityCouldProduce(ability Ability, sourceID uuid.UUID) []Color {
	return newManaCouldProduceQuery(g).types(ability, sourceID)
}

// WaitingManaReplacementPermanentSources reads exact subjects of active mana rules.
// Phasing keeps a saved subject. Departure ends its duration.
func (g *Game) WaitingManaReplacementPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*tappedManaTypeReplacement)
		if !ok || !rule.exact {
			return nil
		}
		return []capturedPermanent{{rule.subject, rule.incarnation}}
	})
}

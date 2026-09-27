package mage

import "github.com/google/uuid"

type opponentsRef struct{ player PlayerRef }

// OpponentsOf selects the other player in this two-player engine.
// An absent player produces an empty set.
func OpponentsOf(player PlayerRef) PlayerSet { return &opponentsRef{player} }

func (*opponentsRef) playerSet() {}
func (s *opponentsRef) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	id, ok := s.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var opponents []uuid.UUID
	for _, candidate := range EachPlayer.resolvePlayers(ctx) {
		if candidate != id {
			opponents = append(opponents, candidate)
		}
	}
	return opponents
}
func (s *opponentsRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", Text: "opponents of " + InspectDefinition(s.player).Text, children: []definitionEdge{{"player", s.player}}}
}

type loseLifeEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
	lifeLossResultState
}

type lifeLossResultState struct {
	result  *NumberBinding
	count   int
	invalid bool
}

// LifeLossResultOption selects the total life lost from one set instruction.
type LifeLossResultOption interface {
	lifeLossResultOption()
	applyLifeLossResult(*lifeLossResultState)
}

type lifeLostOption struct{ binding *NumberBinding }

// LifeLostInto saves the total life lost by the selected players.
func LifeLostInto(binding *NumberBinding) LifeLossResultOption { return lifeLostOption{binding} }
func (lifeLostOption) lifeLossResultOption()                   {}
func (o lifeLostOption) applyLifeLossResult(e *lifeLossResultState) {
	e.result = o.binding
	e.count++
	if o.binding == nil {
		e.invalid = true
	}
}

// LoseLifeEach makes each selected player lose the specified amount of life.
// The instruction reads the amount once in the current resolution frame.
// It captures the players and amount before it changes life totals.
func LoseLifeEach(subjects PlayerSet, amount ResolutionValue, options ...LifeLossResultOption) ResolvingEffect {
	e := &loseLifeEachEffect{subjects: subjects, amount: amount}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyLifeLossResult(&e.lifeLossResultState)
	}
	return e
}

func (e *loseLifeEachEffect) Text() string {
	amount := ""
	if !nilDefinition(e.amount) {
		amount = e.amount.Text()
	}
	return "lose " + amount + " life for " + InspectDefinition(e.subjects).Text
}
func (*loseLifeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *loseLifeEachEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	ids := e.subjects.resolvePlayers(ctx)
	if len(ids) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	players := make([]Player, 0, len(ids))
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			players = append(players, player)
		}
	}
	lost := ctx.Game.playersLoseLife(players, amount)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = lost
	}
	return nil
}
func (e *loseLifeEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "lose-life-each", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
	if e.invalid {
		n.problem = "supply a non-nil life loss result binding"
	} else if e.count > 1 {
		n.problem = "supply the life loss result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}

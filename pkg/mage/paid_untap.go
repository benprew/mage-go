package mage

import (
	"fmt"
	"math"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// UntapWithManaPayment keeps the chosen objects and their total payment together.
// The player can choose zero objects. All choices precede payment and untapping.
func UntapWithManaPayment(player PlayerRef, filter PermanentFilter, cost core.ManaCost) ResolvingEffect {
	cost.Hybrid = slices.Clone(cost.Hybrid)
	return &paidUntapEffect{player: player, filter: filter, cost: cost}
}

type paidUntapEffect struct {
	player PlayerRef
	filter PermanentFilter
	cost   core.ManaCost
}

func (*paidUntapEffect) Text() string { return "choose permanents, pay mana for each, and untap them" }
func (*paidUntapEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *paidUntapEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "paid-untap", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if e.filter.IsZero() {
		n.problem = "supply a permanent filter"
	}
	if e.cost.HasX || e.cost.XCount != 0 || e.cost.Generic < 0 || e.cost.White < 0 || e.cost.Blue < 0 || e.cost.Black < 0 || e.cost.Red < 0 || e.cost.Green < 0 {
		n.problem = "supply a nonnegative fixed mana cost"
	}
	return n
}
func (e *paidUntapEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	g := ctx.Game
	player := g.GetPlayer(id)
	if player == nil {
		return nil
	}
	candidates := PermanentsControlledBy(e.player, And(IsTapped, e.filter)).resolvePermanents(ctx)
	chosen := make([]capturedPermanent, 0, len(candidates))
	for len(candidates) > 0 && player.ChooseMayAbility(fmt.Sprintf("Choose another permanent and pay %s for each chosen permanent", e.cost.String())) {
		selection := player.ChoosePermanent(candidates, "Choose a permanent to untap", g)
		index := slices.IndexFunc(candidates, func(p *Permanent) bool {
			return selection != nil && p.ID() == selection.ID() && p.incarnationID == selection.incarnationID
		})
		if index < 0 {
			return nil
		}
		p := candidates[index]
		chosen = append(chosen, capturedPermanent{p.ID(), p.incarnationID})
		candidates = slices.Delete(candidates, index, index+1)
	}
	if len(chosen) == 0 {
		return nil
	}
	for _, saved := range chosen {
		p := g.FindPermanent(saved.id)
		if p == nil || p.PhasedOut || p.incarnationID != saved.incarnation || p.ControllerID() != id || !p.Tapped || !e.filter.Match(p, g) {
			return nil
		}
	}
	total := e.cost
	count := len(chosen)
	for _, amount := range []int{total.Generic, total.White, total.Blue, total.Black, total.Red, total.Green, len(total.Hybrid)} {
		if amount > math.MaxInt/count {
			return nil
		}
	}
	total.Generic *= count
	total.White *= count
	total.Blue *= count
	total.Black *= count
	total.Red *= count
	total.Green *= count
	total.Hybrid = nil
	for range count {
		total.Hybrid = append(total.Hybrid, e.cost.Hybrid...)
	}
	// One transaction prevents an incomplete payment from untapping some objects.
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: ctx.SourceID, Controller: id, Costs: []Cost{&ManaCostPayment{MC: total}}})
	paid := err == nil && tx.Commit() == nil
	if !paid {
		return nil
	}
	live := make([]*Permanent, 0, len(chosen))
	for _, saved := range chosen {
		p := g.FindPermanent(saved.id)
		if p != nil && !p.PhasedOut && p.incarnationID == saved.incarnation && p.Tapped {
			live = append(live, p)
		}
	}
	g.untapPermanentsTogether(live)
	return nil
}

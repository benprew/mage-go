package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SpendAnyManaForOneSpellThisTurn permits one optional mana-cost payment.
// Added costs keep their normal requirements. Source departure does not end it.
func SpendAnyManaForOneSpellThisTurn(player PlayerRef) ResolvingEffect {
	return &oneSpellManaInstallation{player: player}
}

type oneSpellManaInstallation struct{ player PlayerRef }

func (*oneSpellManaInstallation) Text() string {
	return "spend any mana for one spell's mana cost this turn"
}
func (*oneSpellManaInstallation) Properties() EffectProperties { return EffectProperties{} }
func (e *oneSpellManaInstallation) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *oneSpellManaInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "one-spell-mana", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}
func (e *oneSpellManaInstallation) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if ok && ctx.Game.GetPlayer(player) != nil {
		ctx.Game.oneSpellManaPermissions = append(ctx.Game.oneSpellManaPermissions, oneSpellManaPermission{id: uuid.New(), player: player})
	}
	return nil
}

type oneSpellManaPermission struct{ id, player uuid.UUID }

func (g *Game) chooseOneSpellMana(player uuid.UUID) uuid.UUID {
	for _, p := range g.oneSpellManaPermissions {
		if p.player == player && g.GetPlayer(player).ChooseMayAbility("Spend mana of any type for this spell's mana cost?") {
			return p.id
		}
	}
	return uuid.Nil
}

func (g *Game) hasOneSpellMana(player, id uuid.UUID) bool {
	return slices.Contains(g.oneSpellManaPermissions, oneSpellManaPermission{id: id, player: player})
}

func (g *Game) consumeOneSpellMana(player, id uuid.UUID) {
	g.oneSpellManaPermissions = slices.DeleteFunc(g.oneSpellManaPermissions, func(p oneSpellManaPermission) bool { return p.id == id && p.player == player })
}

// relaxSpellManaCost changes only the payment demand after costs are locked.
// Generic reductions must not reduce colored symbols through this permission.
func relaxSpellManaCost(total, base core.ManaCost, baseX int, ctx *core.SpellPaymentContext) core.ManaCost {
	for _, part := range []struct {
		total *int
		base  int
	}{
		{&total.White, base.White}, {&total.Blue, base.Blue}, {&total.Black, base.Black}, {&total.Red, base.Red}, {&total.Green, base.Green},
	} {
		amount := min(*part.total, part.base)
		*part.total -= amount
		total.Generic += amount
	}
	// The base symbols precede added symbols in the locked total.
	count := min(len(base.Hybrid), len(total.Hybrid))
	total.Generic += count
	total.Hybrid = slices.Clone(total.Hybrid[count:])
	if ctx != nil && ctx.XSpendingRestricted {
		ctx.RestrictedGeneric = max(0, ctx.RestrictedGeneric-baseX)
	}
	return total
}

func (tx *spellPaymentTransaction) checkOneSpellMana() error {
	if tx.manaPermission != uuid.Nil && !tx.game.hasOneSpellMana(tx.controller, tx.manaPermission) {
		return fmt.Errorf("one-spell mana permission is absent or expired")
	}
	return nil
}

// CanPaySpellManaCost checks the printed mana cost with an announced X.
// It includes unused permissions, but makes no choices and spends no resources.
// Added costs, taxes and reductions require the shared casting transaction.
func (g *Game) CanPaySpellManaCost(player uuid.UUID, card Card, x int) bool {
	if card == nil || x < 0 || g.GetPlayer(player) == nil {
		return false
	}
	cost := resolvedSpellManaCost(card.ManaCost(), x)
	ctx := SpellContextForCard(card)
	ctx.RestrictedGeneric = card.ManaCost().XCount * x
	if g.CanAfford(player, cost, ctx) {
		return true
	}
	for _, permission := range g.oneSpellManaPermissions {
		if permission.player == player {
			cost = relaxSpellManaCost(cost, cost, ctx.RestrictedGeneric, ctx)
			return g.CanAfford(player, cost, ctx)
		}
	}
	return false
}

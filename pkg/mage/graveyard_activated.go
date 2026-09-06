package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// GraveyardActivatedAbility wraps a SimpleActivatedAbility so it is
// activatable while the source card is in its owner's graveyard rather than
// on the battlefield (CR 112.6 — "An ability that can be activated only
// while its source is in a particular zone…").
//
// Use [WithGraveyardActivatedAbility] to attach one to a card. The
// activator pays costs at sorcery speed; targets, if any, are validated
// using the card (not a permanent) as the source.
type GraveyardActivatedAbility struct {
	*SimpleActivatedAbility
}

// WithGraveyardActivatedAbility attaches a graveyard-zone activated
// ability to a card. The ability is sorcery-speed by default and is
// looked up by [Game.ActivateGraveyardAbility]. The cost typically
// includes [ExileSelfFromGraveyardCost] to prevent infinite reuse.
func WithGraveyardActivatedAbility(effect Effect, cost Cost, opts ...AbilityOption) CardOption {
	return func(c *BaseCard) {
		opts = append(opts, WithSorcerySpeed())
		base := NewActivatedAbility(effect, cost, opts...)
		c.AddAbility(&GraveyardActivatedAbility{SimpleActivatedAbility: base})
	}
}

// ExileSelfFromGraveyardCost exiles the source card from its owner's
// graveyard. Used as a cost component for graveyard-zone activated
// abilities (Ghoulcaller's Accomplice: "Exile this card from your graveyard:
// …"). If the card is not currently in a graveyard the cost is unpayable.
func ExileSelfFromGraveyardCost() Cost { return &exileSelfFromGraveyardCost{} }

type exileSelfFromGraveyardCost struct{}

func (c *exileSelfFromGraveyardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.AllPlayers() {
		for _, card := range p.Graveyard() {
			if card.ID() == sourceID {
				return true
			}
		}
	}
	return false
}

func (c *exileSelfFromGraveyardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	for _, p := range g.AllPlayers() {
		for _, card := range p.Graveyard() {
			if card.ID() == sourceID {
				if removed, ok := g.MoveFromGraveyard(p.PlayerID(), sourceID, ZoneExile); ok {
					g.ExileCard(removed, sourceID)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("source card not found in any graveyard")
}

func (c *exileSelfFromGraveyardCost) Text() string { return "Exile this card from your graveyard" }

// ActivateGraveyardAbility activates the indexed graveyard-zone ability of
// the card identified by cardID, owned/controlled by playerID. The card
// must currently be in playerID's graveyard. Costs are paid in order; if
// any cost fails the activation aborts. On success, the ability's effects
// are pushed onto the stack as an activated-ability stack object.
func (g *Game) ActivateGraveyardAbility(playerID, cardID uuid.UUID, abilityIdx int, targets []uuid.UUID) error {
	pl := g.GetPlayer(playerID)
	if pl == nil {
		return ErrPlayerNotFound
	}
	var card Card
	for _, c := range pl.Graveyard() {
		if c.ID() == cardID {
			card = c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("card %s not in %s graveyard", cardID, playerID)
	}

	abilities := card.Abilities()
	if abilityIdx < 0 || abilityIdx >= len(abilities) {
		return fmt.Errorf("invalid ability index %d", abilityIdx)
	}
	gaa, ok := abilities[abilityIdx].(*GraveyardActivatedAbility)
	if !ok {
		return fmt.Errorf("ability %d is not a graveyard-zone activated ability", abilityIdx)
	}
	if !gaa.CanActivate(playerID, g) {
		return fmt.Errorf("cannot activate graveyard ability")
	}

	if gaa.SorcerySpeed() {
		if !g.step.IsMainPhase() {
			return ErrSorcerySpeed
		}
		if g.ActivePlayerObj().PlayerID() != playerID {
			return ErrSorcerySpeed
		}
		if !g.stack.IsEmpty() {
			return ErrSorcerySpeed
		}
	}

	preparedCosts := g.prepareActionCosts(gaa.Costs(), targets, g.resolution.X())
	payment, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller:               playerID,
		SourceID:                 cardID,
		Costs:                    preparedCosts,
		Targets:                  targets,
		XValue:                   g.resolution.X(),
		ApplyActivationReduction: true,
	})
	if err != nil {
		return err
	}
	if err := payment.Commit(); err != nil {
		return err
	}

	gaa.MarkActivated()

	obj := &StackObject{
		ID:         uuid.New(),
		Controller: playerID,
		SourceID:   cardID,
		IsAbility:  true,
		Targets:    targets,
		XValue:     g.resolution.X(),
	}
	obj.Effects = append(obj.Effects, gaa.Effects()...)
	g.pushStack(obj)
	return nil
}

// FindGraveyardActivatableCard returns (cardID, abilityIdx) for the first
// card in playerID's graveyard whose graveyard-zone activated ability
// matches the given name and is currently activatable. Returns false if no
// match. Used by harnesses / AIs to discover legal plays.
func (g *Game) FindGraveyardActivatableCard(playerID uuid.UUID, name string) (uuid.UUID, int, bool) {
	pl := g.GetPlayer(playerID)
	if pl == nil {
		return uuid.Nil, -1, false
	}
	for _, c := range pl.Graveyard() {
		if name != "" && c.Name() != name {
			continue
		}
		for i, a := range c.Abilities() {
			gaa, ok := a.(*GraveyardActivatedAbility)
			if !ok {
				continue
			}
			gaa.source = c.ID()
			gaa.controller = playerID
			if gaa.CanActivate(playerID, g) {
				return c.ID(), i, true
			}
		}
	}
	return uuid.Nil, -1, false
}

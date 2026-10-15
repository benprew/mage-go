package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureEnchantRemoval(t *testing.T) {
	for _, state := range []string{"normal", "phase", "cycle", "return", "expiry", "clone", "other-enchant"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
			options := []CardOption{WithCastTarget(CreatureEnchantTarget()), WithAbility(At(EachUpkeep, GainLifeFor(You, Fixed(1))))}
			if state == "other-enchant" {
				options = append(options, func(c *BaseCard) { c.castTargets = append(c.castTargets, LandEnchantTarget()) })
			}
			card := NewAura("Rule", "", options...)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if !p.HasCreatureEnchantAbility() {
				t.Fatal("missing printed enchant")
			}
			effect := Until(core.EndOfTurn, RemoveCreatureEnchant(This))
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if p.HasCreatureEnchantAbility() {
				t.Fatal("enchant not removed")
			}
			if len(p.RuntimeAbilities) != 1 || len(card.CastTargets()) == 0 {
				t.Fatal("other ability or printed declaration changed")
			}
			if !auraHostIsLegal(card, host, g) {
				t.Fatal("removed enchant still restricts attachment")
			}
			clone := g.Clone()
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.ApplyContinuousEffects()
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "expiry":
				g.effects.RemoveEndOfTurn()
				g.effects.Apply(g)
			case "clone":
				clone.effects.RemoveEndOfTurn()
				clone.effects.Apply(clone)
			case "other-enchant":
				creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
				if auraHostIsLegal(card, creature, g) {
					t.Fatal("other enchant was removed")
				}
			}
			if state == "phase" {
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("phased visit remains present")
				}
				return
			}
			p = g.FindPermanent(p.ID())
			if state == "return" || state == "expiry" {
				if !p.HasCreatureEnchantAbility() {
					t.Fatal("enchant did not return")
				}
			} else if p.HasCreatureEnchantAbility() {
				t.Fatal("removal ended early")
			}
			if state == "clone" && !clone.FindPermanent(p.ID()).HasCreatureEnchantAbility() {
				t.Fatal("clone expiry failed")
			}
		})
	}
}

func TestCreatureEnchantRemovalDeclarations(t *testing.T) {
	for _, subject := range []PermanentRef{nil, From(BindPermanent("missing"))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Until(core.Indefinite, RemoveCreatureEnchant(subject)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid subject accepted")
		}
	}
}

func TestCreatureEnchantRemovalAttachmentLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
	card := NewAura("Aura", "", WithCastTarget(CreatureEnchantTarget()))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(p.ID(), land.ID())
	if p.AttachedTo == land.ID() {
		t.Fatal("printed enchant permitted a land")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveCreatureEnchant(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.Attach(p.ID(), land.ID())
	if p.AttachedTo != land.ID() {
		t.Fatal("removed enchant prevented attachment")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(p.ID()) == nil {
		t.Fatal("state action ignored removal")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.CheckStateBasedActions()
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("restored enchant did not prohibit land")
	}
}

func TestCreatureEnchantRemovalWaitingLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
	recipient := g.PutOnBattlefield(NewAura("Recipient", "", WithCastTarget(CreatureEnchantTarget())), a.PlayerID())
	donorCard := NewAura("Donor", "", WithCastTarget(TargetPermanent(AnyPermanent)))
	donorCard.SetOwner(a.PlayerID())
	donor := g.PutOnBattlefield(donorCard, a.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	delayed := AtNext(EachEndStep, Until(core.Indefinite, RemoveCreatureEnchant(HostOfPermanent(This))))
	if err := ApplyEffect(g, delayed, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(donor)
	for range 2 {
		found := false
		for _, source := range g.WaitingDelayedPermanentSources() {
			if source.ID() == donor.ID() {
				found = true
			}
		}
		if !found || !g.FindPermanent(recipient.ID()).HasCreatureEnchantAbility() {
			t.Fatal("waiting query lost source or executed body")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(recipient.ID()).HasCreatureEnchantAbility() || !auraHostIsLegal(recipient.Card, host, g) {
		t.Fatal("saved attachment did not supply recipient")
	}
}

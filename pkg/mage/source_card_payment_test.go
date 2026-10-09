package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceCardPayment(t *testing.T) {
	for _, payment := range []string{"sacrifice", "exile"} {
		for _, state := range []string{"normal", "returned", "missing", "token", "copy", "failed", "phased", "replacement"} {
			t.Run(payment+"/"+state, func(t *testing.T) {
				g, a, b := anteEffectTestGame(t)
				paid := BindCard("paid source")
				cost := SacrificeSourceCostWithCard(paid)
				if payment == "exile" {
					cost = ExileSourceCostWithCard(paid)
				}
				target := DeclareAnteCardTarget("ante")
				card := NewArtifact("source", "", WithAbility(Activate(Costs(ManaCostOf("{1}"), cost), ExchangeCardOwnership(FromCard(paid), target), WithTargets(target))))
				if state == "token" {
					card.isToken = true
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				other := ownedTestCard(b.PlayerID(), "other")
				b.AddToAnte(other)
				if state != "failed" {
					a.ManaPool().Add(core.Colorless, 1)
				}
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				if state == "replacement" {
					g.AddReplacementEffect(&requiredSacrificeSourceReplacement{replacementBase: replacementBase{sourceID: p.ID(), duration: core.Indefinite}})
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{other.ID()}); state == "failed" || state == "phased" {
					if err == nil || g.StackSize() != 0 || g.FindCardAnywhere(other.ID()).Owner() != b.PlayerID() {
						t.Fatal("failed payment changed state", err)
					}
					return
				} else if err != nil {
					t.Fatal(err)
				}
				if g.FindCardAnywhere(other.ID()).Owner() != b.PlayerID() {
					t.Fatal("exchange ran before resolution")
				}
				if state == "returned" || state == "missing" {
					c := g.FindCardAnywhere(card.ID())
					if g.FindExiledCard(card.ID()) != nil {
						g.RemoveFromExile(card.ID())
						if state == "returned" {
							g.ExileCard(c, a.PlayerID())
						}
					} else {
						a.RemoveFromGraveyard(card.ID())
						if state == "returned" {
							a.AddToGraveyard(c)
						}
					}
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				g.ResolveStack()
				swapped := state == "normal" || state == "replacement"
				if (g.FindCardAnywhere(other.ID()).Owner() == a.PlayerID()) != swapped {
					t.Fatal("wrong exact visit or copied exchange")
				}
				if clone.FindCardAnywhere(other.ID()).Owner() != b.PlayerID() {
					t.Fatal("clone shared owners")
				}
				clone.ResolveStack()
				if (clone.FindCardAnywhere(other.ID()).Owner() == a.PlayerID()) != swapped {
					t.Fatal("clone lost result")
				}
			})
		}
	}
}

func TestSourceCardPaymentDeclarations(t *testing.T) {
	for _, cost := range []Cost{SacrificeSourceCostWithCard(nil), ExileSourceCostWithCard(nil), SacrificeSourceCostWithCard(BindCard(""))} {
		card := NewArtifact("invalid", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition("invalid", card) == nil {
			t.Fatal("accepted invalid result")
		}
	}
	result := BindCard("source")
	card := NewArtifact("invalid", "", WithAbility(Activate(Costs(SacrificeSourceCostWithCard(result), ExileSourceCostWithCard(result)), GainLifeFor(You, Fixed(1)))))
	if ValidateCardDefinition("invalid", card) == nil {
		t.Fatal("accepted duplicate result")
	}
}

func TestSourceCardPaymentStaleLock(t *testing.T) {
	for _, state := range []string{"returned", "phased", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := anteEffectTestGame(t)
			card := ownedTestCard(a.PlayerID(), "source")
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			result := BindCard("source result")
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Mana: core.ParseManaCost("{1}"), Costs: []Cost{SacrificeSourceCostWithCard(result)}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(source.ID())
			case "control":
				if err := Until(core.EndOfTurn, ChangeController(This, You)).Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if err := tx.Commit(); err == nil || a.ManaPool().TotalMana() != 1 || len(tx.paidCards) != 0 {
				t.Fatal("stale source spent resources", err)
			}
		})
	}
}

func TestSourceCardPaymentEventIdentity(t *testing.T) {
	g, a, _ := anteEffectTestGame(t)
	card := ownedTestCard(a.PlayerID(), "source")
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindCard("source")
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == card.ID() {
			c, ok := a.RemoveFromGraveyard(card.ID())
			if !ok {
				t.Fatal("missing first visit")
			}
			a.AddToGraveyard(c)
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("watcher", "", WithAbility(watcher)), a.PlayerID())
	identity, err := SacrificeSourceCostWithCard(result).(*sourceCardCost).payCard(p.ID(), a.PlayerID(), g)
	if err != nil {
		t.Fatal(err)
	}
	g.resolution.cardBindings = map[*CardBinding]capturedCard{result: identity}
	if _, ok := FromCard(result).resolveCard(&EffectContext{Game: g, Controller: a.PlayerID()}); ok {
		t.Fatal("event replaced saved first visit")
	}
}

func TestSourceCardPaymentExilePermission(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	result := BindCard("source")
	target := DeclareAnteCardTarget("ante")
	action := Activate(Costs(ExileSourceCostWithCard(result)), ExchangeCardOwnership(FromCard(result), target), WithTargets(target))
	WithAnyPlayerMay()(action)
	source := NewArtifact("source", "", WithAbility(action))
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	other := ownedTestCard(b.PlayerID(), "other")
	b.AddToAnte(other)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindExiledCard(source.ID()).Card.Owner() != b.PlayerID() || g.FindCardAnywhere(other.ID()).Owner() != a.PlayerID() {
		t.Fatal("permitted payer lost source reference")
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func handTargetFixture(t *testing.T) (*Game, *BasePlayer, *BasePlayer, *BaseCard, *StackObject) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Hand target", "{R}", 1, 1)
	a.AddToHand(card)
	keeper := NewArtifact("Keeper", "", WithAbility(NewActivated(GenericCost(0), GainLifeFor(You, Fixed(1)), WithTarget(TargetCardInHand()))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	return g, a, b, card, g.StackPeek()
}

func TestHandTargetSourcesIdentity(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, card, object := handTargetFixture(t)
			saved := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
			if got := g.StackHandSourcesFor(a.PlayerID()); len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("announced hand target missing")
			}
			if got := g.StackHandSourcesFor(b.PlayerID())[0]; got.Name() != "" || got.HasColor(core.Red) {
				t.Fatal("private target revealed")
			}
			clone := g.Clone()
			card.AddType(core.TypeArtifact)
			if clone.StackHandSourcesFor(a.PlayerID())[0].HasType(core.TypeArtifact) {
				t.Fatal("clone shares live facts")
			}
			if state != "live" {
				a.RemoveFromHand(card.ID())
			}
			if state == "returned" {
				card.SetColorOverride([]core.Color{core.Blue})
				a.AddToHand(card)
			}
			copied := g.CopyStackObjectDirect(object, a.PlayerID(), false)
			if copied == nil {
				t.Fatal("copy failed")
			}
			sources := g.StackHandSourcesFor(a.PlayerID())
			if len(sources) != 1 || !sources[0].SameObject(saved) || !sources[0].HasColor(core.Red) || !sources[0].HasType(core.TypeArtifact) {
				t.Fatal("target followed new visit or lost facts")
			}
			sources[0].card.AddType(core.TypeLand)
			if g.StackHandSourcesFor(a.PlayerID())[0].HasType(core.TypeLand) {
				t.Fatal("query changed saved facts")
			}
			after := g.Clone()
			g.GetStack().RemoveByID(object.ID)
			if len(g.StackHandSources()) != 1 {
				t.Fatal("copy lost reference")
			}
			g.GetStack().RemoveByID(copied.ID)
			if len(g.StackHandSources()) != 0 || len(after.StackHandSources()) != 1 {
				t.Fatal("final reference lifetime crossed clone")
			}
		})
	}
}

func TestHandTargetSourcesLegalityAndRetarget(t *testing.T) {
	for _, selection := range []string{"same visit", "new visit", "other card", "cancel"} {
		t.Run(selection, func(t *testing.T) {
			g, a, _, card, original := handTargetFixture(t)
			saved := g.StackHandSourcesFor(a.PlayerID())
			if len(saved) != 1 {
				t.Fatal("target missing")
			}
			chooser := newRecordingPlayer("Copy controller")
			chooser.BasePlayer = a
			g.players[0] = chooser
			if selection != "same visit" {
				a.RemoveFromHand(card.ID())
				a.AddToHand(card)
			}
			other := NewCreature("Other hand target", "{U}", 1, 1)
			a.AddToHand(other)
			chosen := card.ID()
			if selection == "other card" {
				chosen = other.ID()
			}
			chooser.chooseQueue = [][]uuid.UUID{{chosen}}
			if selection == "cancel" {
				chooser.chooseQueue = [][]uuid.UUID{nil}
			}
			copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
			if copied == nil {
				t.Fatal("copy failed")
			}
			g.GetStack().RemoveByID(original.ID)
			got := g.StackHandSourcesFor(a.PlayerID())
			if len(got) != 1 {
				t.Fatal("copy reference missing")
			}
			if selection == "cancel" || selection == "same visit" {
				if !got[0].SameObject(saved[0]) {
					t.Fatal("kept target changed visit")
				}
			} else if got[0].SameObject(saved[0]) || got[0].ID() != chosen {
				t.Fatal("retarget retained old visit")
			}
			g.ResolveTopOfStack()
			want := 21
			if selection == "cancel" {
				want = 20
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
	g, a, _, card, _ := handTargetFixture(t)
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	g.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("returned target remained legal")
	}
}

func TestHandTargetSourcesBoundaries(t *testing.T) {
	g, a, b, card, _ := handTargetFixture(t)
	g.GetStack().Pop()
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID(), card.ID(), b.PlayerID(), uuid.New()}})
	if len(g.StackHandSources()) != 1 {
		t.Fatal("target deduplication failed")
	}
	g.GetStack().Pop()
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID()}, targetIncarnations: map[uuid.UUID]uuid.UUID{card.ID(): uuid.New()}})
	if len(g.StackHandSources()) != 0 {
		t.Fatal("wrong visit accepted")
	}
	g.GetStack().Pop()
	a.RemoveFromHand(card.ID())
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID()}})
	if len(g.StackHandSources()) != 0 {
		t.Fatal("history created an unannounced target")
	}
}

func TestHandTargetSourcesVisibility(t *testing.T) {
	for _, public := range []bool{false, true} {
		t.Run(map[bool]string{false: "private", true: "public"}[public], func(t *testing.T) {
			g, a, b, card, _ := handTargetFixture(t)
			if public {
				g.PutOnBattlefield(NewEnchantment("Public hands", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
			}
			a.RemoveFromHand(card.ID())
			if !public {
				g.PutOnBattlefield(NewEnchantment("Later public hands", "", WithAbility(Static(RevealAllHands()))), a.PlayerID())
			}
			source := g.StackHandSourcesFor(b.PlayerID())[0]
			if source.HasColor(core.Red) != public {
				t.Fatal("departure visibility followed later reveal")
			}
			binding := BindDamageSource("target source")
			choice := ChooseDamageSourceFromInspectableStackHandReferences(You, DamageSourceHasColor(core.Red), binding)
			if err := choice.Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if g.resolution.damageSourceBindings[binding].Valid() != public {
				t.Fatal("choice bypassed visibility")
			}
			g.GetStack().Pop()
			if err := choice.Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if g.resolution.damageSourceBindings[binding].Valid() {
				t.Fatal("empty query kept old binding")
			}
		})
	}
}

func TestHandTargetSourcesResolutionAndClone(t *testing.T) {
	g, a, _, card, object := handTargetFixture(t)
	runs := 0
	object.Effects = []Effect{FuncEffect("Inspect resolving target", EffectProperties{}, func(game *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		runs++
		if got := game.StackHandSourcesFor(controller); len(got) != 1 || got[0].ID() != card.ID() {
			t.Fatal("resolution lost target frame")
		}
		clone := game.Clone()
		if len(clone.StackHandSourcesFor(controller)) != 1 {
			t.Fatal("clone lost resolving frame")
		}
		a.RemoveFromHand(card.ID())
		if len(game.StackHandSourcesFor(controller)) != 1 {
			t.Fatal("departure lost resolving reference")
		}
		return nil
	})}
	g.ResolveTopOfStack()
	if runs != 1 || len(g.StackHandSources()) != 0 {
		t.Fatal("resolution frame lifetime failed")
	}
}

func TestHandTargetSourcesSpellCopy(t *testing.T) {
	g, a, _, card, _ := handTargetFixture(t)
	g.GetStack().Pop()
	spell := NewInstant("Target hand spell", "", NewTargetedSpell(TargetCardInHand(), GainLifeFor(You, Fixed(1))))
	a.AddToHand(spell)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{card.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	saved := g.StackHandSourcesFor(a.PlayerID())[0]
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	chooser := newRecordingPlayer("Spell copy controller")
	chooser.BasePlayer = a
	chooser.chooseQueue = [][]uuid.UUID{{card.ID()}}
	g.players[0] = chooser
	clone := g.Clone()
	copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if copied == nil {
		t.Fatal("spell copy failed")
	}
	g.GetStack().RemoveByID(original.ID)
	if got := g.StackHandSourcesFor(a.PlayerID()); len(got) != 1 || got[0].SameObject(saved) {
		t.Fatal("spell copy did not select new visit")
	}
	if got := clone.StackHandSourcesFor(a.PlayerID()); len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("retarget changed clone")
	}
	g.ResolveTopOfStack()
	if a.Life() != 21 {
		t.Fatal("retargeted spell failed resolution")
	}
}

func TestHandTargetSourcesPaymentDeparture(t *testing.T) {
	for _, kind := range []string{"ability", "spell", "alternate spell"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := NewCreature("Paid target", "{R}", 1, 1)
			a.AddToHand(card)
			saved := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
			if kind == "ability" {
				keeper := g.PutOnBattlefield(NewArtifact("Discard keeper", "", WithAbility(NewActivated(DiscardCost(1), GainLifeFor(You, Fixed(1)), WithTarget(TargetCardInHand())))), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, []uuid.UUID{card.ID()}); err != nil {
					t.Fatal(err)
				}
			} else {
				spell := NewInstant("Discard spell", "", NewTargetedSpell(TargetCardInHand(), GainLifeFor(You, Fixed(1))), WithAdditionalCost(DiscardCost(1)))
				if kind == "alternate spell" {
					a.AddToGraveyard(spell)
					if err := g.CastCardFromZoneWithoutPaying(a.PlayerID(), spell.ID(), core.ZoneGraveyard, []uuid.UUID{card.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					a.AddToHand(spell)
					if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{card.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				}
			}
			got := g.StackHandSourcesFor(a.PlayerID())
			if len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("payment erased announced target visit")
			}
			if len(g.StackGraveyardSources()) != 0 {
				t.Fatal("hand target became a graveyard reference during payment")
			}
			if len(a.Hand()) != 0 || len(a.Graveyard()) != 1 {
				t.Fatal("discard payment failed")
			}
			a.RemoveFromGraveyard(card.ID())
			a.AddToHand(card)
			g.ResolveTopOfStack()
			if a.Life() != 20 {
				t.Fatal("payment target followed return")
			}
		})
	}
}

func TestHandTargetSourcesFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Unpaid target", "{R}", 1, 1)
	a.AddToHand(card)
	saved := g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID()))
	keeper := g.PutOnBattlefield(NewArtifact("Unpaid keeper", "", WithAbility(NewActivated(GenericCost(1), WithCost(DiscardCost(1)), GainLifeFor(You, Fixed(1)), WithTarget(TargetCardInHand())))), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, []uuid.UUID{card.ID()}); err == nil {
		t.Fatal("unpaid targeted ability succeeded")
	}
	if g.StackSize() != 0 || len(g.StackHandSources()) != 0 || len(a.Hand()) != 1 || !g.CaptureHandDamageSourceFor(a.PlayerID(), a.handVisit(card.ID())).SameObject(saved) {
		t.Fatal("failed payment published a frame or changed visit")
	}
}

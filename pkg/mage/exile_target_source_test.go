package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func exileTargetFixture(t *testing.T) (*Game, *BasePlayer, *BasePlayer, *BaseCard, *StackObject) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Exile target", "{R}", 1, 1)
	g.ExileCard(card, uuid.Nil)
	keeper := NewArtifact("Keeper", "", WithAbility(NewActivated(GenericCost(0), GainLifeFor(You, Fixed(1)), WithTarget(&exileReferenceTestTarget{BaseTarget: BaseTarget{min: 1, max: 1}}))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, []uuid.UUID{card.ID()}); err != nil {
		t.Fatal(err)
	}
	return g, a, b, card, g.StackPeek()
}

func TestExileTargetSourcesIdentity(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned"} {
		t.Run(state, func(t *testing.T) {
			g, a, _, card, object := exileTargetFixture(t)
			saved := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
			if got := g.StackBoundExileSources(); len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("announced exile target missing")
			}
			clone := g.Clone()
			card.AddType(core.TypeArtifact)
			if clone.StackBoundExileSources()[0].HasType(core.TypeArtifact) {
				t.Fatal("clone shares live facts")
			}
			if state != "live" {
				g.RemoveFromExile(card.ID())
			}
			if state == "returned" {
				card.SetColorOverride([]core.Color{core.Blue})
				g.ExileCard(card, uuid.Nil)
			}
			copied := g.CopyStackObjectDirect(object, a.PlayerID(), false)
			if copied == nil {
				t.Fatal("copy failed")
			}
			sources := g.StackBoundExileSources()
			if len(sources) != 1 || !sources[0].SameObject(saved) || !sources[0].HasColor(core.Red) || !sources[0].HasType(core.TypeArtifact) {
				t.Fatal("target followed new visit or lost facts")
			}
			sources[0].card.AddType(core.TypeLand)
			if g.StackBoundExileSources()[0].HasType(core.TypeLand) {
				t.Fatal("query changed saved facts")
			}
			after := g.Clone()
			g.GetStack().RemoveByID(object.ID)
			if len(g.StackBoundExileSources()) != 1 {
				t.Fatal("copy lost reference")
			}
			g.GetStack().RemoveByID(copied.ID)
			if len(g.StackBoundExileSources()) != 0 || len(after.StackBoundExileSources()) != 1 {
				t.Fatal("final reference lifetime crossed clone")
			}
		})
	}
}

func TestExileTargetSourcesLegalityAndRetarget(t *testing.T) {
	for _, selection := range []string{"same visit", "new visit", "other card", "cancel"} {
		t.Run(selection, func(t *testing.T) {
			g, a, _, card, original := exileTargetFixture(t)
			saved := g.StackBoundExileSources()
			if len(saved) != 1 {
				t.Fatal("target missing")
			}
			chooser := newRecordingPlayer("Copy controller")
			chooser.BasePlayer = a
			g.players[0] = chooser
			if selection != "same visit" {
				g.RemoveFromExile(card.ID())
				g.ExileCard(card, uuid.Nil)
			}
			other := NewCreature("Other exile target", "{U}", 1, 1)
			g.ExileCard(other, uuid.Nil)
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
			got := g.StackBoundExileSources()
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
	g, a, _, card, _ := exileTargetFixture(t)
	g.RemoveFromExile(card.ID())
	g.ExileCard(card, uuid.Nil)
	g.ResolveTopOfStack()
	if a.Life() != 20 {
		t.Fatal("returned target remained legal")
	}
}

func TestExileTargetSourcesBoundaries(t *testing.T) {
	g, _, b, card, _ := exileTargetFixture(t)
	g.GetStack().Pop()
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID(), card.ID(), b.PlayerID(), uuid.New()}})
	if len(g.StackBoundExileSources()) != 1 {
		t.Fatal("target deduplication failed")
	}
	g.GetStack().Pop()
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID()}, targetIncarnations: map[uuid.UUID]uuid.UUID{card.ID(): uuid.New()}})
	if len(g.StackBoundExileSources()) != 0 {
		t.Fatal("wrong visit accepted")
	}
	g.GetStack().Pop()
	g.RemoveFromExile(card.ID())
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID()}})
	if len(g.StackBoundExileSources()) != 0 {
		t.Fatal("history created an unannounced target")
	}
}

func TestExileTargetSourcesResolutionAndClone(t *testing.T) {
	g, _, _, card, object := exileTargetFixture(t)
	runs := 0
	object.Effects = []Effect{FuncEffect("Inspect resolving target", EffectProperties{}, func(game *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		runs++
		if got := game.StackBoundExileSources(); len(got) != 1 || got[0].ID() != card.ID() {
			t.Fatal("resolution lost target frame")
		}
		clone := game.Clone()
		if len(clone.StackBoundExileSources()) != 1 {
			t.Fatal("clone lost resolving frame")
		}
		game.RemoveFromExile(card.ID())
		if len(game.StackBoundExileSources()) != 1 {
			t.Fatal("departure lost resolving reference")
		}
		return nil
	})}
	g.ResolveTopOfStack()
	if runs != 1 || len(g.StackBoundExileSources()) != 0 {
		t.Fatal("resolution frame lifetime failed")
	}
}

func TestExileTargetSourcesSpellCopy(t *testing.T) {
	g, a, _, card, _ := exileTargetFixture(t)
	g.GetStack().Pop()
	spell := NewInstant("Target exile spell", "", NewTargetedSpell(&exileReferenceTestTarget{BaseTarget: BaseTarget{min: 1, max: 1}}, GainLifeFor(You, Fixed(1))))
	a.AddToHand(spell)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{card.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	if len(g.StackBoundExileSources()) != 1 {
		t.Fatal("exile target missing")
	}
	saved := g.StackBoundExileSources()[0]
	g.RemoveFromExile(card.ID())
	g.ExileCard(card, uuid.Nil)
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
	if got := g.StackBoundExileSources(); len(got) != 1 || got[0].SameObject(saved) {
		t.Fatal("spell copy did not select new visit")
	}
	if got := clone.StackBoundExileSources(); len(got) != 1 || !got[0].SameObject(saved) {
		t.Fatal("retarget changed clone")
	}
	g.ResolveTopOfStack()
	if a.Life() != 21 {
		t.Fatal("retargeted spell failed resolution")
	}
}

type exileReferenceTestTarget struct{ BaseTarget }

func (t *exileReferenceTestTarget) Possible(_ uuid.UUID, _ Card, g *Game) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(g.GetExile()))
	for _, object := range g.GetExile() {
		result = append(result, object.Card.ID())
	}
	return result
}
func (t *exileReferenceTestTarget) Choose(_ uuid.UUID, _ Card, _ *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

func TestExileTargetSourcesHidden(t *testing.T) {
	g, a, _, card, _ := exileTargetFixture(t)
	g.GetStack().Pop()
	g.RemoveFromExile(card.ID())
	g.ExileCardFaceDown(card, uuid.Nil, a.PlayerID())
	g.PushStack(&StackObject{ID: uuid.New(), Targets: []uuid.UUID{card.ID()}})
	got := g.StackBoundExileSources()
	if len(got) != 1 || got[0].Name() != "" || got[0].HasColor(core.Red) {
		t.Fatal("face-down target revealed")
	}
	g.RemoveFromExile(card.ID())
	g.ExileCard(card, uuid.Nil)
	got = g.StackBoundExileSources()
	if len(got) != 1 || got[0].Name() != "" || got[0].HasColor(core.Red) {
		t.Fatal("returned face-up card revealed former visit")
	}
}

type exileTargetDepartureCost struct{ id uuid.UUID }

func (c *exileTargetDepartureCost) Text() string { return "Return an exile target to hand" }
func (c *exileTargetDepartureCost) CanPay(_, _ uuid.UUID, g *Game) bool {
	return g.FindExiledCard(c.id) != nil
}
func (c *exileTargetDepartureCost) Pay(_, controller uuid.UUID, g *Game) error {
	card := g.FindExiledCard(c.id).Card
	g.RemoveFromExile(c.id)
	g.GetPlayer(controller).AddToHand(card)
	return nil
}

func TestExileTargetSourcesPaymentDeparture(t *testing.T) {
	for _, kind := range []string{"ability", "spell", "alternate spell", "unpaid"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := NewCreature("Payment target", "{R}", 1, 1)
			g.ExileCard(card, uuid.Nil)
			saved := g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit)
			target := &exileReferenceTestTarget{BaseTarget: BaseTarget{min: 1, max: 1}}
			cost := &exileTargetDepartureCost{id: card.ID()}
			var err error
			if kind == "ability" || kind == "unpaid" {
				mana := 0
				if kind == "unpaid" {
					mana = 1
				}
				keeper := g.PutOnBattlefield(NewArtifact("Payment keeper", "", WithAbility(NewActivated(GenericCost(mana), WithCost(cost), GainLifeFor(You, Fixed(1)), WithTarget(target)))), a.PlayerID())
				err = g.ActivateAbilityByIndex(a.PlayerID(), keeper.ID(), 0, []uuid.UUID{card.ID()})
			} else {
				spell := NewInstant("Payment spell", "", NewTargetedSpell(target, GainLifeFor(You, Fixed(1))), WithAdditionalCost(cost))
				if kind == "spell" {
					a.AddToHand(spell)
					err = g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{card.ID()}, 0)
				} else {
					a.AddToGraveyard(spell)
					err = g.CastCardFromZoneWithoutPaying(a.PlayerID(), spell.ID(), core.ZoneGraveyard, []uuid.UUID{card.ID()}, 0)
				}
			}
			if kind == "unpaid" {
				if err == nil || g.StackSize() != 0 || len(g.StackBoundExileSources()) != 0 || len(a.Hand()) != 0 || !g.CaptureExiledDamageSource(g.FindExiledCard(card.ID()).Visit).SameObject(saved) {
					t.Fatal("failed payment changed target or published reference", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := g.StackBoundExileSources()
			if len(got) != 1 || !got[0].SameObject(saved) || len(g.StackHandSources()) != 0 || len(g.StackGraveyardSources()) != 0 {
				t.Fatal("payment changed target zone or visit")
			}
			a.RemoveFromHand(card.ID())
			g.ExileCard(card, uuid.Nil)
			g.ResolveTopOfStack()
			if a.Life() != 20 {
				t.Fatal("payment target followed return")
			}
		})
	}
}

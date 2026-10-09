package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnteReferenceVisits(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(b.PlayerID(), "second")
	a.SetLibrary([]Card{first})
	b.SetLibrary([]Card{second})
	g, err := NewGameWithAnte(a, b, []Card{first}, []Card{second})
	if err != nil {
		t.Fatal(err)
	}
	visit := g.anteVisit(first.ID())
	if visit == uuid.Nil || visit == g.anteVisit(second.ID()) {
		t.Fatal("missing separate ante visits")
	}
	for range 2 {
		if g.anteVisit(first.ID()) != visit {
			t.Fatal("query changed visit")
		}
	}
	clone := g.Clone()
	if clone.anteVisit(first.ID()) != visit {
		t.Fatal("clone lost visit")
	}
	if err := g.ChangeOwner(first.ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if g.anteVisit(first.ID()) != visit || clone.FindCardAnywhere(first.ID()).Owner() != a.PlayerID() {
		t.Fatal("ownership changed visit or clone")
	}
	removed, err := g.RemoveFromAnte(first.ID(), core.ZoneHand)
	if err != nil {
		t.Fatal(err)
	}
	if g.anteVisit(first.ID()) != uuid.Nil || clone.anteVisit(first.ID()) != visit {
		t.Fatal("departure changed clone or kept visit")
	}
	b.AddToAnte(removed)
	if g.anteVisit(first.ID()) == visit || g.anteVisit(first.ID()) == uuid.Nil {
		t.Fatal("return reused visit")
	}
	clone.GetPlayer(a.PlayerID()).RemoveFromAnte(first.ID())
	if g.anteVisit(first.ID()) == uuid.Nil {
		t.Fatal("clone removal changed original")
	}
}

func TestAnteReferenceTypedTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	card := ownedTestCard(b.PlayerID(), "opponent ante")
	b.AddToAnte(card)
	target := DeclareAnteCardTarget("ante")
	possible := target.Possible(a.PlayerID(), nil, g)
	if len(possible) != 1 || possible[0] != card.ID() {
		t.Fatal("opponent ante excluded")
	}
	visit := g.anteVisit(card.ID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: card.ID(), incarnation: visit}}}}
	if got, ok := target.resolveCard(ctx); !ok || got.ID() != card.ID() {
		t.Fatal("typed target cannot read ante")
	}
	if err := g.ChangeOwner(card.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if got, ok := target.resolveCard(ctx); !ok || got.Owner() != a.PlayerID() {
		t.Fatal("typed target lost current owner")
	}
	removed, err := g.RemoveFromAnte(card.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := target.resolveCard(ctx); ok {
		t.Fatal("absent target resolved")
	}
	a.AddToAnte(removed)
	if _, ok := target.resolveCard(ctx); ok {
		t.Fatal("returned target resolved")
	}
	plain := NewGame(NewBasePlayer("C"), NewBasePlayer("D"))
	if len(target.Possible(a.PlayerID(), nil, plain)) != 0 {
		t.Fatal("disabled ante supplied target")
	}
	g.anteSettled = true
	if len(target.Possible(a.PlayerID(), nil, g)) != 0 {
		t.Fatal("settled ante supplied target")
	}
}

type anteReturnCost struct {
	card         uuid.UUID
	returnToAnte bool
}

func (anteReturnCost) Text() string                        { return "test ante return" }
func (anteReturnCost) CanPay(_, _ uuid.UUID, _ *Game) bool { return true }
func (c anteReturnCost) Pay(_, _ uuid.UUID, g *Game) error {
	card, err := g.RemoveFromAnte(c.card)
	if err != nil {
		return err
	}
	if c.returnToAnte {
		g.GetPlayer(card.Owner()).AddToAnte(card)
	} else {
		g.GetPlayer(card.Owner()).AddToHand(card)
	}
	return nil
}

func TestAnteReferenceCaptureBeforePayment(t *testing.T) {
	for _, activation := range []bool{false, true} {
		for _, returned := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(2)
			card := ownedTestCard(b.PlayerID(), "ante")
			b.AddToAnte(card)
			visit := g.anteVisit(card.ID())
			target := DeclareAnteCardTarget("ante")
			body := GainLifeFor(You, Fixed(3))
			var err error
			if activation {
				source := NewArtifact("activation", "", WithAbility(Activate(Costs(anteReturnCost{card.ID(), returned}), body, WithTargets(target))))
				p := g.PutOnBattlefield(source, a.PlayerID())
				err = g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{card.ID()})
			} else {
				source := NewSorcery("spell", "", WithAbility(Spell(body, WithTargets(target))), WithAdditionalCost(anteReturnCost{card.ID(), returned}))
				source.SetOwner(a.PlayerID())
				a.AddToHand(source)
				err = g.CastSpellByID(a.PlayerID(), source.ID(), []uuid.UUID{card.ID()}, 0)
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.anteVisit(card.ID()) == visit {
				t.Fatal("cost did not change ante visit")
			}
			if obj := g.StackPeek(); obj.TargetZones[card.ID()] != core.ZoneAnte || obj.targetIncarnations[card.ID()] != visit {
				t.Fatal("payment changed saved target zone or visit")
			}
			g.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("payment return replaced announced target")
			}
		}
	}
}

func TestAnteReferenceCopiesAndRetarget(t *testing.T) {
	a, b := NewBasePlayer("A"), newRecordingPlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	card := ownedTestCard(a.PlayerID(), "ante")
	a.AddToAnte(card)
	old := g.anteVisit(card.ID())
	target := DeclareAnteCardTarget("ante")
	obj := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Targets: []uuid.UUID{card.ID()}, TargetSpecs: []Target{target}, Effects: []Effect{GainLifeFor(You, Fixed(3))}}
	g.PushStack(obj)
	a.RemoveFromAnte(card.ID())
	a.AddToAnte(card)
	unchanged := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if unchanged == nil || unchanged.targetAnteCards[card.ID()].visit != old {
		t.Fatal("copy followed returned visit")
	}
	b.chooseQueue = [][]uuid.UUID{{card.ID()}}
	changed := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if changed == nil || changed.targetAnteCards[card.ID()].visit != g.anteVisit(card.ID()) {
		t.Fatal("legal retarget did not save new visit")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 23 {
		t.Fatal("copies used wrong ante visits")
	}
}

func TestAnteReferenceEntryPaths(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	top := ownedTestCard(a.PlayerID(), "top")
	a.SetLibrary([]Card{top})
	g.AnteLibraryTops([]uuid.UUID{a.PlayerID()})
	if g.anteVisit(top.ID()) == uuid.Nil {
		t.Fatal("library-top ante lacks visit")
	}
	card := ownedTestCard(a.PlayerID(), "permanent")
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.MoveToAnte(a.PlayerID(), p.ID()); err != nil {
		t.Fatal(err)
	}
	if g.anteVisit(card.ID()) == uuid.Nil || g.anteVisit(card.ID()) == g.anteVisit(top.ID()) {
		t.Fatal("permanent ante lacks separate visit")
	}
	visit := g.anteVisit(top.ID())
	a.SetAnte([]Card{top})
	if g.anteVisit(top.ID()) != visit || g.anteVisit(card.ID()) != uuid.Nil {
		t.Fatal("zone replacement lost retained visit or kept removed card")
	}
	a.SetAnte(nil)
	a.SetAnte([]Card{top})
	if g.anteVisit(top.ID()) == visit {
		t.Fatal("zone return kept removed visit")
	}
	target := DeclareAnteCardTarget("ante")
	if err := ValidateCardDefinition("valid", NewSorcery("valid", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), WithTargets(target))))); err != nil {
		t.Fatal(err)
	}
	if err := target.spec.Choose(a.PlayerID(), nil, g, []uuid.UUID{card.ID()}); err == nil {
		t.Fatal("target chose absent card")
	}
	if err := target.spec.Choose(a.PlayerID(), nil, g, []uuid.UUID{top.ID()}); err != nil {
		t.Fatal(err)
	}
}

func TestAnteReferenceCopyCannotReferToLaterSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	card := ownedTestCard(a.PlayerID(), "ante")
	a.AddToAnte(card)
	target := DeclareAnteCardTarget("ante")
	obj := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Targets: []uuid.UUID{card.ID()}, TargetSpecs: []Target{target}}
	g.PushStack(obj)
	a.RemoveFromAnte(card.ID())
	g.PushStack(&StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()})
	copied := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copied == nil || len(copied.targetSpellSources) != 0 || copied.TargetZones[card.ID()] != core.ZoneAnte {
		t.Fatal("copy referred to a later spell")
	}
}

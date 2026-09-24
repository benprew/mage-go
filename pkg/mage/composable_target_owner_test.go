package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedTargetYouOwnCandidates(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewInstant("Source", "")
	var owned []uuid.UUID
	for _, owner := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			card := NewCreature("Candidate", "", 2, 2)
			card.SetOwner(owner)
			g.PutOnBattlefield(card, controller)
			if owner == a.PlayerID() {
				owned = append(owned, card.ID())
			}
		}
	}
	target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn())
	got := target.Possible(a.PlayerID(), source, g)
	if len(got) != 2 || !slices.Contains(got, owned[0]) || !slices.Contains(got, owned[1]) {
		t.Fatalf("owned candidates = %v", got)
	}
	both := DeclarePermanentTargets("both", 1, 2, AnyPermanent, TargetYouOwn(), TargetYouControl())
	got = both.Possible(a.PlayerID(), source, g)
	if len(got) != 1 || got[0] != owned[0] {
		t.Fatalf("owned and controlled candidates = %v", got)
	}
	opposing := DeclarePermanentTarget("opposing", AnyPermanent, TargetYouOwn(), TargetOpponentControls())
	got = opposing.Possible(a.PlayerID(), source, g)
	if len(got) != 1 || got[0] != owned[1] {
		t.Fatalf("owned opposing candidates = %v", got)
	}
	g.PhaseOut(owned[0])
	if len(both.Possible(a.PlayerID(), source, g)) != 0 {
		t.Fatal("phased target is available")
	}
}

func TestTypedTargetYouOwnResolution(t *testing.T) {
	for _, change := range []string{"none", "owner", "owner restored", "control", "phase cycle", "return"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, err := NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			card := NewCreature("Owned", "", 2, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn())
			ability := Spell(ReturnToHand(target), WithTargets(target))
			spell := NewInstant("Return", "", WithAbility(ability))
			obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			clone := g.Clone()
			switch change {
			case "owner", "owner restored":
				if err := clone.ChangeOwner(card.ID(), b.PlayerID()); err != nil {
					t.Fatal(err)
				}
				if change == "owner restored" {
					if err := clone.ChangeOwner(card.ID(), a.PlayerID()); err != nil {
						t.Fatal(err)
					}
				}
			case "control":
				clone.AddControlEffect(ControlEffectSpec{SourceID: spell.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phase cycle":
				clone.PhaseIn(clone.PhaseOut(card.ID()))
			case "return":
				clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
				clone.PutOnBattlefield(card, a.PlayerID())
			}
			clone.ResolveStack()
			want := 1
			if change == "owner" || change == "return" {
				want = 0
			}
			if got := len(clone.GetPlayer(a.PlayerID()).Hand()); got != want {
				t.Fatalf("hand = %d, want %d", got, want)
			}
			if g.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()).Card.Owner() != a.PlayerID() || len(a.Hand()) != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestTypedTargetYouOwnCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn())
	ability := Spell(ReturnToHand(target), WithTargets(target))
	card := NewInstant("Return", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != second.ID() || obj.Targets[0] != first.ID() {
		t.Fatal("copy did not use its controller for ownership")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("copies did not return their respective owned targets")
	}
}

func TestTypedTargetYouOwnActivation(t *testing.T) {
	for _, scenario := range []string{"paid", "not owned", "not controlled", "no mana", "control lost", "source exit"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("owned", AnyPermanent, TargetYouOwn(), TargetYouControl())
			source := NewArtifact("Return source", "", WithAbility(Activate(Costs(ManaPayment("{6}"), TapCost(This)), ReturnToHand(target), WithTargets(target))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			card := NewCreature("Owned", "", 2, 2)
			owner, controller := a.PlayerID(), a.PlayerID()
			if scenario == "not owned" {
				owner = b.PlayerID()
			}
			if scenario == "not controlled" {
				controller = b.PlayerID()
			}
			card.SetOwner(owner)
			g.PutOnBattlefield(card, controller)
			mana := 6
			if scenario == "no mana" {
				mana = 0
			}
			a.ManaPool().Add(Colorless, mana)
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{card.ID()})
			if scenario == "not owned" || scenario == "not controlled" || scenario == "no mana" {
				if err == nil || a.ManaPool().TotalMana() != mana || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("invalid activation succeeded or paid costs")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 1 || g.FindPermanent(card.ID()) == nil {
				t.Fatal("incorrect payment or stack timing")
			}
			if scenario == "control lost" {
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			if scenario == "source exit" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			want := 1
			if scenario == "control lost" {
				want = 0
			}
			if len(a.Hand()) != want {
				t.Fatalf("hand = %d, want %d", len(a.Hand()), want)
			}
		})
	}
}

package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCollectionActionOperands(t *testing.T) {
	for name, build := range map[string]func(PermanentSet) ResolvingEffect{
		"tap": TapEach, "untap": UntapEach, "return": ReturnEachToHand,
		"destroy":         func(s PermanentSet) ResolvingEffect { return DestroyEach(s) },
		"no regeneration": func(s PermanentSet) ResolvingEffect { return DestroyEachWithoutRegeneration(s) },
		"damage recipients": func(s PermanentSet) ResolvingEffect {
			return DealDamageEach(DamageSourceOf(AttachedPermanent), PermanentDamageRecipients(s), Fixed(0))
		},
	} {
		for shape, set := range map[string]PermanentSet{
			"singleton": SingletonPermanent(This), "auras": AttachedAuras(This),
			"attachments": AttachmentsOn(SingletonPermanent(This)), "blockers": BlockersOf(This),
			"blocked": BlockedAttackersOf(This), "combat": BlockingOrBlockedBy(This),
			"history": BlockedByThisTurn(This), "excluded": WithoutPermanent(MatchingPermanents(IsCreature), This),
			"lowest":     LowestPower(SingletonPermanent(This)),
			"controller": PermanentsControlledBy(ControllerOfWithLKI(This), AnyPermanent),
			"owner":      PermanentsOwnedBy(OwnerOf(This), AnyPermanent),
			"opponents":  PermanentsControlledByPlayers(OpponentsOf(ControllerOfWithLKI(This)), AnyPermanent),
		} {
			t.Run(name+"/"+shape, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "", 1, 4)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				original := g.CaptureDamageSource(p.ID())
				body := build(set)
				if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))); err != nil {
					t.Fatal(err)
				}
				if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, b.PlayerID())
				g.ChangeObjectColor(returned.ID(), core.Red)
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(original) || got[0].HasColor(core.Red) {
						t.Fatal("collection action lost exact source declaration")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shares mutable facts")
					}
				}
				clone := g.Clone()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("clone shares delayed lifetime")
				}
			})
		}
	}
}

func TestWaitingDelayedCollectionActionPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("must not run", func(*Permanent, *Game) bool { reads++; return true })
	set := IntersectPermanents(FilterPermanentSet(SingletonPermanent(This), filter), UnionPermanents(AttachmentsOn(SingletonPermanent(This)), LowestPower(SingletonPermanent(This))))
	if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(set)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || p.Tapped {
			t.Fatal("query evaluated collection or body")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, set := range []PermanentSet{SingletonPermanent(AttachedPermanent), MatchingPermanents(filter), waitingMultiEmptySet{&reads}} {
		if err := ApplyEffect(g, AtNext(EachEndStep, TapEach(set)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred implicit or opaque source")
	}
	if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, TapEach(SingletonPermanent(This)))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("future body supplied a reference before creation")
	}
	for _, body := range []ResolvingEffect{TapEach(nil), UntapEach(nil), ReturnEachToHand(nil), DestroyEach(nil), DestroyEachWithoutRegeneration(nil), DealDamageEach(This, PermanentDamageRecipients(nil), Fixed(1))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid collection declaration accepted")
		}
	}
	g.triggers.SetDelayed(nil)
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("First", TapEach(SingletonPermanent(This))), ModeOf("Second", UntapEach(SingletonPermanent(This)))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 || p.Tapped {
		t.Fatal("saved modes lost declaration or ran body")
	}
}

package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCombatPreventionOperands(t *testing.T) {
	shapes := map[string]ResolvingEffect{
		"to-and-by":         PreventCombatDamageToAndBy(This),
		"single-combat":     PreventCombatDamageBy(This),
		"single-turn":       PreventCombatDamageByThisTurn(This),
		"collection-combat": PreventCombatDamageByEach(BlockersOf(This)),
		"collection-turn":   PreventCombatDamageByEachThisTurn(BlockedAttackersOf(This)),
		"singleton":         PreventCombatDamageByEachThisTurn(SingletonPermanent(This)),
		"attachments":       PreventCombatDamageByEachThisTurn(AttachmentsOn(SingletonPermanent(This))),
		"excluded":          PreventCombatDamageByEachThisTurn(WithoutPermanent(MatchingPermanents(IsCreature), This)),
		"controller":        PreventCombatDamageByEachThisTurn(PermanentsControlledBy(ControllerOfWithLKI(This), AnyPermanent)),
		"limit":             PreventCombatDamageByEachThisTurn(ManaValueAtMost(MatchingPermanents(IsCreature), PowerOf(This))),
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 1, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(returned.ID(), core.Red)
			for range 3 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) {
					t.Fatal("combat prevention lost exact source declaration")
				}
				got[0].card.AddType(core.TypeEnchantment)
				if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
					t.Fatal("query shares mutable facts")
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedCombatPreventionPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("query must not call filter", func(*Permanent, *Game) bool { reads++; return false })
	body := PreventCombatDamageByEachThisTurn(FilterPermanentSet(UnionPermanents(SingletonPermanent(This), SingletonPermanent(This)), filter))
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("first", body), ModeOf("second", PreventCombatDamageToAndBy(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(g.effects.replacements) != 0 {
			t.Fatal("query installed prevention or lost empty mode declaration")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{PreventAllCombatDamage(), PreventCombatDamageToAndBy(AttachedPermanent), PreventCombatDamageBy(AttachedPermanent), PreventCombatDamageByEachThisTurn(MatchingPermanents(filter)), PreventCombatDamageByEach(waitingMultiEmptySet{&reads}), AtNext(EachEndStep, PreventCombatDamageBy(This))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 || len(g.effects.replacements) != 0 {
		t.Fatal("query inferred implicit opaque or future source")
	}
	for _, body := range []ResolvingEffect{PreventCombatDamageToAndBy(nil), PreventCombatDamageBy(nil), PreventCombatDamageByEach(nil), PreventCombatDamageByThisTurn(nil), PreventCombatDamageByEachThisTurn(nil), PreventCombatDamageByEachThisTurn(SingletonPermanent(nil))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid combat prevention declaration accepted")
		}
	}
}

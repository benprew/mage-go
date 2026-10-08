package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedLayerOperands(t *testing.T) {
	shapes := map[string]ResolvingEffect{
		"zero-value":      Until(core.EndOfTurn, ModifyPT(AttachedPermanent, MultiplyValues(Fixed(0), PowerOfWithLKI(This)), Fixed(0))),
		"singleton":       Until(core.EndOfTurn, GrantEachKeyword(SingletonPermanent(This), core.Flying)),
		"subject":         Until(core.EndOfTurn, GrantKeywordModification(This, core.Flying)),
		"power":           Until(core.EndOfTurn, ModifyPT(AttachedPermanent, PowerOfWithLKI(This), Fixed(0))),
		"toughness":       Until(core.EndOfTurn, SetBasePowerToughness(AttachedPermanent, Fixed(1), ToughnessOfWithLKI(This))),
		"controller":      Until(core.EndOfTurn, ChangeController(AttachedPermanent, ControllerOfWithLKI(This))),
		"collection":      Until(core.EndOfTurn, GrantEachKeyword(WithoutPermanent(MatchingPermanents(IsCreature), This), core.Flying)),
		"upkeep":          UntilNextUpkeep(ControllerOfWithLKI(This), GrantKeywordModification(AttachedPermanent, core.Flying)),
		"end-upkeep":      UntilEndOfNextUpkeep(OwnerOf(This), GrantKeywordModification(AttachedPermanent, core.Flying)),
		"source-duration": UntilSourceLeaves(This, GrantKeywordModification(AttachedPermanent, core.Flying)),
		"tapped-duration": UntilSourceUntaps(This, GrantKeywordModification(AttachedPermanent, core.Flying)),
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
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
					t.Fatal("layer lost exact source declaration")
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

func TestWaitingDelayedLayerPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("must not run", func(*Permanent, *Game) bool { reads++; return false })
	body := Until(core.EndOfTurn, ModifyEachPT(FilterPermanentSet(WithoutPermanent(MatchingPermanents(filter), This), filter), PowerOfWithLKI(This), waitingMultiAmount{&reads}))
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("first", body), ModeOf("second", body)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query ran filter or lost empty set declaration")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		Until(core.EndOfTurn, GrantKeywordModification(AttachedPermanent, core.Flying)),
		Until(core.EndOfTurn, GrantEachKeyword(MatchingPermanents(filter), core.Flying)),
		AtNext(EachEndStep, Until(core.EndOfTurn, GrantKeywordModification(This, core.Flying))),
		Until(core.EndOfTurn, ActivatedAbilityModification(AttachedPermanent, Costs(), TapSubject(This))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred implicit opaque or future reads")
	}
	for _, body := range []ResolvingEffect{Until(core.EndOfTurn), Until(core.EndOfTurn, ModifyPT(nil, Fixed(1), Fixed(1))), UntilNextUpkeep(nil, GrantKeywordModification(This, core.Flying))} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid layer accepted")
		}
	}
}

package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCollectionCountReferences(t *testing.T) {
	for name, subjects := range map[string]PermanentSet{
		"singleton":   SingletonPermanent(This),
		"auras":       AttachedAuras(This),
		"attachments": AttachmentsOn(SingletonPermanent(This)),
		"blockers":    BlockersOf(This),
		"blocked":     BlockedAttackersOf(This),
		"combat":      BlockingOrBlockedBy(This),
		"history":     BlockedByThisTurn(This),
		"excluded":    WithoutPermanent(MatchingPermanents(IsCreature), This),
		"lowest":      LowestPower(SingletonPermanent(This)),
		"players":     PermanentsControlledByPlayers(OpponentsOf(ControllerOfWithLKI(This)), IsCreature),
	} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "{U}", 1, 4)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				original := g.CaptureDamageSource(p.ID())
				body := GainLifeFor(You, CountPermanents(subjects))
				if err := ValidateCardDefinition("Source", NewCreature("Source", "", 1, 4, WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))); err != nil {
					t.Fatal(err)
				}
				if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				for range 3 {
					got := g.WaitingDelayedPermanentSources()
					if len(got) != 1 || !got[0].SameObject(original) {
						t.Fatal("collection operand lost saved source")
					}
					if state == "returned" && got[0].HasColor(core.Red) {
						t.Fatal("returned object replaced saved facts")
					}
					got[0].card.AddType(core.TypeEnchantment)
					if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
						t.Fatal("query shares mutable facts")
					}
				}
				if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 0 {
					t.Fatal("query executed body")
				}
				clone := g.Clone()
				for _, game := range []*Game{clone, g} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					if len(game.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("detected delay still waits")
					}
					if game == clone && len(g.WaitingDelayedPermanentSources()) != 1 {
						t.Fatal("clone consumed original delay")
					}
					game.PutTriggersOnStack()
					game.ResolveStack()
					count := 0
					if (name == "singleton" || name == "lowest") && (state == "live" || state == "cycle" || state == "control") {
						count = 1
					}
					if (name == "players" || name == "excluded") && state == "returned" {
						count = 1
					}
					if game.GetPlayer(a.PlayerID()).Life() != 20+count {
						t.Fatal("wrong resolution membership", game.GetPlayer(a.PlayerID()).Life(), count)
					}
				}
			})
		}
	}
}

func TestWaitingDelayedCollectionCountPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("query must not run", func(*Permanent, *Game) bool { reads++; return true })
	set := IntersectPermanents(FilterPermanentSet(SingletonPermanent(This), filter), UnionPermanents(AttachmentsOn(SingletonPermanent(This)), LowestPower(SingletonPermanent(This))))
	value := MultiplyValues(Fixed(0), CountPermanents(set))
	if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, value)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || a.Life() != 20 {
			t.Fatal("query evaluated collection")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, set := range []PermanentSet{SingletonPermanent(AttachedPermanent), AttachedAuras(AttachedPermanent), BlockersOf(AttachedPermanent), PermanentsControlledByPlayers(OpponentsOf(You), filter), waitingMultiEmptySet{&reads}, MatchingPermanents(filter)} {
		if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, CountPermanents(set))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred implicit source")
	}
	for _, set := range []PermanentSet{SingletonPermanent(nil), AttachedAuras(nil), AttachmentsOn(nil), BlockersOf(nil), BlockedByThisTurn(nil), LowestPower(nil), WithoutPermanent(MatchingPermanents(IsCreature), nil), PermanentsControlledByPlayers(nil, IsCreature)} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, GainLifeFor(You, CountPermanents(set))))))) == nil {
			t.Fatal("invalid collection accepted")
		}
	}
}

func TestWaitingDelayedCollectionCountLateRelations(t *testing.T) {
	for name, set := range map[string]PermanentSet{
		"auras": AttachedAuras(This), "attachments": AttachmentsOn(SingletonPermanent(This)),
		"blockers": BlockersOf(This), "blocked": BlockedAttackersOf(This),
		"combat": BlockingOrBlockedBy(This), "history": BlockedByThisTurn(This),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 1, 4), a.PlayerID())
			delay := AtNext(EachEndStep, GainLifeFor(You, CountPermanents(set)))
			if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 1 || a.Life() != 20 {
				t.Fatal("empty relation lost source")
			}
			switch name {
			case "auras", "attachments":
				aura := g.PutOnBattlefield(NewEnchantment("Late Aura", "", WithSubTypes("Aura")), a.PlayerID())
				g.Attach(aura.ID(), p.ID())
			default:
				other := g.PutOnBattlefield(NewCreature("Other", "", 1, 4), b.PlayerID())
				if name == "blockers" {
					g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
					g.combat.AddBlockerIncarnation(other.ID(), other.incarnationID, p.ID())
				} else {
					g.combat.AddAttackerIncarnation(other.ID(), other.incarnationID, a.PlayerID())
					g.combat.AddBlockerIncarnation(p.ID(), p.incarnationID, other.ID())
					g.recordBlockHistory(p, other)
				}
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 21 {
					t.Fatal("late relation was not read at resolution", game.GetPlayer(a.PlayerID()).Life())
				}
			}
		})
	}
}

package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetPlayerReferenceEvent(t *testing.T) {
	for _, scenario := range []string{"normal", "control", "return", "phase", "cycle", "shroud", "source-return", "copy", "no-target"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			defender := BindPlayer("defender")
			target := DeclarePermanentTarget("victim", IsCreature, TargetControlledBy(FromPlayer(defender)))
			card := NewCreature("Watcher", "{B}", 2, 2, WithAbility(WhenEvent(AttacksUnblocked(This).WithDefendingPlayer(defender),
				Sequence(TapSubject(target), GainLifeFor(FromPlayer(defender), Fixed(1))), WithTargets(target))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			own := g.PutOnBattlefield(NewCreature("Own", "", 2, 4), a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 4)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if scenario == "no-target" {
				g.PhaseOut(victim.ID())
			}
			g.FireEvent(GameEvent{Type: EvtAttacksUnblocked, SourceID: source.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			g.PutTriggersOnStack()
			if scenario == "no-target" {
				if g.StackSize() != 0 {
					t.Fatal("targetless trigger stacked")
				}
				return
			}
			if g.StackSize() != 1 || !slices.Equal(g.StackPeek().Targets, []uuid.UUID{victim.ID()}) {
				t.Fatal("defender was unavailable at target selection")
			}
			legal := true
			switch scenario {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: own.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
				legal = false
			case "return":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, b.PlayerID())
				legal = false
			case "phase":
				g.PhaseOut(victim.ID())
				legal = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "shroud":
				victim.GrantBaseAttr(Shroud)
				legal = false
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("live resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*Game{g, clone} {
				want := 20
				if legal {
					want++
				}
				if scenario == "copy" {
					want++
				}
				if game.GetPlayer(b.PlayerID()).Life() != want || game.GetPlayer(a.PlayerID()).Life() != 20 || game.FindPermanent(own.ID()).Tapped {
					t.Fatal("target legality or captured defender changed")
				}
			}
		})
	}
}

func TestTargetPlayerReferenceDefinition(t *testing.T) {
	player := BindPlayer("missing")
	for _, ref := range []PlayerRef{nil, FromPlayer(nil), FromPlayer(player), ControllerOf(This)} {
		target := DeclarePermanentTarget("victim", IsCreature, TargetControlledBy(ref))
		card := NewInstant("Invalid", "", Spell(TapSubject(target), WithTargets(target)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid player reference accepted")
		}
	}
	target := DeclarePermanentTarget("own", IsCreature, TargetControlledBy(You))
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", Spell(TapSubject(target), WithTargets(target)))); err != nil {
		t.Fatal(err)
	}
}

func TestTargetPlayerReferenceSnapshotsAndQueries(t *testing.T) {
	for _, random := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		player := BindPlayer("step player")
		target := DeclarePermanentTarget("victim", IsCreature, TargetControlledBy(FromPlayer(player)))
		if random {
			target.spec = TargetRandom(target.spec)
		}
		card := NewArtifact("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, player),
			Sequence(TapSubject(target), GainLifeFor(FromPlayer(player), Fixed(1))), WithTargets(target))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(card, a.PlayerID())
		own := g.PutOnBattlefield(NewCreature("Own", "", 2, 3), a.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), b.PlayerID())
		g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{player: uuid.New()}
		originalBinding := g.resolution.playerBindings[player]
		g.SetRandomResults([]int{0, 0, 7})
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
		g.PutTriggersOnStack()
		if g.StackSize() != 2 || g.resolution.playerBindings[player] != originalBinding {
			t.Fatal("target query changed live frame or mixed pending events")
		}
		if random && g.RandIntn(10) != 7 {
			t.Fatal("random selection did not consume live results")
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		for _, game := range []*Game{g, clone} {
			if !game.FindPermanent(own.ID()).Tapped || !game.FindPermanent(other.ID()).Tapped || game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 21 {
				t.Fatal("event snapshots crossed stack objects")
			}
		}
	}
}

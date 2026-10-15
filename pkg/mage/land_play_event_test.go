package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func landPlayWatcher(afterFirst bool) Card {
	player, land, ordinal := BindPlayer("player"), BindPermanent("land"), BindNumber("ordinal")
	pattern := ControllerPlaysLand(player, land, ordinal)
	if afterFirst {
		pattern = pattern.AfterFirst()
	}
	return NewEnchantment("Watcher", "", WithAbility(WhenEvent(pattern, Sequence(GainLifeFor(FromPlayer(player), FromNumber(ordinal)), TapSubject(From(land))))))
}

func TestLandPlayEventSnapshot(t *testing.T) {
	for _, change := range []string{"none", "source leaves", "source returns", "control", "phase", "land returns", "land phase"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card := landPlayWatcher(true)
			source := g.PutOnBattlefield(card, a.PlayerID())
			land := NewLand("Land")
			a.AddToHand(land)
			g.GrantExtraLandPlay(a.PlayerID(), 3)
			g.SetLandsPlayedThisTurn(1)
			if err := g.playLandCore(a.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 {
				t.Fatalf("stack=%d", g.StackSize())
			}
			g.SetLandsPlayedThisTurn(0)
			switch change {
			case "source leaves", "source returns":
				g.RemoveFromBattlefield(source)
				if change == "source returns" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(source.ID())
			case "land returns":
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				g.PutOnBattlefield(land, a.PlayerID())
			case "land phase":
				g.PhaseOut(land.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 20 {
				t.Fatalf("saved player/ordinal lost: %d/%d", a.Life(), b.Life())
			}
			if change == "land phase" {
				g.PhaseIn([]uuid.UUID{land.ID()})
			}
			wantTapped := change != "land returns" && change != "land phase"
			if g.FindPermanent(land.ID()).Tapped != wantTapped {
				t.Fatal("land visit was not preserved")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("clone lost event")
			}
		})
	}
}

func TestLandPlayEventMatching(t *testing.T) {
	for _, afterFirst := range []bool{false, true} {
		for _, scenario := range []string{"first", "second", "direct", "opponent", "phase", "phase cycle", "removed ability", "failed", "two sources"} {
			t.Run(scenario+map[bool]string{false: "/all", true: "/later"}[afterFirst], func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				g.SetStep(core.PrecombatMain)
				source := g.PutOnBattlefield(landPlayWatcher(afterFirst), a.PlayerID())
				land := NewLand("Land")
				a.AddToHand(land)
				want := 1
				switch scenario {
				case "first":
					if afterFirst {
						want = 0
					}
				case "second":
					g.SetLandsPlayedThisTurn(1)
					g.GrantExtraLandPlay(a.PlayerID(), 1)
					want = 2
				case "direct":
					g.PutOnBattlefield(NewLand("Direct"), a.PlayerID())
					want = 0
				case "opponent":
					g.SetActivePlayerIndex(1)
					a.RemoveFromHand(land.ID())
					land.SetOwner(b.PlayerID())
					b.AddToHand(land)
					want = 0
				case "phase":
					g.PhaseOut(source.ID())
					want = 0
				case "phase cycle":
					g.PhaseOut(source.ID())
					g.PhaseIn([]uuid.UUID{source.ID()})
					if afterFirst {
						want = 0
					}
				case "removed ability":
					g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
					g.ApplyContinuousEffects()
					want = 0
				case "failed":
					g.SetLandsPlayedThisTurn(1)
					want = 0
				case "two sources":
					g.PutOnBattlefield(landPlayWatcher(afterFirst), a.PlayerID())
					if afterFirst {
						want = 0
					} else {
						want = 2
					}
				}
				if scenario != "direct" {
					actor := a.PlayerID()
					if scenario == "opponent" {
						actor = b.PlayerID()
					}
					err := g.PlayLand(actor, land.ID())
					if (err != nil) != (scenario == "failed") {
						t.Fatal(err)
					}
				}
				if a.Life() != 20+want || b.Life() != 20 {
					t.Fatalf("life=%d/%d want gain %d", a.Life(), b.Life(), want)
				}
			})
		}
	}
}

func TestLandPlayEventInvalid(t *testing.T) {
	p, l, n := BindPlayer("player"), BindPermanent("land"), BindNumber("ordinal")
	for _, pattern := range []EventPattern{ControllerPlaysLand(nil, l, n), ControllerPlaysLand(p, nil, n), ControllerPlaysLand(p, l, nil)} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("nil binding accepted")
		}
	}
	card := NewEnchantment("Bad", "", WithAbility(WhenEvent(ControllerPlaysLand(p, l, n), TapSubject(From(BindPermanent("unbound"))))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound read accepted")
	}
}

func TestLandPlayEventIndependentPending(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.PutOnBattlefield(landPlayWatcher(true), a.PlayerID())
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtLandPlayed, SourceID: first.ID(), SourceIncarnation: first.incarnationID, PlayerID: a.PlayerID(), Amount: 2})
	g.FireEvent(core.GameEvent{Type: core.EvtLandPlayed, SourceID: second.ID(), SourceIncarnation: second.incarnationID, PlayerID: a.PlayerID(), Amount: 3})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("events did not remain separate")
	}
	g.SetLandsPlayedThisTurn(0)
	g.ResolveStack()
	if a.Life() != 25 || !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("pending events shared a result")
	}
}

func TestLandPlayEventLibraryAndFailure(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(landPlayWatcher(false), a.PlayerID())
	land := NewLand("Library land")
	land.SetOwner(a.PlayerID())
	a.SetLibrary([]Card{land})
	g.AddPlayLandsFromZone(a.PlayerID(), core.ZoneLibrary)
	if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || len(a.Library()) != 0 || g.GetLandsPlayedThisTurn() != 1 {
		t.Fatal("library land play not counted")
	}
	other := NewLand("Failed land")
	a.AddToHand(other)
	if err := g.PlayLand(a.PlayerID(), other.ID()); err == nil {
		t.Fatal("second land was allowed")
	}
	if a.Life() != 21 || len(a.Hand()) != 1 || g.GetLandsPlayedThisTurn() != 1 || g.StackSize() != 0 {
		t.Fatal("failed play published an event")
	}
}

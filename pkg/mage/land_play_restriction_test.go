package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandPlayRestrictionLifecycle(t *testing.T) {
	for _, active := range []int{0, 1} {
		for _, state := range []string{"normal", "tapped", "face-down", "control", "phased", "cycle", "removed", "restored", "departed", "returned", "clone"} {
			t.Run(string(rune('A'+active))+"/"+state, func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				g.SetStep(core.PrecombatMain)
				g.SetActivePlayerIndex(active)
				filter := NewCardFilter("blocked name", func(c Card) bool { return c.Name() == "Blocked" })
				card := NewArtifact("Restriction", "", WithAbility(Static(PreventLandPlays(filter))))
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				blocked := true
				switch state {
				case "face-down":
					g.MutablePermanent(source.ID()).FaceDown = true
					g.ApplyContinuousEffects()
					blocked = false
				case "tapped":
					g.TapPermanent(source)
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phased":
					g.PhaseOut(source.ID())
					blocked = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "removed", "restored":
					if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					g.ApplyContinuousEffects()
					blocked = false
					if state == "restored" {
						g.GetEffects().RemoveEndOfTurn()
						g.ApplyContinuousEffects()
						blocked = true
					}
				case "departed", "returned":
					g.BouncePermanentToHand(source)
					blocked = false
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
						blocked = true
					}
				case "clone":
					clone := g.Clone()
					clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
					if clone.IsLandPlayProhibited(NewLand("Blocked")) {
						t.Fatal("clone retained departed restriction")
					}
				}
				player := g.ActivePlayerObj()
				land := NewLand("Blocked")
				player.AddToHand(land)
				handCount := len(player.Hand())
				g.GrantExtraLandPlay(player.PlayerID(), 3)
				for range 2 {
					if g.IsLandPlayProhibited(land) != blocked {
						t.Fatal("wrong restriction query")
					}
					if (len(g.GetPlayableLands(player.PlayerID())) == 0) != blocked {
						t.Fatal("wrong playable land list")
					}
				}
				if g.GetLandsPlayedThisTurn() != 0 || len(player.Hand()) != handCount {
					t.Fatal("query changed state")
				}
				err := g.PlayLand(player.PlayerID(), land.ID())
				if (err != nil) != blocked {
					t.Fatalf("play error %v, blocked %v", err, blocked)
				}
				if blocked && (len(player.Hand()) != handCount || g.GetLandsPlayedThisTurn() != 0 || g.FindPermanent(land.ID()) != nil) {
					t.Fatal("rejected play changed state")
				}
				other := NewLand("Allowed")
				player.AddToHand(other)
				if err := g.PlayLand(player.PlayerID(), other.ID()); err != nil {
					t.Fatal(err)
				}
				direct := g.PutOnBattlefield(NewLand("Blocked"), player.PlayerID())
				if direct == nil {
					t.Fatal("land-play rule prevented direct entry")
				}
			})
		}
	}
}

func TestLandPlayRestrictionLibraryAndOverlap(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	filter := NewCardFilter("all", func(Card) bool { return true })
	source := g.PutOnBattlefield(NewArtifact("First", "", WithAbility(Static(PreventLandPlays(filter)))), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", "", WithAbility(Static(PreventLandPlays(filter)))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(PlayLandsFromTopOfLibrary())), a.PlayerID())
	first, next := NewLand("First land"), NewLand("Next land")
	a.SetLibrary([]Card{first, next})
	for _, remove := range []bool{false, true} {
		if remove {
			g.RemoveFromBattlefield(source)
		}
		if err := g.PlayLand(a.PlayerID(), first.ID()); err == nil {
			t.Fatal("library play was allowed")
		}
		if len(a.Library()) != 2 || a.Library()[0].ID() != first.ID() || a.Library()[1].ID() != next.ID() || g.GetLandsPlayedThisTurn() != 0 {
			t.Fatal("rejection changed library order or history")
		}
	}
	clone := g.Clone()
	if !clone.IsLandPlayProhibited(first) {
		t.Fatal("clone lost active restriction")
	}
	g.RemoveFromBattlefield(second)
	if err := g.PlayLand(a.PlayerID(), first.ID()); err != nil {
		t.Fatal(err)
	}
	if err := clone.PlayLand(a.PlayerID(), first.ID()); err == nil {
		t.Fatal("original changed clone")
	}
}

func TestLandPlayRestrictionInvalid(t *testing.T) {
	card := NewArtifact("Bad", "", WithAbility(Static(PreventLandPlays(CardFilter{}))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("empty filter accepted")
	}
}

func TestLandPlayRestrictionHandOrderAndPermissions(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(NewArtifact("Block", "", WithAbility(Static(PreventLandPlays(NewCardFilter("all", func(Card) bool { return true }))))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Unlimited", "", WithStaticAbility(AllowUnlimitedLandPlays())), a.PlayerID())
	first, land, last := NewInstant("First", ""), NewLand("Land"), NewInstant("Last", "")
	a.AddToHand(first)
	a.AddToHand(land)
	a.AddToHand(last)
	for _, candidate := range []Card{land, first} {
		if err := g.PlayLand(a.PlayerID(), candidate.ID()); err == nil {
			t.Fatal("invalid play succeeded")
		}
		hand := a.Hand()
		if len(hand) != 3 || hand[0].ID() != first.ID() || hand[1].ID() != land.ID() || hand[2].ID() != last.ID() {
			t.Fatal("rejected play changed hand order")
		}
	}
	if g.GetLandsPlayedThisTurn() != 0 || len(g.GetPlayableLands(a.PlayerID())) != 0 || g.StackSize() != 0 {
		t.Fatal("unlimited permission overrode restriction or failed play changed history")
	}
	if g.IsLandPlayProhibited(first) || g.IsLandPlayProhibited(nil) {
		t.Fatal("nonland restriction query")
	}
}

package mage

import (
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestUnlimitedLandPlayLifecycle(t *testing.T) {
	for _, state := range []string{"normal", "tapped", "control", "phased", "phase-cycle", "removed", "restored", "departed", "returned", "clone", "face-down"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			card := NewEnchantment("Permission", "", WithStaticAbility(AllowUnlimitedLandPlaysFor(You)))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			wantA, wantB := true, false
			switch state {
			case "tapped":
				g.TapPermanent(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				wantA, wantB = false, true
			case "phased":
				g.PhaseOut(source.ID())
				wantA = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "removed", "restored":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
				wantA = false
				if state == "restored" {
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					wantA = true
				}
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				wantA = false
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
					wantA = true
				}
			case "face-down":
				g.MutablePermanent(source.ID()).FaceDown = true
				g.ApplyContinuousEffects()
				wantA = false
			case "clone":
				clone := g.Clone()
				clone.PhaseOut(source.ID())
				if clone.HasUnlimitedLandPlays(a.PlayerID()) {
					t.Fatal("clone kept phased permission")
				}
			}
			for range 2 {
				if g.HasUnlimitedLandPlays(a.PlayerID()) != wantA || g.HasUnlimitedLandPlays(b.PlayerID()) != wantB {
					t.Fatal("permission has wrong player or lifetime")
				}
			}
			if g.GetLandsPlayedThisTurn() != 0 {
				t.Fatal("query changed play history")
			}
			for active, allowed := range []bool{wantA, wantB} {
				g.SetActivePlayerIndex(active)
				g.SetLandsPlayedThisTurn(1000)
				g.GrantExtraLandPlay(g.ActivePlayerObj().PlayerID(), 2)
				if allowed && g.MaxLandPlays() != math.MaxInt {
					t.Fatal("unlimited allowance has a finite card limit")
				}
				player := g.ActivePlayerObj()
				land := NewLand("Land")
				player.AddToHand(land)
				handCount := len(player.Hand())
				if (len(g.GetPlayableLands(player.PlayerID())) == 1) != allowed {
					t.Fatal("wrong land query")
				}
				err := g.PlayLand(player.PlayerID(), land.ID())
				if (err == nil) != allowed {
					t.Fatalf("play: %v, allowed %v", err, allowed)
				}
				if !allowed && (len(player.Hand()) != handCount || g.GetLandsPlayedThisTurn() != 1000) {
					t.Fatal("failed play changed state")
				}
			}
		})
	}
}

func TestUnlimitedLandPlayRestrictionsAndOverlap(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	first := g.PutOnBattlefield(NewArtifact("First", "", WithStaticAbility(AllowUnlimitedLandPlaysFor(You))), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", "", WithStaticAbility(AllowUnlimitedLandPlaysFor(You))), a.PlayerID())
	g.RemoveFromBattlefield(first)
	if !g.HasUnlimitedLandPlays(a.PlayerID()) {
		t.Fatal("one source removed another permission")
	}
	g.SetLandsPlayedThisTurn(1)
	land := NewLand("Land")
	a.AddToHand(land)
	b.AddToHand(NewLand("Other land"))
	if err := g.PlayLand(b.PlayerID(), b.Hand()[0].ID()); err == nil {
		t.Fatal("off-turn land played")
	}
	g.SetStep(core.Upkeep)
	if err := g.PlayLand(a.PlayerID(), land.ID()); err == nil || len(g.GetPlayableLands(a.PlayerID())) != 0 {
		t.Fatal("permission bypassed main phase")
	}
	g.SetStep(core.PrecombatMain)
	blocker := g.PutOnBattlefield(NewArtifact("Prohibition", "", WithAbility(Static(PreventLandPlays(NewCardFilter("all", func(Card) bool { return true }))))), b.PlayerID())
	if err := g.PlayLand(a.PlayerID(), land.ID()); err == nil || len(g.GetPlayableLands(a.PlayerID())) != 0 {
		t.Fatal("permission bypassed prohibition")
	}
	if len(a.Hand()) != 1 || g.GetLandsPlayedThisTurn() != 1 {
		t.Fatal("failed play changed hand or history")
	}
	if g.PutOnBattlefield(NewLand("Direct entry"), a.PlayerID()) == nil || g.GetLandsPlayedThisTurn() != 1 {
		t.Fatal("direct entry used land allowance")
	}
	g.RemoveFromBattlefield(blocker)
	if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(second)
	if g.HasUnlimitedLandPlays(a.PlayerID()) || g.MaxLandPlays() != 1 {
		t.Fatal("departure kept permission")
	}
}

func TestUnlimitedLandPlayInvalid(t *testing.T) {
	for _, subject := range []PlayerRef{nil, FromPlayer(BindPlayer("unbound"))} {
		card := NewArtifact("Invalid", "", WithStaticAbility(AllowUnlimitedLandPlaysFor(subject)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
}

func TestUnlimitedLandPlayLegacy(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.PutOnBattlefield(NewArtifact("Legacy", "", WithStaticAbility(AllowUnlimitedLandPlays())), a.PlayerID())
	for active := range 2 {
		g.SetActivePlayerIndex(active)
		if !g.HasUnlimitedLandPlays(g.ActivePlayerObj().PlayerID()) || g.MaxLandPlays() != math.MaxInt {
			t.Fatal("legacy global permission changed")
		}
	}
}

func TestUnlimitedLandPlayStackTiming(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(AllowUnlimitedLandPlaysFor(You))), a.PlayerID())
	spell := NewInstant("Pending spell", "")
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	land := NewLand("Land")
	a.AddToHand(land)
	if len(g.GetPlayableLands(a.PlayerID())) != 0 {
		t.Error("land offered with a pending spell")
	}
	if err := g.PlayLand(a.PlayerID(), land.ID()); err == nil {
		t.Error("land played with a pending spell")
	}
	if g.StackSize() != 1 || len(a.Hand()) != 1 || g.GetLandsPlayedThisTurn() != 0 || g.FindPermanent(land.ID()) != nil {
		t.Fatal("rejected land play changed state")
	}
	g.ResolveStack()
	if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
}

func TestUnlimitedLandPlayCleanupAndFiniteGrants(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(AllowUnlimitedLandPlaysFor(You))), a.PlayerID())
	g.GrantExtraLandPlay(a.PlayerID(), 3)
	g.SetLandsPlayedThisTurn(4)
	g.PhaseOut(source.ID())
	if g.MaxLandPlays() != 4 || g.GetLandsPlayedThisTurn() != 4 {
		t.Fatal("permission changed finite grants or history")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.doCleanupActions()
	if !g.HasUnlimitedLandPlays(a.PlayerID()) || g.ExtraLandPlaysGrantedThisTurn(a.PlayerID()) != 0 {
		t.Fatal("cleanup changed lasting permission or kept finite grant")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if g.MaxLandPlays() != 1 {
		t.Fatal("cleanup retained finite grant")
	}
}

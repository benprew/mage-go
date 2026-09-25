package astral

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestWhimsyRandomTapMembership(t *testing.T) {
	for _, kind := range []string{"artifact", "creature", "land", "enchantment", "tapped", "phased"} {
		for _, opponent := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/own", true: "/opponent"}[opponent], func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				owner := a.PlayerID()
				if opponent {
					owner = b.PlayerID()
				}
				var card mage.Card
				switch kind {
				case "artifact":
					card = mage.NewArtifact("Candidate", "")
				case "land":
					card = mage.NewLand("Candidate")
				case "enchantment":
					card = mage.NewEnchantment("Candidate", "")
				default:
					card = mage.NewCreature("Candidate", "", 1, 1)
				}
				card.SetOwner(owner)
				p := g.PutOnBattlefield(card, owner)
				p.GrantBaseAttr(core.Shroud)
				p.Tapped = kind == "tapped"
				p.PhasedOut = kind == "phased"
				eligible := kind == "artifact" || kind == "creature" || kind == "land"
				g.SetRandomResults([]int{1, 0})
				if err := applyWhimsyAction(g, uuid.New(), a.PlayerID(), 2); err != nil {
					t.Fatal(err)
				}
				if got, want := g.FindPermanentIncludingPhased(p.ID()).Tapped, eligible || kind == "tapped"; got != want {
					t.Fatalf("tapped = %v, want %v", got, want)
				}
				wantRandom := 1
				if eligible {
					wantRandom = 0
				}
				if got := g.RandIntn(2); got != wantRandom {
					t.Fatalf("next random result = %d, want %d", got, wantRandom)
				}
			})
		}
	}
}

func TestWhimsyRandomTapEventsAndRepeatedSelection(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	host := dsl.BindPermanent("host")
	aura := mage.NewAura("Watcher", "", mage.WithCastTarget(mage.TargetLand()), mage.WithAbility(dsl.WhenEvent(
		dsl.AttachedBecomesTapped(host), dsl.GainLife(dsl.You, dsl.Fixed(1)),
	)))
	aura.SetOwner(a.PlayerID())
	watcher := g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(watcher.ID(), land.ID())
	g.SetRandomResults([]int{0, 1})
	clone := g.Clone()
	for range 2 {
		if err := applyWhimsyAction(clone, uuid.New(), a.PlayerID(), 2); err != nil {
			t.Fatal(err)
		}
	}
	clone.PutTriggersOnStack()
	if clone.StackSize() != 1 {
		t.Fatalf("tap queued %d triggers, want 1", clone.StackSize())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 || g.FindPermanent(land.ID()).Tapped {
		t.Fatal("tap event or clone isolation failed")
	}
	if clone.RandIntn(2) != 1 {
		t.Fatal("second action selected an already tapped permanent")
	}
}

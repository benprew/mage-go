package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLiveCountValueEternalFlame(t *testing.T) {
	for _, state := range []string{"normal", "phase land", "control land", "return land", "copy", "clone", "illegal target", "shield"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			lands := make([]*mage.Permanent, 0, 3)
			for range 3 {
				lands = append(lands, g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID()))
			}
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
			walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID())
			obj := castRONSpell(t, g, a, repositoryCard(t, "DRK", "Eternal Flame"), walker.ID())
			count := 3
			switch state {
			case "phase land":
				g.PhaseOut(lands[0].ID())
				count = 2
			case "control land":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: lands[0].ID(), TargetID: lands[0].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				count = 2
			case "return land":
				g.RemoveFromBattlefield(lands[0])
				g.PutOnBattlefield(lands[0].Card, a.PlayerID())
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				count = 1
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "illegal target":
				g.PhaseOut(walker.ID())
			case "shield":
				g.AddPreventionShield(walker.ID(), 2)
			}
			groups, ok := mage.InspectPacketDamageInstructions(mage.Sequence(obj.Effects...))
			if !ok || len(groups) != 1 || len(groups[0].Packets) != 2 {
				t.Fatal("RON lost packet declaration")
			}
			for range 2 {
				if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
					t.Fatal("real RON live count packet query unknown")
				}
			}
			if g.GetPlayer(a.PlayerID()).Life() != 20 || g.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("query dealt damage")
			}
			g.ResolveTopOfStack()
			wantA, wantB, wantL := 20-(count+1)/2, 20, 10-count
			if state == "copy" {
				wantA, wantB = 20, 20-(count+1)/2
			}
			if state == "illegal target" {
				wantA, wantL = 20, 10
			}
			if state == "shield" {
				wantL = 9
			}
			if g.GetPlayer(a.PlayerID()).Life() != wantA || g.GetPlayer(b.PlayerID()).Life() != wantB || g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != wantL {
				t.Fatal("live RON damage differs")
			}
			if state == "clone" && (a.Life() != 20 || walker.Counters[core.Loyalty] != 10) {
				t.Fatal("clone crossed state")
			}
		})
	}
}

func TestRONLiveCountValueInvalid(t *testing.T) {
	for _, value := range []string{
		`TheNumberOfPermanentsOnTheBattlefield()`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))`,
		`HalfRoundedUp(TheNumberOfPermanentsOnTheBattlefield(Unknown))`,
		`HalfRoundedUp(TheNumberOfPermanentsOnTheBattlefield(AnyPermanent),Integer(2))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(`+value+`,Player(You))])]))])`), "invalid-count.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("invalid declaration accepted: %s", value)
		}
	}
}

package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStackTargetSpellSourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Counterspell", "Spell Blast", "Chaoslace"} {
		for _, change := range []string{"live", "departed", "recast", "recolored", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				bolt := repositoryCard(t, "LEA", "Lightning Bolt")
				original := castRONSpell(t, g, a, bolt, b.PlayerID())
				source := g.CaptureSpellDamageSource(original.ID)
				card := repositoryCard(t, "LEA", name)
				card.SetOwner(b.PlayerID())
				b.AddToHand(card)
				if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{original.ID}, 0); err == nil || g.StackSize() != 1 || len(b.Hand()) != 1 {
					t.Fatal("unpaid cast changed resources")
				}
				b.ManaPool().Add(core.Blue, 2)
				b.ManaPool().Add(core.Red, 1)
				x := 0
				if name == "Spell Blast" {
					x = 1
					b.ManaPool().Add(core.Colorless, 1)
				}
				if err := g.CastSpellByID(b.PlayerID(), card.ID(), []uuid.UUID{original.ID}, x); err != nil {
					t.Fatal(err)
				}
				referer := g.StackPeek()
				switch change {
				case "departed", "recast":
					g.CounterStackObject(original.ID)
					if change == "recast" {
						a.RemoveFromGraveyard(bolt.ID())
						castRONSpell(t, g, a, bolt, b.PlayerID())
					}
				case "recolored":
					g.ChangeObjectColor(original.ID, core.Blue)
				case "copy":
					if g.CopyStackObjectDirect(referer, a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				filter, _ := chosenSourceFilter(t, "LEA", "Circle of Protection: Red")
				chosen := mage.BindDamageSource("spell target")
				fixture := mage.NewArtifact("Choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(
					mage.ChooseDamageSourceFromStackSpellTargets(mage.You, filter, chosen),
					mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
				if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
					t.Fatal(err)
				}
				watcher := g.PutOnBattlefield(fixture, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), watcher.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				if chooser.calls != 0 || a.Life() != 20 {
					t.Fatal("choice ran before resolution")
				}
				g.ResolveTopOfStack()
				want := 1
				if change == "recolored" {
					want = 0
				}
				if chooser.calls != want || a.Life() != 20-2*want {
					t.Fatal("wrong source filter or resolution timing")
				}
				if want == 1 && !chooser.source.SameObject(source) {
					t.Fatal("departed spell was replaced by recast")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("choice changed pending clone")
				}
				g.ResolveStack()
				clone.ResolveStack()
				wantLife := 20
				if change == "recast" || name == "Chaoslace" && (change == "live" || change == "recolored" || change == "copy") {
					wantLife = 17
				}
				if b.Life() != wantLife || clone.GetPlayer(b.PlayerID()).Life() != wantLife || len(g.StackTargetSpellSources()) != 0 {
					t.Fatalf("RON gameplay or final-reference lifetime: life=%d want=%d", b.Life(), wantLife)
				}
			})
		}
	}
}

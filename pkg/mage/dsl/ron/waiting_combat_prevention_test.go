package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingCombatPreventionReferences(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"DRK", "Maze of Ith"}, {"ARN", "Ebony Horse"}, {"FEM", "Elvish Scout"},
		{"LEG", "Horn of Deafening"}, {"LEG", "Lady Evangela"}, {"LEG", "Subdue"},
		{"LEG", "Telekinesis"}, {"LEG", "Feint"}, {"PROMOS", "Sewers of Estark"},
	} {
		for _, state := range []string{"live", "depart", "return", "phase", "cycle", "illegal", "copy", "combat-end", "cleanup"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				subjectCard := mage.NewCreature("Subject", "{R}", 2, 20)
				subject := g.PutOnBattlefield(subjectCard, a.PlayerID())
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "{U}", 1, 20), b.PlayerID())
				subject.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{subject.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: subject.ID()}})
				target := subject
				collection := tc.name == "Feint" || tc.name == "Sewers of Estark"
				if tc.name == "Sewers of Estark" {
					target = blocker
				}
				saved := g.CaptureDamageSource(target.ID())
				spell := tc.name == "Subdue" || tc.name == "Telekinesis" || collection
				var source *mage.Permanent
				if spell {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
						t.Fatal("unpaid cast succeeded")
					}
				} else {
					source = g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					if tc.name != "Maze of Ith" {
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
							t.Fatal("unpaid activation succeeded")
						}
					}
				}
				for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
					a.ManaPool().Add(color, 4)
				}
				var err error
				if spell {
					err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0)
				} else {
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()})
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(g.WaitingPreventionPermanentSources()) != 0 {
					t.Fatal("announcement exposed unresolved subject")
				}
				if state == "illegal" {
					g.PhaseOut(target.ID())
				}
				if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("stack copy failed")
				}
				g.ResolveStack()
				switch state {
				case "depart", "return":
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					if state == "return" {
						g.PutOnBattlefield(target.Card, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(target.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "combat-end":
					g.GetEffects().RemoveEndOfCombat()
				case "cleanup":
					g.GetEffects().RemoveEndOfCombat()
					g.GetEffects().ClearReplacementsEndOfTurn()
				}
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					got := game.WaitingPreventionPermanentSources()
					want := 1
					if collection {
						want = 2
					}
					if state == "illegal" || state == "cleanup" || state == "combat-end" && tc.name == "Sewers of Estark" {
						want = 0
					}
					if len(got) != want {
						t.Fatalf("references=%d want=%d", len(got), want)
					}
					if want == 0 {
						continue
					}
					found := false
					for _, ref := range got {
						found = found || ref.SameObject(saved)
					}
					if !found {
						t.Fatal("returned object replaced retained subject")
					}
					binding := mage.BindDamageSource("red combat subject")
					effect := mage.Sequence(mage.ChooseDamageSourceFromWaitingPreventionReferences(mage.You, mage.DamageSourceHasColor(core.Red), binding), mage.DealDamageFrom(mage.FromDamageSource(binding), mage.You, mage.Fixed(1)))
					if err := mage.ApplyEffect(game, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("retained combat subject could not supply noncombat damage")
					}
				}
			})
		}
	}
}

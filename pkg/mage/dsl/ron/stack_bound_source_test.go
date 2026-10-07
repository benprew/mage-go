package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronBoundBlockPlayer struct {
	*ronSourceChoicePlayer
	assignments []mage.BlockAssignment
}

func (p *ronBoundBlockPlayer) DeclareBlockers(*mage.Game) []mage.BlockAssignment {
	return p.assignments
}

func TestRONStackBoundSourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Battering Ram", "Infernal Medusa"} {
		for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "control", "shroud", "copy", "source-leaves"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				defender := &ronBoundBlockPlayer{ronSourceChoicePlayer: chooser}
				g := mage.NewGame(a, defender)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				set, generic, black, opponent := "LEG", 3, 2, "Water Elemental"
				if name == "Battering Ram" {
					set, generic, black, opponent = "ATQ", 2, 0, "Wall of Air"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid casting changed resources")
				}
				a.ManaPool().Add(core.Colorless, generic)
				a.ManaPool().Add(core.Black, black)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("early entry")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				blockerCard := repositoryCard(t, "LEB", opponent)
				blockerCard.SetOwner(b.PlayerID())
				q := g.PutOnBattlefield(blockerCard, b.PlayerID())
				original := g.CaptureDamageSource(q.ID())
				g.RunStepWithPriority(core.BeginCombat)
				if name == "Battering Ram" && !g.FindPermanent(p.ID()).HasKeyword(core.Banding) {
					t.Fatal("missing combat banding")
				}
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})

				defender.assignments = []mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: p.ID()}}
				var clone *mage.Game
				checked := false
				g.SetOnPriority(func(game *mage.Game, _ int, _ bool) mage.PriorityAction {
					if checked {
						return mage.PriorityAction{Type: mage.PriorityPass}
					}
					checked = true

					if g.StackSize() != 1 || len(g.StackTargetPermanentSources()) != 0 {
						t.Fatalf("pair trigger missing or treated as targeted: stack=%d targets=%d attackers=%v combat=%+v", g.StackSize(), len(g.StackTargetPermanentSources()), g.GetCombat().Attackers, g.GetCombat())
					}
					if got := g.StackBoundPermanentSources(); len(got) != 1 || !got[0].SameObject(original) {
						t.Fatal("RON event binding not enumerated")
					}
					if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					switch change {
					case "departed", "returned":
						g.BouncePermanentToHand(g.FindPermanent(q.ID()))
						if change == "returned" {
							b.RemoveFromHand(blockerCard.ID())
							g.PutOnBattlefield(blockerCard, a.PlayerID())
							g.ChangeObjectColor(q.ID(), core.Red)
						}
					case "phased", "phase-cycle":
						ids := g.PhaseOut(q.ID())
						if change == "phase-cycle" {
							g.PhaseIn(ids)
						}
					case "recolored":
						g.ChangeObjectColor(q.ID(), core.Red)
					case "control":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), q.ID(), a.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					case "shroud":
						g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
					case "source-leaves":
						g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					}
					filter, _ := chosenSourceFilter(t, "LEA", "Circle of Protection: Blue")
					chosen := mage.BindDamageSource("bound choice")
					fixture := mage.NewEnchantment("Bound choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(
						mage.ChooseDamageSourceFromStackBindings(mage.You, filter, chosen),
						mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
					if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
						t.Fatal(err)
					}
					watcher := g.PutOnBattlefield(fixture, b.PlayerID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), watcher.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					clone = g.Clone()
					if chooser.calls != 0 || a.Life() != 20 {
						t.Fatal("choice ran before resolution")
					}
					g.ResolveTopOfStack()
					want := 1
					if change == "recolored" {
						want = 0
					}
					if chooser.calls != want || a.Life() != 20-2*want {
						t.Fatal("wrong resolution filter or damage")
					}
					if want == 1 {
						controller := b.PlayerID()
						if change == "control" {
							controller = a.PlayerID()
						}
						if !chooser.source.SameObject(original) || chooser.source.Controller() != controller {
							t.Fatal("choice lost original binding or current controller")
						}
					}
					if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.StackSize() != g.StackSize()+1 {
						t.Fatal("choice changed pending clone")
					}
					return mage.PriorityAction{Type: mage.PriorityPass}
				})
				g.RunStepWithPriority(core.DeclareBlockers)
				if !checked || clone == nil {
					t.Fatal("blocker priority was not reached")
				}

				for _, game := range []*mage.Game{g, clone} {
					game.ResolveStack()
					if len(game.StackBoundPermanentSources()) != 0 {
						t.Fatal("resolved reference remained in stack category")
					}
					game.RunStepWithPriority(core.EndCombat)
					survives := change == "returned"
					if (game.FindPermanent(q.ID()) != nil) != survives {
						t.Fatal("RON delayed destruction lost identity")
					}
					if change == "phased" {
						game.PhaseIn([]uuid.UUID{q.ID()})
						if game.FindPermanent(q.ID()) == nil {
							t.Fatal("phased object was destroyed")
						}
					}
				}
				if clone.GetPlayer(a.PlayerID()).Life() != a.Life() {
					t.Fatal("clone lost pending source choice")
				}
			})
		}
	}
}

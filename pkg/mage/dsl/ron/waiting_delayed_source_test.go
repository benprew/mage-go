package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONWaitingDelayedPermanentSources(t *testing.T) {
	for _, name := range []string{"Stone Giant", "Goblin Kites", "War Barge"} {
		for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "source-leaves"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(2)
				set := map[string]string{"Stone Giant": "LEB", "Goblin Kites": "FEM", "War Barge": "DRK"}[name]
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				saved := repositoryCard(t, "LEB", "Prodigal Sorcerer")
				saved.SetOwner(a.PlayerID())
				q := g.PutOnBattlefield(saved, a.PlayerID())
				original := g.CaptureDamageSource(q.ID())
				if name != "Stone Giant" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil || len(g.DelayedTriggers()) != 0 || g.StackSize() != 0 {
						t.Fatal("unpaid activation changed state")
					}
				}
				if name == "Goblin Kites" {
					a.ManaPool().Add(core.Red, 1)
				}
				if name == "War Barge" {
					a.ManaPool().Add(core.Colorless, 3)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delayed ability exists before resolution")
				}
				g.ResolveStack()
				switch change {
				case "departed", "returned":
					g.BouncePermanentToHand(q)
					if change == "returned" {
						a.RemoveFromHand(saved.ID())
						g.PutOnBattlefield(saved, b.PlayerID())
						g.ChangeObjectColor(q.ID(), core.Red)
					}
				case "phased", "phase-cycle":
					ids := g.PhaseOut(q.ID())
					if change == "phase-cycle" {
						g.PhaseIn(ids)
					}
				case "recolored":
					g.ChangeObjectColor(q.ID(), core.Red)
				case "source-leaves":
					g.BouncePermanentToHand(p)
				}
				filter, _ := chosenSourceFilter(t, "LEB", "Circle of Protection: Blue")
				result := mage.BindDamageSource("waiting choice")
				effect := mage.Sequence(mage.ChooseDamageSourceFromWaitingDelayedBindings(mage.You, filter, result), mage.DealDamageFrom(mage.FromDamageSource(result), mage.ActivePlayer, mage.Fixed(2)))
				fixture := mage.NewArtifact("Choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), effect)))
				if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
					t.Fatal(err)
				}
				watcher := g.PutOnBattlefield(fixture, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), watcher.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				if chooser.calls != 0 {
					t.Fatal("choice ran at announcement")
				}
				g.ResolveTopOfStack()
				want := 1
				if change == "recolored" {
					want = 0
				}
				if chooser.calls != want || a.Life() != 20-2*want || (want == 1 && !chooser.source.SameObject(original)) {
					t.Fatal("waiting source choice lost its saved target")
				}
				clone.ResolveTopOfStack()
				for _, game := range []*mage.Game{g, clone} {
					if name == "War Barge" && change != "source-leaves" {
						game.BouncePermanentToHand(game.FindPermanent(p.ID()))
					}
					if name != "War Barge" {
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					}
					game.PutTriggersOnStack()
					game.SetCoinFlipResults([]bool{false})
					game.ResolveStack()
					survives := change == "returned" || change == "phased"
					if (game.FindPermanentIncludingPhased(q.ID()) != nil) != survives || game.GetPlayer(a.PlayerID()).Life() != 20-2*want {
						t.Fatal("native delayed body or clone failed")
					}
				}
			})
		}
	}
}

func TestRONWaitingDelayedPermanentSourceInstances(t *testing.T) {
	g, a, b := ronGame()
	originals := make([]mage.DamageSource, 0, 2)
	declarations := make([]mage.Target, 0, 2)
	for _, player := range []*mage.BasePlayer{a, b} {
		card := repositoryCard(t, "FEM", "Goblin Kites")
		card.SetOwner(player.PlayerID())
		declarations = append(declarations, card.Abilities()[0].(*mage.ActionDefinition).Targets()[0])
		p := g.PutOnBattlefield(card, player.PlayerID())
		saved := repositoryCard(t, "LEB", "Prodigal Sorcerer")
		saved.SetOwner(player.PlayerID())
		q := g.PutOnBattlefield(saved, player.PlayerID())
		originals = append(originals, g.CaptureDamageSource(q.ID()))
		player.ManaPool().Add(core.Red, 1)
		if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if declarations[0] == declarations[1] {
		t.Fatal("factories share a target declaration")
	}
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 2 || !got[0].SameObject(originals[0]) || !got[1].SameObject(originals[1]) {
		t.Fatal("registration order or independent frames failed")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 2 {
		t.Fatal("instance or clone lifetime failed")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackObjects()[0].Controller != a.PlayerID() || g.StackObjects()[1].Controller != b.PlayerID() {
		t.Fatal("APNAP placement failed")
	}
	g.SetCoinFlipResults([]bool{false, false})
	g.ResolveStack()
	for _, source := range originals {
		if g.FindPermanent(source.ID()) != nil || clone.FindPermanent(source.ID()) == nil {
			t.Fatal("independent sacrifice bodies failed")
		}
	}
}

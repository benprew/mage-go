package ron

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStackDelayedPermanentSources(t *testing.T) {
	for _, name := range []string{"Stone Giant", "Goblin Kites", "War Barge"} {
		for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "recolored", "control", "shroud", "copy", "source-leaves"} {
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
				targetCard := repositoryCard(t, "LEB", "Prodigal Sorcerer")
				targetCard.SetOwner(a.PlayerID())
				q := g.PutOnBattlefield(targetCard, a.PlayerID())
				original := g.CaptureDamageSource(q.ID())
				if name != "Stone Giant" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err == nil || len(g.DelayedTriggers()) != 0 || g.StackSize() != 0 {
						t.Fatal("unpaid activation changed resources")
					}
				}
				switch name {
				case "Goblin Kites":
					a.ManaPool().Add(core.Red, 1)
				case "War Barge":
					a.ManaPool().Add(core.Colorless, 3)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if len(g.DelayedTriggers()) != 1 || len(g.StackBoundPermanentSources()) != 0 {
					t.Fatal("waiting delayed frame entered stack category")
				}
				switch change {
				case "departed", "returned":
					g.BouncePermanentToHand(g.FindPermanent(q.ID()))
					if change == "returned" {
						a.RemoveFromHand(targetCard.ID())
						g.PutOnBattlefield(targetCard, b.PlayerID())
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
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: q.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "shroud":
					g.MutablePermanent(q.ID()).GrantBaseAttr(core.Shroud)
				case "source-leaves":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				}
				if name == "War Barge" && change != "source-leaves" {
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				} else if name != "War Barge" {
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				}
				if len(g.StackBoundPermanentSources()) != 0 {
					t.Fatal("pending frame entered stack category")
				}
				g.PutTriggersOnStack()
				got := g.StackBoundPermanentSources()
				if g.StackSize() != 1 || len(got) != 1 || !got[0].SameObject(original) {
					t.Fatal("real RON saved target frame is absent")
				}
				if len(g.StackPeek().TargetSpecs) != 0 {
					t.Fatal("saved reference became a new target")
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				filter, _ := chosenSourceFilter(t, "LEB", "Circle of Protection: Blue")
				chosen := mage.BindDamageSource("delayed frame choice")
				fixture := mage.NewArtifact("Choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(
					mage.ChooseDamageSourceFromStackBindings(mage.You, filter, chosen),
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
					t.Fatal("filtered source choice or damage failed")
				}
				if want == 1 {
					controller := a.PlayerID()
					if change == "control" {
						controller = b.PlayerID()
					}
					if !chooser.source.SameObject(original) || chooser.source.Controller() != controller {
						t.Fatal("choice followed a returned object or old control")
					}
				}
				for _, game := range []*mage.Game{g, clone} {
					if name == "Goblin Kites" {
						game.SetCoinFlipResults([]bool{false, false})
					}
					game.ResolveStack()
					survives := change == "returned" || change == "phased" || (name == "Goblin Kites" && change == "control")
					if (game.FindPermanentIncludingPhased(q.ID()) != nil) != survives || len(game.StackBoundPermanentSources()) != 0 {
						t.Fatal("delayed body or reference lifetime failed")
					}
					if game.GetPlayer(a.PlayerID()).Life() != 20-2*want {
						t.Fatal("clone changed source choice")
					}
				}
			})
		}
	}
}

func TestRONStackDelayedPermanentSourceInstances(t *testing.T) {
	g, a, b := ronGame()
	originals := make([]mage.DamageSource, 0, 2)
	ids := make([]uuid.UUID, 0, 2)
	declarations := make([]mage.Target, 0, 2)
	for _, player := range []*mage.BasePlayer{a, b} {
		card := repositoryCard(t, "FEM", "Goblin Kites")
		card.SetOwner(player.PlayerID())
		declarations = append(declarations, card.Abilities()[0].(*mage.ActionDefinition).Targets()[0])
		p := g.PutOnBattlefield(card, player.PlayerID())
		targetCard := repositoryCard(t, "LEB", "Prodigal Sorcerer")
		targetCard.SetOwner(player.PlayerID())
		q := g.PutOnBattlefield(targetCard, player.PlayerID())
		originals = append(originals, g.CaptureDamageSource(q.ID()))
		ids = append(ids, q.ID())
		player.ManaPool().Add(core.Red, 1)
		if err := g.ActivateAbilityByIndex(player.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if declarations[0] == declarations[1] {
		t.Fatal("factory instances share target declarations")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	got := g.StackBoundPermanentSources()
	if len(got) != 2 || g.StackSize() != 2 || g.StackObjects()[0].Controller != a.PlayerID() || g.StackObjects()[1].Controller != b.PlayerID() {
		t.Fatal("independent delayed frames or APNAP placement failed")
	}
	for _, source := range originals {
		if !slices.ContainsFunc(got, source.SameObject) {
			t.Fatal("one instance replaced the other frame")
		}
	}
	clone := g.Clone()
	g.CounterStackObject(g.StackPeek().ID)
	if got := g.StackBoundPermanentSources(); len(got) != 1 || !got[0].SameObject(originals[0]) {
		t.Fatal("countering removed the wrong saved frame")
	}
	g.SetCoinFlipResults([]bool{false})
	clone.SetCoinFlipResults([]bool{false, false})
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(ids[0]) != nil || g.FindPermanent(ids[1]) == nil || clone.FindPermanent(ids[0]) != nil || clone.FindPermanent(ids[1]) != nil {
		t.Fatal("independent sacrifice bodies or clone isolation failed")
	}
}

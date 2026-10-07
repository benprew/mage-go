package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStackBoundSpellSourcePrerequisite(t *testing.T) {
	for _, name := range []string{"Presence of the Master", "Nether Void"} {
		for _, change := range []string{"live", "countered", "recast", "color", "control", "copy", "departed", "phased"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				_, a, b := ronGame()
				chooser := &ronSourceChoicePlayer{BasePlayer: b}
				g := mage.NewGame(a, chooser)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, "LEG", name)
				card.SetOwner(a.PlayerID())
				observer := g.PutOnBattlefield(card, a.PlayerID())
				spell := repositoryCard(t, "LEB", "Crusade")
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(g.StackBoundSpellSources()) != 0 {
					t.Fatal("unpaid cast rollback")
				}
				a.ManaPool().Add(core.White, 2)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				victim := g.StackPeek()
				saved := g.CaptureSpellDamageSource(victim.ID)
				if len(g.StackBoundSpellSources()) != 0 {
					t.Fatal("pending trigger admitted")
				}
				g.PutTriggersOnStack()
				trigger := g.StackPeek()
				if len(trigger.Targets) != 0 {
					t.Fatal("event binding became a target")
				}
				switch change {
				case "countered", "recast":
					g.CounterStackObject(victim.ID)
					if change == "recast" {
						g.GetStack().RemoveByID(trigger.ID)
						a.RemoveFromGraveyard(spell.ID())
						a.AddToHand(spell)
						a.ManaPool().Add(core.White, 2)
						if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
							t.Fatal(err)
						}
						g.ChangeObjectColor(g.StackPeek().ID, core.Blue)
						g.PushStack(trigger)
					}
				case "color":
					g.ChangeObjectColor(victim.ID, core.Blue)
				case "control":
					victim.Controller = b.PlayerID()
				case "copy":
					if g.CopyStackObjectDirect(trigger, b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "departed":
					g.DestroyPermanent(observer)
				case "phased":
					g.PhaseOut(observer.ID())
				}
				got := g.StackBoundSpellSources()
				if len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("exact event identity or duplicate copy")
				}
				filter, _ := chosenSourceFilter(t, "LEB", "Circle of Protection: White")
				chosen := mage.BindDamageSource("chosen")
				fixture := mage.NewArtifact("Choice fixture", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(mage.ChooseDamageSourceFromStackSpellBindings(mage.You, filter, chosen), mage.DealDamageFrom(mage.FromDamageSource(chosen), mage.ActivePlayer, mage.Fixed(2))))))
				if err := mage.ValidateCardDefinition(fixture.Name(), fixture); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(fixture, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if chooser.calls != 0 {
					t.Fatal("choice before resolution")
				}
				clone := g.Clone()
				g.ResolveTopOfStack()
				want := 1
				if change == "color" {
					want = 0
				}
				if chooser.calls != want || a.Life() != 20-2*want {
					t.Fatal("filtered resolution choice or actual damage")
				}
				if want == 1 && !chooser.source.SameObject(saved) {
					t.Fatal("recast replaced saved source")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone changed")
				}
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if game.GetPlayer(a.PlayerID()).Life() != 20-2*want || game.StackSize() != 0 || len(game.StackBoundSpellSources()) != 0 || game.FindPermanent(spell.ID()) != nil {
						t.Fatal("native RON counter or clone lifetime")
					}
				}
			})
		}
	}
}

func TestRONStackBoundSpellSourceIndependentEvents(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	for _, player := range []*mage.BasePlayer{a, b} {
		observer := repositoryCard(t, "LEG", "Nether Void")
		observer.SetOwner(player.PlayerID())
		g.PutOnBattlefield(observer, player.PlayerID())
	}
	spells := make([]mage.DamageSource, 0, 2)
	for range 2 {
		card := repositoryCard(t, "LEB", "Lightning Bolt")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Red, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		spells = append(spells, g.CaptureSpellDamageSource(g.StackPeek().ID))
		g.PutTriggersOnStack()
	}
	got := g.StackBoundSpellSources()
	if len(got) != 2 || !got[0].SameObject(spells[0]) || !got[1].SameObject(spells[1]) || g.StackSize() != 6 {
		t.Fatal("factory/event identity or deduplication")
	}
	clone := g.Clone()
	for range 2 {
		g.ResolveTopOfStack()
	}
	got = g.StackBoundSpellSources()
	if len(got) != 1 || !got[0].SameObject(spells[0]) || len(clone.StackBoundSpellSources()) != 2 {
		t.Fatal("second event replaced first or clone changed")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(b.PlayerID()).Life() != 20 || len(game.StackBoundSpellSources()) != 0 || game.StackSize() != 0 {
			t.Fatal("native counter/event or clone separation")
		}
	}
}

package ron

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTargetingAbilityGameplay(t *testing.T) {
	for _, name := range []string{"Ali Baba", "Goblin Digging Team"} {
		t.Run(name, func(t *testing.T) {
			set := "ARN"
			if name == "Goblin Digging Team" {
				set = "DRK"
			}
			g, a, b := ronGame()
			card := repositoryCard(t, set, name)
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.GrantBaseAttr(core.Haste)
			ability := mage.UnwrapAbility(source.RuntimeAbilities[0]).(mage.ActivatedAbility)
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wall.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			for _, view := range []mage.Card{obj.TargetSource, g.Clone().StackPeek().TargetSource, g.CopyStackObjectDirect(obj, a.PlayerID(), false).TargetSource} {
				id, declared, known := mage.TargetSourceAbility(view)
				if !known || id != ability.AbilityID() || len(declared) != 1 || !slices.Equal(mage.TargetRequiredSubtypes(declared[0]), []string{"Wall"}) {
					t.Fatal("RON activation lost identity or declaration")
				}
			}
			g.ResolveStack()
			if name == "Ali Baba" && !g.FindPermanent(wall.ID()).Tapped {
				t.Fatal("tap failed")
			}
			if name == "Goblin Digging Team" && (g.FindPermanent(wall.ID()) != nil || g.FindPermanent(source.ID()) != nil) {
				t.Fatal("sacrifice or destruction failed")
			}
		})
	}
}

func TestRONTargetingAbilityZeroGroup(t *testing.T) {
	data := `Card(Name:"Zero",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([ManaCostGeneric(0)]),Targeted([NumberTargetPermanents(Integer(0),IsCreatureType(Wall))],ActionList([TapEachPermanent(Ref_TargetPermanents)])))])`
	registry, err := LoadFS(fstest.MapFS{"zero.ron": {Data: []byte(data)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	card, err := registry.CreateCard("Zero")
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(card, a.PlayerID())
	ability := mage.UnwrapAbility(source.RuntimeAbilities[0]).(mage.ActivatedAbility)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	if len(obj.TargetSpecs) != 0 {
		t.Fatal("zero group acquired chosen occurrences")
	}
	for _, view := range []mage.Card{obj.TargetSource, g.Clone().StackPeek().TargetSource, g.CopyStackObjectDirect(obj, a.PlayerID(), true).TargetSource} {
		id, declared, known := mage.TargetSourceAbility(view)
		if !known || id != ability.AbilityID() || len(declared) != 1 || declared[0].Max() != 0 || !slices.Equal(mage.TargetRequiredSubtypes(declared[0]), []string{"Wall"}) {
			t.Fatal("zero group declaration lost")
		}
	}
	g.ResolveStack()
	for _, bad := range []string{"Integer(-1)", "Integer()", "ValueX"} {
		if reg, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(strings.Replace(data, "Integer(0)", bad, 1))}}, "*.ron"); err == nil || reg != nil {
			t.Fatalf("invalid activation declaration %s", bad)
		}
	}
}

func TestRONTargetingAbilityModal(t *testing.T) {
	for _, mode := range []int{0, 1} {
		g, a, b := ronModalGame(mode)
		card := repositoryCard(t, "FEM", "Balm of Restoration")
		source := g.PutOnBattlefield(card, a.PlayerID())
		ability := mage.UnwrapAbility(source.RuntimeAbilities[0]).(mage.ActivatedAbility)
		a.ManaPool().Add(core.Colorless, 1)
		a.targets = []uuid.UUID{b.PlayerID()}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		id, declared, known := mage.TargetSourceAbility(g.StackPeek().TargetSource)
		if !known || id != ability.AbilityID() || len(declared) != mode {
			t.Fatal("selected mode declaration lost")
		}
		if g.FindPermanent(source.ID()) != nil {
			t.Fatal("source sacrifice failed")
		}
		clone := g.Clone()
		copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
		if copied == nil {
			t.Fatal("copied failed")
		}
		for _, view := range []mage.Card{clone.StackPeek().TargetSource, copied.TargetSource} {
			copiedID, copiedDeclared, ok := mage.TargetSourceAbility(view)
			if !ok || copiedID != id || len(copiedDeclared) != mode {
				t.Fatal("copied mode lost")
			}
		}
		g.ResolveStack()
		if mode == 0 && a.Life() != 24 {
			t.Fatal("life mode failed")
		}
		if mode == 1 {
			g.DealDamageToPlayer(b, 3, uuid.Nil)
			if b.Life() != 20 {
				t.Fatal("prevention mode failed")
			}
		}
	}
}

func TestRONTargetingAbilityTrigger(t *testing.T) {
	for _, change := range []string{"present", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			card := repositoryCard(t, "LEG", "The Abyss")
			source := g.PutOnBattlefield(card, a.PlayerID())
			ability := mage.UnwrapAbility(source.RuntimeAbilities[0]).(mage.TriggeredAbility)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 3), a.PlayerID())
			a.requested = []uuid.UUID{victim.ID()}
			g.SetActivePlayerIndex(0)
			g.SetStep(core.Upkeep)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			clone := g.Clone()
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for _, game := range []*mage.Game{g, clone} {
				game.PutTriggersOnStack()
				id, declared, known := mage.TargetSourceAbility(game.StackPeek().TargetSource)
				if !known || id != ability.AbilityID() || len(declared) != 1 {
					t.Fatal("pending RON trigger lost identity")
				}
				game.ResolveStack()
				if game.FindPermanent(victim.ID()) != nil {
					t.Fatal("pending trigger lost destruction")
				}
			}
		})
	}
}

func TestRONTargetingAbilityRecords(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"ARN", "Ali Baba"}, {"LEB", "Dwarven Demolition Team"}, {"FEM", "Dwarven Lieutenant"},
		{"ARN", "Elephant Graveyard"}, {"FEM", "Fungal Bloom"}, {"DRK", "Goblin Digging Team"},
		{"DRK", "Goblin Wizard"}, {"FEM", "Icatian Lieutenant"}, {"ATQ", "Mishra's Factory"},
		{"FEM", "Orcish Captain"}, {"FEM", "Thrull Champion"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, tc.set, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			g, _, _ := ronGame()
			for i, ability := range first.Abilities() {
				if activated, ok := mage.UnwrapAbility(ability).(mage.ActivatedAbility); ok {
					id, declared, known := mage.TargetSourceAbility(g.AbilityTargetSource(first, activated, activated.Targets()))
					if !known || id != activated.AbilityID() || len(declared) != len(activated.Targets()) || id == mage.UnwrapAbility(second.Abilities()[i]).AbilityID() {
						t.Fatal("factory declarations or identities were shared")
					}
					for j, target := range declared {
						if target != activated.Targets()[j] {
							t.Fatal("target group replaced")
						}
					}
				}
			}
		})
	}
	for _, tc := range []struct{ set, name string }{{"LEG", "Wall of Shadows"}, {"DRK", "Reflecting Mirror"}, {"LEG", "Kry Shield"}} {
		if _, err := Compile(staticBaseRecord(t, tc.set, tc.name)); err == nil {
			t.Fatalf("published incomplete %s", tc.name)
		}
	}
}

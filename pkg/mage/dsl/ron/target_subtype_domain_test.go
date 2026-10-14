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

func TestRONTargetSubtypeDomainGameplay(t *testing.T) {
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), b.PlayerID())
	spell := repositoryCard(t, "LEB", "Tunnel")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	spec := spell.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
	if got := mage.TargetRequiredSubtypes(spec); !slices.Equal(got, []string{"Wall"}) {
		t.Fatalf("Tunnel domain = %v", got)
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{other.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("invalid target spent resources")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	clone := g.Clone()
	copiedSpell := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copiedSpell == nil || !slices.Equal(mage.TargetRequiredSubtypes(copiedSpell.TargetSpecs[0]), []string{"Wall"}) || !slices.Equal(mage.TargetRequiredSubtypes(clone.StackPeek().TargetSpecs[0]), []string{"Wall"}) {
		t.Fatal("copy or clone lost the domain")
	}
	g.ResolveStack()
	if g.FindPermanent(wall.ID()) != nil || g.FindPermanent(other.ID()) == nil || clone.FindPermanent(wall.ID()) == nil {
		t.Fatal("Tunnel destruction or clone isolation")
	}
	clone.ResolveStack()
	if clone.FindPermanent(wall.ID()) != nil {
		t.Fatal("clone lost destruction")
	}
}

func TestRONTargetSubtypeDomainConjunction(t *testing.T) {
	for _, name := range []string{"Glyph of Doom", "Glyph of Life", "Glyph of Destruction"} {
		card := repositoryCard(t, "LEG", name)
		for _, ability := range card.Abilities() {
			if action, ok := ability.(*mage.ActionDefinition); ok && len(action.Targets()) > 0 {
				if got := mage.TargetRequiredSubtypes(action.Targets()[0]); !slices.Equal(got, []string{"Wall"}) {
					t.Fatalf("%s domain = %v", name, got)
				}
			}
		}
	}
}

func TestRONTargetSubtypeDomainRecords(t *testing.T) {
	for _, tc := range []struct{ set, name, subtype string }{
		{"ARN", "Ali Baba", "Wall"},
		{"LEB", "Dwarven Demolition Team", "Wall"},
		{"FEM", "Dwarven Lieutenant", "Dwarf"},
		{"ARN", "Elephant Graveyard", "Elephant"},
		{"FEM", "Fungal Bloom", "Fungus"},
		{"LEG", "Glyph of Destruction", "Wall"},
		{"LEG", "Glyph of Doom", "Wall"},
		{"LEG", "Glyph of Life", "Wall"},
		{"DRK", "Goblin Digging Team", "Wall"},
		{"DRK", "Goblin Wizard", "Goblin"},
		{"FEM", "Icatian Lieutenant", "Soldier"},
		{"ATQ", "Mishra's Factory", "Assembly-Worker"},
		{"FEM", "Orcish Captain", "Orc"},
		{"FEM", "Thrull Champion", "Thrull"},
		{"LEB", "Tunnel", "Wall"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := staticBaseRecord(t, tc.set, tc.name)
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			found := false
			for i, ability := range first.Abilities() {
				if action, ok := ability.(*mage.ActionDefinition); ok {
					for j, target := range action.Targets() {
						if slices.Contains(mage.TargetRequiredSubtypes(target), tc.subtype) {
							found = true
							if !mage.TargetRequiresSubtype(target, tc.subtype) {
								t.Fatal("missing subtype proof")
							}
							result := mage.TargetRequiredSubtypes(target)
							result[0] = "Changed"
							if !slices.Contains(mage.TargetRequiredSubtypes(second.Abilities()[i].(*mage.ActionDefinition).Targets()[j]), tc.subtype) {
								t.Fatal("factory instances shared domain state")
							}
						}
					}
				}
			}
			if !found {
				t.Fatal("missing domain")
			}
		})
	}
	for _, name := range []string{"Glyph of Delusion", "Glyph of Reincarnation"} {
		if _, err := Compile(staticBaseRecord(t, "LEG", name)); err == nil {
			t.Fatalf("published incomplete %s", name)
		}
	}
}

func TestRONTargetSubtypeDomainInvalidAndBoundaries(t *testing.T) {
	base := `Card(Name:"Tap",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCreatureType(Wall))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
	for _, filter := range []string{"IsCreatureType", "IsCreatureType()", "IsCreatureType(Wall,Elf)", "IsCreatureType(Wall())", `IsCreatureType("Wall")`} {
		data := strings.Replace(base, "IsCreatureType(Wall)", filter, 1)
		if registry, err := LoadFS(fstest.MapFS{"valid.ron": {Data: []byte(base)}, "invalid.ron": {Data: []byte(data)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("published invalid declaration %s", filter)
		}
	}
	for _, filter := range []string{"IsNonCreatureType(Wall)", "Or([IsCreatureType(Wall),IsCreatureType(Elf)])", "AnyPermanent"} {
		data := strings.Replace(base, "IsCreatureType(Wall)", filter, 1)
		nodes, err := Parse(strings.NewReader(data), "boundary.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		if got := mage.TargetRequiredSubtypes(factory().Abilities()[0].(*mage.ActionDefinition).Targets()[0]); len(got) != 0 {
			t.Fatalf("inferred a conjunctive domain for %s: %v", filter, got)
		}
	}
	for _, invalid := range []string{
		strings.Replace(base, "TargetPermanent(IsCreatureType(Wall))", "TargetSpell(IsCreatureType(Wall))", 1),
		strings.Replace(base, "TapPermanent(Ref_TargetPermanent)", "IsCreatureType(Wall)", 1),
	} {
		nodes, err := Parse(strings.NewReader(invalid), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted incompatible context")
		}
	}
}

func TestRONTargetSubtypeDomainActivation(t *testing.T) {
	for _, scenario := range []string{"present", "returned", "phased", "phase-cycle"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Goblin Digging Team"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{other.ID()}); err == nil || g.FindPermanent(source.ID()) == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("illegal target paid sacrifice or tap cost")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wall.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()) != nil || !slices.Equal(mage.TargetRequiredSubtypes(g.StackPeek().TargetSpecs[0]), []string{"Wall"}) {
				t.Fatal("sacrificed source lost its target declaration")
			}
			switch scenario {
			case "returned":
				g.RemoveFromBattlefield(wall)
				g.PutOnBattlefield(wall.Card, b.PlayerID())
			case "phased":
				g.PhaseOut(wall.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(wall.ID()))
			}
			g.ResolveStack()
			if (g.FindPermanentIncludingPhased(wall.ID()) == nil) != (scenario == "present" || scenario == "phase-cycle") || g.FindPermanent(other.ID()) == nil {
				t.Fatal("activation target identity or resolution")
			}
		})
	}
}

func TestRONTargetSubtypeDomainResolutionRecheck(t *testing.T) {
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
	spell := repositoryCard(t, "LEB", "Tunnel")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	g.AddContinuousEffect(mage.SetSubtypes(wall.ID(), mage.SubtypeCreature, core.Indefinite, "Elf"))
	if !slices.Equal(mage.TargetRequiredSubtypes(obj.TargetSpecs[0]), []string{"Wall"}) {
		t.Fatal("live subtype change replaced the declared domain")
	}
	g.ResolveStack()
	if g.FindPermanent(wall.ID()) == nil {
		t.Fatal("resolution failed to recheck the enforced subtype")
	}
}

func TestRONTargetSubtypeDomainCountDeclarations(t *testing.T) {
	for _, count := range []string{"Integer(0)", "Integer(2)", "ValueX"} {
		t.Run(count, func(t *testing.T) {
			data := `Card(Name:"Walls",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(Targeted([NumberTargetPermanents(` + count + `,And([IsCreatureType(Wall),IsCardtype(Creature)]))],ActionList([TapEachPermanent(Ref_TargetPermanents)])))])`
			nodes, err := Parse(strings.NewReader(data), "count.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(mage.TargetRequiredSubtypes(card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]), []string{"Wall"}) {
				t.Fatal("count declaration lost subtype constraint")
			}
			g, a, _ := ronGame()
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), a.PlayerID())
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			var targets []uuid.UUID
			x := 0
			if count != "Integer(0)" {
				targets = []uuid.UUID{wall.ID(), second.ID()}
			}
			if count == "ValueX" {
				x = 2
				a.ManaPool().Add(core.Colorless, 2)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, x); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(wall.ID()).Tapped != (len(targets) > 0) || g.FindPermanent(second.ID()).Tapped != (len(targets) > 0) {
				t.Fatal("counted target execution")
			}
		})
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTargetDomainProof(t *testing.T) {
	for _, tc := range []struct {
		filter string
		want   bool
	}{
		{`IsCreatureType(Wall)`, true},
		{`Or([And([IsCreatureType(Wall),IsColor(Red)]),And([IsCreatureType(Wall),IsCardtype(Creature)])])`, true},
		{`Or([IsCreatureType(Wall),IsCreatureType(Elf)])`, false},
		{`And([Or([IsCreatureType(Wall),IsCreatureType(Elf)]),IsNonCreatureType(Elf)])`, true},
		{`And([IsCreatureType(Wall),IsNonCreatureType(Wall)])`, false},
		{`IsCardtype(Creature)`, false},
	} {
		t.Run(tc.filter, func(t *testing.T) {
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(0)]),Targeted([TargetPermanent(`+tc.filter+`)],ActionList([TapPermanent(Ref_TargetPermanent)])))`)()
			spec := card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
			if mage.TargetRequiresSubtype(spec, "Wall") != tc.want {
				t.Fatal("incorrect declaration proof")
			}
		})
	}
	for _, filter := range []string{`Or([])`, `And([])`, `Or([IsCreatureType(Wall),IsCreatureType(Wall())])`, `Or([IsCreatureType(Wall),IsCreatureType("Elf")])`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
		valid := strings.Replace(src, `Name:"Bad"`, `Name:"Good"`, 1)
		valid = strings.Replace(valid, filter, `IsCreatureType(Wall)`, 1)
		if registry, err := LoadFS(fstest.MapFS{"good.ron": {Data: []byte(valid)}, "bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("published invalid domain %s", filter)
		}
	}
}

func TestRONTargetDomainProofAura(t *testing.T) {
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 5, mage.WithSubTypes("Wall")), b.PlayerID())
	elf := g.PutOnBattlefield(mage.NewCreature("Elf", "", 1, 5, mage.WithSubTypes("Elf")), b.PlayerID())
	card := repositoryCard(t, "LEB", "Animate Wall")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if !mage.TargetRequiresSubtype(card.CastTargets()[0], "Wall") {
		t.Fatal("Aura enchant domain absent")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{elf.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("illegal Aura spent resources")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	clone := g.Clone()
	copied := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copied == nil || !mage.TargetRequiresSubtype(copied.TargetSpecs[0], "Wall") || !mage.TargetRequiresSubtype(clone.StackPeek().TargetSpecs[0], "Wall") {
		t.Fatal("copy or clone lost Aura domain")
	}
	g.ResolveStack()
	if len(g.FindPermanent(wall.ID()).Attachments) == 0 || !g.FindPermanent(wall.ID()).HasAttr(core.AttrCanAttack) {
		t.Fatal("Aura did not attach")
	}
	g.AddContinuousEffect(mage.SetSubtypes(wall.ID(), mage.SubtypeCreature, core.Indefinite, "Elf"))
	g.CheckStateBasedActions()
	if len(g.FindPermanent(wall.ID()).Attachments) != 0 || len(clone.FindPermanent(wall.ID()).Attachments) != 0 {
		t.Fatal("attachment legality or clone isolation")
	}
	clone.ResolveStack()
	if len(clone.FindPermanent(wall.ID()).Attachments) == 0 || !clone.FindPermanent(wall.ID()).HasAttr(core.AttrCanAttack) {
		t.Fatal("clone lost attachment")
	}
}

func TestRONTargetDomainProofUnionGameplay(t *testing.T) {
	for _, scenario := range []string{"present", "efreet", "returned", "phased", "phase-cycle", "subtype-change"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ARN", "King Suleiman"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			subtype := "Djinn"
			if scenario == "efreet" {
				subtype = "Efreet"
			}
			target := g.PutOnBattlefield(mage.NewCreature(subtype, "", 1, 5, mage.WithSubTypes(subtype)), b.PlayerID())
			efreet := g.PutOnBattlefield(mage.NewCreature("Efreet", "", 1, 5, mage.WithSubTypes("Efreet")), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), b.PlayerID())
			spec := source.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0]
			for _, subtype := range []string{"Wall", "Djinn", "Efreet"} {
				if mage.TargetRequiresSubtype(spec, subtype) {
					t.Fatal("union became a single-subtype declaration")
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{other.ID()}); err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
				t.Fatal("invalid union target paid cost")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			switch scenario {
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				g.PutOnBattlefield(target.Card, b.PlayerID())
			case "phased":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "subtype-change":
				g.AddContinuousEffect(mage.SetSubtypes(target.ID(), mage.SubtypeCreature, core.Indefinite, "Wall"))
			}
			clone := g.Clone()
			g.ResolveStack()
			destroyed := scenario == "present" || scenario == "efreet" || scenario == "phase-cycle"
			if (g.FindPermanentIncludingPhased(target.ID()) == nil) != destroyed || g.FindPermanent(efreet.ID()) == nil || clone.FindPermanentIncludingPhased(target.ID()) == nil {
				t.Fatal("union resolution identity, legality or clone isolation")
			}
			clone.ResolveStack()
			if (clone.FindPermanentIncludingPhased(target.ID()) == nil) != destroyed {
				t.Fatal("clone lost union predicate")
			}
		})
	}
}

func TestRONTargetDomainProofAuraShapes(t *testing.T) {
	for _, filter := range []string{`Or([IsCreatureType(Wall),And([IsCreatureType(Wall),IsColor(Red)])])`, `And([Or([IsCreatureType(Wall),IsCreatureType(Elf)]),IsNonCreatureType(Elf)])`} {
		src := `Card(Name:"Aura",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(` + filter + `),PermanentLayerEffect(HostPermanent,[AdjustPT(1,1)])])`
		nodes, err := Parse(strings.NewReader(src), "aura.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		first, second := factory(), factory()
		if !mage.TargetRequiresSubtype(first.CastTargets()[0], "Wall") || first.CastTargets()[0] == second.CastTargets()[0] {
			t.Fatal("Aura domain or factory isolation")
		}
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
	}
}

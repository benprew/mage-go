package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func attachRONTestAura(t *testing.T, g *mage.Game, host *mage.Permanent, player uuid.UUID) *mage.Permanent {
	t.Helper()
	card := repositoryCard(t, "LEB", "Holy Strength")
	card.SetOwner(player)
	aura := g.PutOnBattlefield(card, player)
	g.Attach(aura.ID(), host.ID())
	return aura
}

func TestRONEnchantedFilterRamses(t *testing.T) {
	for _, change := range []string{"none", "aura-leaves", "aura-phases", "target-phases", "target-returns", "control", "source-leaves", "clone", "copy", "regenerate"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			ramsesCard := repositoryCard(t, "LEG", "Ramses Overdark")
			ramsesCard.SetOwner(a.PlayerID())
			ramses := g.PutOnBattlefield(ramsesCard, a.PlayerID())
			ramses.RevokeBaseAttr(core.AttrSummonSick)
			hostCard := mage.NewCreature("Host", "", 2, 6)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			aura := attachRONTestAura(t, g, host, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), ramses.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
				t.Fatal(err)
			}
			survives := false
			switch change {
			case "aura-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				survives = true
			case "aura-phases":
				g.PhaseOut(aura.ID())
				survives = true
			case "target-phases":
				g.PhaseOut(host.ID())
				survives = true
			case "target-returns":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				host = g.PutOnBattlefield(hostCard, b.PlayerID())
				attachRONTestAura(t, g, host, a.PlayerID())
				survives = true
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: ramses.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(ramses.ID()))
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
				clone.ResolveStack()
				if clone.FindPermanent(host.ID()) == nil || g.FindPermanent(host.ID()) == nil {
					t.Fatal("clone changed original or lost its illegal target")
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "regenerate":
				g.AddRegenerationShield(host.ID())
				survives = true
			}
			g.ResolveStack()
			if change == "target-phases" {
				g.PhaseIn([]uuid.UUID{host.ID()})
			}
			if (g.FindPermanent(host.ID()) != nil) != survives {
				t.Fatalf("survival differs for %s", change)
			}
		})
	}
}

func TestRONEnchantedFilterLegalityRollback(t *testing.T) {
	for _, kind := range []string{"plain", "equipment", "phased-aura", "shroud", "noncreature", "sick"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ramses Overdark"), a.PlayerID())
			if kind != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			card := mage.NewCreature("Host", "", 2, 6)
			if kind == "noncreature" {
				card = mage.NewArtifact("Host", "")
			}
			host := g.PutOnBattlefield(card, b.PlayerID())
			if kind == "equipment" {
				equipment := g.PutOnBattlefield(mage.NewArtifact("Equipment", "", mage.WithSubTypes("Equipment")), b.PlayerID())
				g.Attach(equipment.ID(), host.ID())
			} else if kind != "plain" {
				aura := attachRONTestAura(t, g, host, b.PlayerID())
				if kind == "phased-aura" {
					g.PhaseOut(aura.ID())
				}
			}
			if kind == "shroud" {
				host.GrantBaseAttr(core.Shroud)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{host.ID()}); err == nil {
				t.Fatal("illegal activation accepted")
			}
			if g.FindPermanent(source.ID()).Tapped || g.StackPeek() != nil {
				t.Fatal("failed activation changed game")
			}
		})
	}
}

func TestRONEnchantedFilterWall(t *testing.T) {
	for _, change := range []string{"enchanted", "plain", "equipment", "aura-leaves", "aura-phases", "control", "phase-cycle", "ability-removal", "clone", "source-leaves", "source-returns"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Putrid Flesh"), a.PlayerID())
			sourceCard := mage.NewCreature("Source", "{B}", 1, 6)
			sourceCard.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(sourceCard, b.PlayerID())
			var aura *mage.Permanent
			if change != "plain" && change != "equipment" {
				aura = attachRONTestAura(t, g, source, a.PlayerID())
			}
			if change == "equipment" {
				equipment := g.PutOnBattlefield(mage.NewArtifact("Equipment", "", mage.WithSubTypes("Equipment")), b.PlayerID())
				g.Attach(equipment.ID(), source.ID())
			}
			prevented := change != "plain" && change != "equipment"
			switch change {
			case "aura-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				prevented = false
			case "aura-phases":
				g.PhaseOut(aura.ID())
				prevented = false
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "ability-removal":
				removal := mage.RemoveAllAbilities(wall.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(source.ID())
				g.AddContinuousEffect(removal)
				prevented = false
			}
			captured := g.CaptureDamageSource(source.ID())
			if change == "source-leaves" || change == "source-returns" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "source-returns" {
					g.PutOnBattlefield(sourceCard, a.PlayerID())
				}
			}
			if change == "clone" {
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
				clone.DealSimultaneousDamage(mage.PermanentDamageAssignment(clone.FindPermanent(wall.ID()), 1, captured))
				if clone.FindPermanent(wall.ID()).Damage != 1 || g.FindPermanent(wall.ID()).Damage != 0 {
					t.Fatal("clone isolation")
				}
			}
			for range 2 {
				g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(wall.ID()), 1, captured), mage.PlayerDamageAssignment(a, 1, captured))
			}
			want := 2
			if prevented {
				want = 0
			}
			if g.FindPermanent(wall.ID()).Damage != want || a.Life() != 18 {
				t.Fatalf("damage=%d want=%d life=%d", g.FindPermanent(wall.ID()).Damage, want, a.Life())
			}
		})
	}
}

func TestRONEnchantedFilterDeclarations(t *testing.T) {
	for _, filter := range []string{"IsEnchanted()", "IsEnchanted(AnyPermanent)", "IsEnchanted(ThisPermanent)", "IsEnchanted(You)", "IsEnchanted(1)"} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(`+filter+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
	for _, filter := range []string{"IsEnchanted", "And([IsEnchanted,IsCardtype(Creature)])", "Or([IsEnchanted,IsTapped])"} {
		nodes, err := Parse(strings.NewReader(filter), "filter.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = compileFilter(nodes[0]); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONEnchantedFilterWallCombatAndOtherRules(t *testing.T) {
	for _, enchanted := range []bool{false, true} {
		t.Run(fmt.Sprint(enchanted), func(t *testing.T) {
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Putrid Flesh"), a.PlayerID())
			wall.RevokeBaseAttr(core.AttrSummonSick)
			if wall.CanDeclareAsAttacker(g) {
				t.Fatal("defender can attack")
			}
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "{B}", 1, 10), b.PlayerID())
			if enchanted {
				attachRONTestAura(t, g, attacker, b.PlayerID())
			}
			g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
			g.GetCombat().AddBlocker(wall.ID(), attacker.ID())
			g.ExecuteCombatDamage()
			want := 1
			if enchanted {
				want = 0
			}
			if g.FindPermanent(wall.ID()).Damage != want || g.FindPermanent(attacker.ID()).Damage != 2 {
				t.Fatal("combat prevention or wall power")
			}
			white := g.PutOnBattlefield(mage.NewCreature("White", "{W}", 1, 10), b.PlayerID())
			g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(wall.ID()), 1, g.CaptureDamageSource(white.ID())))
			if g.FindPermanent(wall.ID()).Damage != want {
				t.Fatal("protection from white")
			}
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			bolt.SetOwner(b.PlayerID())
			b.AddToHand(bolt)
			b.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if (g.FindPermanent(wall.ID()) != nil) != enchanted {
				t.Fatal("permanent prevention changed spell damage")
			}
		})
	}
}

func TestRONEnchantedFilterReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Putrid Flesh"), b.PlayerID())
		source := g.PutOnBattlefield(mage.NewCreature("Source", "{B}", 1, 10), a.PlayerID())
		attachRONTestAura(t, g, source, a.PlayerID())
		g.SetCreatureDamageRedirect(wall.ID(), b.PlayerID())
		g.DealSimultaneousDamage(mage.PermanentDamageAssignment(wall, 1, g.CaptureDamageSource(source.ID())))
		want := 20
		if mode == 0 {
			want = 19
		}
		if g.FindPermanent(wall.ID()).Damage != 0 || b.Life() != want || b.prompts != 1 {
			t.Fatal("prevention order or recipient recheck")
		}
	}
}

func TestRONEnchantedFilterStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name != "Ramses Overdark" && name != "Wall of Putrid Flesh" {
			continue
		}
		lines := strings.Split(string(data), "\n")
		files[name+".ron"] = &fstest.MapFile{Data: []byte(lines[node.Position.Line-1])}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("strict load count")
	}
	for _, name := range registry.RegisteredCardNames() {
		g, a, _ := ronGame()
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green, core.Colorless} {
			a.ManaPool().Add(color, 8)
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(card.ID()) == nil {
			t.Fatalf("%s did not resolve", name)
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, entry := range report.Signatures {
		if entry.Name == "IsEnchanted" {
			if entry.Role != "permanent_filter" {
				t.Fatal("filter inventory")
			}
			for _, name := range entry.Cards {
				found[name] = true
			}
		}
	}
	if len(found) != 2 {
		t.Fatal("missing inventory filter")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}

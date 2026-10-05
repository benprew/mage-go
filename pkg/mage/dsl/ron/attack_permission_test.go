package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONAttackPermissionRealCards(t *testing.T) {
	for _, name := range []string{"Animate Wall", "Instill Energy"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Wall", "", 2, 4, mage.WithSubTypes("Wall"), mage.WithKeyword(core.Defender))
			if name == "Instill Energy" {
				hostCard = mage.NewCreature("Host", "", 2, 4)
			}
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			if name == "Animate Wall" {
				g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			card := repositoryCard(t, "LEB", name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			color := core.White
			if name == "Instill Energy" {
				color = core.Green
			}
			a.ManaPool().Add(color, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("permission before resolution")
			}
			g.ResolveStack()
			if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("missing permission")
			}
			if name == "Instill Energy" {
				if g.FindPermanent(host.ID()).CanTapForEffect(g) || g.FindPermanent(host.ID()).HasKeyword(core.Haste) {
					t.Fatal("attack permission granted full haste")
				}
				g.TapPermanent(g.FindPermanent(host.ID()))
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(host.ID()).Tapped {
					t.Fatal("untap before resolution")
				}
				g.ResolveStack()
				if g.FindPermanent(host.ID()).Tapped {
					t.Fatal("host not untapped")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 1, nil); err == nil {
					t.Fatal("second use accepted")
				}
				g.SetTurn(2)
				g.SetActivePlayerIndex(1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 1, nil); err == nil {
					t.Fatal("opponent turn accepted")
				}
			}
			clone := g.Clone()
			phased := g.PhaseOut(card.ID())
			g.ApplyContinuousEffects()
			if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("phased source retained permission")
			}
			if !clone.FindPermanent(host.ID()).CanDeclareAsAttacker(clone) {
				t.Fatal("clone changed")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("phase cycle lost permission")
			}
			otherCard := mage.NewCreature("Other", "", 2, 4)
			if name == "Animate Wall" {
				otherCard = mage.NewCreature("Other Wall", "", 2, 4, mage.WithSubTypes("Wall"), mage.WithKeyword(core.Defender))
			}
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			g.Attach(card.ID(), other.ID())
			if name == "Animate Wall" {
				g.MutablePermanent(other.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			g.ApplyContinuousEffects()
			if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) || !g.FindPermanent(other.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("attachment membership")
			}
			g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
			g.ApplyContinuousEffects()
			if g.FindPermanent(other.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("source ability removal retained permission")
			}
		})
	}
}

func TestRONAttackPermissionInvalid(t *testing.T) {
	for _, operation := range []string{"CanAttackAsThoughItDidntHaveDefender", "CanAttackAsThoughItHadHaste"} {
		for _, rule := range []string{
			fmt.Sprintf(`PermanentRuleEffect(HostPermanent,[%s()])`, operation),
			fmt.Sprintf(`PermanentRuleEffect(HostPermanent,[%s(Integer(1))])`, operation),
			fmt.Sprintf(`PermanentRuleEffect(Ref_TargetPermanent,[%s])`, operation),
			fmt.Sprintf(`PermanentRuleEffect(HostPermanent(),[%s])`, operation),
			fmt.Sprintf(`PermanentRuleEffect(HostPermanent,[%s,%s])`, operation, operation),
			fmt.Sprintf(`EachPermanentRuleEffect(AnyPermanent,[%s])`, operation),
		} {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),`+rule+`])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s", rule)
			}
		}
		assertConditionalRejected(t, fmt.Sprintf(`PermanentRuleEffect(HostPermanent,[%s])`, operation))
	}
}

func TestRONAttackPermissionLifecycle(t *testing.T) {
	for _, name := range []string{"Animate Wall", "Instill Energy"} {
		for _, change := range []string{"host-abilities", "host-phase", "host-control", "source-control", "source-return", "host-return"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				hostCard := mage.NewCreature("Host", "", 2, 4)
				if name == "Animate Wall" {
					hostCard = mage.NewCreature("Wall", "", 2, 4, mage.WithSubTypes("Wall"), mage.WithKeyword(core.Defender))
				}
				host := g.PutOnBattlefield(hostCard, a.PlayerID())
				if name == "Animate Wall" {
					host.RevokeBaseAttr(core.AttrSummonSick)
				}
				card := repositoryCard(t, "LEB", name)
				source := g.PutOnBattlefield(card, b.PlayerID())
				g.Attach(source.ID(), host.ID())
				switch change {
				case "host-abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.EndOfTurn))
				case "host-phase":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "host-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					if name == "Animate Wall" {
						g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
					}
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: host.ID(), TargetID: source.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					source = g.PutOnBattlefield(card, b.PlayerID())
				case "host-return":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					host = g.PutOnBattlefield(hostCard, a.PlayerID())
					if name == "Animate Wall" {
						host.RevokeBaseAttr(core.AttrSummonSick)
					}
				}
				g.ApplyContinuousEffects()
				want := change != "source-return" && change != "host-return"
				if g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) != want {
					t.Fatal("permission lifecycle", change)
				}
				if change == "source-return" {
					g.Attach(source.ID(), host.ID())
					if !g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
						t.Fatal("fresh attachment missing permission")
					}
				}
			})
		}
	}
}

func TestRONAttackPermissionNativeAttack(t *testing.T) {
	for _, name := range []string{"Animate Wall", "Instill Energy"} {
		a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
		g := mage.NewGame(a, b)
		g.SetTurn(1)
		card := mage.NewCreature("Host", "", 2, 4)
		if name == "Animate Wall" {
			card = mage.NewCreature("Host", "", 2, 4, mage.WithSubTypes("Wall"), mage.WithKeyword(core.Defender))
		}
		host := g.PutOnBattlefield(card, a.PlayerID())
		if name == "Animate Wall" {
			host.RevokeBaseAttr(core.AttrSummonSick)
		}
		a.SetAttackers(1, []string{"Host"})
		g.RunStepWithPriority(core.DeclareAttackers)
		if g.GetCombat().IsAttacking(host.ID()) {
			t.Fatal("attack before permission")
		}
		source := g.PutOnBattlefield(repositoryCard(t, "LEB", name), a.PlayerID())
		g.Attach(source.ID(), host.ID())
		g.RunStepWithPriority(core.DeclareAttackers)
		if !g.GetCombat().IsAttacking(host.ID()) || !g.FindPermanent(host.ID()).Tapped {
			t.Fatal("legal native attack rejected")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		if !g.GetCombat().IsAttacking(host.ID()) {
			t.Fatal("permission removal erased declared attack")
		}
		g.ExecuteCombatDamage()
		if b.Life() != 18 {
			t.Fatal("native combat damage", b.Life())
		}
	}
}

func TestRONAttackPermissionAuraLegality(t *testing.T) {
	for _, name := range []string{"Animate Wall", "Instill Energy"} {
		for _, scenario := range []string{"invalid", "departed", "phased"} {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Wall", "", 2, 4, mage.WithSubTypes("Wall"))
			if scenario == "invalid" {
				if name == "Animate Wall" {
					hostCard = mage.NewCreature("Not Wall", "", 2, 4)
				} else {
					hostCard = mage.NewLand("Land")
				}
			}
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := repositoryCard(t, "LEB", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			color := core.White
			if name == "Instill Energy" {
				color = core.Green
			}
			a.ManaPool().Add(color, 1)
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
			if scenario == "invalid" {
				if err == nil || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("illegal target rollback")
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "departed" {
				g.RemoveFromBattlefield(host)
			} else {
				g.PhaseOut(host.ID())
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("illegal Aura resolved")
			}
		}
	}
}

func TestRONAttackPermissionReprints(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	collection := fstest.MapFS{}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		var records []string
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Animate Wall" && name != "Instill Energy" {
				continue
			}
			count++
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if first.ID() == second.ID() {
				t.Fatal("shared factory identity")
			}
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
			}
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Animate Wall"`) || strings.Contains(line, `Name: "Instill Energy"`) {
				records = append(records, line)
			}
		}
		if len(records) > 0 {
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
		}
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if count != 12 || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprint count", count)
	}
	collection["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unsupported])`)}
	if registry, err := LoadFS(collection, "*.ron"); err == nil || registry != nil {
		t.Fatal("published incomplete collection")
	}
}

func TestRONAttackPermissionSourceInstances(t *testing.T) {
	for _, operation := range []string{"CanAttackAsThoughItDidntHaveDefender", "CanAttackAsThoughItHadHaste"} {
		prefix := ""
		if operation == "CanAttackAsThoughItDidntHaveDefender" {
			prefix = "Defender,"
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[`+prefix+`PermanentRuleEffect(ThisPermanent,[`+operation+`])])`), "source.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		first := g.PutOnBattlefield(factory(), a.PlayerID())
		second := g.PutOnBattlefield(factory(), b.PlayerID())
		if operation == "CanAttackAsThoughItDidntHaveDefender" {
			first.RevokeBaseAttr(core.AttrSummonSick)
			second.RevokeBaseAttr(core.AttrSummonSick)
		}
		if !first.CanDeclareAsAttacker(g) || !second.CanDeclareAsAttacker(g) {
			t.Fatal("factory permission missing")
		}
		g.AddContinuousEffect(mage.RemoveAllAbilities(first.ID(), core.EndOfTurn))
		g.ApplyContinuousEffects()
		if !g.FindPermanent(second.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("factory instances share permission")
		}
		if operation == "CanAttackAsThoughItHadHaste" && g.FindPermanent(first.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("removed source retained permission")
		}
	}
}

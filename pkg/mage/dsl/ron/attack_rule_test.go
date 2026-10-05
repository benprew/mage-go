package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONAttackRuleCards(t *testing.T) {
	for _, name := range []string{"Moat", "Evil Eye of Orms-by-Gore", "Akron Legionnaire"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
			if err := mage.ValidateCardDefinition(source.Card.Name(), source.Card); err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"plain", "flying", "eye", "artifact", "named", "named-other", "land"} {
				for _, own := range []bool{false, true} {
					c := mage.NewCreature("Subject", "", 2, 4)
					switch kind {
					case "flying":
						c = mage.NewCreature("Subject", "", 2, 4, mage.WithKeyword(core.Flying))
					case "eye":
						c.AddSubType("Eye")
					case "artifact":
						c.AddType(core.TypeArtifact)
					case "named":
						c = mage.NewCreature("Akron Legionnaire", "", 2, 4)
					case "named-other":
						c = mage.NewCreature("Different Legionnaire", "", 2, 4)
					case "land":
						c = mage.NewLand("Land")
					}
					player := b
					if own {
						player = a
					}
					p := g.PutOnBattlefield(c, player.PlayerID())
					p.RevokeBaseAttr(core.AttrSummonSick)
					blocked := kind == "land"
					switch name {
					case "Moat":
						blocked = blocked || kind != "flying"
					case "Evil Eye of Orms-by-Gore":
						blocked = blocked || (own && kind != "eye")
					case "Akron Legionnaire":
						blocked = blocked || (own && kind != "named" && kind != "artifact")
					}
					if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) == blocked {
						t.Fatalf("%s own=%v blocked=%v", kind, own, blocked)
					}
				}
			}
			if name == "Evil Eye of Orms-by-Gore" {
				wallCard := mage.NewCreature("Wall", "", 1, 6)
				wallCard.AddSubType("Wall")
				wall := g.PutOnBattlefield(wallCard, b.PlayerID())
				plain := g.PutOnBattlefield(mage.NewCreature("Plain", "", 1, 6), b.PlayerID())
				if !mage.CanBlock(wall, g.FindPermanent(source.ID()), g) || mage.CanBlock(plain, g.FindPermanent(source.ID()), g) {
					t.Fatal("wall blocking restriction")
				}
			}
		})
	}
}

func TestRONAttackRuleInvalid(t *testing.T) {
	for _, body := range []string{
		`EachPermanentRuleEffect(AnyPermanent,[CantAttack()])`,
		`EachPermanentRuleEffect(AnyPermanent,[CantAttack,CantAttack])`,
		`EachPermanentRuleEffect(AnyPermanent,[CantAttack(Integer(1))])`,
		`EachPermanentRuleEffect(ExceptFor(ControlledByAPlayer(SinglePlayer(You))),[CantAttack])`,
		`EachPermanentRuleEffect(And([]),[CantAttack])`,
		`EachPermanentRuleEffect(IsNamed("Name"),[CantAttack])`,
		`EachPermanentRuleEffect(IsNamed(NamedCard(You)),[CantAttack])`,
		`EachPermanentRuleEffect(IsNamed(NamedCard("")),[CantAttack])`,
		`EachPermanentRuleEffect(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)),[CantAttack])`,
		`EachPermanentRuleEffect(Or([AnyPermanent,ControlledByAPlayer(SinglePlayer(You))]),[CantAttack])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[%s])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONAttackRuleNative(t *testing.T) {
	for _, name := range []string{"Moat", "Evil Eye of Orms-by-Gore", "Akron Legionnaire"} {
		t.Run(name, func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
			a.SetAttackers(1, []string{"Llanowar Elves"})
			g.RunStepWithPriority(core.DeclareAttackers)
			if g.GetCombat().IsAttacking(p.ID()) || g.FindPermanent(p.ID()).Tapped {
				t.Fatal("restricted attack declared")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.RunStepWithPriority(core.DeclareAttackers)
			if !g.GetCombat().IsAttacking(p.ID()) || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("legal attack not declared")
			}
			g.PutOnBattlefield(source.Card, a.PlayerID())
			if !g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("new restriction removed attacker")
			}
		})
	}
}

func TestRONAttackRuleLifecycle(t *testing.T) {
	for _, name := range []string{"Moat", "Evil Eye of Orms-by-Gore", "Akron Legionnaire"} {
		for _, change := range []string{"source-phase", "source-phase-cycle", "source-control", "recipient-control", "source-left", "source-return", "source-abilities", "recipient-abilities", "recipient-phase-cycle", "recipient-return", "clone", "second-instance"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, "LEG", name), a.PlayerID())
				c := mage.NewCreature("Subject", "", 2, 4)
				p := g.PutOnBattlefield(c, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				check := func(game *mage.Game, blocked bool) {
					t.Helper()
					if game.FindPermanent(p.ID()).CanDeclareAsAttacker(game) == blocked {
						t.Fatalf("blocked=%v", blocked)
					}
				}
				check(g, true)
				clone := g.Clone()
				check(clone, true)
				blocked := true
				switch change {
				case "source-phase":
					g.PhaseOut(source.ID())
					g.ApplyContinuousEffects()
					blocked = false
				case "source-phase-cycle":
					phased := g.PhaseOut(source.ID())
					g.PhaseIn(phased)
					g.ApplyContinuousEffects()
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					blocked = name == "Moat"
				case "recipient-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.FindPermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
					blocked = name == "Moat"
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					blocked = false
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(source.Card, a.PlayerID())
				case "source-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					blocked = false
				case "recipient-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "recipient-phase-cycle":
					phased := g.PhaseOut(p.ID())
					g.PhaseIn(phased)
					g.ApplyContinuousEffects()
				case "recipient-return":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					p = g.PutOnBattlefield(c, a.PlayerID())
					p.RevokeBaseAttr(core.AttrSummonSick)
				case "clone":
					clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
					check(clone, false)
				case "second-instance":
					second := g.PutOnBattlefield(repositoryCard(t, "LEG", name), b.PlayerID())
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					blocked = name == "Moat"
					check(g, blocked)
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: second.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					blocked = true
				}
				check(g, blocked)
				if change != "clone" && change != "recipient-return" {
					check(clone, true)
				}
			})
		}
	}
}

func TestRONAttackRuleFlyingChanges(t *testing.T) {
	for _, early := range []bool{false, true} {
		g, a, b := ronGame()
		c := mage.NewCreature("Subject", "", 2, 4)
		p := g.PutOnBattlefield(c, a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		var source *mage.Permanent
		if early {
			source = g.PutOnBattlefield(repositoryCard(t, "LEG", "Moat"), b.PlayerID())
		}
		if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Flying)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if !early {
			source = g.PutOnBattlefield(repositoryCard(t, "LEG", "Moat"), b.PlayerID())
		}
		if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("flying grant did not permit attack")
		}
		if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("removed flying still permitted attack")
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("departed Moat still restricted attack")
		}
	}
}

func TestRONAttackRuleStaticFilterScopes(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentRuleEffect(And([IsCardtype(Creature),Other(ThisPermanent),ExceptFor(IsNamed(NamedCard("Exempt"))),ControlledByAPlayer(SinglePlayer(You))]),[CantAttack])`)
	g, a, b := ronGame()
	first, second := factory(), factory()
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	one := g.PutOnBattlefield(first, a.PlayerID())
	two := g.PutOnBattlefield(second, b.PlayerID())
	exempt := g.PutOnBattlefield(mage.NewCreature("Exempt", "", 2, 4), a.PlayerID())
	exempt.RevokeBaseAttr(core.AttrSummonSick)
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 2, 4), a.PlayerID())
	own.RevokeBaseAttr(core.AttrSummonSick)
	if !exempt.CanDeclareAsAttacker(g) || own.CanDeclareAsAttacker(g) {
		t.Fatal("typed static collection")
	}
	g.RemoveFromBattlefield(g.FindPermanent(one.ID()))
	if !g.FindPermanent(own.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("second instance used first controller")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: two.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(own.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("second instance did not follow current controller")
	}
}

func TestRONAttackRuleStrictLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	lines := strings.Split(string(data), "\n")
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name == "Moat" || name == "Evil Eye of Orms-by-Gore" || name == "Akron Legionnaire" {
			files[name+".ron"] = &fstest.MapFile{Data: []byte(lines[node.Position.Line-1])}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 3 {
		t.Fatalf("strict subset: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 3 || len(report.Issues) != 0 {
		t.Fatal("accepted subset report")
	}
	roles := map[string]string{"CantAttack": "permanent_rule_operation", "ExceptFor": "permanent_filter", "IsNamed": "permanent_filter", "NamedCard": "card_name"}
	for name, role := range roles {
		seen := false
		for _, entry := range report.Signatures {
			if entry.Name == name {
				seen = true
				if entry.Role != role {
					t.Fatalf("%s: role %s", name, entry.Role)
				}
			}
		}
		if !seen {
			t.Fatal("missing", name)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[EachPermanentRuleEffect(AnyPermanent,[CantAttack()])])`)}
	if bad, err := LoadFS(files, "*.ron"); err == nil || bad != nil {
		t.Fatal("invalid collection accepted")
	}
}

func TestRONAttackRuleNativeCast(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEG", "Moat")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	subject := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 4), a.PlayerID())
	subject.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 || !g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("unpaid cast changed state")
	}
	a.ManaPool().Add(core.White, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 || !g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("restriction began before resolution")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || g.FindPermanent(subject.ID()).CanDeclareAsAttacker(g) || a.ManaPool().TotalMana() != 0 {
		t.Fatal("restriction did not begin on entry")
	}
}

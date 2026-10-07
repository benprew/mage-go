package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONSourceCombatAttributeCast(t *testing.T) {
	for _, name := range []string{"Juggernaut", "Two-Headed Giant of Foriys"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "LEB", name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Colorless, 4)
			if name != "Juggernaut" {
				a.ManaPool().Add(core.Red, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("source entered before resolution")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil {
				t.Fatal("source did not enter")
			}
			attr := core.AttrMustAttack
			power, toughness := 5, 3
			if name != "Juggernaut" {
				attr, power, toughness = core.CanBlockAdditional, 4, 4
				if !p.HasAttr(core.Trample) {
					t.Fatal("missing trample")
				}
			}
			if !p.HasAttr(attr) || p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness {
				t.Fatal("source combat attribute or metadata")
			}
		})
	}
}

func TestRONSourceCombatAttributeRequiredAttack(t *testing.T) {
	for _, change := range []string{"able", "sick", "tapped", "phased", "restricted", "disabled", "unpaid-cost", "second-combat"} {
		t.Run(change, func(t *testing.T) {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			card := repositoryCard(t, "LEB", "Juggernaut")
			if change == "unpaid-cost" {
				card.AddAbility(mage.StaticAbility(mage.SourceCantAttackUnlessPays(mage.GenericCost(2))))
				a.ManaPool().Add(core.Colorless, 1)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if change != "sick" {
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			switch change {
			case "tapped":
				g.TapPermanent(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "restricted":
				g.PutOnBattlefield(mage.NewCreature("Restriction", "", 1, 1, mage.WithAbility(mage.Static(mage.CantAttackEach(mage.MatchingPermanents(mage.AnyPermanent))))), b.PlayerID())
			case "disabled":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			g.ApplyContinuousEffects()
			if change == "unpaid-cost" && (g.PayAttackCosts(g.FindPermanent(p.ID()), a.PlayerID(), false) || a.ManaPool().TotalMana() != 1 || g.FindPermanent(p.ID()).Tapped) {
				t.Fatal("unpaid attack changed resources")
			}
			g.RunStepWithPriority(core.DeclareAttackers)
			want := change == "able" || change == "second-combat"
			if g.GetCombat().IsAttacking(p.ID()) != want {
				t.Fatal("required attack ignored ability or legality", change)
			}
			if change == "unpaid-cost" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("unpaid attack tapped source")
			}
			if want {
				if !g.FindPermanent(p.ID()).Tapped {
					t.Fatal("attacker did not tap")
				}
				wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 8, mage.WithSubTypes("Wall")), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
				if mage.CanBlock(wall, g.FindPermanent(p.ID()), g) || !mage.CanBlock(other, g.FindPermanent(p.ID()), g) {
					t.Fatal("Juggernaut blocking restriction")
				}
				if change == "second-combat" {
					g.GetCombat().Reset()
					g.UntapPermanent(g.FindPermanent(p.ID()))
					g.RunStepWithPriority(core.DeclareAttackers)
					if !g.GetCombat().IsAttacking(p.ID()) {
						t.Fatal("requirement did not apply in later combat")
					}
				}
			}
		})
	}
}

func TestRONSourceCombatAttributeAttackControlAndFinality(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	card := repositoryCard(t, "LEB", "Juggernaut")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.RunStepWithPriority(core.DeclareAttackers)
	if g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("requirement followed owner instead of controller")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.DeclareAttackers)
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("controller's required attack missing")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
	g.ApplyContinuousEffects()
	if !g.GetCombat().IsAttacking(p.ID()) {
		t.Fatal("ability removal erased declared attack")
	}
	g.ExecuteCombatDamage()
	if a.Life() != 15 || b.Life() != 20 {
		t.Fatal("damage used wrong defender")
	}
}

type ronCombatBlockPlayer struct {
	*mage.BasePlayer
	assignments []mage.BlockAssignment
}

func (p *ronCombatBlockPlayer) DeclareBlockers(*mage.Game) []mage.BlockAssignment {
	return p.assignments
}

func TestRONSourceCombatAttributeAdditionalBlock(t *testing.T) {
	for _, change := range []string{"able", "disabled", "tapped", "phased", "flying"} {
		t.Run(change, func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &ronCombatBlockPlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Two-Headed Giant of Foriys"), b.PlayerID())
			attackers := make([]*mage.Permanent, 0, 3)
			for i := range 3 {
				card := mage.NewCreature(fmt.Sprintf("Attacker %d", i), "", 1, 6)
				if change == "flying" && i == 0 {
					card = mage.NewCreature("Flying attacker", "", 1, 6, mage.WithKeyword(core.Flying))
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				attackers = append(attackers, p)
				g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
				b.assignments = append(b.assignments, mage.BlockAssignment{BlockerID: blocker.ID(), AttackerID: p.ID()})
			}
			switch change {
			case "disabled":
				g.AddContinuousEffect(mage.RemoveAllAbilities(blocker.ID(), core.Indefinite))
			case "tapped":
				g.TapPermanent(blocker)
			case "phased":
				g.PhaseOut(blocker.ID())
			}
			g.ApplyContinuousEffects()
			g.RunStepWithPriority(core.DeclareBlockers)
			count := 0
			for _, p := range attackers {
				if len(g.GetCombat().GroupFor(p.ID()).BlockerIDs) != 0 {
					count++
				}
			}
			want := 2
			switch change {
			case "disabled":
				want = 1
			case "tapped", "phased":
				want = 0
			}
			if count != want || (change == "flying" && len(g.GetCombat().GroupFor(attackers[0].ID()).BlockerIDs) != 0) {
				t.Fatal("block capacity or legality", count, want)
			}
		})
	}
}

func TestRONSourceCombatAttributeInvalid(t *testing.T) {
	for _, op := range []string{"MustAttack", "CanBlockAnAdditionalCreature"} {
		for _, rule := range []string{
			`PermanentRuleEffect(ThisPermanent,[` + op + `()])`,
			`PermanentRuleEffect(ThisPermanent,[` + op + `(Integer(1))])`,
			`PermanentRuleEffect(ThisPermanent(),[` + op + `])`,
			`PermanentRuleEffect(HostPermanent,[` + op + `])`,
			`PermanentRuleEffect(Ref_TargetPermanent,[` + op + `])`,
			`PermanentRuleEffect(ThisPermanent,[` + op + `,` + op + `])`,
			`EachPermanentRuleEffect(AnyPermanent,[` + op + `])`,
		} {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid-combat.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid declaration accepted", rule)
			}
		}
	}
}

func TestRONSourceCombatAttributeLifecycle(t *testing.T) {
	for _, name := range []string{"Juggernaut", "Two-Headed Giant of Foriys"} {
		for _, change := range []string{"disabled", "phase-cycle", "returned", "control", "departed"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEB", name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				other := g.PutOnBattlefield(repositoryCard(t, "LEB", name), b.PlayerID())
				attribute := core.AttrMustAttack
				if name != "Juggernaut" {
					attribute = core.CanBlockAdditional
				}
				clone := g.Clone()
				switch change {
				case "disabled":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					incarnation := g.CaptureDamageSource(p.ID()).Incarnation()
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, a.PlayerID())
					if g.CaptureDamageSource(p.ID()).Incarnation() == incarnation {
						t.Fatal("return retained incarnation")
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "departed":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				}
				g.ApplyContinuousEffects()
				if !clone.FindPermanent(p.ID()).HasAttr(attribute) || !g.FindPermanent(other.ID()).HasAttr(attribute) {
					t.Fatal("state shared between sources or clones")
				}
				if change == "departed" {
					if g.FindPermanent(p.ID()) != nil {
						t.Fatal("departed source still present")
					}
					return
				}
				want := change != "disabled"
				if g.FindPermanent(p.ID()).HasAttr(attribute) != want {
					t.Fatal("static attribute lifecycle")
				}
				if change == "disabled" {
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					if !g.FindPermanent(p.ID()).HasAttr(attribute) {
						t.Fatal("expired removal did not restore printed rule")
					}
				}
				if change == "control" && g.FindPermanent(p.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("control fixture")
				}
			})
		}
	}
}

func TestRONSourceCombatAttributeDeclaredBlockFinality(t *testing.T) {
	a := mage.NewBasePlayer("A")
	b := &ronCombatBlockPlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "Two-Headed Giant of Foriys"), b.PlayerID())
	for i := range 2 {
		p := g.PutOnBattlefield(mage.NewCreature(fmt.Sprintf("Attacker %d", i), "", 1, 6), a.PlayerID())
		g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
		b.assignments = append(b.assignments, mage.BlockAssignment{BlockerID: blocker.ID(), AttackerID: p.ID()})
	}
	g.RunStepWithPriority(core.DeclareBlockers)
	g.AddContinuousEffect(mage.RemoveAllAbilities(blocker.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	for _, pair := range b.assignments {
		if len(g.GetCombat().GroupFor(pair.AttackerID).BlockerIDs) != 1 {
			t.Fatal("ability removal erased declared block")
		}
	}
	clone := g.Clone()
	g.ExecuteCombatDamage()
	if g.FindPermanent(blocker.ID()).Damage != 2 || clone.FindPermanent(blocker.ID()).Damage != 0 {
		t.Fatal("combat damage or clone isolation")
	}
}

func TestRONSourceCombatAttributeGiantTrample(t *testing.T) {
	a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Two-Headed Giant of Foriys"), a.PlayerID())
	g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 1), b.PlayerID())
	a.SetAttackers(1, []string{p.Name()})
	b.SetBlockers(1, map[string]string{"Blocker": p.Name()})
	g.RunStepWithPriority(core.DeclareAttackers)
	g.RunStepWithPriority(core.DeclareBlockers)
	g.ExecuteCombatDamage()
	if b.Life() != 17 {
		t.Fatal("complete Giant record did not trample", b.Life())
	}
}

func TestRONSourceCombatAttributeReprints(t *testing.T) {
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
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Juggernaut" && name != "Two-Headed Giant of Foriys" {
				continue
			}
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
			count++
		}
		var records []string
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Juggernaut"`) || strings.Contains(line, `Name: "Two-Headed Giant of Foriys"`) {
				records = append(records, line)
			}
		}
		if len(records) != 0 {
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
		}
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil || registry == nil {
		t.Fatal("complete selected collection", err)
	}
	if count != 11 || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprint counts", count)
	}
	collection["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unsupported])`)}
	if registry, err := LoadFS(collection, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}

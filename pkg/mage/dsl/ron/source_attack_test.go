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
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

var sourceAttackCards = []struct{ set, name string }{
	{"DRK", "Cave People"}, {"ARN", "Hasran Ogress"}, {"ARN", "Merchant Ship"}, {"DRK", "Murk Dwellers"},
}

func TestRONSourceAttackNative(t *testing.T) {
	for _, tc := range sourceAttackCards {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/blocked=%t", tc.name, blocked), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), b.PlayerID())
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 8), b.PlayerID())
				a.SetAttackers(1, []string{tc.name})
				if blocked {
					b.SetBlockers(1, map[string]string{"Blocker": tc.name})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				if !g.GetCombat().IsAttacking(p.ID()) {
					t.Fatal("attack rejected")
				}
				power, toughness := card.Power(), card.Toughness()
				life := 20
				if tc.name == "Cave People" {
					power++
					toughness -= 2
				}
				if tc.name == "Hasran Ogress" {
					life = 17
				}
				if g.FindPermanent(p.ID()).CurrentPower(g) != power || g.FindPermanent(p.ID()).CurrentToughness(g) != toughness || a.Life() != life {
					t.Fatal("attack timing or effect")
				}
				g.RunStepWithPriority(core.DeclareBlockers)
				if tc.name == "Merchant Ship" && !blocked {
					life += 2
				}
				if tc.name == "Murk Dwellers" && !blocked {
					power += 2
				}
				if a.Life() != life || g.FindPermanent(p.ID()).CurrentPower(g) != power {
					t.Fatal("unblocked timing or effect")
				}
				if blocked && !g.GetCombat().IsBlocking(blocker.ID()) {
					t.Fatal("block rejected")
				}
				g.RunStepWithPriority(core.EndCombat)
				if tc.name == "Murk Dwellers" && g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
					t.Fatal("combat bonus survived combat")
				}
				if tc.name == "Cave People" && g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
					t.Fatal("turn bonus ended at combat")
				}
				g.RunStepWithPriority(core.Cleanup)
				if g.FindPermanent(p.ID()).CurrentPower(g) != card.Power() {
					t.Fatal("bonus survived cleanup")
				}
			})
		}
	}
}

func TestRONSourceAttackIdentity(t *testing.T) {
	for _, scenario := range []string{"normal", "other", "leave", "return", "phase", "phase-cycle", "control", "removed", "removed-after", "inactive", "copy", "two"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Cave People")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(repositoryCard(t, "DRK", "Cave People"), a.PlayerID())
			if scenario == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			if scenario == "inactive" {
				g.PhaseOut(p.ID())
			}
			source := p.ID()
			if scenario == "other" {
				source = other.ID()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: source, PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			if scenario == "two" {
				g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: other.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			}
			switch scenario {
			case "leave", "return":
				g.RemoveFromBattlefield(p)
				if scenario == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			want := 1
			if scenario == "removed" || scenario == "inactive" {
				want = 0
			}
			if scenario == "two" {
				want = 2
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want=%d", g.StackSize(), want)
			}
			if scenario == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if live := g.FindPermanent(p.ID()); live != nil && live.CurrentPower(g) != 1 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				live := game.FindPermanent(p.ID())
				absent := scenario == "leave" || scenario == "phase" || scenario == "inactive" || scenario == "copy"
				if (live == nil) != absent {
					t.Fatal("wrong source presence after resolution")
				}
				if live != nil {
					power := 1
					if scenario == "normal" || scenario == "removed-after" || scenario == "phase-cycle" || scenario == "control" || scenario == "two" {
						power = 2
					}
					if scenario == "copy" {
						power = 3
					}
					if live.CurrentPower(game) != power {
						t.Fatalf("power=%d want=%d", live.CurrentPower(game), power)
					}
				}
				otherPower := 1
				if scenario == "other" || scenario == "two" {
					otherPower = 2
				}
				if game.FindPermanent(other.ID()).CurrentPower(game) != otherPower {
					t.Fatal("event crossed instances")
				}
			}
		})
	}
}

func TestRONSourceAttackOtherRules(t *testing.T) {
	for _, choice := range []string{"paid", "declined", "unaffordable", "prevent", "returned"} {
		t.Run("Ogress/"+choice, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "declined"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "ARN", "Hasran Ogress")
			p := g.PutOnBattlefield(card, a.PlayerID())
			if choice == "paid" || choice == "declined" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredAttacker, SourceID: p.ID(), PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			g.PutTriggersOnStack()
			if a.offers != 0 || a.Life() != 20 {
				t.Fatal("payment before resolution")
			}
			if choice == "prevent" {
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			if choice == "returned" {
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, b.PlayerID())
			}
			g.ResolveStack()
			life := 17
			if choice == "paid" {
				life = 20
			}
			if choice == "prevent" {
				life = 19
			}
			if a.Life() != life || b.Life() != 20 {
				t.Fatal("payment or damage recipient")
			}
			mana := 0
			if choice == "declined" {
				mana = 2
			}
			if a.ManaPool().TotalMana() != mana {
				t.Fatal("payment rollback")
			}
		})
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "DRK", "Cave People"), a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	target := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 2), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
		t.Fatal("unpaid activation")
	}
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("failed cost tapped source")
	}
	a.ManaPool().Add(core.Red, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(target.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("landwalk activation")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(target.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("landwalk survived cleanup")
	}
	g, a, _ = ronGame()
	g.PutOnBattlefield(repositoryCard(t, "ARN", "Merchant Ship"), a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(g.AllBattlefield()) != 0 {
		t.Fatal("ship no-land sacrifice")
	}
}

func TestRONSourceAttackInvalid(t *testing.T) {
	valid := `TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent)),ActionList([CreatePermanentLayerEffectUntil(Trigger_ThatCreature,[AdjustPT(1,0)],UntilEndOfTurn)]))`
	for _, rule := range []string{
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(HostPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent())", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent,ThisPermanent)", 1),
		strings.Replace(valid, "SinglePermanent(ThisPermanent)", "IsCardtype(Creature)", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_ThatCreature()", 1),
		strings.Replace(valid, "Trigger_ThatCreature", "Trigger_ThatPermanent", 1),
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat()", 1),
		strings.Replace(valid, "CreatePermanentLayerEffectUntil(Trigger_ThatCreature,[AdjustPT(1,0)],UntilEndOfTurn)", "PlayerAction(Trigger_DefendingPlayer,GainLife(Integer(1)))", 1),
		valid + `,Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatCreature)]))`,
		strings.Replace(valid, "WhenACreatureAttacks(SinglePermanent(ThisPermanent))", "WhenACreatureAttacksAndIsntBlocked(SinglePermanent(ThisPermanent),You)", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:4),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONSourceAttackCastingAndRegistry(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range sourceAttackCards {
		nodes, err := Parse(strings.NewReader(mustRONSet(t, tc.set)), tc.set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if node.Fields["Name"].String == tc.name {
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				first, second := factory(), factory()
				if first.Abilities()[0] == second.Abilities()[0] {
					t.Fatal("factory shared ability")
				}
			}
		}
		for _, accept := range []bool{false, true} {
			g, a, _ := ronGame()
			if tc.name == "Merchant Ship" {
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
				g.TapPermanent(land)
			}
			spell := repositoryCard(t, tc.set, tc.name)
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if accept {
				for _, color := range []core.Color{core.Red, core.Blue, core.Black, core.Colorless} {
					a.ManaPool().Add(color, 10)
				}
			}
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0)
			if !accept {
				if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("failed casting rollback")
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(spell.ID()) == nil {
				t.Fatal("cast did not enter")
			}
			if g.StackSize() != 0 || a.Life() != 20 {
				t.Fatal("entry triggered attack")
			}
		}
	}
	for _, set := range []string{"DRK", "ARN"} {
		files[set+".ron"] = &fstest.MapFile{Data: []byte(selectedSourceAttackRecords(t, set))}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 4 {
		t.Fatal("fixture registry count")
	}
}

func mustRONSet(t *testing.T, set string) string {
	t.Helper()
	data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func selectedSourceAttackRecords(t *testing.T, set string) string {
	t.Helper()
	var records []string
	for line := range strings.SplitSeq(mustRONSet(t, set), "\n") {
		nodes, err := Parse(strings.NewReader(line), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			for _, tc := range sourceAttackCards {
				if tc.set == set && node.Fields["Name"].String == tc.name {
					records = append(records, line)
				}
			}
		}
	}
	return strings.Join(records, "\n")
}

func TestRONSourceAttackNoDeclaration(t *testing.T) {
	for _, tc := range sourceAttackCards {
		g, a, b := ronGame()
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Island"), a.PlayerID())
		p := g.PutOnBattlefieldAttacking(repositoryCard(t, tc.set, tc.name), a.PlayerID(), b.PlayerID())
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("entry attacking triggered declaration")
		}
		if tc.name == "Cave People" && p.CurrentPower(g) != 1 {
			t.Fatal("entry attacking received bonus")
		}
		g.ExecuteBlockers(nil)
		if g.StackSize() != 0 {
			t.Fatal("simulation left triggers unresolved")
		}
		if tc.name == "Hasran Ogress" && a.Life() != 20 {
			t.Fatal("entry attacking paid attack cost")
		}
		if tc.name == "Cave People" && g.FindPermanent(p.ID()).CurrentPower(g) != 1 {
			t.Fatal("entry attacking received attack bonus")
		}
		if tc.name == "Merchant Ship" && a.Life() != 22 {
			t.Fatal("entry attacking life gain")
		}
		if tc.name == "Murk Dwellers" && g.FindPermanent(p.ID()).CurrentPower(g) != 4 {
			t.Fatal("entry attacking unblocked bonus")
		}
	}
}

func TestRONSourceAttackInventory(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(selectedSourceAttackRecords(t, "DRK"))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 {
		t.Fatal("inspection did not compile real records")
	}
	for _, name := range []string{"WhenACreatureAttacks", "WhenACreatureAttacksAndIsntBlocked"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == "SinglePermanent" && sig.Role == "permanent_filter" {
				if strings.Contains(sig.Context, name) {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("missing combat subject inventory for %s", name)
		}
	}
}

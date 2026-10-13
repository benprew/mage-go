package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const distributedDamageBody = `TargetedDistributed([AnyNumberOfAnyTargets], DistributeNumberAmongTargets(Integer(4)), ActionList([SpellDealsDistributedDamage(ThisSpell)]))`

type ronDistributionPlayer struct {
	*mage.BasePlayer
	choose  func([]uuid.UUID, int, *mage.Game) map[uuid.UUID]int
	prompts int
	targets []uuid.UUID
}

func (p *ronDistributionPlayer) ChooseTargets(_ []uuid.UUID, _, _ int, _ *mage.Game) []uuid.UUID {
	return p.targets
}

func (p *ronDistributionPlayer) ChooseDamageDistribution(ids []uuid.UUID, total int, _ string, g *mage.Game) map[uuid.UUID]int {
	p.prompts++
	return p.choose(ids, total, g)
}

func distributedDamageGame() (*mage.Game, *ronDistributionPlayer, *mage.BasePlayer) {
	a := &ronDistributionPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetActivePlayerIndex(0)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	return g, a, b
}

func TestRONDistributedDamagePyrotechnics(t *testing.T) {
	for _, scenario := range []string{"normal", "empty", "four", "walker", "partial", "all-illegal", "return", "phase", "phase-cycle", "prevention", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := distributedDamageGame()
			card := repositoryCard(t, "LEG", "Pyrotechnics")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, 4)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 9), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 9), b.PlayerID())
			walker := g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 9), b.PlayerID())
			ids := []uuid.UUID{first.ID(), b.PlayerID()}
			shares := map[uuid.UUID]int{first.ID(): 3, b.PlayerID(): 1}
			if scenario == "empty" {
				ids = nil
				shares = nil
			}
			if scenario == "walker" {
				ids[0] = walker.ID()
				shares = map[uuid.UUID]int{walker.ID(): 3, b.PlayerID(): 1}
			}
			if scenario == "four" {
				ids = []uuid.UUID{first.ID(), second.ID(), walker.ID(), b.PlayerID()}
				shares = map[uuid.UUID]int{first.ID(): 1, second.ID(): 1, walker.ID(): 1, b.PlayerID(): 1}
			}
			if scenario == "all-illegal" {
				ids[1] = second.ID()
				shares = map[uuid.UUID]int{first.ID(): 3, second.ID(): 1}
			}
			a.choose = func(chosen []uuid.UUID, total int, game *mage.Game) map[uuid.UUID]int {
				if total != 4 || len(chosen) != len(ids) || a.ManaPool().TotalMana() != 5 || game.StackSize() != 0 || first.Damage != 0 {
					t.Error("allocation did not precede payment and publication")
				}
				return shares
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0); err != nil {
				t.Fatal(err)
			}
			wantPrompts := 1
			if scenario == "empty" {
				wantPrompts = 0
			}
			if a.prompts != wantPrompts || a.ManaPool().TotalMana() != 0 {
				t.Fatal("allocation or payment count")
			}
			if scenario != "empty" {
				shares[ids[0]] = 99
			}
			switch scenario {
			case "partial":
				first.GrantBaseAttr(core.Shroud)
			case "all-illegal":
				first.GrantBaseAttr(core.Shroud)
				second.GrantBaseAttr(core.Shroud)
			case "return":
				g.BouncePermanentToHand(first)
				g.PutOnBattlefield(first.Card, b.PlayerID())
			case "phase":
				g.PhaseOut(first.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "prevention":
				g.AddPreventionShield(first.ID(), 2)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone shared damage")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				wantFirst, wantLife, wantWalker := 3, 19, 9
				switch scenario {
				case "empty":
					wantFirst, wantLife = 0, 20
				case "four":
					wantFirst, wantWalker = 1, 8
				case "walker":
					wantFirst, wantWalker = 0, 6
				case "partial", "return", "phase":
					wantFirst = 0
				case "all-illegal":
					wantFirst, wantLife = 0, 20
				case "prevention":
					wantFirst = 1
				case "copy":
					wantFirst, wantLife = 6, 18
				}
				if game.FindPermanentIncludingPhased(first.ID()).Damage != wantFirst || game.GetPlayer(b.PlayerID()).Life() != wantLife || game.FindPermanent(walker.ID()).Counters[core.Loyalty] != wantWalker {
					t.Fatalf("damage/life/loyalty mismatch for %s", scenario)
				}
			}
			if a.prompts != wantPrompts {
				t.Fatal("allocation repeated at resolution or copy")
			}
		})
	}
}

func TestRONDistributedDamageAnnouncement(t *testing.T) {
	g, a, b := distributedDamageGame()
	card := repositoryCard(t, "LEG", "Pyrotechnics")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 4)
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 9), b.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 9), b.PlayerID())
	g.PhaseOut(phased.ID())
	hexproof := g.PutOnBattlefield(mage.NewCreature("Hexproof", "", 1, 9, mage.WithKeyword(core.Hexproof)), b.PlayerID())
	extra := g.PutOnBattlefield(mage.NewCreature("Extra", "", 1, 9), b.PlayerID())
	for _, ids := range [][]uuid.UUID{
		{creature.ID(), creature.ID()}, {land.ID()}, {phased.ID()}, {hexproof.ID()}, {uuid.New()},
		{creature.ID(), extra.ID(), a.PlayerID(), b.PlayerID(), g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 9), b.PlayerID()).ID()},
	} {
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 0); err == nil {
			t.Fatal("illegal announcement accepted")
		}
		if a.prompts != 0 || a.ManaPool().TotalMana() != 5 || g.StackSize() != 0 || len(a.Hand()) != 1 {
			t.Fatal("illegal announcement spent resources or requested shares")
		}
	}
	a.ManaPool().Clear()
	a.choose = func([]uuid.UUID, int, *mage.Game) map[uuid.UUID]int { return map[uuid.UUID]int{b.PlayerID(): 4} }
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 || b.Life() != 20 {
		t.Fatal("failed payment changed game")
	}
}

func TestRONDistributedDamageRetargetAndInstances(t *testing.T) {
	g, a, b := distributedDamageGame()
	card := repositoryCard(t, "LEG", "Pyrotechnics")
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 12), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 12), b.PlayerID())
	a.ManaPool().Add(core.Red, 2)
	a.ManaPool().Add(core.Colorless, 8)
	a.choose = func([]uuid.UUID, int, *mage.Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{first.ID(): 3, b.PlayerID(): 1}
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.targets = []uuid.UUID{second.ID(), a.PlayerID()}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	if copied == nil || copied.DamageDistribution[second.ID()] != 3 || copied.DamageDistribution[a.PlayerID()] != 1 || original.DamageDistribution[first.ID()] != 3 {
		t.Fatal("copy did not keep shares by target position")
	}
	g.ResolveStack()
	if first.Damage != 3 || second.Damage != 3 || a.Life() != 19 || b.Life() != 19 || a.prompts != 1 {
		t.Fatal("retargeted copy used original recipients or repeated allocation")
	}
	fresh := rebuildRONDefinition(t, card)
	a.AddToHand(fresh)
	a.choose = func([]uuid.UUID, int, *mage.Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{second.ID(): 1, a.PlayerID(): 3}
	}
	if err := g.CastSpellByID(a.PlayerID(), fresh.ID(), []uuid.UUID{second.ID(), a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if first.Damage != 3 || second.Damage != 4 || a.Life() != 16 || b.Life() != 19 || a.prompts != 2 {
		t.Fatal("rebuilt instance used prior bindings")
	}
}

func TestRONDistributedDamageInvalid(t *testing.T) {
	for _, body := range []string{
		strings.Replace(distributedDamageBody, "Integer(4)", "Integer(0)", 1),
		strings.Replace(distributedDamageBody, "Integer(4)", "Integer(-1)", 1),
		strings.Replace(distributedDamageBody, "Integer(4)", "ValueX", 1),
		strings.Replace(distributedDamageBody, "[AnyNumberOfAnyTargets]", "[]", 1),
		strings.Replace(distributedDamageBody, "[AnyNumberOfAnyTargets]", "[AnyNumberOfAnyTargets, AnyTarget]", 1),
		strings.Replace(distributedDamageBody, "AnyNumberOfAnyTargets", "AnyNumberOfAnyTargets()", 1),
		strings.Replace(distributedDamageBody, "ThisSpell", "Ref_TargetSpell", 1),
		strings.Replace(distributedDamageBody, "SpellDealsDistributedDamage(ThisSpell)", "", 1),
		strings.Replace(distributedDamageBody, "SpellDealsDistributedDamage(ThisSpell)", "SpellDealsDistributedDamage(ThisSpell), SpellDealsDistributedDamage(ThisSpell)", 1),
		`ActionList([SpellDealsDistributedDamage(ThisSpell)])`,
		`TargetedDistributed([AnyNumberOfAnyTargets], DistributeNumberAmongTargets(Integer(4)))`,
		strings.Replace(distributedDamageBody, "Integer(4)", `Integer("4")`, 1),
		strings.Replace(distributedDamageBody, "DistributeNumberAmongTargets(Integer(4))", "DistributeNumberAmongTargets(Integer(4), Integer(1))", 1),
		strings.Replace(distributedDamageBody, "SpellDealsDistributedDamage(ThisSpell)", "SpellDealsDistributedDamage(ThisSpell, Ref_AnyTargets)", 1),
		`Targeted([AnyTarget], ` + distributedDamageBody + `)`,
		`Modal_ChooseOne([` + distributedDamageBody + `, ActionList([GainLife(Integer(1))])])`,
	} {
		src := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Sorcery], Subtypes: []), Rules: [SpellActions(` + body + `)])`
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron"); err == nil || !strings.Contains(err.Error(), "invalid.ron:1:") {
			t.Fatalf("missing positioned rejection: %s: %v", body, err)
		}
	}
	for _, rule := range []string{
		`Activated(PayMana([ManaCostR]), ` + distributedDamageBody + `)`,
		`Trigger(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ` + distributedDamageBody + `)`,
	} {
		src := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [` + rule + `])`
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Fatalf("unsupported context accepted: %s", rule)
		}
	}
}

func TestRONDistributedDamageInventory(t *testing.T) {
	src := `Card(Name: "Distribution", Typeline: (Supertypes: [], Cardtypes: [Sorcery], Subtypes: []), Rules: [SpellActions(` + distributedDamageBody + `)])`
	report, err := InspectFS(fstest.MapFS{"distribution.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v, %v", report, err)
	}
	want := map[string]string{"AnyNumberOfAnyTargets": "target", "DistributeNumberAmongTargets": "target_distribution", "Integer": "announcement_value", "SpellDealsDistributedDamage": "action", "ThisSpell": "spell_reference"}
	for _, entry := range report.Signatures {
		if role, ok := want[entry.Name]; ok {
			if entry.Role != role {
				t.Fatalf("%s role=%s want=%s", entry.Name, entry.Role, role)
			}
			delete(want, entry.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing roles: %v", want)
	}
}

package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatCardinalityCaverns(t *testing.T) {
	for _, change := range []string{"normal", "unpaid", "phase", "cycle", "remove", "return", "abilities", "clone", "copy", "world"} {
		t.Run(change, func(t *testing.T) {
			factory, err := Compile(hostUntapRecord(t, "LEG", "Caverns of Despair"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if change != "unpaid" {
				a.ManaPool().Add(core.Red, 2)
				a.ManaPool().Add(core.Colorless, 2)
			}
			err = g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
			if change == "unpaid" {
				if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("cast rollback")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				switch change {
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "world":
					g.PutOnBattlefield(mage.NewEnchantment("New World", "", mage.WithSuperTypes(core.SuperWorld)), b.PlayerID())
					g.CheckStateBasedActions()
					if g.FindPermanent(card.ID()) != nil {
						t.Fatal("world rule kept the older enchantment")
					}
				case "remove":
					g.RemoveFromBattlefield(p)
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, a.PlayerID())
				case "abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				case "clone":
					clone := g.Clone()
					clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
					g = clone
				}
			}
			want := 2
			if change == "unpaid" || change == "phase" || change == "remove" || change == "abilities" || change == "clone" || change == "world" {
				want = 3
			}
			attackers := make([]uuid.UUID, 0, 4)
			blockers := make([]uuid.UUID, 0, 3)
			for range 3 {
				atk := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
				atk.RevokeBaseAttr(core.AttrSummonSick)
				attackers = append(attackers, atk.ID())
				blk := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
				blockers = append(blockers, blk.ID())
			}
			g.ExecuteAttackers(a.PlayerID(), append(attackers, attackers[0]))
			if len(g.GetCombat().Groups) != want {
				t.Fatalf("attackers=%d want %d", len(g.GetCombat().Groups), want)
			}
			proposal := make([]mage.BlockAssignment, 0, len(blockers))
			for _, id := range blockers {
				proposal = append(proposal, mage.BlockAssignment{BlockerID: id, AttackerID: attackers[0]})
			}
			g.ExecuteBlockers(proposal)
			if got := len(g.GetCombat().GroupFor(attackers[0]).BlockerIDs); got != want {
				t.Fatalf("blockers=%d want %d", got, want)
			}
		})
	}
}

func TestRONCombatCardinalityInvalid(t *testing.T) {
	for _, name := range []string{"NoMoreThanNumberCreaturesCanAttack", "NoMoreThanNumberCreaturesCanBlock"} {
		for _, arg := range []string{"", "()", "(Integer(-1))", "(ValueX)", "(Integer(2,3))", "(Integer(2),Integer(3))", "(2)"} {
			source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + name + arg + `])`
			nodes, err := Parse(strings.NewReader(source), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s%s", name, arg)
			}
		}
		for _, rule := range []string{`SpellActions(ActionList([` + name + `(Integer(2))]))`, `If(IsPlayersTurn(You),[` + name + `(Integer(2))])`} {
			factory := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[` + rule + `])`
			nodes, err := Parse(strings.NewReader(factory), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("accepted unsupported context")
			}
		}
	}
}

func TestRONCombatCardinalityFactoryAndAtomicLoad(t *testing.T) {
	node := hostUntapRecord(t, "LEG", "Caverns of Despair")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.ID() == second.ID() {
		t.Fatal("shared card identity")
	}
	for i, ability := range first.Abilities() {
		if ability.AbilityID() == second.Abilities()[i].AbilityID() {
			t.Fatal("shared ability identity")
		}
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{"good.ron": {Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"NoMoreThanNumberCreaturesCanAttack", "NoMoreThanNumberCreaturesCanBlock"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == "rule" && signature.CompiledOccurrences == 1 {
				found = true
			}
		}
		if !found {
			t.Fatal("missing compiled role", name)
		}
	}
	numericOccurrences := 0
	for _, signature := range report.Signatures {
		if signature.Name == "Integer" && signature.Role == "value" {
			numericOccurrences += signature.CompiledOccurrences
		}
	}
	if numericOccurrences != 2 {
		t.Fatal("missing numeric role")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[NoMoreThanNumberCreaturesCanBlock(Integer(-1))])`)}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Caverns of Despair"); err != nil {
		t.Fatal("failed load erased registry", err)
	}
}

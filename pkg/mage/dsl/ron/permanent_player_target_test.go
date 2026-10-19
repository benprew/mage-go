package ron

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentPlayerTargetLowering(t *testing.T) {
	for _, syntax := range []string{
		`ControlledByAPlayer(SinglePlayer(You))`, `ControlledByAPlayer(Opponent)`,
		`OwnedByAPlayer(SinglePlayer(You))`, `OwnedByAPlayer(Opponent)`,
		`And([IsCardtype(Creature),ControlledByAPlayer(Opponent),IsNonCreatureType(Wall)])`,
		`And([OwnedByAPlayer(SinglePlayer(You)),And([ControlledByAPlayer(SinglePlayer(You)),Or([IsColor(Red),IsLandType(Forest)])])])`,
	} {
		filter, options, err := compilePermanentTargetFilter(nameScopeNode(t, syntax), scope{})
		if err != nil {
			t.Fatal(syntax, err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("player", filter, options...)); !ok {
			t.Fatal("player declaration lost", syntax)
		}
	}
}

func TestRONPermanentPlayerTargetReturn(t *testing.T) {
	for _, name := range []string{"Obelisk of Undoing", "Witch Hunter"} {
		for _, state := range []string{"legal", "control", "owner", "source control", "source leaves", "phase", "phase cycle", "return", "copy", "clone"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				set, index, mana, controller := "ATQ", 0, 6, a.PlayerID()
				if name == "Witch Hunter" {
					set, index, mana, controller = "DRK", 1, 1, b.PlayerID()
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				victimCard := mage.NewCreature("Victim", "", 2, 5)
				victimCard.SetOwner(a.PlayerID())
				victim := g.PutOnBattlefield(victimCard, controller)
				a.ManaPool().Add(core.Colorless, mana)
				if name == "Witch Hunter" {
					a.ManaPool().Add(core.White, 2)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				want := true
				switch state {
				case "control":
					newController := b.PlayerID()
					if name == "Witch Hunter" {
						newController = a.PlayerID()
					}
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: newController, Duration: core.Indefinite})
					want = false
				case "owner":
					g.MutablePermanent(victim.ID()).Card.SetOwner(b.PlayerID())
					want = name == "Witch Hunter"
				case "source control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source leaves":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "phase", "phase cycle":
					g.PhaseOut(victim.ID())
					want = state == "phase cycle"
					if want {
						g.PhaseIn([]uuid.UUID{victim.ID()})
					}
				case "return":
					g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
					g.PutOnBattlefield(victimCard, controller)
					want = false
				case "copy":
					g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
				}
				clone := g.Clone()
				if state == "clone" {
					clone.PhaseOut(victim.ID())
					clone.ResolveStack()
					if clone.FindPermanentIncludingPhased(victim.ID()) == nil {
						t.Fatal("clone resolved illegal target")
					}
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != want {
					t.Fatal("return ignored current target domain or exact visit")
				}
				if want {
					owner := a.PlayerID()
					if state == "owner" {
						owner = b.PlayerID()
					}
					if len(g.GetPlayer(owner).Hand()) != 1 {
						t.Fatal("return used controller instead of owner")
					}
				}
			})
		}
	}
}

func TestRONPermanentPlayerTargetRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil || len(files) == 0 {
		t.Fatal("missing collection", err)
	}
	names := map[string]bool{}
	records := 0
	var hasPlayerDomain func(Node) bool
	hasPlayerDomain = func(n Node) bool {
		return n.Name == "ControlledByAPlayer" || n.Name == "OwnedByAPlayer" || slices.ContainsFunc(n.Args, hasPlayerDomain)
	}
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
			found := false
			var walk func(Node)
			walk = func(n Node) {
				if n.Name == "TargetPermanent" && len(n.Args) == 1 && hasPlayerDomain(n.Args[0]) {
					filter, options, err := compilePermanentTargetFilter(n.Args[0], scope{})
					if err == nil {
						_, foundHere := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("audit", filter, options...))
						found = found || foundHere
					}
				}
				for _, child := range n.Args {
					walk(child)
				}
				for _, child := range n.Fields {
					walk(child)
				}
			}
			walk(node)
			if found {
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				if err := mage.ValidateCardDefinition(factory().Name(), factory()); err != nil {
					t.Fatal(err)
				}
				names[node.Fields["Name"].String] = true
				records++
			}
		}
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	slices.Sort(list)
	t.Logf("player targets: %d names / %d records: %v", len(names), records, list)
	if len(names) != 19 || records != 25 || !names["Goblin Kites"] || !names["Miracle Worker"] {
		t.Fatal("player target coverage changed")
	}
}

func TestRONPermanentPlayerTargetBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))])),TargetPermanent(And([IsCardtype(Land),OwnedByAPlayer(SinglePlayer(You))]))],ActionList([DestroyPermanent(Ref_TargetPermanent1),DestroyPermanentNoRegen(Ref_TargetPermanent2)])))`)
	g, a, b := ronGame()
	card := mage.NewLand("Subject")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	first := castRONSpell(t, g, a, factory(), p.ID(), p.ID())
	second := castRONSpell(t, g, a, factory(), p.ID(), p.ID())
	instructions, ok := mage.InspectDirectDestruction(mage.Sequence(first.Effects...))
	if !ok || len(instructions) != 2 {
		t.Fatal("instructions lost")
	}
	control, ok := mage.InspectPermanentTarget(instructions[0].Subject)
	if !ok {
		t.Fatal("controller declaration lost")
	}
	owner, ok := mage.InspectPermanentTarget(instructions[1].Subject)
	if !ok {
		t.Fatal("owner declaration lost")
	}
	if _, known := control.Select(g, second); known {
		t.Fatal("factories shared declaration")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if got, known := control.Select(g, first); !known || len(got) != 0 {
		t.Fatal("controller group stayed legal")
	}
	if got, known := owner.Select(g, first); !known || len(got) != 1 {
		t.Fatal("owner group lost legal visit")
	}
	g.AddRegenerationShield(p.ID())
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("illegal controller group blocked independent owner group")
	}
}

func TestRONPermanentPlayerTargetInvalid(t *testing.T) {
	for _, syntax := range []string{
		`ControlledByAPlayer`, `OwnedByAPlayer()`, `ControlledByAPlayer(Opponent,Opponent)`,
		`ControlledByAPlayer(Opponent())`, `OwnedByAPlayer("Opponent")`,
		`ControlledByAPlayer(SinglePlayer())`, `OwnedByAPlayer(SinglePlayer(You,You))`,
		`OwnedByAPlayer(SinglePlayer(You()))`, `OwnedByAPlayer(SinglePlayer(Unknown))`,
		`And([ControlledByAPlayer(Opponent),ControlledByAPlayer(SinglePlayer(You))])`,
		`And([OwnedByAPlayer(SinglePlayer(You)),And([OwnedByAPlayer(Opponent)])])`,
		`Or([ControlledByAPlayer(Opponent),IsCardtype(Land)])`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("invalid player declaration accepted", syntax)
		}
	}
}

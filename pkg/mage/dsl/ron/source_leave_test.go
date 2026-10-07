package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceLeaveCards(t *testing.T) {
	for _, entry := range []struct{ set, name string }{{"DRK", "Goblin Shrine"}, {"LEG", "Hazezon Tamar"}} {
		t.Run(entry.name, func(t *testing.T) {
			c := repositoryCard(t, entry.set, entry.name)
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRONSourceLeaveCasting(t *testing.T) {
	for _, entry := range []struct{ set, name string }{{"DRK", "Goblin Shrine"}, {"LEG", "Hazezon Tamar"}} {
		t.Run(entry.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, entry.set, entry.name)
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			var targets []uuid.UUID
			if entry.name == "Goblin Shrine" {
				land := g.PutOnBattlefield(mage.NewLand("Host"), b.PlayerID())
				targets = []uuid.UUID{land.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid casting changed state")
			}
			if entry.name == "Goblin Shrine" {
				a.ManaPool().Add(core.Colorless, 1)
				a.ManaPool().Add(core.Red, 2)
				invalid := g.PutOnBattlefield(mage.NewCreature("Invalid", "", 1, 1), a.PlayerID())
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{invalid.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 {
					t.Fatal("invalid enchant target spent mana")
				}
			} else {
				a.ManaPool().Add(core.Colorless, 4)
				a.ManaPool().Add(core.Red, 1)
				a.ManaPool().Add(core.Green, 1)
				a.ManaPool().Add(core.White, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong casting timing or cost")
			}
			g.ResolveStack()
			p := g.FindPermanent(c.ID())
			if p == nil {
				t.Fatal("spell did not enter")
			}
			if entry.name == "Goblin Shrine" {
				if p.AttachedTo != targets[0] {
					t.Fatal("Aura did not attach")
				}
			} else if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 4 || !p.Card.HasSuperType(core.SuperLegendary) || !p.HasSubType("Human") || !p.HasSubType("Warrior") {
				t.Fatal("wrong Hazezon metadata")
			}
		})
	}
}

func TestRONSourceLeaveHazezon(t *testing.T) {
	for _, scenario := range []string{"normal", "early-leave", "before-entry-resolves", "control", "return", "counter-entry", "counter-delayed", "phase"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Hazezon Tamar")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.PutOnBattlefield(mage.NewLand("A land"), a.PlayerID())
			g.PutOnBattlefield(mage.NewLand("B land"), b.PlayerID())
			if scenario == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing entry trigger")
			}
			if scenario == "counter-entry" {
				g.CounterStackObject(g.StackPeek().ID)
			}
			if scenario == "before-entry-resolves" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				g.PutTriggersOnStack()
			}
			g.ResolveStack()
			if countSandWarriors(g, a.PlayerID()) != 0 {
				t.Fatal("tokens created before the next upkeep")
			}
			switch scenario {
			case "early-leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				g.PutTriggersOnStack()
				g.ResolveStack()
				if scenario == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
					g.PutTriggersOnStack()
					g.ResolveStack()
				}
			case "phase":
				g.PhaseOut(p.ID())
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("phasing generated a departure trigger")
				}
			}
			g.PutOnBattlefield(mage.NewLand("Second A land"), a.PlayerID())
			phased := g.PutOnBattlefield(mage.NewLand("Phased A land"), a.PlayerID())
			g.PhaseOut(phased.ID())
			g.SetActivePlayerIndex(1)
			g.SetStep(core.Upkeep)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			wantB := 0
			if scenario == "return" {
				wantB = 1
			}
			if countSandWarriors(g, a.PlayerID()) != 0 || countSandWarriors(g, b.PlayerID()) != wantB {
				t.Fatal("wrong next upkeep or returned source supplied the old controller")
			}
			g.SetActivePlayerIndex(0)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			clone := g.Clone()
			if scenario == "counter-delayed" {
				if g.StackSize() != 1 {
					t.Fatal("missing delayed trigger")
				}
				g.CounterStackObject(g.StackPeek().ID)
			}
			g.ResolveStack()
			wantA := 2
			if scenario == "counter-entry" || scenario == "counter-delayed" {
				wantA = 0
			}
			if countSandWarriors(g, a.PlayerID()) != wantA || countSandWarriors(clone, a.PlayerID()) != 0 {
				t.Fatal("wrong resolution land count or shared clone token creation")
			}
			for _, token := range g.AllBattlefield() {
				if !token.IsToken || !token.HasSubType("Sand") {
					continue
				}
				if token.CurrentPower(g) != 1 || token.CurrentToughness(g) != 1 || !token.HasType(core.TypeCreature) || !token.HasSubType("Warrior") || token.Card.Owner() != token.ControllerID() || len(token.Colors()) != 3 {
					t.Fatal("wrong token characteristics or owner")
				}
			}
			clone.ResolveStack()
			if scenario != "counter-entry" && countSandWarriors(clone, a.PlayerID()) != 2 {
				t.Fatal("clone lost the captured delayed controller")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if countSandWarriors(g, a.PlayerID()) != wantA {
				t.Fatal("delayed trigger repeated")
			}
			if scenario == "normal" || scenario == "control" || scenario == "return" {
				g.ExilePermanent(g.FindPermanent(p.ID()))
				g.PutTriggersOnStack()
				g.ResolveStack()
				if countSandWarriors(g, a.PlayerID()) != 0 || countSandWarriors(g, b.PlayerID()) != 0 {
					t.Fatal("departure did not exile every Sand Warrior")
				}
			}
		})
	}
}

func countSandWarriors(g *mage.Game, controller uuid.UUID) int {
	count := 0
	for _, p := range g.AllBattlefield() {
		if !p.PhasedOut && p.ControllerID() == controller && p.HasSubType("Sand") && p.HasSubType("Warrior") {
			count++
		}
	}
	return count
}

func TestRONSourceLeaveHazezonCollection(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEG", "Hazezon Tamar")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	var members, others []*mage.Permanent
	for i, subtypes := range [][]string{{"Sand", "Warrior"}, {"Sand"}, {"Warrior"}, {"Sand", "Warrior"}} {
		card := mage.NewCreature("Subject", "", 2, 2, mage.WithSubTypes(subtypes...), mage.WithKeyword(core.Shroud), mage.WithKeyword(core.Indestructible))
		card.SetOwner(b.PlayerID())
		q := g.PutOnBattlefield(card, b.PlayerID())
		if i == 0 {
			members = append(members, q)
		} else {
			others = append(others, q)
		}
		if i == 3 {
			g.PhaseOut(q.ID())
		}
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	late := g.PutOnBattlefield(mage.NewCreature("Late member", "", 2, 2, mage.WithSubTypes("Sand", "Warrior")), a.PlayerID())
	members = append(members, late)
	g.PutTriggersOnStack()
	clone := g.Clone()
	g.ResolveStack()
	for _, q := range members {
		if g.FindPermanent(q.ID()) != nil || clone.FindPermanent(q.ID()) == nil {
			t.Fatal("resolution membership, shroud, indestructible or clone exile")
		}
	}
	for _, q := range others {
		if g.FindPermanentIncludingPhased(q.ID()) == nil {
			t.Fatal("wrong subtype intersection or phased membership")
		}
	}
	clone.ResolveStack()
	for _, q := range members {
		if clone.FindPermanent(q.ID()) != nil {
			t.Fatal("clone lost the pending exile collection")
		}
	}
}

func TestRONSourceLeaveStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	paths, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Goblin Shrine"`) || strings.Contains(line, `Name: "Hazezon Tamar"`) {
				files[filepath.Base(path)] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("strict complete records", err)
	}
	first, err := registry.CreateCard("Hazezon Tamar")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Hazezon Tamar")
	if err != nil || first.ID() == second.ID() || first.Abilities()[1] == second.Abilities()[1] {
		t.Fatal("factory instances share source state", err)
	}
	files["unsupported.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnsupportedRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Hazezon Tamar") {
		t.Fatal("rejected collection replaced the installed registry")
	}
}

func TestRONSourceLeaveShrineSourceInformation(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "DRK", "Goblin Shrine")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.MutablePermanent(p.ID()).GrantBaseAttr(core.Lifelink)
	land := g.PutOnBattlefield(mage.NewLand("Host"), a.PlayerID())
	g.Attach(p.ID(), land.ID())
	x := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5, mage.WithSubTypes("Goblin"), mage.WithKeyword(core.Shroud)), a.PlayerID())
	y := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5, mage.WithSubTypes("Goblin")), b.PlayerID())
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(c.ID())
	p = g.PutOnBattlefield(c, b.PlayerID())
	g.Attach(p.ID(), land.ID())
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	callbacks := 0
	g.SetOnDamageDealt(func(_ string, _ string, amount int, combat bool) {
		callbacks++
		if combat || amount != 1 || g.FindPermanent(x.ID()).Damage != g.FindPermanent(y.ID()).Damage {
			t.Error("collection damage was not one simultaneous batch")
		}
	})
	g.ResolveStack()
	if callbacks != 4 || a.Life() != 24 || b.Life() != 20 || g.FindPermanent(x.ID()).Damage != 2 || g.FindPermanent(y.ID()).Damage != 2 {
		t.Fatal("returned source replaced old lifelink/controller or copied damage")
	}
}

func TestRONSourceLeaveShrineMembershipAndBonus(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "DRK", "Goblin Shrine")
	c.SetOwner(a.PlayerID())
	shrine := g.PutOnBattlefield(c, a.PlayerID())
	basic := g.PutOnBattlefield(mage.NewLand("Basic Mountain", mage.WithSuperTypes(core.SuperBasic), mage.WithSubTypes("Mountain")), a.PlayerID())
	nonbasic := g.PutOnBattlefield(mage.NewLand("Nonbasic Mountain", mage.WithSubTypes("Mountain")), a.PlayerID())
	forest := g.PutOnBattlefield(mage.NewLand("Basic Forest", mage.WithSuperTypes(core.SuperBasic), mage.WithSubTypes("Forest")), a.PlayerID())
	goblin := g.PutOnBattlefield(mage.NewCreature("Goblin", "", 2, 3, mage.WithSubTypes("Goblin")), b.PlayerID())
	tribal := g.PutOnBattlefield(mage.NewArtifact("Goblin artifact", "", mage.WithSubTypes("Goblin")), b.PlayerID())
	for _, host := range []*mage.Permanent{basic, nonbasic, forest} {
		g.Attach(shrine.ID(), host.ID())
		want := 2
		if host.ID() == basic.ID() {
			want = 3
		}
		if g.FindPermanent(goblin.ID()).CurrentPower(g) != want {
			t.Fatal("bonus did not read both Basic and Mountain on the current host")
		}
	}
	g.PhaseOut(goblin.ID())
	g.BouncePermanentToHand(g.FindPermanent(shrine.ID()))
	late := g.PutOnBattlefield(mage.NewCreature("Late Goblin", "", 2, 3, mage.WithSubTypes("Goblin")), a.PlayerID())
	protected := g.PutOnBattlefield(mage.NewCreature("Protected Goblin", "", 2, 3, mage.WithSubTypes("Goblin")), b.PlayerID())
	if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.This, mage.AnyPermanent), protected.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(late.ID()).Damage != 1 || g.FindPermanent(protected.ID()).Damage != 0 || g.FindPermanentIncludingPhased(goblin.ID()).Damage != 0 || g.FindPermanent(tribal.ID()).Damage != 0 {
		t.Fatal("resolution membership, creature filter, phasing or prevention")
	}
}

func TestRONSourceLeaveIndependentEvents(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "DRK", "Goblin Shrine")
	c.SetOwner(a.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Host"), a.PlayerID())
	goblin := g.PutOnBattlefield(mage.NewCreature("Goblin", "", 1, 5, mage.WithSubTypes("Goblin")), b.PlayerID())
	for range 2 {
		p := g.PutOnBattlefield(c, a.PlayerID())
		g.Attach(p.ID(), land.ID())
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
		a.RemoveFromHand(c.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("separate departures lost a pending event")
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(goblin.ID()).Damage != 2 || clone.FindPermanent(goblin.ID()).Damage != 0 {
		t.Fatal("pending events shared scratch state")
	}
	clone.ResolveStack()
	if clone.FindPermanent(goblin.ID()).Damage != 2 {
		t.Fatal("clone lost separate pending source incarnations")
	}
}

func TestRONSourceLeaveInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Hazezon Tamar"`) {
			record = line
		}
	}
	report, err := InspectFS(fstest.MapFS{"hazezon.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	eventSubject, exileSubtype := false, false
	for _, sig := range report.Signatures {
		if sig.Name == "SinglePermanent" && sig.Role == "permanent_filter" && strings.Contains(sig.Examples[0].Path, "Rules[1]") {
			eventSubject = true
		}
		if sig.Name == "Sand" && sig.Role == "creature_subtype" && strings.Contains(sig.Examples[0].Path, "Rules[1]") {
			exileSubtype = true
		}
	}
	if !eventSubject || !exileSubtype {
		t.Fatal("leave subject or nested exile filter has no semantic role")
	}
}

func TestRONSourceLeaveShrine(t *testing.T) {
	for _, move := range []string{"bounce", "destroy", "exile", "return", "control", "phase", "remove-ability", "host-leaves"} {
		t.Run(move, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "DRK", "Goblin Shrine")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Mountain", mage.WithSubTypes("Mountain"), mage.WithSuperTypes(core.SuperBasic)), a.PlayerID())
			land.Card.SetOwner(a.PlayerID())
			g.Attach(p.ID(), land.ID())
			goblin := g.PutOnBattlefield(mage.NewCreature("Goblin", "", 2, 3, mage.WithSubTypes("Goblin")), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
			if g.FindPermanent(goblin.ID()).CurrentPower(g) != 3 {
				t.Fatal("basic Mountain did not supply the global bonus")
			}
			switch move {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: land.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
			case "remove-ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			switch move {
			case "bounce", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			case "host-leaves":
				g.DestroyPermanent(g.FindPermanent(land.ID()))
				g.CheckStateBasedActions()
			case "exile":
				g.ExilePermanent(g.FindPermanent(p.ID()))
			case "phase":
			default:
				g.DestroyPermanent(g.FindPermanent(p.ID()))
			}
			if move == "return" {
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, b.PlayerID())
				g.Attach(p.ID(), land.ID())
			}
			g.PutTriggersOnStack()
			want := 1
			if move == "phase" || move == "remove-ability" {
				want = 0
			}
			if g.StackSize() != want || g.FindPermanent(goblin.ID()).Damage != 0 {
				t.Fatalf("stack=%d want=%d, premature damage", g.StackSize(), want)
			}
			if want == 1 {
				controller := a.PlayerID()
				if move == "control" {
					controller = b.PlayerID()
				}
				if g.StackPeek().Controller != controller {
					t.Fatal("departure trigger lost its controller")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if g.FindPermanent(goblin.ID()).Damage != want || g.FindPermanent(other.ID()).Damage != 0 || clone.FindPermanent(goblin.ID()).Damage != 0 {
				t.Fatal("wrong collection or shared clone damage")
			}
			clone.ResolveStack()
			if clone.FindPermanent(goblin.ID()).Damage != want {
				t.Fatal("clone lost pending source information")
			}
		})
	}
}

func TestRONSourceLeaveInvalid(t *testing.T) {
	for _, event := range []string{
		`WhenAPermanentLeavesTheBattlefield`, `WhenAPermanentLeavesTheBattlefield()`,
		`WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent),AnyPermanent)`,
		`WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent()))`,
		`WhenAPermanentLeavesTheBattlefield(SinglePermanent(HostPermanent))`,
		`WhenAPermanentLeavesTheBattlefield(SinglePermanent(Ref_TargetPermanent))`,
		`WhenAPermanentLeavesTheBattlefield(IsCardtype(Creature))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(` + event + `,ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-leave.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid event accepted", event)
		}
	}
	for _, rule := range []string{
		`Activated(PayMana([]),ActionList([PermanentDealsDamage(Trigger_ThatPermanent,Integer(1),EachPermanent(IsCardtype(Creature)))]))`,
		`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([PermanentDealsDamage(Trigger_ThatCreature,Integer(1),EachPermanent(IsCardtype(Creature)))]))`,
		`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(1))])),Activated(PayMana([]),ActionList([PermanentDealsDamage(Trigger_ThatPermanent,Integer(1),EachPermanent(IsCardtype(Creature)))]))`,
		`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([Exile([EachPermanent(Ref_TargetPermanents)],[])]))`,
		`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([Exile([EachPermanent(And([]))],[])]))`,
		`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([Exile([EachPermanent(IsCreatureType())],[])]))`,
		`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([Exile([EachPermanent(AnyPermanent)],[UnknownModifier])]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid-binding.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unbound event source accepted")
		}
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentManaCostMerseine(t *testing.T) {
	factory, err := Compile(hostUntapRecord(t, "FEM", "Merseine"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "{1}{G}", 2, 4), b.PlayerID())
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Blue, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.AttachedTo != host.ID() || source.Counters[core.Net] != 3 {
		t.Fatal("entry or attachment")
	}
	a.ManaPool().Add(core.Green, 10)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 2, nil); err == nil || a.ManaPool().TotalMana() != 10 {
		t.Fatal("Aura controller paid")
	}
	b.ManaPool().Add(core.Green, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 2, nil); err == nil || b.ManaPool().TotalMana() != 1 {
		t.Fatal("incomplete payment")
	}
	for count := 3; count >= 0; count-- {
		g.TapPermanent(g.FindPermanent(host.ID()))
		g.SetActivePlayerIndex(1)
		g.RunStepWithPriority(core.Untap)
		if g.FindPermanent(host.ID()).Tapped != (count > 0) {
			t.Fatalf("untap count %d", count)
		}
		if count == 0 {
			break
		}
		b.ManaPool().Add(core.Green, 2)
		if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 2, nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(source.ID()).Counters[core.Net] != count {
			t.Fatal("removed before resolution")
		}
		g.ResolveStack()
		if g.FindPermanent(source.ID()).Counters[core.Net] != count-1 {
			t.Fatal("counter removal")
		}
	}
	if b.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong host cost")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 2, nil); err == nil {
		t.Fatal("unpaid zero-counter activation")
	}
	b.ManaPool().Add(core.Green, 2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 2, nil); err != nil {
		t.Fatal("zero counters should not prohibit payment", err)
	}
	g.ResolveStack()
}

func TestRONPermanentManaCostInvalid(t *testing.T) {
	for _, cost := range []string{`PayManaCostOfPermanent`, `PayManaCostOfPermanent()`, `PayManaCostOfPermanent(HostPermanent,ThisPermanent)`, `PayManaCostOfPermanent(HostPermanent())`, `PayManaCostOfPermanent(Ref_TargetPermanent)`, `PayManaCostOfPermanent(ActionPermanent)`, `PayManaCostOfPermanent(Trigger_ThatPermanent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(`+cost+`,ActionList([GainLife(Integer(1))]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", cost)
		}
	}
}

func TestRONPermanentManaCostLifecycle(t *testing.T) {
	factory, err := Compile(hostUntapRecord(t, "FEM", "Merseine"))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"move-before", "control-before", "host-phase", "source-phase", "phase-cycle", "host-return", "source-return", "move-after", "control-after", "source-leaves", "copy", "clone", "two", "abilities-removed"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "{G}", 2, 4), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "{2}{U}", 2, 4), a.PlayerID())
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			a.ManaPool().Add(core.Blue, 3)
			b.ManaPool().Add(core.Green, 1)
			actor := b.PlayerID()
			remaining := 2
			switch mode {
			case "move-before":
				g.Attach(source.ID(), other.ID())
				actor = a.PlayerID()
			case "control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				a.ManaPool().Add(core.Green, 1)
				actor = a.PlayerID()
			case "host-phase":
				g.PhaseOut(host.ID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(host.Card, b.PlayerID())
			case "abilities-removed":
				removal := mage.RemoveAllAbilities(source.ID(), core.Indefinite)
				removal.SetSourceID(other.ID())
				g.AddContinuousEffect(removal)
			}
			err := g.ActivateAbilityByIndex(actor, source.ID(), 2, nil)
			reject := mode == "host-phase" || mode == "source-phase" || mode == "host-return" || mode == "abilities-removed"
			if reject {
				if err == nil || b.ManaPool().TotalMana() != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
					t.Fatal("illegal activation spent resources", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				source = g.PutOnBattlefield(source.Card, a.PlayerID())
				g.Attach(source.ID(), host.ID())
				remaining = 3
			case "move-after":
				g.Attach(source.ID(), other.ID())
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				remaining = 1
			case "two":
				second := g.PutOnBattlefield(factory(), a.PlayerID())
				g.Attach(second.ID(), host.ID())
				if second.Counters[core.Net] != 3 {
					t.Fatal("instances shared counters")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				current := game.FindPermanent(source.ID())
				if mode == "source-leaves" {
					if current != nil {
						t.Fatal("source restored")
					}
					continue
				}
				if current == nil || current.Counters[core.Net] != remaining {
					t.Fatalf("pending source visit: want %d", remaining)
				}
			}
			if mode == "clone" {
				g.MutablePermanent(source.ID()).Counters[core.Net] = 0
				if clone.FindPermanent(source.ID()).Counters[core.Net] != 2 {
					t.Fatal("clone shared counters")
				}
			}
		})
	}
}

func TestRONPermanentManaCostContexts(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayManaCostOfPermanent(ThisPermanent),ActionList([AddMana(ManaProduceG)]))`,
		`Activated(PayManaCostOfPermanent(HostPermanent),ActionList([GainLife(Integer(1))]))`,
		`SpellActions(ActionList([PayManaCostOfPermanent(ThisPermanent)]))`,
		`AsPermanentEnters(ThisPermanent,[PayManaCostOfPermanent(ThisPermanent)])`,
		`CastEffect(PayManaCostOfPermanent(ThisPermanent))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid context accepted", rule)
		}
	}
	factory := permanentScopeFactory(t, "Artifact", `Activated(And([PayManaCostOfPermanent(ThisPermanent),PayMana([ManaCostU])]),ActionList([GainLife(Integer(1))]))`)
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	a.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 {
		t.Fatal("absent source mana cost paid")
	}
}

func TestRONPermanentManaCostStrictLoadAndInventory(t *testing.T) {
	record := hostUntapRecord(t, "FEM", "Merseine")
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Merseine", "Card", "card", "collection", true, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "HostPermanent" && entry.Role == "permanent_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("payment subject role missing")
	}
	src := `Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),ManaCost:[ManaCostGeneric(0)],Rules:[Activated(PayManaCostOfPermanent(ThisPermanent),ActionList([GainLife(Integer(1))]))])`
	registry := mage.NewRegistry()
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["b.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(src, `Name:"Valid"`, `Name:"Invalid"`, 1), "ThisPermanent", "HostPermanent"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Valid") || registry.CardRegistered("Invalid") {
		t.Fatal("invalid collection published")
	}
}

func TestRONPermanentManaCostManaSources(t *testing.T) {
	factory, err := Compile(hostUntapRecord(t, "FEM", "Merseine"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "{1}{G}", 2, 4), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	forest := repositoryCard(t, "LEA", "Forest")
	forest.SetOwner(b.PlayerID())
	first := g.PutOnBattlefield(forest, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 2, nil); err == nil || first.Tapped || b.ManaPool().TotalMana() != 0 {
		t.Fatal("failed automatic mana spent resources")
	}
	second := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || b.ManaPool().TotalMana() != 0 {
		t.Fatal("automatic payment incorrect")
	}
	g.ResolveStack()
	if g.FindPermanent(source.ID()).Counters[core.Net] != 2 {
		t.Fatal("paid effect missing")
	}
}

package ron

import (
	"fmt"
	"strings"
	"testing"

	"os"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDetonateManaValueX(t *testing.T) {
	for _, change := range []string{"normal", "unpaid", "wrong-x", "wrong-type", "shroud", "phase", "phase-cycle", "return", "control", "indestructible", "regen", "clone", "copy", "zero", "mana-change"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			x := 2
			if change == "zero" {
				x = 0
			}
			victimCard := mage.NewArtifact("Victim", fmt.Sprintf("{%d}", x))
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			id := victim.ID()
			if change == "wrong-type" {
				id = g.PutOnBattlefield(mage.NewCreature("Creature", "{2}", 2, 3), b.PlayerID()).ID()
			}
			if change == "wrong-x" {
				x = 3
			}
			spell := repositoryCard(t, "ATQ", "Detonate")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if change != "unpaid" {
				a.ManaPool().Add(core.Red, 1)
				a.ManaPool().Add(core.Colorless, x)
			}
			before := a.ManaPool().TotalMana()
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{id}, x)
			if change == "unpaid" || change == "wrong-x" || change == "wrong-type" {
				if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != before {
					t.Fatal("invalid cast changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.Life() != 20 || g.FindPermanent(id) == nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("cost or pre-resolution timing")
			}
			pending := g.Clone()
			recipient := b.PlayerID()
			switch change {
			case "shroud":
				g.MutablePermanent(id).GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(id)
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(id))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(id))
				g.PutOnBattlefield(victimCard, b.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: id, ControllerID: a.PlayerID(), Duration: core.Indefinite})
				recipient = a.PlayerID()
			case "indestructible":
				g.MutablePermanent(id).GrantBaseAttr(core.Indestructible)
			case "regen":
				g.AddRegenerationShield(id)
			case "mana-change":
				changed := g.MutablePermanent(id).Card.Copy().(*mage.BaseCard)
				changed.CloneFrom(mage.NewArtifact("Changed", "{3}"))
				changed.SetID(id)
				g.MutablePermanent(id).Card = changed
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			illegal := change == "shroud" || change == "phase" || change == "return" || change == "mana-change"
			damage := x
			if illegal {
				damage = 0
			}
			if g.GetPlayer(recipient).Life() != 20-damage {
				t.Fatalf("recipient life=%d", g.GetPlayer(recipient).Life())
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{id})
			}
			remains := illegal || change == "indestructible"
			if (g.FindPermanent(id) != nil) != remains {
				t.Fatal("wrong destruction identity or result")
			}

			if pending.FindPermanent(id) == nil || pending.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution")
			}
			pending.ResolveStack()
			if pending.FindPermanent(id) != nil || pending.GetPlayer(b.PlayerID()).Life() != 20-x {
				t.Fatal("clone lost X or targets")
			}
		})
	}
}

func TestRONSpellBlastManaValueX(t *testing.T) {
	for _, change := range []string{"normal", "wrong-x", "unpaid", "left", "recast", "copy-target", "copy-counter", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			victim := mage.NewInstant("Variable", "{X}{X}{U}", mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(3))))
			victim.SetOwner(b.PlayerID())
			b.AddToHand(victim)
			b.ManaPool().Add(core.Blue, 1)
			b.ManaPool().Add(core.Colorless, 4)
			if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, 2); err != nil {
				t.Fatal(err)
			}
			targetID := g.StackPeek().ID
			if change == "copy-target" {
				targetID = g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false).ID
			}
			spell := repositoryCard(t, "LEA", "Spell Blast")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			x := 5
			if change == "wrong-x" {
				x = 2
			}
			if change != "unpaid" {
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Colorless, x)
			}
			before := a.ManaPool().TotalMana()
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{targetID}, x)
			if change == "wrong-x" || change == "unpaid" {
				if err == nil || len(a.Hand()) != 1 || g.StackSize() != 1 || a.ManaPool().TotalMana() != before {
					t.Fatal("illegal counter changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || b.Life() != 20 {
				t.Fatal("cost or timing")
			}
			clone := g.Clone()
			if change == "left" || change == "recast" {
				g.CounterStackObject(targetID)
			}
			if change == "recast" {
				b.AddToHand(victim)
				b.ManaPool().Add(core.Blue, 1)
				b.ManaPool().Add(core.Colorless, 4)
				if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, 2); err != nil {
					t.Fatal(err)
				}
				g.ResolveTopOfStack()
			}
			if change == "copy-counter" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveTopOfStack()
			if change == "copy-counter" {
				g.ResolveTopOfStack()
			}
			want := 0
			if change == "copy-target" {
				want = 1
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want=%d", g.StackSize(), want)
			}
			if change == "recast" && b.Life() != 23 {
				t.Fatal("counter followed a new stack visit")
			}
			initial := 2
			if change == "copy-target" {
				initial = 3
			}
			if clone.StackSize() != initial {
				t.Fatal("clone shared stack")
			}
			clone.ResolveTopOfStack()
			if clone.StackSize() != want {
				t.Fatal("clone counter failed")
			}
			if change == "copy-target" {
				g.ResolveStack()
				if b.Life() != 23 {
					t.Fatal("countered original instead of copy")
				}
			}
		})
	}
}

func TestRONManaValueXDeclarations(t *testing.T) {
	for _, kind := range []string{"Permanent", "Spell"} {
		for _, filter := range []string{"ManaValueIs", "ManaValueIs()", "ManaValueIs(EqualTo)", "ManaValueIs(EqualTo(ValueX()))", "ManaValueIs(EqualTo(Integer(2)))", "ManaValueIs(LessThan(ValueX))", "ManaValueIs(EqualTo(Ref_TargetPlayer))", "Or([AnySpell,ManaValueIs(EqualTo(ValueX))])"} {
			src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(Targeted([Target%s(%s)],ActionList([])))])`, kind, filter)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatalf("accepted %s", src)
			}
		}
	}
}

func TestRONManaValueXRetargetAndInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewArtifact("First", "{2}"), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewArtifact("Second", "{2}"), b.PlayerID())
	g.PhaseOut(second.ID())
	spell := repositoryCard(t, "ATQ", "Detonate")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID()}, 2); err != nil {
		t.Fatal(err)
	}
	g.PhaseIn([]uuid.UUID{second.ID()})
	g.PhaseOut(first.ID())
	cp := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != second.ID() || cp.XValue != 2 {
		t.Fatal("copy did not use its X to retarget")
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || b.Life() != 18 {
		t.Fatal("copy binding overwrote original")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()) != nil || b.Life() != 16 {
		t.Fatal("original binding or X lost")
	}
}

func TestRONManaValueXStrictLoadAndInventory(t *testing.T) {
	entries, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	records := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Detonate" && name != "Spell Blast" {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, factory()); err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(data), "\n")
			line := lines[node.Position.Line-1]
			file := fmt.Sprintf("%d.ron", records)
			files[file] = &fstest.MapFile{Data: []byte(line)}
			records++
		}
	}
	if records != 7 {
		t.Fatalf("records=%d", records)
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("strict load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 7 || report.UniqueCards != 2 {
		t.Fatalf("inventory: %v", err)
	}
	for _, sig := range report.Signatures {
		if sig.Name == "EqualTo" && sig.Role != "mana_value_comparison" {
			t.Fatalf("comparison role=%s", sig.Role)
		}
		if sig.Name == "ValueX" && sig.Role != "value" {
			t.Fatalf("X role=%s", sig.Role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(ManaValueIs(EqualTo(ValueX)))],ActionList([])))])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection exposed partial registry")
	}
}

func TestRONManaValueXIndependentPendingCounters(t *testing.T) {
	g, a, b := ronGame()
	ids := make([]uuid.UUID, 0, 2)
	for _, cost := range []string{"{U}", "{1}{U}"} {
		victim := mage.NewInstant("Victim "+cost, cost, mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))))
		victim.SetOwner(b.PlayerID())
		b.AddToHand(victim)
		b.ManaPool().Add(core.Blue, 1)
		b.ManaPool().Add(core.Colorless, 1)
		if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, g.StackPeek().ID)
	}
	for i, id := range ids {
		spell := repositoryCard(t, "LEA", "Spell Blast")
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Blue, 1)
		a.ManaPool().Add(core.Colorless, i+1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{id}, i+1); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveTopOfStack()
	if g.StackSize() != 2 || g.GetStack().FindByID(ids[0]) == nil || g.GetStack().FindByID(ids[1]) != nil {
		t.Fatal("second instance used the first target or X")
	}
	g.ResolveTopOfStack()
	if g.StackSize() != 0 || b.Life() != 20 {
		t.Fatal("first instance lost its target or X")
	}
}

func TestRONSpellBlastZeroAndAbilityExclusion(t *testing.T) {
	for _, ability := range []bool{false, true} {
		g, a, b := ronGame()
		var targetID uuid.UUID
		if ability {
			source := g.PutOnBattlefield(mage.NewArtifact("Ability source", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.GainLifeFor(mage.You, mage.Fixed(2))))), b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			targetID = g.StackPeek().ID
		} else {
			victim := mage.NewInstant("Zero spell", "", mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(2))))
			victim.SetOwner(b.PlayerID())
			b.AddToHand(victim)
			if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			targetID = g.StackPeek().ID
		}
		blast := repositoryCard(t, "LEA", "Spell Blast")
		blast.SetOwner(a.PlayerID())
		a.AddToHand(blast)
		a.ManaPool().Add(core.Blue, 1)
		err := g.CastSpellByID(a.PlayerID(), blast.ID(), []uuid.UUID{targetID}, 0)
		if ability {
			if err == nil || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 1 {
				t.Fatal("ability accepted or payment changed")
			}
			g.ResolveStack()
			if b.Life() != 22 {
				t.Fatal("failed cast changed ability")
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if b.Life() != 20 || g.StackSize() != 0 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("zero mana value counter failed")
			}
		}
	}
}

package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttachmentTargetQueryLowering(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"DRK", "Miracle Worker"}, {"LEG", "Enchantment Alteration"}} {
		record := repositoryRecord(t, tc.set, tc.name, "attachment-query.ron")
		count := 0
		var walk func(Node)
		walk = func(n Node) {
			if n.Name == "TargetPermanent" {
				filter, opts, err := compilePermanentTargetFilter(n.Args[0], scope{permanent: true})
				if err != nil {
					t.Fatal(err)
				}
				if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("record", filter, opts...)); !ok {
					t.Fatal(tc.name, "full host predicate opaque")
				}
				count++
			}
			for _, child := range n.Args {
				walk(child)
			}
			for _, child := range n.Fields {
				walk(child)
			}
		}
		walk(record)
		if count != 1 {
			t.Fatal("target count changed", count)
		}
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONAttachmentTargetQueryBindings(t *testing.T) {
	for _, state := range []string{"legal", "host control", "host shroud", "reattach", "phase", "cycle", "return", "source leaves", "copy", "copy controller", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, source, host, aura := attachmentTargetFixture(t, "Miracle Worker")
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{aura.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
			if !ok || len(instructions) != 1 {
				t.Fatal("lowered instruction missing")
			}
			plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
			if !ok {
				t.Fatal("lowered attachment binding opaque")
			}
			want := true
			switch state {
			case "host control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = false
			case "host shroud":
				host.GrantBaseAttr(core.Shroud)
			case "reattach":
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 4), a.PlayerID())
				g.Attach(aura.ID(), other.ID())
			case "phase":
				g.PhaseOut(host.ID())
				want = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "return":
				g.RemoveFromBattlefield(aura)
				aura = g.PutOnBattlefield(aura.Card, b.PlayerID())
				g.Attach(aura.ID(), host.ID())
				want = false
			case "source leaves":
				g.RemoveFromBattlefield(source)
			case "copy":
				obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
			case "copy controller":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				want = false
			case "clone":
				clone := g.Clone()
				clone.PhaseOut(host.ID())
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone did not use its host state")
				}
			}
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 1) != want {
				t.Fatalf("selection %v/%v want %v", got, known, want)
			}
			otherSource := g.PutOnBattlefield(repositoryCard(t, "DRK", "Miracle Worker"), a.PlayerID())
			otherSource.RevokeBaseAttr(core.AttrSummonSick)
			if state == "legal" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), otherSource.ID(), 0, []uuid.UUID{aura.ID()}); err != nil {
					t.Fatal(err)
				}
				if _, known := plan.Select(g, g.StackPeek()); known {
					t.Fatal("factory declaration shared")
				}
			}
			if state == "copy controller" {
				g.ResolveTopOfStack()
				if g.FindPermanent(aura.ID()) == nil {
					t.Fatal("copy used original controller")
				}
				return
			}
			g.ResolveStack()
			if state != "phase" && (g.FindPermanent(aura.ID()) == nil) != want {
				t.Fatal("gameplay differs from query")
			}
		})
	}
}

func TestRONAttachmentTargetQueryInvalid(t *testing.T) {
	for _, syntax := range []string{
		`AttachedToAPermanent()`, `AttachedToAPermanent(IsCardtype())`, `AttachedToAPermanent(IsCardtype(Creature,Land))`, `AttachedToAPermanent(IsCardtype(Creature),IsTapped)`,
		`AttachedToAPermanent(Or([IsCardtype(Land),Unknown]))`,
		`AttachedToAPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You()))]))`,
		`AttachedToAPermanent(Or([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))`,
		`AttachedToAPermanent(AttachedToAPermanent(IsCardtype(Land)))`,
		`AttachedToAPermanent(SinglePermanent(Ref_TargetPermanent))`,
	} {
		_, _, err := compilePermanentTargetFilter(nameScopeNode(t, syntax), scope{})
		if err == nil {
			t.Fatal("invalid host accepted", syntax)
		}
	}
}

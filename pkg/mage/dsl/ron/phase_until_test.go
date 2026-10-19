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

func TestRONPhaseUntilOubliette(t *testing.T) {
	for _, change := range []string{"none", "source-left", "source-returned", "source-phased", "target-left", "target-phased", "shroud", "copy", "token"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			targetCard := mage.NewCreature("Subject", "", 2, 3)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if change == "token" {
				g.MutablePermanent(target.ID()).IsToken = true
			}
			card := repositoryCard(t, "ARN", "Oubliette")
			castRONSpell(t, g, a, card)
			g.ResolveTopOfStack()
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.FindPermanent(target.ID()) == nil {
				t.Fatal("entry trigger timing")
			}
			source := g.FindPermanent(card.ID())
			switch change {
			case "source-left", "source-returned":
				g.RemoveFromBattlefield(source)
				if change == "source-returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "source-phased":
				g.PhaseOut(source.ID())
			case "target-left":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phased":
				g.PhaseOut(target.ID())
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if change == "source-returned" {
				clone.ResolveTopOfStack()
			} else {
				clone.ResolveStack()
			}
			if g.FindPermanentIncludingPhased(target.ID()).PhasedOut != (change == "target-phased") {
				t.Fatal("pending clone changed original")
			}
			if change == "source-returned" {
				g.ResolveTopOfStack()
			} else {
				g.ResolveStack()
			}
			want := change == "token" || change == "none" || change == "source-phased" || change == "copy" || change == "target-phased"
			if clone.FindPermanentIncludingPhased(target.ID()).PhasedOut != want {
				t.Fatal("clone lost target or source identity")
			}
			if g.FindPermanentIncludingPhased(target.ID()).PhasedOut != want {
				t.Fatal("wrong resolution phase status")
			}
			if change == "token" || change == "none" || change == "source-phased" || change == "copy" {
				if change == "source-phased" {
					g.PhaseIn([]uuid.UUID{source.ID()})
				}
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if p := g.FindPermanent(target.ID()); p == nil || !p.Tapped {
					t.Fatal("departure did not return target tapped")
				}
				if change == "token" && !g.FindPermanent(target.ID()).IsToken {
					t.Fatal("phasing lost token status")
				}
			}
		})
	}
}

const phaseUntilAction = `PhaseOutPermanentUntilWithEffects(Ref_TargetPermanent,UntilPermanentLeavesBattlefield(ThisPermanent),TapAsPhasesIn)`

func TestRONPhaseUntilDeclarations(t *testing.T) {
	valid := `Card(Name:"Valid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + phaseUntilAction + `])))])`
	for _, action := range []string{
		`PhaseOutPermanentUntilWithEffects()`,
		strings.Replace(phaseUntilAction, "Ref_TargetPermanent", "Ref_TargetPermanent()", 1),
		strings.Replace(phaseUntilAction, "Ref_TargetPermanent", "Ref_TargetPermanent_2", 1),
		strings.Replace(phaseUntilAction, "Ref_TargetPermanent", "You", 1),
		strings.Replace(phaseUntilAction, "UntilPermanentLeavesBattlefield(ThisPermanent)", "UntilEndOfTurn", 1),
		strings.Replace(phaseUntilAction, "ThisPermanent)", "HostPermanent)", 1),
		strings.Replace(phaseUntilAction, "ThisPermanent)", "ThisPermanent())", 1),
		strings.Replace(phaseUntilAction, "TapAsPhasesIn", "TapAsPhasesIn()", 1),
		strings.Replace(phaseUntilAction, "TapAsPhasesIn", "UntapAsPhasesIn", 1),
		strings.Replace(phaseUntilAction, "UntilPermanentLeavesBattlefield(ThisPermanent)", "UntilPermanentLeavesBattlefield(ThisPermanent,ThisPermanent)", 1),
	} {
		n := nameScopeNode(t, strings.Replace(valid, phaseUntilAction, action, 1))
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted invalid action", action)
		}
	}
	for _, source := range []string{
		strings.Replace(valid, "[Enchantment]", "[Sorcery]", 1),
		strings.Replace(valid, "Targeted([TargetPermanent(IsCardtype(Creature))],ActionList(["+phaseUntilAction+"]))", "ActionList(["+phaseUntilAction+"])", 1),
		`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + strings.Replace(phaseUntilAction, "Ref_TargetPermanent", "ThisPermanent", 1) + `,ActionList([]))])`,
	} {
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatal("accepted invalid context", source)
		}
	}
}

func TestRONPhaseUntilLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Oubliette"`) {
			source = line
			break
		}
	}
	if source == "" {
		t.Fatal("missing complete record")
	}
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Oubliette")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Oubliette")
	if err != nil {
		t.Fatal(err)
	}
	one := first.Abilities()[0].(*mage.GenericTriggered)
	two := second.Abilities()[0].(*mage.GenericTriggered)
	if one.Targets()[0] == two.Targets()[0] {
		t.Fatal("shared target bindings")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "TapAsPhasesIn", "TapAsPhasesIn()", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Oubliette") {
		t.Fatal("load was not atomic")
	}
	delete(files, "invalid.ron")
	inventory, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"PhaseOutPermanentUntilWithEffects": "action", "UntilPermanentLeavesBattlefield": "duration", "TapAsPhasesIn": "phase_in_operation"} {
		found := false
		for _, sig := range inventory.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing inventory role", name, role)
		}
	}
}

func TestRONPhaseUntilCasting(t *testing.T) {
	g, a, b := ronGame()
	target := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 3), b.PlayerID())
	card := repositoryCard(t, "ARN", "Oubliette")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast mutated resources")
	}
	a.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) != nil || g.FindPermanent(target.ID()) == nil {
		t.Fatal("cast payment or timing")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(target.ID()) == nil || g.StackSize() != 1 {
		t.Fatal("entry trigger did not wait")
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()) != nil {
		t.Fatal("trigger did not phase out target")
	}
}

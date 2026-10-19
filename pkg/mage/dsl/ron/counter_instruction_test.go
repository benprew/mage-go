package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterInstructionGameplay(t *testing.T) {
	for _, record := range []struct{ set, name string }{
		{"LEA", "Counterspell"}, {"LEB", "Counterspell"}, {"CED", "Counterspell"}, {"CEI", "Counterspell"}, {"2ED", "Counterspell"}, {"3ED", "Counterspell"},
		{"LEG", "Flash Counter"}, {"LEG", "Remove Soul"}, {"ATQ", "Artifact Blast"},
	} {
		name := record.name
		for _, state := range []string{"normal", "left", "recast", "copy", "clone", "uncounterable"} {
			t.Run(record.set+"/"+name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				victimCard := mage.NewInstant("Victim", "", mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(2)))))
				if name == "Remove Soul" {
					victimCard = mage.NewCreature("Victim", "", 2, 2, mage.WithKeyword(core.Flash))
				}
				if name == "Artifact Blast" {
					victimCard = mage.NewArtifact("Victim", "", mage.WithKeyword(core.Flash))
				}
				if state == "uncounterable" {
					mage.WithUncounterable()(victimCard)
				}
				victim := castRONSpell(t, g, a, victimCard)
				spell := castRONSpell(t, g, b, repositoryCard(t, record.set, name), victim.ID)
				instruction, ok := mage.InspectCounterInstructions(spell.Effects[0])
				if !ok || len(instruction) != 1 {
					t.Fatal("RON lost native counter declaration")
				}
				predicate, ok := mage.InspectSpellTarget(instruction[0].Subject)
				if !ok {
					t.Fatal("RON lost native spell target")
				}
				switch state {
				case "left", "recast":
					g.CounterStackObject(victim.ID)
					if state == "recast" {
						a.RemoveFromGraveyard(victimCard.ID())
						castRONSpell(t, g, a, victimCard)
					}
				case "copy":
					spell = g.CopyStackObjectDirect(spell, b.PlayerID(), false)
				case "clone":
					original := g
					defer func() {
						if original.StackSize() != 2 {
							t.Fatal("clone changed original stack")
						}
					}()
					g = g.Clone()
					spell = g.StackPeek()
				}
				want := victim.ID
				if state == "left" || state == "recast" {
					want = uuid.Nil
				}
				before := g.StackSize()
				for range 2 {
					selected, known := predicate.Select(g, spell)
					if !known || selected != want {
						t.Fatalf("selection = %v/%v", selected, known)
					}
					if certain, known := g.InspectSpellDestruction(spell, nil); certain || !known {
						t.Fatalf("counter inspection = %v/%v", certain, known)
					}
				}
				if before != g.StackSize() {
					t.Fatal("inspection changed stack")
				}
				g.ResolveStack()
				player := g.GetPlayer(a.PlayerID())
				inGraveyard := false
				for _, card := range player.Graveyard() {
					if card.ID() == victimCard.ID() {
						inGraveyard = true
					}
				}
				wantLife := 20
				if (name == "Counterspell" || name == "Flash Counter") && (state == "uncounterable" || state == "recast") {
					wantLife = 22
				}
				if player.Life() != wantLife {
					t.Fatal("countered spell executed or surviving spell did not resolve")
				}
				permanent := name == "Remove Soul" || name == "Artifact Blast"
				survives := permanent && (state == "recast" || state == "uncounterable")
				if inGraveyard == survives {
					t.Fatal("counter execution differs")
				}
				if permanent && (g.FindPermanent(victimCard.ID()) != nil) != survives {
					t.Fatal("wrong permanent resolution")
				}

			})
		}
	}
}

func TestRONCounterInstructionInvalid(t *testing.T) {
	for _, body := range []string{`CounterSpell()`, `CounterSpell(Ref_TargetSpell,Ref_TargetSpell)`, `CounterSpell(You)`, `CounterSpell(Ref_TargetSpell2)`, `CounterSpell(Ref_TargetSpell())`} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([`+body+`])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONCounterInstructionBindings(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "LEA", "Counterspell", "LEA.ron"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	victim := castRONSpell(t, g, a, mage.NewInstant("Victim", ""))
	first := castRONSpell(t, g, b, factory(), victim.ID)
	second := castRONSpell(t, g, a, factory(), first.ID)
	one, ok := mage.InspectCounterInstructions(first.Effects[0])
	if !ok {
		t.Fatal("first declaration missing")
	}
	two, ok := mage.InspectCounterInstructions(second.Effects[0])
	if !ok {
		t.Fatal("second declaration missing")
	}
	plan, ok := mage.InspectSpellTarget(one[0].Subject)
	if !ok {
		t.Fatal("first target missing")
	}
	if _, known := plan.Select(g, second); known {
		t.Fatal("same label supplied foreign declaration")
	}
	other, ok := mage.InspectSpellTarget(two[0].Subject)
	if !ok {
		t.Fatal("second target missing")
	}
	if selected, known := other.Select(g, second); !known || selected != first.ID {
		t.Fatal("instance target lost")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.StackSize() != 3 {
		t.Fatal("clone changed live stack")
	}
	g.ResolveStack()
	if g.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("unrelated state changed")
	}
}

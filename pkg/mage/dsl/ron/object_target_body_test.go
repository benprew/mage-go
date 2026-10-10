package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func objectBodyRule(spell, permanent string) string {
	return `SpellActions(Targeted_Modal([Targeted([TargetSpell(AnySpell)],ActionList([` + spell + `])),Targeted([TargetPermanent(AnyPermanent)],ActionList([` + permanent + `]))]))`
}

func TestRONObjectTargetBodyGameplay(t *testing.T) {
	for _, spellTarget := range []bool{false, true} {
		g, a, b := ronModalGame(0)
		p := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID())
		target := p.ID()
		if spellTarget {
			target = castRONSpell(t, g, b, repositoryCard(t, "LEA", "Air Elemental")).ID
		}
		card := permanentScopeFactory(t, "Instant", objectBodyRule(`PlayerActions(ControllerOfSpell(Ref_TargetSpell),[GainLife(Integer(2))])`, `TapPermanent(Ref_TargetPermanent),PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[GainLife(Integer(3))])`))()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
		if a.prompts != 0 || b.Life() != 20 || p.Tapped {
			t.Fatal("body ran at announcement")
		}
		clone := g.Clone()
		g.ResolveTopOfStack()
		want := 23
		if spellTarget {
			want = 22
		}
		if b.Life() != want || a.Life() != 20 || a.prompts != 0 {
			t.Fatal("wrong branch or chooser")
		}
		if clone.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("clone changed")
		}
		clone.ResolveTopOfStack()
		if clone.GetPlayer(b.PlayerID()).Life() != want {
			t.Fatal("clone lost union binding")
		}
		if !spellTarget && !g.FindPermanent(p.ID()).Tapped {
			t.Fatal("typed permanent projection failed")
		}
	}
}

func TestRONObjectTargetBodyRealChoices(t *testing.T) {
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		record := staticBaseRecord(t, "LEA", name)
		for i := range record.Fields["Rules"].Args[0].Args[0].Args[0].Args {
			body := &record.Fields["Rules"].Args[0].Args[0].Args[0].Args[i].Args[1].Args[0]
			body.Args = body.Args[:1]
		}
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		for _, spellTarget := range []bool{false, true} {
			g, base, b := ronGame()
			a := &ronTextWordTester{Player: base}
			g.SetPlayerAt(0, a)
			target := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID()).ID()
			if spellTarget {
				g.SetActivePlayerIndex(1)
				target = castRONSpell(t, g, b, repositoryCard(t, "LEA", "Air Elemental")).ID
			}
			card := factory()
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("choice at announcement")
			}
			g.ResolveTopOfStack()
			if a.calls != 2 {
				t.Fatal("union made a mode choice or lost word choices")
			}
		}
		_, err = Compile(staticBaseRecord(t, "LEA", name))
		if err != nil {
			t.Fatalf("complete text consumer failed: %v", err)
		}
	}
}

func TestRONObjectTargetBodyInvalid(t *testing.T) {
	valid := objectBodyRule(`GainLife(Integer(1))`, `TapPermanent(Ref_TargetPermanent)`)
	for _, rule := range []string{
		strings.Replace(valid, "AnySpell", "IsCardtype(Instant)", 1),
		strings.Replace(valid, "AnyPermanent", "IsCardtype(Artifact)", 1),
		strings.Replace(valid, "TargetPermanent(AnyPermanent)", "TargetSpell(AnySpell)", 1),
		strings.Replace(valid, "TapPermanent(Ref_TargetPermanent)", "PlayerActions(ControllerOfSpell(Ref_TargetSpell),[GainLife(Integer(1))])", 1),
		strings.Replace(valid, "GainLife(Integer(1))", "TapPermanent(Ref_TargetPermanent)", 1),
		strings.Replace(valid, "ActionList([GainLife(Integer(1))])", "Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(Integer(1))]))", 1),
		strings.Replace(valid, "TapPermanent(Ref_TargetPermanent)", "GainLife(TheFirstChosenColorWord)", 1),
		strings.Replace(valid, "ActionList([GainLife(Integer(1))])", "ActionList([])", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONObjectTargetBodyLegalityAndPayment(t *testing.T) {
	for _, state := range []string{"phased", "returned", "countered", "unpaid"} {
		g, base, b := ronGame()
		a := &ronTextWordTester{Player: base}
		g.SetPlayerAt(0, a)
		p := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID())
		target := p.ID()
		if state == "countered" {
			g.SetActivePlayerIndex(1)
			target = castRONSpell(t, g, b, repositoryCard(t, "LEA", "Air Elemental")).ID
		}
		rule := objectBodyRule(`ChooseTwoColorWords,GainLife(Integer(1))`, `ChooseTwoBasicLandTypes,TapPermanent(Ref_TargetPermanent)`)
		nodes, err := Parse(strings.NewReader(`Card(Name:"Union",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostU],Rules:[`+rule+`])`), "payment.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		a.AddToHand(card)
		if state != "unpaid" {
			a.ManaPool().Add(core.Blue, 1)
		}
		err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0)
		if state == "unpaid" {
			if err == nil || len(a.Hand()) != 1 || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("failed payment changed state")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		switch state {
		case "phased":
			g.PhaseOut(p.ID())
		case "returned":
			g.BouncePermanentToHand(p)
			g.PutOnBattlefield(p.Card, b.PlayerID())
		case "countered":
			g.CounterStackObject(target)
		}
		g.ResolveTopOfStack()
		if a.calls != 0 || a.Life() != 20 {
			t.Fatal("illegal target ran body")
		}
	}
}

func TestRONObjectTargetBodyRetargetCopy(t *testing.T) {
	for _, fromSpell := range []bool{false, true} {
		g, a, b := ronModalGame(0)
		g.SetActivePlayerIndex(1)
		spell := castRONSpell(t, g, b, repositoryCard(t, "LEA", "Air Elemental"))
		p := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID())
		first, second := p.ID(), spell.ID
		if fromSpell {
			first, second = second, first
		}
		card := permanentScopeFactory(t, "Instant", objectBodyRule(`PlayerActions(ControllerOfSpell(Ref_TargetSpell),[GainLife(Integer(2))])`, `TapPermanent(Ref_TargetPermanent)`))()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first}, 0); err != nil {
			t.Fatal(err)
		}
		original := g.StackPeek()
		a.targets = []uuid.UUID{second}
		copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
		if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != second || original.Targets[0] != first {
			t.Fatal("copy target isolation")
		}
		g.ResolveTopOfStack()
		if fromSpell {
			if !g.FindPermanent(p.ID()).Tapped || b.Life() != 20 {
				t.Fatal("copy did not switch to permanent")
			}
		} else if b.Life() != 22 || g.FindPermanent(p.ID()).Tapped {
			t.Fatal("copy did not switch to spell")
		}
		g.ResolveTopOfStack()
		if !g.FindPermanent(p.ID()).Tapped || b.Life() != 22 || a.prompts != 0 {
			t.Fatal("original branch changed")
		}
	}
}

func TestRONObjectTargetBodyDepartureInformation(t *testing.T) {
	for _, spellTarget := range []bool{false, true} {
		g, a, b := ronModalGame(0)
		g.SetActivePlayerIndex(1)
		p := g.PutOnBattlefield(mage.NewArtifact("subject", ""), b.PlayerID())
		target := p.ID()
		if spellTarget {
			target = castRONSpell(t, g, b, repositoryCard(t, "LEA", "Air Elemental")).ID
		}
		card := permanentScopeFactory(t, "Instant", objectBodyRule(`CounterSpell(Ref_TargetSpell),PlayerActions(ControllerOfSpell(Ref_TargetSpell),[GainLife(Integer(2))])`, `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent),PlayerActions(ControllerOfPermanent(Ref_TargetPermanent),[GainLife(Integer(3))])`))()
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveTopOfStack()
		want := 23
		if spellTarget {
			want = 22
		}
		if b.Life() != want || g.FindPermanent(p.ID()) != nil && !spellTarget {
			t.Fatal("body lost original controller after departure")
		}
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronColorEntryPlayer struct {
	mage.Player
	answer core.Color
	calls  int
}

func (p *ronColorEntryPlayer) ChooseManaColor(string) core.Color { p.calls++; return p.answer }

const colorEntryRule = `AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor)])`

func colorEntryFixture(rule string) string {
	return `Card(Name:"Color entry",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),ManaCost:[ManaCostU],Rules:[` + rule + `])`
}

func TestRONColorEntryGameplay(t *testing.T) {
	registry, err := LoadFS(fstest.MapFS{"entry.ron": {Data: []byte(colorEntryFixture(colorEntryRule))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Black}
	g.SetPlayerAt(0, chooser)
	card, err := registry.CreateCard("Color entry")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || chooser.calls != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid cast changed state or made a choice")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 {
		t.Fatal("choice at announcement")
	}
	pending := g.Clone()
	other := &ronColorEntryPlayer{Player: pending.GetPlayer(a.PlayerID()), answer: core.Red}
	pending.SetPlayerAt(0, other)
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.ChosenColor != core.Black || chooser.calls != 1 || g.StackSize() != 0 {
		t.Fatal("choice timing or value")
	}
	pending.ResolveStack()
	if pending.FindPermanent(card.ID()).ChosenColor != core.Red || p.ChosenColor != core.Black || other.calls != 1 {
		t.Fatal("pending clones share entry choices")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if chooser.calls != 1 || g.FindPermanent(p.ID()).ChosenColor != core.Black {
		t.Fatal("phase cycle repeated choice")
	}
	chooser.answer = core.Green
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.ChosenColor != core.Green || chooser.calls != 2 {
		t.Fatal("return kept previous choice")
	}
}

func TestRONColorEntryInvalid(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[ChooseAColor])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor()])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor())])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(Colorless)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(Red)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor,AnyColor)])`,
		`AsPermanentEnters(ThisPermanent(),[ChooseAColor(AnyColor)])`,
		`AsPermanentEnters(HostPermanent,[ChooseAColor(AnyColor)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(Opponent)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),EntersTapped])`,
		colorEntryRule + `,` + colorEntryRule,
		colorEntryRule + `,AsPermanentEnters(ThisPermanent,[ChooseABasicLandType])`,
		`SpellActions(ActionList([ChooseAColor(AnyColor)]))`,
		colorEntryRule + `,TriggerA(AtTheBeginningOfYourUpkeep,ActionList([GainLife(TheChosenColor)]))`,
	} {
		nodes, err := Parse(strings.NewReader(colorEntryFixture(rule)), "invalid-color-entry.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted invalid declaration: %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(strings.Replace(colorEntryFixture(colorEntryRule), "[Enchantment]", "[Instant]", 1)), "invalid-color-entry.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("nonpermanent entry accepted")
	}
}

func TestRONColorEntryInventoryAndRealRequirements(t *testing.T) {
	report, err := InspectFS(fstest.MapFS{"entry.ron": {Data: []byte(colorEntryFixture(colorEntryRule))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("entry fixture rejected", report.Issues)
	}
	found := false
	for _, expression := range report.Signatures {
		if expression.Name == "AnyColor" && expression.Role == "color_choice_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("entry color filter has wrong semantic role")
	}
	for _, tc := range []struct{ set, name string }{{"DRK", "Psychic Allergy"}, {"ARN", "Jihad"}} {
		node := repositoryRecord(t, tc.set, tc.name, tc.set+".ron")
		if _, err := Compile(node); err == nil {
			t.Fatalf("claimed complete unsupported record: %s", tc.name)
		}
		rules := node.Fields["Rules"].Args
		entry := rules[0]
		if tc.name == "Psychic Allergy" {
			factory, err := compilePermanentEntry(entry)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			chooser := &ronColorEntryPlayer{Player: a, answer: core.Blue}
			g.SetPlayerAt(0, chooser)
			p := g.PutOnBattlefield(mage.NewEnchantment("Entry requirement", "", factory()), a.PlayerID())
			if p.ChosenColor != core.Blue || chooser.calls != 1 {
				t.Fatal("real entry requirement failed")
			}
		} else if _, err := compilePermanentEntry(entry); err == nil {
			t.Fatal("partial combined entry accepted")
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewArtifact("Retained", "") })
	if err := LoadInto(registry, fstest.MapFS{"entry.ron": {Data: []byte(colorEntryFixture(colorEntryRule) + "\n" + colorEntryFixture(`AsPermanentEnters(ThisPermanent,[ChooseAColor(Colorless)])`))}}, "*.ron"); err == nil {
		t.Fatal("mixed collection accepted")
	}
	if _, err := registry.CreateCard("Retained"); err != nil {
		t.Fatal("rejected collection changed registry")
	}
}

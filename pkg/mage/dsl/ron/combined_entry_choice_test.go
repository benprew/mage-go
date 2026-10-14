package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const combinedEntryRule = `AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(Opponent)])`

func TestRONCombinedEntryChoiceGameplay(t *testing.T) {
	registry, err := LoadFS(fstest.MapFS{"entry.ron": {Data: []byte(colorEntryFixture(combinedEntryRule))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
	g.SetPlayerAt(0, chooser)
	card, err := registry.CreateCard("Color entry")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || chooser.calls != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("unpaid casting changed state")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 {
		t.Fatal("choice at announcement")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("permanent spell copy failed")
	}
	copiedChooser := &ronColorEntryPlayer{Player: b, answer: core.Green}
	g.SetPlayerAt(1, copiedChooser)
	pending := g.Clone()
	other := &ronColorEntryPlayer{Player: pending.GetPlayer(a.PlayerID()), answer: core.Black}
	pending.SetPlayerAt(0, other)
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.ChosenColor != core.Red || p.ChosenPlayer != b.PlayerID() || chooser.calls != 1 || g.StackSize() != 0 {
		t.Fatal("joint entry choices missing")
	}
	copiedCount := 0
	for _, member := range g.AllBattlefield() {
		if member.IsToken && member.Name() == "Color entry" {
			copiedCount++
			if member.ChosenColor != core.Green || member.ChosenPlayer != a.PlayerID() {
				t.Fatal("spell copy inherited entry choices")
			}
		}
	}
	if copiedCount != 1 || copiedChooser.calls != 1 {
		t.Fatal("spell copy did not make fresh entry choices")
	}
	pending.ResolveStack()
	if pending.FindPermanent(card.ID()).ChosenColor != core.Black || p.ChosenColor != core.Red || other.calls != 1 {
		t.Fatal("pending clones share choices")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).ChosenColor != core.Red || g.FindPermanent(p.ID()).ChosenPlayer != b.PlayerID() || chooser.calls != 1 {
		t.Fatal("control or phasing changed choices")
	}
	second, err := registry.CreateCard("Color entry")
	if err != nil {
		t.Fatal(err)
	}
	secondChooser := &ronColorEntryPlayer{Player: b, answer: core.Green}
	g.SetPlayerAt(1, secondChooser)
	second.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	if q.ChosenPlayer != a.PlayerID() || q.ChosenColor != core.Green || p.ChosenColor != core.Red {
		t.Fatal("factory instances share choices or use owner")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.ChosenColor != core.Green || returned.ChosenPlayer != a.PlayerID() || secondChooser.calls != 2 {
		t.Fatal("return kept old choices")
	}
}

func TestRONCombinedEntryChoiceInvalid(t *testing.T) {
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent),ChooseAColor(AnyColor)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(AnyPlayer)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(Colorless),ChooseAPlayer(Opponent)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(Opponent())])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAColor(AnyColor)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(Opponent),ChooseAPlayer(Opponent)])`,
		`AsPermanentEnters(HostPermanent,[ChooseAColor(AnyColor),ChooseAPlayer(Opponent)])`,
		combinedEntryRule + `,` + colorEntryRule,
		combinedEntryRule + `,AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])`,
		combinedEntryRule + `,AsPermanentEnters(ThisPermanent,[EntersTapped])`,
		`AsPermanentEnters(ThisPermanent,[MayActions([ChooseAColor(AnyColor),ChooseAPlayer(Opponent)])])`,
	} {
		nodes, err := Parse(strings.NewReader(colorEntryFixture(rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted invalid declaration: %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(strings.Replace(colorEntryFixture(combinedEntryRule), "[Enchantment]", "[Instant]", 1)), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("nonpermanent accepted")
	}
}

func TestRONCombinedEntryChoiceJihad(t *testing.T) {
	record := repositoryRecord(t, "ARN", "Jihad", "ARN.lines.ron")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported static condition player") {
		t.Fatalf("expected remaining Jihad static blocker: %v", err)
	}
	entry := record.Fields["Rules"].Args[0]
	if err := validateColorEntryDeclaration([]Node{entry}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := compileOpponentEntryDeclaration([]Node{entry}, true); err != nil {
		t.Fatal(err)
	}
	option, err := compilePermanentEntry(entry)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Blue}
	g.SetPlayerAt(0, chooser)
	if err := mage.ValidateCardDefinition("Entry requirement", mage.NewEnchantment("Entry requirement", "", option())); err != nil {
		t.Fatal(err)
	}
	factory := staticBaseSubrule(t, record, entry)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if p.ChosenColor != core.Blue || p.ChosenPlayer != b.PlayerID() || chooser.calls != 1 {
		t.Fatal("exact Jihad entry failed")
	}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewArtifact("Retained", "") })
	if err := LoadInto(registry, fstest.MapFS{"entry.ron": {Data: []byte(colorEntryFixture(combinedEntryRule) + "\n" + colorEntryFixture(combinedEntryRule+`,`+colorEntryRule))}}, "*.ron"); err == nil {
		t.Fatal("mixed collection accepted")
	}
	if _, err := registry.CreateCard("Retained"); err != nil {
		t.Fatal("failed collection changed registry")
	}
}

func TestRONCombinedEntryChoiceConsumers(t *testing.T) {
	rules := combinedEntryRule + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([TapEachPermanent(IsColor(TheChosenColor))]))`
	registry, err := LoadFS(fstest.MapFS{"entry.ron": {Data: []byte(colorEntryFixture(rules))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
	g.SetPlayerAt(0, chooser)
	card, err := registry.CreateCard("Color entry")
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 4), a.PlayerID())
	blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 4), b.PlayerID())
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("unchosen upkeep triggered")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 1 || red.Tapped {
		t.Fatal("chosen event timing")
	}
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	chooser.answer = core.Blue
	g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	if !red.Tapped || blue.Tapped {
		t.Fatal("joint choice consumers followed returned source")
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONChosenStaticJihadRequirement(t *testing.T) {
	record := repositoryRecord(t, "ARN", "Jihad", "ARN.lines.ron")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "unsupported no-land trigger player reference") {
		t.Fatalf("remaining state-event gap: %v", err)
	}
	rules := record.Fields["Rules"].Args
	record.Fields["Rules"] = Node{Kind: List, Args: rules[:2]}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
	g.SetPlayerAt(0, chooser)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 0 {
		t.Fatal("static query made entry choice at announcement")
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	own := g.PutOnBattlefield(mage.NewCreature("White A", "{W}", 1, 4), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("White B", "{W}", 1, 4), b.PlayerID())
	nonwhite := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 4), a.PlayerID())
	check := func(game *mage.Game, want int) {
		t.Helper()
		game.ApplyContinuousEffects()
		for _, p := range []*mage.Permanent{own, other} {
			q := game.FindPermanent(p.ID())
			if q.CurrentPower(game) != want || q.CurrentToughness(game) != 4+(want-1)/2 {
				t.Fatalf("white bonus: %d/%d want power %d", q.CurrentPower(game), q.CurrentToughness(game), want)
			}
		}
		if game.FindPermanent(nonwhite.ID()).CurrentPower(game) != 1 {
			t.Fatal("bonus affected nonwhite creature")
		}
	}
	check(g, 1)
	token := g.PutOnBattlefield(mage.NewCreature("Token", "{R}", 1, 4), b.PlayerID())
	token.IsToken = true
	check(g, 1)
	member := g.PutOnBattlefield(mage.NewArtifact("Red artifact", "{R}"), b.PlayerID())
	check(g, 3)
	clone := g.Clone()
	clone.MutablePermanent(source.ID()).ChosenColor = core.Green
	check(clone, 1)
	check(g, 3)
	if chooser.calls != 1 {
		t.Fatal("query repeated entry choice")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	check(g, 3)
	phased := g.PhaseOut(member.ID())
	check(g, 1)
	g.PhaseIn(phased)
	check(g, 3)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Blue)), member.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check(g, 1)
	g.RemoveFromBattlefield(g.FindPermanent(member.ID()))
	member = g.PutOnBattlefield(mage.NewArtifact("Late red", "{R}"), b.PlayerID())
	check(g, 3)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: member.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	check(g, 1)
	secondChooser := &ronColorEntryPlayer{Player: b, answer: core.Red}
	g.SetPlayerAt(1, secondChooser)
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	check(g, 3)
	phased = g.PhaseOut(second.ID())
	check(g, 1)
	g.PhaseIn(phased)
	check(g, 3)
	removal := mage.RemoveAllAbilities(second.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(nonwhite.ID())
	g.AddContinuousEffect(removal)
	check(g, 1)
	g.RemoveFromBattlefield(g.FindPermanent(nonwhite.ID()))
	nonwhite = g.PutOnBattlefield(mage.NewCreature("Blue replacement", "{U}", 1, 4), a.PlayerID())
	check(g, 3)
	g.BouncePermanentToHand(g.FindPermanent(second.ID()))
	check(g, 1)
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	a.RemoveFromHand(card.ID())
	chooser.answer = core.Blue
	returned := g.PutOnBattlefield(card, a.PlayerID())
	check(g, 1)
	if returned.ChosenColor != core.Blue || returned.ChosenPlayer != b.PlayerID() || chooser.calls != 2 {
		t.Fatal("return did not reset choices")
	}
	g.PutOnBattlefield(mage.NewArtifact("Blue member", "{U}"), b.PlayerID())
	check(g, 3)
}

func TestRONChosenStaticInvalid(t *testing.T) {
	body := `,[EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(2,1)])])`
	for _, rules := range []string{
		`If(PlayerPassesFilter(TheChosenPlayer,ControlsA(IsColor(Red)))` + body,
		combinedEntryRule + `,If(PlayerPassesFilter(TheChosenPlayer(),ControlsA(IsColor(TheChosenColor)))` + body,
		`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)]),If(PlayerPassesFilter(TheChosenPlayer,ControlsA(IsColor(TheChosenColor)))` + body,
		combinedEntryRule + `,If(PlayerPassesFilter(TheChosenPlayer,ControlsA(IsColor(TheChosenColor())))` + body,
		combinedEntryRule + `,If(PlayerPassesFilter(TheChosenPlayer,ControlsA(Or([IsNonToken,IsColor(TheChosenColor)])))` + body,
		combinedEntryRule + `,If(PlayerPassesFilter(ActionPlayer,ControlsA(IsColor(TheChosenColor)))` + body,
		combinedEntryRule + `,If(PlayerPassesFilter(TheChosenPlayer,ControlsA(And([])))` + body,
	} {
		nodes, err := Parse(strings.NewReader(colorEntryFixture(rules)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted invalid rule: %s", rules)
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewArtifact("Retained", "") })
	if err := LoadInto(registry, fstest.MapFS{"invalid.ron": {Data: []byte(colorEntryFixture(`If(PlayerPassesFilter(TheChosenPlayer,ControlsA(IsColor(Red)))` + body))}}, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Retained"); err != nil {
		t.Fatal("rejected load changed registry")
	}
}

func TestRONChosenStaticPlayerAndColorContexts(t *testing.T) {
	for _, tc := range []struct {
		entry, player, filter, selector string
		want                            int
	}{
		{combinedEntryRule, "TheChosenPlayer", "IsColor(Red)", "ControlsA", 3},
		{colorEntryRule, "You", "And([IsNonToken,And([IsColor(TheChosenColor),IsCardtype(Artifact)])])", "ControlsA", 3},
		{combinedEntryRule, "TheChosenPlayer", "IsColor(TheChosenColor)", "ControlsNo", 1},
		{combinedEntryRule, "TheChosenPlayer", "Or([IsColor(Red),IsColor(Green)])", "ControlsA", 3},
	} {
		t.Run(tc.player+tc.selector+tc.filter, func(t *testing.T) {
			rule := tc.entry + `,If(PlayerPassesFilter(` + tc.player + `,` + tc.selector + `(` + tc.filter + `)),[EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(2,1)])])`
			registry, err := LoadFS(fstest.MapFS{"choice.ron": {Data: []byte(colorEntryFixture(rule))}}, "*.ron")
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
			g.PutOnBattlefield(card, a.PlayerID())
			recipient := g.PutOnBattlefield(mage.NewCreature("Recipient", "{U}", 1, 4), a.PlayerID())
			controller := b.PlayerID()
			if tc.player == "You" {
				controller = a.PlayerID()
			}
			member := g.PutOnBattlefield(mage.NewArtifact("Member", "{R}"), controller)
			g.ApplyContinuousEffects()
			if recipient.CurrentPower(g) != tc.want || chooser.calls != 1 {
				t.Fatal("wrong live player or color context")
			}
			g.RemoveFromBattlefield(member)
			want := 1
			if tc.selector == "ControlsNo" {
				want = 3
			}
			if recipient.CurrentPower(g) != want {
				t.Fatal("condition did not update after departure")
			}
		})
	}
}

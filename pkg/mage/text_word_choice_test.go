package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type textWordTester struct {
	Player
	answers []string
	calls   int
	options [][]string
}

func (p *textWordTester) ChooseMode(options []string, _ string) int {
	p.options = append(p.options, slices.Clone(options))
	answer := p.answers[p.calls%len(p.answers)]
	p.calls++
	index := slices.Index(options, answer)
	options[0] = "changed"
	return index
}

func TestTextWordChoiceResults(t *testing.T) {
	a := &textWordTester{Player: NewBasePlayer("A"), answers: []string{"Red", "Blue", "Forest", "Island"}}
	g := NewGame(a, NewBasePlayer("B"))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	colors, lands := BindColorWordPair("colors"), BindBasicLandWordPair("lands")
	if _, ok := FirstColorWord(colors).Resolve(ctx); ok {
		t.Fatal("read before choice")
	}
	for _, effect := range []ResolvingEffect{ChooseColorWordPair(You, colors), ChooseBasicLandWordPair(You, lands)} {
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if first, ok := FirstColorWord(colors).Resolve(ctx); !ok || first != core.Red {
		t.Fatal("wrong first color")
	}
	if second, ok := SecondColorWord(colors).Resolve(ctx); !ok || second != core.Blue {
		t.Fatal("wrong second color")
	}
	if first, ok := FirstBasicLandWord(lands).Resolve(ctx); !ok || first != core.ForestLandType {
		t.Fatal("wrong first land")
	}
	if second, ok := SecondBasicLandWord(lands).Resolve(ctx); !ok || second != core.IslandLandType {
		t.Fatal("wrong second land")
	}
	if len(a.options[0]) != 5 || len(a.options[1]) != 4 || slices.Contains(a.options[1], "Red") {
		t.Fatal("illegal color options")
	}
	clone := g.Clone()
	clone.resolution.Clear()
	if _, ok := FirstColorWord(colors).Resolve(&EffectContext{Game: clone}); ok {
		t.Fatal("cleared clone retained result")
	}
	if first, _ := FirstColorWord(colors).Resolve(ctx); first != core.Red {
		t.Fatal("clone changed original")
	}
	if err := ChooseColorWordPair(ControllerOf(This), colors).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FirstColorWord(colors).Resolve(ctx); ok {
		t.Fatal("absent chooser retained result")
	}
	a.answers = []string{"invalid"}
	if err := ChooseColorWordPair(You, colors).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	first, _ := FirstColorWord(colors).Resolve(ctx)
	second, _ := SecondColorWord(colors).Resolve(ctx)
	if first != core.White || second != core.Blue {
		t.Fatal("invalid response did not use distinct legal defaults")
	}
}

func TestTextWordChoiceDeclarations(t *testing.T) {
	for _, effect := range []Effect{ChooseColorWordPair(nil, BindColorWordPair("x")), ChooseColorWordPair(You, nil), ChooseBasicLandWordPair(You, BindBasicLandWordPair(""))} {
		card := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	binding := BindColorWordPair("duplicate")
	card := NewInstant("Invalid", "", Spell(Sequence(ChooseColorWordPair(You, binding), ChooseColorWordPair(You, binding))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("duplicate producer accepted")
	}
}

type textWordCheck struct {
	first, second         ColorWordRef
	wantFirst, wantSecond core.Color
	t                     *testing.T
	calls                 *int
}

func (*textWordCheck) Text() string                 { return "check saved words" }
func (*textWordCheck) Properties() EffectProperties { return EffectProperties{} }
func (e *textWordCheck) Apply(ctx *EffectContext) error {
	first, ok := e.first.Resolve(ctx)
	second, otherOK := e.second.Resolve(ctx)
	if !ok || !otherOK || first != e.wantFirst || second != e.wantSecond {
		e.t.Fatalf("saved words = %v %v, present = %v %v", first, second, ok, otherOK)
	}
	if e.calls != nil {
		*e.calls++
	}
	return nil
}
func (e *textWordCheck) inspectDefinition() definitionNode {
	return definitionNode{Role: "word-test", children: []definitionEdge{{"first", e.first}, {"second", e.second}}}
}
func TestTextWordChoiceSavedFrames(t *testing.T) {
	a := &textWordTester{Player: NewBasePlayer("A"), answers: []string{"Red", "Blue"}}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	pair := BindColorWordPair("saved")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choose := ChooseColorWordPair(You, pair)
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	calls := 0
	check := &textWordCheck{FirstColorWord(pair), SecondColorWord(pair), core.Red, core.Blue, t, &calls}
	if err := AtNext(EachEndStep, check).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.answers = []string{"Green", "Black"}
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(g, choose, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if first, _ := FirstColorWord(pair).Resolve(ctx); first != core.Green {
		t.Fatal("nested frame changed outer result")
	}
	g.PutOnBattlefield(NewArtifact("member", ""), a.PlayerID())
	a.answers = []string{"White", "Red"}
	if err := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), choose).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if first, _ := FirstColorWord(pair).Resolve(ctx); first != core.Green {
		t.Fatal("loop changed outer result")
	}
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if calls != 2 {
		t.Fatal("saved delayed frames did not run separately")
	}
}
func TestTextWordChoiceReadDeclarations(t *testing.T) {
	pair := BindColorWordPair("words")
	check := &textWordCheck{first: FirstColorWord(pair), second: SecondColorWord(pair)}
	invalid := NewInstant("Invalid", "", Spell(check))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("read before producer accepted")
	}
	valid := NewInstant("Valid", "", Spell(Sequence(ChooseColorWordPair(You, pair), check)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid = NewInstant("Invalid", "", Spell(&textWordCheck{first: FirstColorWord(nil), second: SecondColorWord(nil)}))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("nil read accepted")
	}
}
func TestTextWordChoiceWaitingSource(t *testing.T) {
	a := &textWordTester{Player: NewBasePlayer("A"), answers: []string{"Red", "Blue"}}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("source", ""), a.PlayerID())
	original := g.CaptureDamageSource(source.ID())
	body := ChooseColorWordPair(ControllerOfWithLKI(This), BindColorWordPair("choice"))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	for range 2 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(original) || a.calls != 0 {
			t.Fatal("query chose words or lost saved source")
		}
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(source.ID())
	g.PutOnBattlefield(source.Card, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.calls != 2 {
		t.Fatal("delay followed returned source controller")
	}
}

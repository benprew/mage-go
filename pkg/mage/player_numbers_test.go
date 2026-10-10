package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type perPlayerDiscardChooser struct {
	*BasePlayer
	count int
	order *[]string
}

func (p *perPlayerDiscardChooser) ChooseNumber(minimum, maximum int, _ string) int {
	if p.order != nil {
		*p.order = append(*p.order, p.Name())
	}
	return max(minimum, min(maximum, p.count))
}
func TestPlayerDiscardResultsBatch(t *testing.T) {
	for _, replace := range []bool{false, true} {
		order := []string{}
		a := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("A"), count: 1, order: &order}
		b := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("B"), count: 2, order: &order}
		g := NewGame(a, b)
		for range 3 {
			a.AddToHand(NewInstant("A card", ""))
			b.AddToHand(NewInstant("B card", ""))
		}
		if replace {
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		}
		seen := 0
		watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			seen++
			if len(a.Hand()) != 2 || len(b.Hand()) != 1 {
				t.Error("partial discard batch")
			}
			return false
		})
		g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
		result := BindPlayerNumbers("discarded")
		body := Sequence(DiscardUpToEach(EachPlayer, Fixed(3), result), GainLifeFor(You, PlayerNumber(result, You)))
		if err := ValidateCardDefinition("Valid", NewSorcery("Valid", "", Spell(body))); err != nil {
			t.Fatal(err)
		}
		if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 21 || PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}) != 2 || seen != 3 || len(order) != 2 || order[0] != "A" || order[1] != "B" {
			t.Fatalf("results or order: life=%d events=%d order=%v", a.Life(), seen, order)
		}
	}
}
func TestPlayerDiscardResultsDeclaration(t *testing.T) {
	result := BindPlayerNumbers("discarded")
	for _, body := range []ResolvingEffect{
		DiscardUpToEach(nil, Fixed(3), result), DiscardUpToEach(EachPlayer, nil, result), DiscardUpToEach(EachPlayer, Fixed(3), nil),
		DiscardUpToEach(EachPlayer, Fixed(3), BindPlayerNumbers("")),
		GainLifeFor(You, PlayerNumber(result, You)),
		Sequence(GainLifeFor(You, PlayerNumber(result, You)), DiscardUpToEach(EachPlayer, Fixed(3), result)),
		Sequence(DiscardUpToEach(EachPlayer, Fixed(3), result), GainLifeFor(You, PlayerNumber(result, nil))),
	} {
		if err := ValidateCardDefinition("Invalid", NewSorcery("Invalid", "", Spell(body))); err == nil {
			t.Fatalf("accepted %T", body)
		}
	}
}

type playerDiscardReturningReplacement struct {
	replacementBase
	card uuid.UUID
}

func (r *playerDiscardReturningReplacement) Matches(a Action, _ GameReader) bool {
	d, ok := a.(*DiscardAction)
	return ok && d.CardID() == r.card
}
func (r *playerDiscardReturningReplacement) Replace(a Action, g *Game) Action {
	d := a.(*DiscardAction)
	p := g.GetPlayer(d.PlayerID())
	c, ok := p.RemoveFromHand(d.CardID())
	if ok {
		p.AddToHand(c)
	}
	return a
}
func (*playerDiscardReturningReplacement) IsActive(GameReader) bool   { return true }
func (r *playerDiscardReturningReplacement) Clone() ReplacementEffect { clone := *r; return &clone }

func TestPlayerDiscardResultsIdentity(t *testing.T) {
	a := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("A"), count: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewInstant("Card", "")
	a.AddToHand(card)
	g.effects.AddReplacement(&playerDiscardReturningReplacement{card: card.ID()})
	result := BindPlayerNumbers("discarded")
	body := Sequence(DiscardUpToEach(EachPlayer, Fixed(3), result), GainLifeFor(You, PlayerNumber(result, You)))
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Life() != 20 {
		t.Fatal("discard followed a returned hand visit")
	}
}
func TestPlayerDiscardResultsCloneClearAndCancel(t *testing.T) {
	a := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("A"), count: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewInstant("Card", ""))
	result := BindPlayerNumbers("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	effect := DiscardUpToEach(EachPlayer, Fixed(3), result)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if got := PlayerNumber(result, You).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != 1 {
		t.Fatalf("clone result %d", got)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if PlayerNumber(result, You).resolveValue(ctx) != 0 || PlayerNumber(result, You).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 1 {
		t.Fatal("clear changed clone")
	}
	g.resolution.Clear()
	if len(g.resolution.numberBindings) != 0 {
		t.Fatal("results survived clear")
	}
	cancelled := NewInstant("Cancelled", "")
	a.AddToHand(cancelled)
	g.effects.AddReplacement(&preventRandomDiscardReplacement{card: cancelled.ID()})
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || PlayerNumber(result, You).resolveValue(ctx) != 0 {
		t.Fatal("cancelled discard counted")
	}
}

func TestPlayerDiscardResultsDelayedAndLoopFrames(t *testing.T) {
	a := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("A"), count: 1}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewInstant("One", ""))
	a.AddToHand(NewInstant("Two", ""))
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	result := BindPlayerNumbers("discarded")
	body := Sequence(DiscardUpToEach(EachPlayer, Fixed(1), result), AtNext(YourUpkeep, GainLifeFor(You, PlayerNumber(result, You))))
	if err := ValidateCardDefinition("Valid", NewSorcery("Valid", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}); err != nil {
		t.Fatal(err)
	}
	member := BindPermanent("member")
	if err := ForEach(MatchingPermanents(IsArtifact), member, DiscardUpToEach(EachPlayer, Fixed(0), result)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: a.PlayerID()}) != 1 {
		t.Fatal("loop changed outer results")
	}
	g.resolution.Clear()
	clone := g.Clone()
	g.DestroyPermanent(source)
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("delayed body lost result or controller")
		}
	}
}

func TestPlayerDiscardResultsOrderEmptyAndDuplicateSets(t *testing.T) {
	order := []string{}
	a := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("A"), count: 1, order: &order}
	b := &perPlayerDiscardChooser{BasePlayer: NewBasePlayer("B"), count: 1, order: &order}
	g := NewGame(a, b)
	g.SetActivePlayerIndex(1)
	a.AddToHand(NewInstant("A card", ""))
	b.AddToHand(NewInstant("B card", ""))
	result := BindPlayerNumbers("discarded")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DiscardUpToEach(playerDamageTestSet{[]uuid.UUID{a.PlayerID(), b.PlayerID(), a.PlayerID(), uuid.New()}}, Fixed(3), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "B" || order[1] != "A" || PlayerNumber(result, You).resolveValue(ctx) != 1 {
		t.Fatal("not active-player order or duplicate choices")
	}
	if err := DiscardUpToEach(playerDamageTestSet{}, Fixed(3), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if PlayerNumber(result, You).resolveValue(ctx) != 0 {
		t.Fatal("empty set kept result")
	}
}

func TestPlayerDiscardResultsWaitingReads(t *testing.T) {
	for _, change := range []string{"live", "phased", "departed", "returned"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			result := BindPlayerNumbers("discarded")
			player := ControllerOfWithLKI(This)
			body := Sequence(DiscardUpToEach(OpponentsOf(player), Fixed(0), result), GainLifeFor(You, PlayerNumber(result, player)))
			if err := AtNext(YourUpkeep, body).Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "phased":
				g.PhaseOut(source.ID())
			case "departed", "returned":
				g.DestroyPermanent(source)
				if change == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			first := g.WaitingDelayedPermanentSources()
			second := g.WaitingDelayedPermanentSources()
			if len(first) != 1 || len(second) != 1 || !first[0].SameObject(second[0]) || first[0].Incarnation() != source.incarnationID {
				t.Fatal("waiting query lost exact source")
			}
			if len(g.resolution.numberBindings) != 0 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("waiting query executed body")
			}
		})
	}
}

package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSetChoiceRepeatedAPNAPAndOverlap(t *testing.T) {
	for _, active := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		order := []string{}
		pa, pb := &playerTapChooser{Player: a, count: 2, order: &order}, &playerTapChooser{Player: b, count: 1, order: &order}
		g.SetPlayerAt(0, pa)
		g.SetPlayerAt(1, pb)
		for _, player := range []Player{a, b} {
			g.PutOnBattlefield(NewCreature("Creature", "", 1, 1), player.PlayerID())
		}
		for range 4 {
			g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
		}
		counts, result := BindPlayerNumbers("counts"), BindPermanentSet("chosen")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		body := Sequence(TapAnyNumberEach(EachPlayer, MatchingPermanents(IsCreature), counts), ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), Fixed(2), counts, result))
		if err := ValidateCardDefinition("Choice", NewSorcery("Choice", "", Spell(body))); err != nil {
			t.Fatal(err)
		}
		if err := body.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if len(FromPermanentSet(result).resolvePermanents(ctx)) != 2 {
			t.Fatal("overlap was counted twice or opponent lands excluded")
		}
		if len(order) != 4 || order[2] != g.ActivePlayerObj().Name() {
			t.Fatal("separate APNAP operations", order)
		}
		pa.check = func(reason string) {
			if active == 1 && !strings.Contains(reason, "Previous choices:") {
				t.Fatal("prior choices absent")
			}
		}
		order = nil
		g.resolution.numberBindings[counts.members[0]] = 2
		g.resolution.numberBindings[counts.members[1]] = 0
		if err := ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), Fixed(2), counts, result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if len(order) != 2 || len(FromPermanentSet(result).resolvePermanents(ctx)) != 2 {
			t.Fatal("did not repeat distinct up-to choices", order)
		}
	}
}

func TestPermanentSetChoiceVisitsResetCloneAndDelay(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewLand("Chosen"), a.PlayerID())
	other := g.PutOnBattlefield(NewLand("Other"), b.PlayerID())
	chooser := &playerTapChooser{Player: a, count: 1}
	g.SetPlayerAt(0, chooser)
	counts, result := BindPlayerNumbers("counts"), BindPermanentSet("chosen")
	g.resolution.numberBindings = map[*NumberBinding]int{counts.members[0]: 1}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	choose := ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), Fixed(2), counts, result)
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := WithoutPermanents(MatchingPermanents(IsLand), FromPermanentSet(result)).resolvePermanents(ctx); len(got) != 1 || got[0].ID() != other.ID() {
		t.Fatal("wrong exclusion", got)
	}
	clone := g.Clone()
	g.PhaseOut(p.ID())
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
		t.Fatal("phased choice remained actionable")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 1 {
		t.Fatal("phase cycle lost saved visit")
	}
	if err := AtNext(EachEndStep, DestroyEach(WithoutPermanents(MatchingPermanents(IsLand), FromPermanentSet(result)))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	chooser.count = 0
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPermanentSet(result).resolvePermanents(ctx)) != 0 {
		t.Fatal("empty selection retained old result")
	}
	if len(FromPermanentSet(result).resolvePermanents(&EffectContext{Game: clone})) != 1 {
		t.Fatal("clone shared result")
	}
	g.PhaseOut(p.ID())
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.ResolveStack()
	if g.FindPermanent(p.ID()) == nil || g.FindPermanent(other.ID()) != nil {
		t.Fatal("delay lost saved result")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(p.Card, a.PlayerID())
	cloneCard := clone.FindPermanent(p.ID()).Card
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	clone.PutOnBattlefield(cloneCard, a.PlayerID())
	if len(FromPermanentSet(result).resolvePermanents(&EffectContext{Game: clone})) != 0 || g.FindPermanent(returned.ID()) == nil {
		t.Fatal("return inherited old visit")
	}
}

func TestPermanentSetChoiceInvalidDefinitions(t *testing.T) {
	counts, result := BindPlayerNumbers("counts"), BindPermanentSet("chosen")
	for _, body := range []ResolvingEffect{
		ChooseUpToPermanentsEach(nil, MatchingPermanents(IsLand), Fixed(2), counts, result),
		ChooseUpToPermanentsEach(EachPlayer, nil, Fixed(2), counts, result),
		ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), nil, counts, result),
		ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), Fixed(2), nil, result),
		ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), Fixed(2), counts, nil),
		DestroyEach(WithoutPermanents(MatchingPermanents(IsLand), FromPermanentSet(result))),
	} {
		if err := ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(body))); err == nil {
			t.Fatal("invalid or unbound declaration accepted")
		}
	}
}

type permanentSetChooser struct {
	Player
	counts []int
	next   int
	seen   map[uuid.UUID]bool
}

func (p *permanentSetChooser) ChooseNumber(_, _ int, _ string) int {
	value := p.counts[p.next%len(p.counts)]
	p.next++
	return value
}
func (p *permanentSetChooser) ChoosePermanent(members []*Permanent, _ string, _ GameReader) *Permanent {
	for _, m := range members {
		if !p.seen[m.ID()] {
			p.seen[m.ID()] = true
			return m
		}
	}
	return members[0]
}

func TestPermanentSetChoiceBoundsDistinctVisitsAndWaiting(t *testing.T) {
	for _, requested := range []int{-2, 0, 1, 2, 20} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		chooser := &permanentSetChooser{Player: a, counts: []int{requested}, seen: make(map[uuid.UUID]bool)}
		g.SetPlayerAt(0, chooser)
		members := make([]*Permanent, 0, 9)
		for range 4 {
			members = append(members, g.PutOnBattlefield(NewLand("Land"), b.PlayerID()))
		}
		phased := g.PutOnBattlefield(NewLand("Phased"), b.PlayerID())
		g.PhaseOut(phased.ID())
		old := g.PutOnBattlefield(NewLand("Old"), a.PlayerID())
		g.RemoveFromBattlefield(old)
		returned := g.PutOnBattlefield(old.Card, a.PlayerID())
		members = append(members, members[0], phased, old, nil)
		counts, result := BindPlayerNumbers("counts"), BindPermanentSet("kept")
		g.resolution.numberBindings = map[*NumberBinding]int{counts.members[0]: 2}
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		if err := ChooseUpToPermanentsEach(playerDamageTestSet{[]uuid.UUID{a.PlayerID(), a.PlayerID(), uuid.New()}}, fixedTapSet{members}, Fixed(2), counts, result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		selected := FromPermanentSet(result).resolvePermanents(ctx)
		if len(selected) != 2*min(2, max(0, requested)) || chooser.next != 2 {
			t.Fatal("bounds, repeated choices or duplicates", requested, len(selected), chooser.next)
		}
		for _, p := range selected {
			if p.ID() == phased.ID() || p.ID() == returned.ID() {
				t.Fatal("invalid visit selected")
			}
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	counts, result := BindPlayerNumbers("counts"), BindPermanentSet("kept")
	reads := 0
	filter := NewPermanentFilter("query must be pure", func(*Permanent, *Game) bool { reads++; return true })
	body := ChooseUpToPermanentsEach(EachPlayer, FilterPermanentSet(SingletonPermanent(This), filter), Fixed(0), counts, result)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	before := reads
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 {
			t.Fatal("source declaration lost")
		}
	}
	if reads != before || len(g.resolution.createdTokenBindings) != 0 {
		t.Fatal("query evaluated collection or chose")
	}
	g.PhaseOut(source.ID())
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("phasing lost exact source")
	}
}

func TestPermanentSetChoiceMalformedSubsetFallback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	}
	invalid := g.PutOnBattlefield(NewArtifact("Invalid", ""), a.PlayerID())
	chooser := &cardinalityInvalidChooser{Player: a, invalid: invalid}
	g.SetPlayerAt(0, chooser)
	counts, result := BindPlayerNumbers("counts"), BindPermanentSet("result")
	g.resolution.numberBindings = map[*NumberBinding]int{counts.members[0]: 1}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ChooseUpToPermanentsEach(EachPlayer, MatchingPermanents(IsLand), Fixed(2), counts, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := FromPermanentSet(result).resolvePermanents(ctx); len(got) != 2 || got[0].ID() == got[1].ID() || chooser.calls != 2 {
		t.Fatal("malformed answers lacked distinct legal fallbacks", got)
	}
}

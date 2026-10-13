package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type playerTapChooser struct {
	Player
	count   int
	order   *[]string
	check   func(string)
	invalid *Permanent
}

func (p *playerTapChooser) ChooseNumber(_, maximum int, reason string) int {
	if p.check != nil {
		p.check(reason)
	}
	if p.order != nil {
		*p.order = append(*p.order, p.Name())
	}
	return p.count
}
func (p *playerTapChooser) ChoosePermanent(candidates []*Permanent, reason string, _ GameReader) *Permanent {
	if p.check != nil {
		p.check(reason)
	}
	if p.invalid != nil {
		return p.invalid
	}
	return candidates[0]
}
func TestPlayerTapResultsChoicesAndBatch(t *testing.T) {
	for _, active := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		order := []string{}
		pa, pb := &playerTapChooser{Player: a, count: 1, order: &order}, &playerTapChooser{Player: b, count: 2, order: &order}
		g.SetPlayerAt(0, pa)
		g.SetPlayerAt(1, pb)
		members := make([]*Permanent, 0, 6)
		for _, player := range []Player{a, b} {
			for range 3 {
				members = append(members, g.PutOnBattlefield(NewCreature("Member", "", 1, 1), player.PlayerID()))
			}
		}
		check := func(reason string) {
			for _, p := range members {
				if g.FindPermanent(p.ID()).Tapped {
					t.Fatal("choice saw a partial batch")
				}
			}
			if len(order) > 0 && g.ActivePlayerObj().Name() != reason[:1] && !strings.Contains(reason, "Previous choices:") {
				t.Fatal("prior choices not visible", reason)
			}
		}
		pa.check = check
		pb.check = check
		events := 0
		result := BindPlayerNumbers("tapped")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtTapped, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			events++
			count := 0
			for _, p := range members {
				if g.FindPermanent(p.ID()).Tapped {
					count++
				}
			}
			if count != 3 || PlayerNumber(result, You).resolveValue(ctx) != 1 {
				t.Fatal("event saw incomplete transitions or results")
			}
			return false
		}))), a.PlayerID())
		body := Sequence(TapAnyNumberEach(EachPlayer, MatchingPermanents(IsCreature), result), GainLifeFor(You, PlayerNumber(result, You)))
		if err := ValidateCardDefinition("Tap", NewSorcery("Tap", "", Spell(body))); err != nil {
			t.Fatal(err)
		}
		if err := body.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if events != 3 || a.Life() != 21 || len(order) != 2 || order[0] != g.ActivePlayerObj().Name() {
			t.Fatal(events, a.Life(), order)
		}
	}
}
func TestPlayerTapResultsVisitsAndActualTransitions(t *testing.T) {
	for _, change := range []string{"tap", "phase", "return", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Member", "", 1, 1), a.PlayerID())
			pa := &playerTapChooser{Player: a, count: 1}
			g.SetPlayerAt(0, pa)
			pb := &playerTapChooser{Player: b, count: 0}
			g.SetPlayerAt(1, pb)
			fired := false
			pb.check = func(string) {
				if fired {
					return
				}
				fired = true
				switch change {
				case "tap":
					g.TapPermanent(p)
				case "phase":
					g.PhaseOut(p.ID())
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, a.PlayerID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
			}
			g.PutOnBattlefield(NewCreature("B member", "", 1, 1), b.PlayerID())
			result := BindPlayerNumbers("taps")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := TapAnyNumberEach(EachPlayer, MatchingPermanents(IsCreature), result).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if PlayerNumber(result, You).resolveValue(ctx) != 0 {
				t.Fatal("uncompleted tap counted")
			}
			if change == "return" && g.FindPermanent(p.ID()).Tapped {
				t.Fatal("returned visit tapped")
			}
		})
	}
}
func TestPlayerTapResultsBoundsResetAndDeclarations(t *testing.T) {
	for _, count := range []int{-1, 0, 1, 10} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		one := g.PutOnBattlefield(NewCreature("One", "", 1, 1), a.PlayerID())
		two := g.PutOnBattlefield(NewCreature("Two", "", 1, 1), a.PlayerID())
		tapped := g.PutOnBattlefield(NewCreature("Tapped", "", 1, 1), a.PlayerID())
		g.TapPermanent(tapped)
		phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
		g.PhaseOut(phased.ID())
		old := g.PutOnBattlefield(NewCreature("Old", "", 1, 1), a.PlayerID())
		g.RemoveFromBattlefield(old)
		returned := g.PutOnBattlefield(old.Card, a.PlayerID())
		chooser := &playerTapChooser{Player: a, count: count, invalid: tapped}
		g.SetPlayerAt(0, chooser)
		subjects := fixedTapSet{[]*Permanent{one, two, one, tapped, phased, old, nil}}
		result := BindPlayerNumbers("taps")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		effect := TapAnyNumberEach(playerDamageTestSet{[]uuid.UUID{a.PlayerID(), a.PlayerID(), uuid.New()}}, subjects, result)
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if got := PlayerNumber(result, You).resolveValue(ctx); got != min(2, max(0, count)) {
			t.Fatal(count, got)
		}
		if returned.Tapped {
			t.Fatal("old visit followed return")
		}
		chooser.count = 0
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if PlayerNumber(result, You).resolveValue(ctx) != 0 {
			t.Fatal("zero result did not clear old count")
		}
	}
	result := BindPlayerNumbers("taps")
	for _, body := range []ResolvingEffect{TapAnyNumberEach(nil, MatchingPermanents(IsCreature), result), TapAnyNumberEach(EachPlayer, nil, result), TapAnyNumberEach(EachPlayer, MatchingPermanents(IsCreature), nil), GainLifeFor(You, PlayerNumber(result, You))} {
		if err := ValidateCardDefinition("Invalid", NewSorcery("Invalid", "", Spell(body))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestPlayerTapResultsSnapshotAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	pa := &playerTapChooser{Player: a, count: 1}
	pb := &playerTapChooser{Player: b, count: 10}
	g.SetPlayerAt(0, pa)
	g.SetPlayerAt(1, pb)
	var later *Permanent
	pa.check = func(string) {
		if later == nil {
			later = g.PutOnBattlefield(NewCreature("Later", "", 1, 1), b.PlayerID())
		}
	}
	result := BindPlayerNumbers("taps")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := TapAnyNumberEach(EachPlayer, MatchingPermanents(IsCreature), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || later.Tapped || PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}) != 1 {
		t.Fatal("candidate sets were not saved before choices")
	}
	g.MutablePermanent(first.ID()).Tapped = false
	g.RemoveFromBattlefield(second)
	if PlayerNumber(result, You).resolveValue(ctx) != 1 || PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}) != 1 {
		t.Fatal("historical result followed later state")
	}
	sourceCard := NewArtifact("Source", "")
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("waiting query", func(*Permanent, *Game) bool { reads++; return true })
	body := TapAnyNumberEach(EachPlayer, FilterPermanentSet(SingletonPermanent(This), filter), BindPlayerNumbers("delayed taps"))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(source.ID())
	g.PhaseOut(source.ID())
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || reads != 0 {
			t.Fatal("waiting query selected subjects or lost source")
		}
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(sourceCard, b.PlayerID())
	clone := g.Clone()
	if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) || reads != 0 {
		t.Fatal("return replaced waiting identity")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.ResolveStack()
	if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("cloned lifetime shared")
	}
}

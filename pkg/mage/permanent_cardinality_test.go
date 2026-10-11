package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentCardinalityChoicesAndBatch(t *testing.T) {
	for _, active := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		var order []string
		pa := &apnapChoicePlayer{Player: a, order: &order}
		pb := &apnapChoicePlayer{Player: b, order: &order}
		g.SetPlayerAt(0, pa)
		g.SetPlayerAt(1, pb)
		all := make([]*Permanent, 0, 6)
		for _, player := range []Player{pa, pb} {
			for range 3 {
				all = append(all, g.PutOnBattlefield(NewCreature("Member", "", 1, 1, WithKeyword(core.Indestructible)), player.PlayerID()))
			}
		}
		for _, player := range []*apnapChoicePlayer{pa, pb} {
			player.check = func(reason string) {
				for _, p := range all {
					if g.FindPermanent(p.ID()) == nil {
						t.Fatal("choice observed partial sacrifice")
					}
				}
				if len(order) == 2 && !strings.Contains(reason, "Previous choices:") {
					t.Fatal("prior kept choices are not visible", reason)
				}
			}
		}
		events := 0
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtSacrifice, false).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
			events++
			for _, index := range []int{0, 1, 3, 4} {
				if g.FindPermanent(all[index].ID()) != nil || reader.FindPermanent(all[index].ID()) == nil {
					t.Fatal("sacrifice event lost simultaneous movement or LKI")
				}
			}
			return false
		}))), a.PlayerID())
		if err := ApplyEffect(g, SacrificeAllButEach(EachPlayer, MatchingPermanents(IsCreature), Fixed(1)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(order) != 2 || order[0] != g.ActivePlayerObj().Name() || events != 4 {
			t.Fatal("choice order or sacrifice count", order, events)
		}
		for i, p := range all {
			if (g.FindPermanent(p.ID()) != nil) != (i == 2 || i == 5) {
				t.Fatal("chose sacrifices instead of kept subjects", i)
			}
		}
	}
}

func TestPermanentCardinalityLeastAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		g.PutOnBattlefield(NewLand("A land"), a.PlayerID())
	}
	one := g.PutOnBattlefield(NewLand("B land"), b.PlayerID())
	phased := g.PutOnBattlefield(NewLand("Phased"), b.PlayerID())
	g.PhaseOut(phased.ID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	value := LeastControlledPermanentCount(EachPlayer, MatchingPermanents(IsLand))
	if value.resolveValue(ctx) != 1 {
		t.Fatal("least count included phased members")
	}
	if err := SacrificeAllButEach(EachPlayer, MatchingPermanents(IsLand), value).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 2 || g.FindPermanent(one.ID()) == nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
		t.Fatal("least count did not retain one land each")
	}
	for _, body := range []ResolvingEffect{
		SacrificeAllButEach(nil, MatchingPermanents(IsLand), Fixed(1)),
		SacrificeAllButEach(EachPlayer, nil, Fixed(1)),
		SacrificeAllButEach(EachPlayer, MatchingPermanents(IsLand), nil),
		SacrificeAllButEach(EachPlayer, MatchingPermanents(IsLand), LeastControlledPermanentCount(nil, MatchingPermanents(IsLand))),
		SacrificeAllButEach(EachPlayer, MatchingPermanents(IsLand), LeastControlledPermanentCount(EachPlayer, nil)),
	} {
		c := NewSorcery("Invalid", "", Spell(body))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestPermanentCardinalityBoundsIdentityAndFallback(t *testing.T) {
	for _, count := range []int{-1, 0, 1, 2, 3, 10} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		var order []string
		chooser := &apnapChoicePlayer{Player: a, order: &order}
		g.SetPlayerAt(0, chooser)
		members := make([]*Permanent, 0, 7)
		for range 3 {
			members = append(members, g.PutOnBattlefield(NewArtifact("Member", ""), a.PlayerID()))
		}
		old := g.PutOnBattlefield(NewArtifact("Old", ""), a.PlayerID())
		g.RemoveFromBattlefield(old)
		returned := g.PutOnBattlefield(old.Card, a.PlayerID())
		phased := g.PutOnBattlefield(NewArtifact("Phased", ""), a.PlayerID())
		g.PhaseOut(phased.ID())
		members = append(members, members[0], old, phased, nil)
		players := playerDamageTestSet{[]uuid.UUID{a.PlayerID(), a.PlayerID(), uuid.New()}}
		set := fixedTapSet{members}
		if got := LeastControlledPermanentCount(players, set).resolveValue(&EffectContext{Game: g}); got != 3 {
			t.Fatal("duplicate, absent, phased or returned member counted", got)
		}
		if err := ApplyEffect(g, SacrificeAllButEach(players, set, Fixed(count)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Graveyard()) != 3-min(max(count, 0), 3) || g.FindPermanent(returned.ID()) == nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
			t.Fatal("bounds or exact visits", count, len(a.Graveyard()))
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		g.PutOnBattlefield(NewArtifact("Member", ""), a.PlayerID())
	}
	invalid := g.PutOnBattlefield(NewLand("Not a candidate"), b.PlayerID())
	chooser := &cardinalityInvalidChooser{Player: a, invalid: invalid}
	g.SetPlayerAt(0, chooser)
	if err := ApplyEffect(g, SacrificeAllButEach(EachPlayer, MatchingPermanents(IsArtifact), Fixed(2)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 2 || len(a.Graveyard()) != 1 || g.FindPermanent(invalid.ID()) == nil {
		t.Fatal("illegal choices did not receive distinct legal fallback")
	}
}

type cardinalityInvalidChooser struct {
	Player
	invalid *Permanent
	calls   int
}

func (p *cardinalityInvalidChooser) ChoosePermanent(_ []*Permanent, _ string, _ GameReader) *Permanent {
	p.calls++
	return p.invalid
}

func TestPermanentCardinalityReplacementsAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	var order []uuid.UUID
	g.effects.AddReplacement(&collectionSacrificeReplacement{blocked: first.ID(), inspect: func(action Action) {
		if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil {
			t.Fatal("replacement saw partial moves")
		}
		order = append(order, g.FindPermanent(action.(*SacrificePermanentAction).PermanentID()).ControllerID())
	}})
	if err := ApplyEffect(g, SacrificeAllButEach(EachPlayer, MatchingPermanents(IsCreature), Fixed(0)), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != a.PlayerID() || g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("replacement order or cancellation")
	}
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	filter := NewPermanentFilter("must not select during queries", func(*Permanent, *Game) bool { reads++; return true })
	body := SacrificeAllButEach(EachPlayer, MatchingPermanents(IsCreature), LeastControlledPermanentCount(EachPlayer, FilterPermanentSet(SingletonPermanent(This), filter)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(source.ID())
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	for range 3 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || reads != 0 {
			t.Fatal("waiting query lost numeric source or made selections")
		}
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting clone lifetimes shared")
	}
}

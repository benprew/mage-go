package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileControllerCountsBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	stolen := NewCreature("Stolen", "{W}", 1, 1)
	stolen.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(stolen, b.PlayerID())
	token := g.PutOnBattlefield(NewCreature("Token", "{W}", 1, 1), b.PlayerID())
	token.IsToken = true
	phased := g.PutOnBattlefield(NewCreature("Phased", "{W}", 1, 1), a.PlayerID())
	g.PhaseOut(phased.ID())
	g.AddRegenerationShield(first.ID())
	first.GrantBaseAttr(core.Indestructible)
	observations := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if evt.FromZone == core.ZoneBattlefield && evt.ToZone == core.ZoneExile {
			observations++
			if g.FindPermanent(first.ID()) != nil || g.FindPermanent(token.ID()) != nil {
				t.Error("partial exile batch")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	result := BindPlayerNumbers("exiled")
	set := UnionPermanents(MatchingPermanents(IsCreature), MatchingPermanents(IsCreature))
	body := ExileEachWithControllerCounts(set, result)
	if err := ValidateCardDefinition("Valid", NewSorcery("Valid", "", Spell(Sequence(body, GainLifeFor(You, PlayerNumber(result, You)))))); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}); got != 2 {
		t.Fatalf("former controller count %d", got)
	}
	if PlayerNumber(result, You).resolveValue(ctx) != 0 || observations != 2 || len(g.GetExile()) != 1 || g.FindPermanentIncludingPhased(phased.ID()) == nil {
		t.Fatal("token, owner, phasing or events")
	}
	clone := g.Clone()
	if err := body.Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}) != 2 || PlayerNumber(result, You).resolveValue(&EffectContext{Game: clone, Controller: b.PlayerID()}) != 0 {
		t.Fatal("empty result or clone isolation")
	}
}

type exileCountReplacement struct {
	replacementBase
	cancel uuid.UUID
	calls  int
	t      *testing.T
	all    []uuid.UUID
}

func (*exileCountReplacement) IsActive(GameReader) bool { return true }
func (r *exileCountReplacement) Matches(action Action, _ GameReader) bool {
	a, ok := action.(*permanentGraveyardAction)
	return ok && a.destination == core.ZoneExile
}
func (r *exileCountReplacement) Replace(action Action, g *Game) Action {
	r.calls++
	for _, id := range r.all {
		if g.FindPermanent(id) == nil {
			r.t.Fatal("replacement after a departure")
		}
	}
	a := action.(*permanentGraveyardAction)
	if a.subject == r.cancel {
		return nil
	}
	c := *a
	c.destination = core.ZoneGraveyard
	return &c
}
func (r *exileCountReplacement) Clone() ReplacementEffect { c := *r; return &c }
func TestExileControllerCountsReplacements(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("Cancel", "", 1, 1), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Redirect", "", 1, 1), b.PlayerID())
	second.Card.SetOwner(b.PlayerID())
	r := &exileCountReplacement{cancel: first.ID(), t: t, all: []uuid.UUID{first.ID(), second.ID()}}
	g.AddReplacementEffect(r)
	result := BindPlayerNumbers("exiled")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ExileEachWithControllerCounts(MatchingPermanents(IsCreature), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if r.calls != 2 || PlayerNumber(result, You).resolveValue(&EffectContext{Game: g, Controller: b.PlayerID()}) != 0 || g.FindPermanent(first.ID()) == nil || len(b.Graveyard()) != 1 || len(g.GetExile()) != 0 {
		t.Fatal("count used intended moves")
	}
}
func TestExileControllerCountsDeclarations(t *testing.T) {
	result := BindPlayerNumbers("exiled")
	for _, body := range []ResolvingEffect{ExileEachWithControllerCounts(nil, result), ExileEachWithControllerCounts(MatchingPermanents(IsCreature), nil), ExileEachWithControllerCounts(MatchingPermanents(IsCreature), BindPlayerNumbers("")), Sequence(GainLifeFor(You, PlayerNumber(result, You)), ExileEachWithControllerCounts(MatchingPermanents(IsCreature), result))} {
		if ValidateCardDefinition("Invalid", NewSorcery("Invalid", "", Spell(body))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

type exileOrderProbe struct {
	replacementBase
	order *[]uuid.UUID
}

func (*exileOrderProbe) IsActive(GameReader) bool { return true }
func (r *exileOrderProbe) Matches(a Action, _ GameReader) bool {
	v, ok := a.(*permanentGraveyardAction)
	return ok && v.destination == core.ZoneExile
}
func (r *exileOrderProbe) Replace(a Action, _ *Game) Action {
	*r.order = append(*r.order, a.(*permanentGraveyardAction).subject)
	return a
}
func (r *exileOrderProbe) Clone() ReplacementEffect { c := *r; return &c }
func TestExileControllerCountsOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	var order []uuid.UUID
	g.AddReplacementEffect(&exileOrderProbe{order: &order})
	g.ExilePermanents([]*Permanent{second, first})
	if len(order) != 2 || order[0] != first.ID() || order[1] != second.ID() {
		t.Fatal("exile replacement choices did not use turn order")
	}
}

func TestExileControllerCountsIdentityAndHistory(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Return", "", 1, 1)
	card.SetOwner(b.PlayerID())
	original := g.PutOnBattlefield(card, b.PlayerID())
	saved := BindPermanent("saved")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := Snapshot(This, saved).Apply(&EffectContext{Game: g, SourceID: original.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(original)
	b.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	result := BindPlayerNumbers("exiled")
	if err := ExileEachWithControllerCounts(SingletonPermanent(From(saved)), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(returned.ID()) == nil || g.resolution.numberBindings[result.member(g, b.PlayerID())] != 0 {
		t.Fatal("returned visit supplied a saved subject")
	}
	if err := ExileEachWithControllerCounts(MatchingPermanents(IsCreature), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromExile(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	g.GetEffects().Apply(g)
	if g.resolution.numberBindings[result.member(g, b.PlayerID())] != 1 || g.resolution.numberBindings[result.member(g, a.PlayerID())] != 0 {
		t.Fatal("later zone or control change altered historical counts")
	}
}
func TestExileControllerCountsDrawReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	for range 3 {
		a.AddToLibrary(NewInstant("A draw", ""))
		b.AddToLibrary(NewInstant("B draw", ""))
	}
	g.PutOnBattlefield(NewCreature("A victim", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("B victim", "", 1, 1), b.PlayerID())
	drawing := BindPlayer("drawing")
	g.PutOnBattlefield(NewEnchantment("Replace", "", WithAbility(Static(ReplaceDraw(ExceptFirstDrawStepDraw, drawing, GainLifeFor(FromPlayer(drawing), Fixed(2)))))), a.PlayerID())
	result := BindPlayerNumbers("exiled")
	body := Sequence(ExileEachWithControllerCounts(MatchingPermanents(IsCreature), result), DrawEachPlayerNumber(result))
	if err := ValidateCardDefinition("Valid", NewSorcery("Valid", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(b.Hand()) != 0 || a.Life() != 22 || b.Life() != 22 {
		t.Fatal("draw bypassed replacement or lost count frame")
	}
	for _, invalid := range []ResolvingEffect{DrawEachPlayerNumber(nil), DrawEachPlayerNumber(BindPlayerNumbers("")), DrawEachPlayerNumber(BindPlayerNumbers("unbound"))} {
		if ValidateCardDefinition("Invalid", NewSorcery("Invalid", "", Spell(invalid))) == nil {
			t.Fatal("invalid draw count accepted")
		}
	}
}

type exileCompetingReplacement struct {
	replacementBase
	cancel bool
}

func (*exileCompetingReplacement) IsActive(GameReader) bool { return true }
func (*exileCompetingReplacement) Matches(a Action, _ GameReader) bool {
	v, ok := a.(*permanentGraveyardAction)
	return ok && v.destination == core.ZoneExile
}
func (r *exileCompetingReplacement) Replace(a Action, _ *Game) Action {
	if r.cancel {
		return nil
	}
	c := *a.(*permanentGraveyardAction)
	c.destination = core.ZoneGraveyard
	return &c
}
func (r *exileCompetingReplacement) Clone() ReplacementEffect { c := *r; return &c }
func TestExileControllerCountsCompeting(t *testing.T) {
	for _, index := range []int{0, 1} {
		a := NewBasePlayer("A")
		b := &deathExileOrderPlayer{BasePlayer: NewBasePlayer("B"), index: index}
		g := NewGame(a, b)
		card := NewCreature("Subject", "", 1, 1)
		card.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(card, b.PlayerID())
		g.AddReplacementEffect(&exileCompetingReplacement{cancel: true})
		g.AddReplacementEffect(&exileCompetingReplacement{})
		result := BindPlayerNumbers("exiled")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		if err := ExileEachWithControllerCounts(MatchingPermanents(IsCreature), result).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if b.choices != 1 || len(g.GetExile()) != 0 || (g.FindPermanent(p.ID()) != nil) != (index == 0) || len(b.Graveyard()) != index || g.resolution.numberBindings[result.member(g, b.PlayerID())] != 0 {
			t.Fatal("replacement controller, recheck or actual count")
		}
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func lifeDrawLibrary(p *BasePlayer, count int) {
	for range count {
		p.AddToLibrary(NewLand("Draw"))
	}
}

func TestLifeGainDrawLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lifeDrawLibrary(a, 30)
	lifeDrawLibrary(b, 30)
	card := NewArtifact("Rule", "", WithStaticAbility(DrawInsteadOfLifeGain(You)))
	card.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	for _, n := range []int{0, -1, 2} {
		g.PlayerGainLife(a, n)
	}
	if a.Life() != 20 || len(a.Hand()) != 2 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 0 || g.StackSize() != 0 {
		t.Fatal("positive gain must draw immediately without life gain")
	}
	g.PlayerGainLife(b, 1)
	if b.Life() != 21 || len(b.Hand()) != 0 {
		t.Fatal("wrong subject")
	}
	clone := g.Clone()
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.PlayerGainLife(a, 1)
	g.PlayerGainLife(b, 2)
	if a.Life() != 21 || b.Life() != 21 || len(b.Hand()) != 2 {
		t.Fatal("current controller was not used")
	}
	clone.PlayerGainLife(clone.GetPlayer(a.PlayerID()), 1)
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 3 || len(a.Hand()) != 2 {
		t.Fatal("clone state leaked")
	}
	ids := g.PhaseOut(source.ID())
	g.PlayerGainLife(b, 1)
	g.PhaseIn(ids)
	g.PlayerGainLife(b, 1)
	if b.Life() != 22 || len(b.Hand()) != 3 {
		t.Fatal("phase cycle")
	}
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
	g.PlayerGainLife(b, 1)
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PlayerGainLife(b, 1)
	if b.Life() != 23 || len(b.Hand()) != 4 {
		t.Fatal("ability removal and restoration")
	}
	var stale ReplacementEffect
	for _, r := range g.effects.cycleReplacements {
		if _, ok := r.(*lifeGainDrawReplacement); ok {
			stale = r
			break
		}
	}
	if stale == nil {
		t.Fatal("missing rule")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	b.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if stale.IsActive(g) || stale.Matches(NewLifeGainAction(uuid.Nil, a.PlayerID(), 1), g) {
		t.Fatal("old visit matched returned source")
	}
	g.PlayerGainLife(a, 1)
	if a.Life() != 21 || len(a.Hand()) != 3 {
		t.Fatal("new visit did not get rule")
	}
}

func TestLifeGainDrawDeclarationsAndCopies(t *testing.T) {
	for _, subject := range []PlayerRef{nil, FromPlayer(BindPlayer("unbound"))} {
		card := NewArtifact("Invalid", "", WithStaticAbility(DrawInsteadOfLifeGain(subject)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	if ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithStaticAbility(DrawInsteadOfLifeGain(You)))) == nil {
		t.Fatal("spell source accepted")
	}
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	lifeDrawLibrary(a.BasePlayer, 10)
	lifeDrawLibrary(b, 10)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithStaticAbility(DrawInsteadOfLifeGain(You))} })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	original := g.PutOnBattlefield(donor, b.PlayerID())
	copyPlan := entryCopyPlan(t, "Copy", func() []CardOption { return []CardOption{WithAbility(MayCopyAsEnters(IsCreature))} })
	card, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(card, a.PlayerID())
	g.PlayerGainLife(a, 2)
	g.PlayerGainLife(b, 1)
	if a.Life() != 20 || b.Life() != 20 || len(a.Hand()) != 2 || len(b.Hand()) != 1 {
		t.Fatal("copy lost or shared rule")
	}
	g.PhaseOut(original.ID())
	g.PlayerGainLife(a, 1)
	g.PhaseOut(copied.ID())
	g.PlayerGainLife(a, 1)
	if a.Life() != 21 || len(a.Hand()) != 3 {
		t.Fatal("copy lifecycle")
	}
}

type lifeDrawRewrite struct {
	replacementBase
	replace func(*LifeGainAction, *Game) Action
}

func (*lifeDrawRewrite) Matches(a Action, _ GameReader) bool { _, ok := a.(*LifeGainAction); return ok }
func (*lifeDrawRewrite) IsActive(GameReader) bool            { return true }
func (r *lifeDrawRewrite) Clone() ReplacementEffect          { c := *r; return &c }
func (r *lifeDrawRewrite) Replace(a Action, g *Game) Action  { return r.replace(a.(*LifeGainAction), g) }

func TestLifeGainDrawOrderingAndRechecks(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		lifeDrawLibrary(a.BasePlayer, 10)
		g.PutOnBattlefield(NewArtifact("Rule", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
		g.AddReplacementEffect(&lifeDrawRewrite{replace: func(gain *LifeGainAction, _ *Game) Action {
			return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), gain.Amount()*2)
		}})
		g.PlayerGainLife(a, 2)
		want := 4
		if choice == 1 {
			want = 2
		}
		if a.Life() != 20 || len(a.Hand()) != want {
			t.Fatalf("choice %d: hand %d, want %d", choice, len(a.Hand()), want)
		}
	}
	for _, change := range []string{"zero", "remove", "control"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		lifeDrawLibrary(a, 10)
		p := g.PutOnBattlefield(NewArtifact("Rule", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
		g.AddReplacementEffect(&lifeDrawRewrite{replace: func(gain *LifeGainAction, g *Game) Action {
			switch change {
			case "zero":
				return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), 0)
			case "remove":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			return gain
		}})
		g.PlayerGainLife(a, 2)
		want := 22
		if change == "zero" {
			want = 20
		}
		if a.Life() != want || len(a.Hand()) != 0 {
			t.Fatalf("%s: stale replacement applied", change)
		}
	}
}

func TestLifeGainDrawNestedDraws(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lifeDrawLibrary(a, 10)
	g.SetStep(core.Upkeep)
	g.PutOnBattlefield(NewArtifact("Life rule", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
	player := BindPlayer("drawing player")
	g.PutOnBattlefield(NewArtifact("Draw rule", "", WithAbility(Static(ReplaceDraw(ExceptFirstDrawStepDraw, player, GainLifeFor(FromPlayer(player), Fixed(1)))))), a.PlayerID())
	g.PlayerGainLife(a, 2)
	if a.Life() != 20 || len(a.Hand()) != 2 || len(g.resolution.playerBindings) != 0 || len(g.resolution.drawReplacementRules) != 0 {
		t.Fatal("nested gain/draw replacements or scope restoration")
	}
	g.PlayerGainLife(a, 1)
	if len(a.Hand()) != 3 {
		t.Fatal("replacement exclusion escaped event")
	}
	a.SetLibrary(nil)
	g.PlayerGainLife(a, 1)
	g.CheckStateBasedActions()
	if a.IsAlive() {
		t.Fatal("replacement bypassed empty-library draw loss")
	}
}

type lifeDrawDepartOnDraw struct {
	replacementBase
	subject uuid.UUID
	count   int
}

func (*lifeDrawDepartOnDraw) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*DrawCardAction)
	return ok
}
func (*lifeDrawDepartOnDraw) IsActive(GameReader) bool   { return true }
func (r *lifeDrawDepartOnDraw) Clone() ReplacementEffect { c := *r; return &c }
func (r *lifeDrawDepartOnDraw) Replace(a Action, g *Game) Action {
	r.count++
	if p := g.FindPermanent(r.subject); p != nil {
		g.BouncePermanentToHand(p)
	}
	return a
}

func TestLifeGainDrawCommittedAmountAndQueries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lifeDrawLibrary(a, 10)
	p := g.PutOnBattlefield(NewArtifact("Rule", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
	action := NewLifeGainAction(uuid.Nil, a.PlayerID(), 3)
	for range 2 {
		candidates := g.effects.replacementCandidates(action, g, nil)
		if len(candidates) != 1 || !candidates[0].Matches(action, g) || !candidates[0].Clone().IsActive(g) {
			t.Fatal("live query failed")
		}
	}
	if len(a.Hand()) != 0 || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("query changed state")
	}
	observer := &lifeDrawDepartOnDraw{subject: p.ID()}
	g.AddReplacementEffect(observer)
	result := BindNumber("gain")
	if err := ApplyEffect(g, Sequence(GainLifeForWithResult(You, Fixed(3), LifeGainedInto(result)), LoseLifeFor(You, FromNumber(result))), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observer.count != 3 || a.Life() != 20 || b.Life() != 20 || len(a.Hand()) != 4 {
		t.Fatal("source departure changed committed draws")
	}
	g.PlayerGainLife(a, 1)
	if observer.count != 3 || a.Life() != 21 {
		t.Fatal("later gain inherited departed rule")
	}
}

type lifeDrawChoicePlayer struct {
	*BasePlayer
	choose func() int
}

func (p *lifeDrawChoicePlayer) ChooseMode([]string, string) int { return p.choose() }

func TestLifeGainDrawSelectedRuleRemoval(t *testing.T) {
	a := &lifeDrawChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	lifeDrawLibrary(a.BasePlayer, 10)
	first := g.PutOnBattlefield(NewArtifact("First", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Second", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
	choices := 0
	a.choose = func() int {
		choices++
		g.AddContinuousEffect(RemoveAllAbilities(first.ID(), core.EndOfTurn))
		return 0
	}
	g.PlayerGainLife(a, 2)
	if choices != 1 || a.Life() != 20 || len(a.Hand()) != 2 {
		t.Fatal("selected stale rule blocked live replacement")
	}
	g.PlayerGainLife(a, 1)
	if choices != 1 || len(a.Hand()) != 3 {
		t.Fatal("live rule did not continue independently")
	}
}

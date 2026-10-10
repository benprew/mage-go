package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDrawReplacementScopeTiming(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	player := BindPlayer("drawing player")
	rule := ReplaceDraw(ExceptFirstDrawStepDraw, player, GainLifeFor(FromPlayer(player), Fixed(2)))
	card := NewEnchantment("Rule", "", WithAbility(Static(rule)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	for range 6 {
		a.AddToLibrary(NewLand("Draw"))
		b.AddToLibrary(NewLand("Draw"))
	}
	g.SetStep(core.Upkeep)
	g.PlayerDrawCard(a)
	if a.Life() != 22 || len(a.Hand()) != 0 {
		t.Fatal("upkeep draw did not replace")
	}
	g.SetStep(core.Draw)
	g.PlayerDrawCard(a)
	if a.Life() != 22 || len(a.Hand()) != 1 {
		t.Fatal("first effect draw during draw step must survive")
	}
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	if a.Life() != 24 || b.Life() != 22 || len(b.Hand()) != 0 {
		t.Fatal("event scope or active-player timing")
	}
	cloned := g.Clone()
	cloned.PlayerDrawCard(cloned.GetPlayer(a.PlayerID()))
	if cloned.GetPlayer(a.PlayerID()).Life() != 26 || a.Life() != 24 {
		t.Fatal("clone history or bindings")
	}
	g.SetStep(core.PrecombatMain)
	g.SetStep(core.Draw)
	g.PlayerDrawCard(a)
	if len(a.Hand()) != 2 {
		t.Fatal("extra draw step did not reset first draw")
	}
	if len(g.resolution.playerBindings) != 0 {
		t.Fatal("event binding escaped")
	}
}

func TestDrawReplacementScopeGeneratedDraw(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.Draw)
	player := BindPlayer("drawing player")
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ControllerDrawStep, player, Sequence(GainLifeFor(FromPlayer(player), Fixed(1)), DrawFor(FromPlayer(player), Fixed(1))))))), a.PlayerID())
	a.AddToLibrary(NewLand("Draw"))
	g.PlayerDrawCard(a)
	if a.Life() != 21 || len(a.Hand()) != 1 {
		t.Fatal("replacement reapplied to its generated draw")
	}
	a.AddToLibrary(NewLand("Draw"))
	g.PlayerDrawCard(a)
	if a.Life() != 22 || len(a.Hand()) != 2 {
		t.Fatal("exclusion escaped its event")
	}
	if len(g.resolution.drawReplacementRules) != 0 {
		t.Fatal("draw ancestry escaped")
	}
}

func TestDrawReplacementScopeInvalid(t *testing.T) {
	player := BindPlayer("event")
	for _, rule := range []StaticRule{
		ReplaceDraw(DrawReplacementWindow(99), player, GainLifeFor(You, Fixed(1))),
		ReplaceDraw(ControllerDrawStep, nil, GainLifeFor(You, Fixed(1))),
		ReplaceDraw(ControllerDrawStep, BindPlayer(""), GainLifeFor(You, Fixed(1))),
		ReplaceDraw(ControllerDrawStep, player, nil),
		ReplaceDraw(ControllerDrawStep, player, GainLifeFor(FromPlayer(BindPlayer("absent")), Fixed(1))),
	} {
		if err := ValidateCardDefinition("Invalid", NewEnchantment("Invalid", "", WithAbility(Static(rule)))); err == nil {
			t.Fatal("invalid draw replacement accepted")
		}
	}
	if err := ApplyEffect(NewGame(NewBasePlayer("A"), NewBasePlayer("B")), GainLifeFor(FromPlayer(player), Fixed(1)), uuid.Nil, uuid.Nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDrawReplacementScopeLifecycle(t *testing.T) {
	for _, state := range []string{"control", "phase", "depart-return", "remove-ability", "clone"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(3)
			g.SetStep(core.Draw)
			player := BindPlayer("drawing player")
			card := NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ControllerDrawStep, player, GainLifeFor(FromPlayer(player), Fixed(2))))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.AddToLibrary(NewLand("Draw"))
			b.AddToLibrary(NewLand("Draw"))
			switch state {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
			case "phase":
				g.PhaseOut(source.ID())
			case "depart-return":
				g.DestroyPermanent(source)
				g.PutOnBattlefield(card, b.PlayerID())
				g.SetActivePlayerIndex(1)
			case "remove-ability":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
			case "clone":
				g = g.Clone()
			}
			id := a.PlayerID()
			if state == "control" || state == "depart-return" {
				id = b.PlayerID()
			}
			g.PlayerDrawCard(g.GetPlayer(id))
			want := 22
			hand := 0
			if state == "phase" || state == "remove-ability" {
				want = 20
				hand = 1
			}
			if g.GetPlayer(id).Life() != want || len(g.GetPlayer(id).Hand()) != hand {
				t.Fatal("source lifecycle", g.GetPlayer(id).Life(), len(g.GetPlayer(id).Hand()))
			}
			if state == "clone" && (a.Life() != 20 || len(a.Library()) != 1) {
				t.Fatal("clone changed live game")
			}
			if state == "phase" {
				g.PhaseIn([]uuid.UUID{source.ID()})
				g.PlayerDrawCard(a)
				if a.Life() != 22 {
					t.Fatal("phase-in did not restore rule")
				}
			}
		})
	}
}

type drawReplacementChooser struct {
	*BasePlayer
	choice int
	offers int
}

func (p *drawReplacementChooser) ChooseMode(options []string, _ string) int {
	p.offers++
	return p.choice
}

func TestDrawReplacementScopeOrderingAndInstances(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		g := NewGame(a, NewBasePlayer("B"))
		g.SetTurn(3)
		g.SetStep(core.Draw)
		player := BindPlayer("drawing player")
		card := NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ControllerDrawStep, player, Sequence(GainLifeFor(FromPlayer(player), Fixed(1)), DrawFor(FromPlayer(player), Fixed(1)))))))
		first := g.PutOnBattlefield(card, a.PlayerID())
		second := g.PutOnBattlefield(newPermanentSpellCopyCard(card.Copy()), a.PlayerID())
		if first.incarnationID == second.incarnationID {
			t.Fatal("copy did not create another visit")
		}
		a.AddToLibrary(NewLand("Draw"))
		g.PlayerDrawCard(a)
		if a.Life() != 22 || len(a.Hand()) != 1 || a.offers != 1 {
			t.Fatal("copied rule exclusion or replacement ordering", a.Life(), len(a.Hand()), a.offers)
		}
		a.AddToLibrary(NewLand("Draw"))
		g.PlayerDrawCard(a)
		if a.Life() != 24 || len(a.Hand()) != 2 || a.offers != 2 {
			t.Fatal("independent draw events")
		}
	}
}

func TestDrawReplacementScopeReplacedFirstAndFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.Draw)
	player := BindPlayer("drawing player")
	source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ExceptFirstDrawStepDraw, player, GainLifeFor(FromPlayer(player), Fixed(2)))))), a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) {})
	g.PlayerDrawCard(a)
	a.AddToLibrary(NewLand("Draw"))
	g.PlayerDrawCard(a)
	if a.Life() != 20 || len(a.Hand()) != 1 {
		t.Fatal("replaced first proposal counted as a draw")
	}
	a.AddToLibrary(NewLand("Another"))
	outer := BindPlayer("outer")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{outer: b.PlayerID()}
	g.PlayerDrawCard(a)
	if a.Life() != 22 || g.resolution.playerBindings[outer] != b.PlayerID() || len(g.resolution.playerBindings) != 1 {
		t.Fatal("outer frame lost or event player leaked")
	}
	b.AddToLibrary(NewLand("Draw"))
	if card, ok := g.PlayerDrawCard(b); ok || card != nil || b.Life() != 22 {
		t.Fatal("replaced draw supplied output")
	}
}

func TestDrawReplacementScopeConsecutiveSteps(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	player := BindPlayer("drawing player")
	g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ExceptFirstDrawStepDraw, player, GainLifeFor(FromPlayer(player), Fixed(2)))))), a.PlayerID())
	a.AddToLibrary(NewLand("One"))
	a.AddToLibrary(NewLand("Two"))
	g.RunStepWithPriority(core.Draw)
	g.RunStepWithPriority(core.Draw)
	if a.Life() != 20 || len(a.Hand()) != 2 {
		t.Fatal("consecutive new draw steps shared history")
	}
}

func TestDrawReplacementScopeStaleSelectedVisit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.Draw)
	player := BindPlayer("drawing player")
	card := NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ControllerDrawStep, player, GainLifeFor(FromPlayer(player), Fixed(2))))))
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	action := g.drawAction(a.PlayerID(), false)
	candidates := g.effects.replacementCandidates(action, g, nil)
	if len(candidates) != 1 {
		t.Fatal("missing draw replacement")
	}
	selected := candidates[0]
	g.DestroyPermanent(source)
	g.PutOnBattlefield(card, b.PlayerID())
	if result := selected.Replace(action, g); result != action || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("selected replacement used a returned source visit")
	}
}

func TestDrawReplacementScopeSelectedAbilityRemoval(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.Draw)
	player := BindPlayer("drawing player")
	source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(Static(ReplaceDraw(ControllerDrawStep, player, GainLifeFor(FromPlayer(player), Fixed(2)))))), a.PlayerID())
	action := g.drawAction(a.PlayerID(), false)
	candidates := g.effects.replacementCandidates(action, g, nil)
	if len(candidates) != 1 {
		t.Fatal("missing draw replacement")
	}
	selected := candidates[0]
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
	if result := selected.Replace(action, g); result != action || a.Life() != 20 {
		t.Fatal("selected replacement used a removed ability")
	}
}

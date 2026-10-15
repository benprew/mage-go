package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPentaclePlayer struct {
	*mage.BasePlayer
	target           uuid.UUID
	source           uuid.UUID
	targets, sources int
}

func (p *ronPentaclePlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *mage.Game) []uuid.UUID {
	p.targets++
	for _, id := range possible {
		if id == p.target {
			return []uuid.UUID{id}
		}
	}
	return p.BasePlayer.ChooseTargets(possible, minimum, maximum, g)
}
func (p *ronPentaclePlayer) ChooseDamageSource(sources []mage.DamageSource, _ *mage.Game) int {
	p.sources++
	for i, source := range sources {
		if source.ID() == p.source {
			return i
		}
	}
	return -1
}

func TestRONSelectedPermanentRedirectRecord(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "LEG", "Nova Pentacle"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPentaclePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronPentaclePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	pentacle := g.FindPermanent(card.ID())
	target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 20), a.PlayerID())
	source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 20), b.PlayerID())
	b.target = target.ID()
	a.source = source.ID()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), pentacle.ID(), 0, nil); err == nil || pentacle.Tapped || g.StackSize() != 0 || a.sources != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), pentacle.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.targets != 0 || b.targets == 0 || a.sources != 0 || b.sources != 0 || len(g.WaitingRedirectionPermanentSources()) != 0 {
		t.Fatal("wrong target chooser or early source choice")
	}
	g.RemoveFromBattlefield(pentacle)
	g.ResolveStack()
	if a.sources != 1 || b.sources != 0 {
		t.Fatal("source was not chosen by the saved activator at resolution")
	}
	baseline := g.Clone()
	for _, game := range []*mage.Game{g, g.Clone()} {
		game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 3, source.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.FindPermanent(target.ID()).Damage != 3 {
			t.Fatal("damage did not redirect after source departure")
		}
		game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 2, source.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 18 {
			t.Fatal("one-shot rule did not expire")
		}
	}
	if baseline.GetPlayer(a.PlayerID()).Life() != 20 || baseline.FindPermanent(target.ID()).Damage != 0 {
		t.Fatal("clone state crossed")
	}
}

const selectedPermanentRedirectRON = `CreateFutureReplaceWouldDealDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(TheChosenDamageSource,Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)])`

func TestRONSelectedPermanentRedirectInvalid(t *testing.T) {
	for _, text := range []string{
		strings.Replace(selectedPermanentRedirectRON, "Player(You)", "Player()", 1),
		strings.Replace(selectedPermanentRedirectRON, "ThisPermanent)", "Trigger_ThatPermanent)", 1),
		strings.Replace(selectedPermanentRedirectRON, "[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)]", "[]", 1),
		strings.Replace(selectedPermanentRedirectRON, "ThisPermanent)", "ThisPermanent,ThisPermanent)", 1),
	} {
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = compileEffect(nodes[0], scope{permanent: true, resolving: true, chosenDamageSource: &chosenDamageSourcePlan{}}); err == nil {
			t.Fatal("invalid declaration accepted", text)
		}
	}
	nodes, err := Parse(strings.NewReader(selectedPermanentRedirectRON), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{
		{permanent: true, resolving: true},
		{permanent: true, chosenDamageSource: &chosenDamageSourcePlan{}},
		{permanent: true, resolving: true, enteringSource: true, chosenDamageSource: &chosenDamageSourcePlan{}},
		{permanent: true, resolving: true, drawReplacementBody: true, chosenDamageSource: &chosenDamageSourcePlan{}},
	} {
		if _, err = compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
	if _, err = compileEffect(nodes[0], scope{permanent: true, resolving: true, chosenDamageSource: &chosenDamageSourcePlan{}}); err != nil {
		t.Fatal(err)
	}
}

func TestRONSelectedPermanentRedirectLegality(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "LEG", "Nova Pentacle"))
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"illegal-at-resolution", "hexproof-after-resolution", "target-phase", "phase-cycle", "target-return", "source-control", "copied"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPentaclePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &ronPentaclePlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			pentacle := g.PutOnBattlefield(factory(), a.PlayerID())
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 20), b.PlayerID())
			source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 20), b.PlayerID())
			a.source = source.ID()
			b.target = target.ID()
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), pentacle.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if state == "illegal-at-resolution" {
				g.PhaseOut(target.ID())
			}
			if state == "source-control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: pentacle.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if state == "copied" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			wantChoices := 1
			if state == "illegal-at-resolution" {
				wantChoices = 0
			}
			if state == "copied" {
				wantChoices = 2
			}
			if a.sources != wantChoices || b.sources != 0 {
				t.Fatal("wrong resolution choices")
			}
			switch state {
			case "illegal-at-resolution":
				g.PhaseIn([]uuid.UUID{target.ID()})
			case "hexproof-after-resolution":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "target-phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(target.Card, b.PlayerID())
			}
			g.DealDamageToPlayer(a, 2, source.ID())
			want := 20
			if state == "illegal-at-resolution" || state == "target-phase" || state == "target-return" {
				want = 18
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
			if state == "copied" {
				g.DealDamageToPlayer(a, 3, source.ID())
				if a.Life() != 20 || g.FindPermanent(target.ID()).Damage != 5 {
					t.Fatal("copies shared consumption")
				}
			}
		})
	}
}

func TestRONSelectedPermanentRedirectInstances(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "LEG", "Nova Pentacle"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPentaclePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronPentaclePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	targetA := g.PutOnBattlefield(mage.NewCreature("A target", "", 1, 20), a.PlayerID())
	targetB := g.PutOnBattlefield(mage.NewCreature("B target", "", 1, 20), b.PlayerID())
	a.target = targetB.ID()
	b.target = targetA.ID()
	a.source = targetB.ID()
	b.source = targetA.ID()
	a.ManaPool().Add(core.Colorless, 3)
	b.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	for _, game := range []*mage.Game{g, g.Clone()} {
		game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 2, targetB.ID())
		game.DealDamageToPlayer(game.GetPlayer(b.PlayerID()), 3, targetA.ID())
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanent(targetA.ID()).Damage != 2 || game.FindPermanent(targetB.ID()).Damage != 3 {
			t.Fatal("factory bindings crossed")
		}
	}
}

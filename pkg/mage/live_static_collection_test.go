package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLiveStaticCollectionPrevention(t *testing.T) {
	for _, change := range []string{"live", "control", "phase", "phase-cycle", "ability-loss", "depart", "return", "clone", "recheck", "unpreventable", "source-lki"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			set := PermanentsControlledBy(You, IsCreature)
			card := NewArtifact("Rule", "", WithAbility(Static(PreventDamageFromMatchingPermanentsToSetIf(set, IsLand, PermanentIsUntapped(This)))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := NewCreature("Victim", "", 1, 20)
			v := g.PutOnBattlefield(victimCard, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
			source := g.PutOnBattlefield(NewLand("Source"), b.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			want := 0
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = 1
			case "phase":
				g.PhaseOut(p.ID())
				want = 1
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "ability-loss":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.Indefinite))
				want = 1
			case "depart":
				g.RemoveFromBattlefield(p)
				want = 1
			case "return":
				g.RemoveFromBattlefield(v)
				v = g.PutOnBattlefield(victimCard, a.PlayerID())
			case "clone":
				clone := g.Clone()
				clone.MutablePermanent(p.ID()).Tapped = true
				clone.DealDamageToPermanentFromSource(clone.FindPermanent(v.ID()), 1, saved)
				if clone.FindPermanent(v.ID()).Damage != 1 || g.FindPermanent(v.ID()).Damage != 0 {
					t.Fatal("clone state shared")
				}
			case "recheck":
				want = 1
				g.AddReplacementEffect(&testPrioritizedReplacement{replacementBase: replacementBase{duration: core.Indefinite, priority: ReplacementPrioritySelf}, matchFunc: func(action Action) bool { _, ok := action.(*DamageToCreatureAction); return ok }, replaceFunc: func(action Action, game *Game) Action { game.MutablePermanent(p.ID()).Tapped = true; return action }})
			case "unpreventable":
				g.MutablePermanent(v.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				want = 1
			case "source-lki":
				g.RemoveFromBattlefield(source)
			}
			g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(v.ID()), 1, saved), PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, saved), PlayerDamageAssignment(a, 1, saved))
			wantOther := 1
			if change == "control" {
				wantOther = 0
			}
			if g.FindPermanent(v.ID()).Damage != want || g.FindPermanent(other.ID()).Damage != wantOther || a.Life() != 19 {
				t.Fatal("live membership or packet scope")
			}
		})
	}
}

func TestLiveStaticCollectionCondition(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Rule", "", 1, 20)
	p := g.PutOnBattlefield(card, a.PlayerID())
	condition := PlayerControlsAny(You, WithoutPermanents(MatchingPermanents(IsCreature), SingletonPermanent(This)))
	ctx := StateContext{SourceID: p.ID(), SourceIncarnation: p.incarnationID, ControllerID: a.PlayerID()}
	if condition.checkState(g, ctx) {
		t.Fatal("source supplied other creature")
	}
	q := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), a.PlayerID())
	if !condition.checkState(g, ctx) {
		t.Fatal("missing live member")
	}
	clone := g.Clone()
	clone.PhaseOut(q.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone membership shared")
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) {
		t.Fatal("returned source supplied old visit")
	}
}

func TestLiveStaticCollectionInvalid(t *testing.T) {
	for _, set := range []PermanentSet{nil, SingletonPermanent(DeclarePermanentTarget("target", IsCreature)), FromPermanentSet(BindPermanentSet("unbound"))} {
		card := NewArtifact("Invalid", "", WithAbility(Static(PreventDamageFromMatchingPermanentsToSetIf(set, IsLand, PermanentIsUntapped(This)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid set accepted")
		}
		card = NewArtifact("Invalid", "", WithAbility(Static(While(PlayerControlsAny(You, set), ModifyPT(This, Fixed(1), Fixed(1))))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid condition accepted")
		}
	}
	card := NewArtifact("Invalid", "", WithAbility(Static(PreventDamageFromMatchingPermanentsToSetIf(MatchingPermanents(IsCreature), IsLand, nil))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("nil condition accepted")
	}
}

func TestLiveStaticCollectionWaitingIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subjects := WithoutPermanents(PermanentsControlledBy(You, IsCreature), SingletonPermanent(This))
	card := NewCreature("Rule", "", 1, 20, WithAbility(Static(PreventDamageFromMatchingPermanentsToSetIf(subjects, IsLand, PermanentIsUntapped(This)))))
	p := g.PutOnBattlefield(card, a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Member", "", 1, 20), a.PlayerID())
	var saved *staticDamagePreventionReplacement
	for _, r := range g.effects.cycleReplacements {
		if r, ok := r.(*staticDamagePreventionReplacement); ok {
			saved = r.Clone().(*staticDamagePreventionReplacement)
		}
	}
	if saved == nil {
		t.Fatal("no collection registration")
	}
	check := func(want *Permanent) {
		t.Helper()
		g.effects.cycleReplacements = []ReplacementEffect{saved}
		refs := g.WaitingPreventionPermanentSources()
		if want == nil {
			if len(refs) != 0 {
				t.Fatal("inactive waiting reference")
			}
		} else if len(refs) != 1 || !refs[0].SameObject(g.CaptureDamageSource(want.ID())) {
			t.Fatal("wrong waiting member")
		}
	}
	check(q)
	g.PhaseOut(q.ID())
	check(nil)
	g.PhaseIn([]uuid.UUID{q.ID()})
	check(q)
	g.MutablePermanent(p.ID()).Tapped = true
	check(nil)
	g.MutablePermanent(p.ID()).Tapped = false
	check(q)
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, a.PlayerID())
	check(nil)
}

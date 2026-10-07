package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerDamageHistoryValues(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	total, artifact := DamageTakenThisTurn(You), ArtifactDamageTakenThisTurn(You)
	check := func(wantTotal, wantArtifact int) {
		t.Helper()
		if total.resolveValue(ctx) != wantTotal || artifact.resolveValue(ctx) != wantArtifact {
			t.Fatalf("history = %d/%d, want %d/%d", total.resolveValue(ctx), artifact.resolveValue(ctx), wantTotal, wantArtifact)
		}
	}
	check(0, 0)
	source := g.PutOnBattlefield(NewCreature("Artifact", "", 1, 10, WithCardType(core.TypeArtifact)), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 10), b.PlayerID())
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(a, 3, g.CaptureDamageSource(other.ID())), PlayerDamageAssignment(b, 4, g.CaptureDamageSource(source.ID())))
	check(5, 2)
	a.LoseLife(1)
	a.GainLife(2)
	check(5, 2)
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(a, 3, source.ID())
	check(6, 3)
	if err := ApplyEffect(g, PreventDamageThisTurn(You, PermanentFilter{}), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(a, 9, source.ID())
	check(6, 3)
	clone := g.Clone()
	clone.effects.ClearReplacementsEndOfTurn()
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 1, source.ID())
	if total.resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 7 {
		t.Fatal("clone lost history")
	}
	check(6, 3)
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	check(0, 0)
	if total.resolveValue(&EffectContext{Game: g, Controller: uuid.New()}) != 0 {
		t.Fatal("absent player supplied history")
	}
	if DamageTakenThisTurn(FromPlayer(BindPlayer("missing"))).resolveValue(ctx) != 0 {
		t.Fatal("unbound player supplied history")
	}
	for _, value := range []ResolutionValue{DamageTakenThisTurn(nil), ArtifactDamageTakenThisTurn(nil)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, value))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("nil player accepted")
		}
	}
}

func TestPlayerDamageHistorySourceInformation(t *testing.T) {
	for _, state := range []string{"departed", "returned", "phased", "control"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 1, 10, WithCardType(core.TypeArtifact))
			p := g.PutOnBattlefield(card, b.PlayerID())
			captured := g.CaptureDamageSource(p.ID())
			switch state {
			case "departed", "returned":
				g.RemoveFromBattlefield(p)
				if state == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				if err := ApplyEffect(g, Until(core.EndOfTurn, ChangeController(This, You)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.DealDamageToPlayerFromSource(a, 3, captured)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if ArtifactDamageTakenThisTurn(You).resolveValue(ctx) != 3 {
				t.Fatal("source information lost")
			}
		})
	}
}

func TestPlayerDamageHistoryTypesAndSpells(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Changing source", "", 1, 10), b.PlayerID())
	g.DealDamageToPlayer(a, 1, source.ID())
	g.MutablePermanent(source.ID()).GrantBaseAttr(core.AttrIsArtifact)
	g.DealDamageToPlayer(a, 2, source.ID())
	g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
	g.DealDamageToPlayer(a, 3, source.ID())
	card := NewArtifact("Artifact spell", "")
	card.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: b.PlayerID()}
	g.pushStack(obj)
	saved := g.CaptureSpellDamageSource(obj.ID)
	g.GetStack().RemoveByID(obj.ID)
	g.DealDamageToPlayerFromSource(a, 4, saved)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if DamageTakenThisTurn(You).resolveValue(ctx) != 10 || ArtifactDamageTakenThisTurn(You).resolveValue(ctx) != 6 {
		t.Fatal("history used present types or lost spell information")
	}
	if DamageTakenThisTurn(You).resolveValue(ctx) != 10 {
		t.Fatal("query changed history")
	}
	g.RunStepWithPriority(core.Cleanup)
	if DamageTakenThisTurn(You).resolveValue(ctx) != 0 || ArtifactDamageTakenThisTurn(You).resolveValue(ctx) != 0 {
		t.Fatal("native cleanup retained history")
	}
}

func TestPlayerDamageHistoryReplacementOrdering(t *testing.T) {
	for choice := range 2 {
		t.Run(string(rune('0'+choice)), func(t *testing.T) {
			a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 10, WithCardType(core.TypeArtifact)), b.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), a.PlayerID())
			g.SetArtifactDamageRedirect(a.PlayerID(), victim.ID())
			g.effects.AddReplacement(&damagePreventionRuleReplacement{from: IsArtifact, toPlayerID: a.PlayerID()})
			g.DealDamageToPlayer(a, 3, source.ID())
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if DamageTakenThisTurn(You).resolveValue(ctx) != 0 || ArtifactDamageTakenThisTurn(You).resolveValue(ctx) != 0 || a.prompts != 1 {
				t.Fatal("prevented or redirected damage entered player history")
			}
		})
	}
}

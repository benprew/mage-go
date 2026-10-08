package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func selectedSourceFixture(t *testing.T, exact bool) (*Game, *BasePlayer, *StackObject) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Selected", "{R}", 2, 10)
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID()}
	g.stack.Push(obj)
	binding := BindDamageSource("selected")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: g.CaptureSpellDamageSource(obj.ID)}
	effect := PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red))
	if exact {
		effect = PreventNextDamageFromExactSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red))
	}
	if err := effect.Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	return g, b, obj
}

func TestSelectedSourcePreventionContinuity(t *testing.T) {
	for _, state := range []string{"live", "depart", "return", "phase", "phase-cycle", "blue", "cleanup", "consumed-spell", "exact"} {
		t.Run(state, func(t *testing.T) {
			g, b, obj := selectedSourceFixture(t, state == "exact")
			spell := g.CaptureSpellDamageSource(obj.ID)
			if state == "consumed-spell" {
				g.DealDamageToPlayerFromSource(b, 1, spell)
			}
			g.ResolveTopOfStack()
			p := g.FindPermanent(obj.Card.ID())
			if p == nil {
				t.Fatal("spell did not enter")
			}
			saved := g.CaptureDamageSource(p.ID())
			switch state {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if state == "return" {
					g.PutOnBattlefield(p.Card, obj.Controller)
					g.DealDamageToPlayer(b, 1, p.ID())
				}
				g.zones.ClearLKI()
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "blue":
				g.MutablePermanent(p.ID()).ColorOverride = &[]core.Color{core.Blue}
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			for _, game := range []*Game{g, g.Clone()} {
				recipient := game.GetPlayer(b.PlayerID())
				before := recipient.Life()
				game.DealDamageToPlayerFromSource(recipient, 0, saved)
				game.DealSimultaneousDamage(PlayerDamageAssignment(recipient, 2, saved), PlayerDamageAssignment(recipient, 3, saved))
				want := before
				if state == "blue" || state == "cleanup" || state == "consumed-spell" || state == "exact" {
					want -= 5
				}
				if recipient.Life() != want {
					t.Fatalf("life=%d want=%d", recipient.Life(), want)
				}
				if state == "blue" {
					game.MutablePermanent(p.ID()).ColorOverride = &[]core.Color{core.Red}
					game.DealDamageToPlayerFromSource(recipient, 2, saved)
					if recipient.Life() != want {
						t.Fatal("mismatch consumed shield")
					}
				}
				game.DealDamageToPlayerFromSource(recipient, 1, saved)
				if recipient.Life() != want-1 {
					t.Fatal("shield not consumed")
				}
			}
		})
	}
}

func TestSelectedSourcePreventionCopyAndClone(t *testing.T) {
	g, b, obj := selectedSourceFixture(t, false)
	copyObj := g.CopyStackObjectDirect(obj, obj.Controller, false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveTopOfStack()
		var copied *Permanent
		for _, p := range game.AllBattlefield() {
			if p.IsToken {
				copied = p
			}
		}
		if copied == nil {
			t.Fatal("copy did not become token")
		}
		recipient := game.GetPlayer(b.PlayerID())
		game.DealDamageToPlayer(recipient, 1, copied.ID())
		if recipient.Life() != 19 {
			t.Fatal("separate copy inherited shield")
		}
		game.ResolveTopOfStack()
		game.DealDamageToPlayer(recipient, 2, obj.Card.ID())
		if recipient.Life() != 19 {
			t.Fatal("original did not inherit shield")
		}
	}
}

func TestSelectedSourcePreventionDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		PreventNextDamageFromSelectedSource(nil, You, DamageSourceFilter{}),
		PreventNextDamageFromSelectedSource(This, nil, DamageSourceFilter{}),
		PreventNextDamageFromSelectedSource(FromDamageSource(BindDamageSource("unbound")), You, DamageSourceFilter{}),
		PreventNextDamageFromSelectedSource(This, You, DamageSourceHasColor(core.Colorless)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestSelectedSourcePreventionChosenCopy(t *testing.T) {
	g, b, obj := selectedSourceFixture(t, false)
	copied := g.CopyStackObjectDirect(obj, obj.Controller, false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	binding := BindDamageSource("copy")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: g.CaptureSpellDamageSource(copied.ID)}
	if err := PreventNextDamageFromSelectedSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	for _, p := range g.AllBattlefield() {
		if !p.IsToken {
			t.Fatal("copy is not a token")
		}
		if p.ID() == obj.Card.ID() {
			t.Fatal("copy kept the original card identity")
		}
		refs := g.WaitingPreventionPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(g.CaptureDamageSource(p.ID())) {
			t.Fatal("continued token missing from waiting references")
		}
		g.DealDamageToPlayer(b, 2, p.ID())
		if b.Life() != 20 {
			t.Fatal("chosen copy shield did not follow its new card identity")
		}
		g.DealDamageToPlayer(b, 1, p.ID())
		if b.Life() != 19 {
			t.Fatal("copy shield not consumed")
		}
	}
	g.ResolveTopOfStack()
	g.DealDamageToPlayer(b, 2, obj.Card.ID())
	if b.Life() != 19 {
		t.Fatal("original shield lost its separate identity")
	}
}

func TestSelectedSourcePreventionEntryTimingAndCounter(t *testing.T) {
	t.Run("entry-damage", func(t *testing.T) {
		g, b, obj := selectedSourceFixture(t, false)
		obj.Controller = b.PlayerID()
		card := NewCreature("Entry source", "{R}", 2, 10, WithAbility(ETBEffect(DealDamageFrom(This, You, Fixed(2)))))
		card.SetID(obj.Card.ID())
		card.SetOwner(obj.Controller)
		obj.Card = card
		g.ResolveTopOfStack()
		if b.Life() != 20 {
			t.Fatal("continuity was installed after entry damage")
		}
		g.DealDamageToPlayer(b, 1, card.ID())
		if b.Life() != 19 {
			t.Fatal("entry damage did not consume shield")
		}
	})
	t.Run("counter-and-recast", func(t *testing.T) {
		g, b, obj := selectedSourceFixture(t, false)
		g.CounterStackObject(obj.ID)
		next := *obj
		next.ID = uuid.New()
		g.stack.Push(&next)
		g.ResolveTopOfStack()
		g.DealDamageToPlayer(b, 2, obj.Card.ID())
		if b.Life() != 18 {
			t.Fatal("recast inherited countered spell's shield")
		}
	})
}

type selectedSourceRejectedEntry struct{ BaseAbility }

func (*selectedSourceRejectedEntry) OnEnter(*Game, *Permanent) bool { return false }

func TestSelectedSourcePreventionRejectedEntry(t *testing.T) {
	g, b, obj := selectedSourceFixture(t, false)
	card := NewCreature("No entry", "{R}", 2, 10, WithAbility(&selectedSourceRejectedEntry{}))
	card.SetID(obj.Card.ID())
	card.SetOwner(obj.Controller)
	obj.Card = card
	g.ResolveTopOfStack()
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("entry was not rejected")
	}
	if len(g.WaitingPreventionPermanentSources()) != 0 {
		t.Fatal("failed entry created continuity")
	}
	direct := NewCreature("Direct entry", "{R}", 2, 10)
	direct.SetID(card.ID())
	direct.SetOwner(obj.Controller)
	g.PutOnBattlefield(direct, obj.Controller)
	g.DealDamageToPlayer(b, 2, direct.ID())
	if b.Life() != 18 {
		t.Fatal("direct entry inherited a spell shield")
	}
}

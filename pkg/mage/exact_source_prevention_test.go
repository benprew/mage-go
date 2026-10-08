package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExactSourcePreventionIdentity(t *testing.T) {
	for _, state := range []string{"live", "depart", "return", "phase", "phase-cycle", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{R}", 1, 10)
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			binding := BindDamageSource("selected")
			effect := Sequence(SnapshotDamageSource(This, binding), PreventNextDamageFromExactSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)))
			if err := ValidateCardDefinition("Shield", NewArtifact("Shield", "", WithAbility(Activate(Costs(), effect)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, p.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if state == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
				g.zones.ClearLKI()
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			for _, game := range []*Game{g, g.Clone()} {
				recipient := game.GetPlayer(b.PlayerID())
				if state == "return" {
					game.DealDamageToPlayer(recipient, 1, p.ID())
				}
				before := recipient.Life()
				game.DealDamageToPlayerFromSource(recipient, 0, saved)
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
				game.DealSimultaneousDamage(PlayerDamageAssignment(recipient, 2, saved), PlayerDamageAssignment(recipient, 3, saved))
				want := before
				if state == "cleanup" {
					want -= 5
				}
				if recipient.Life() != want {
					t.Fatalf("life=%d want=%d", recipient.Life(), want)
				}
				game.DealDamageToPlayerFromSource(recipient, 1, saved)
				if recipient.Life() != want-1 {
					t.Fatal("shield was not consumed")
				}
			}
		})
	}
}

func TestExactSourcePreventionSpells(t *testing.T) {
	for _, state := range []string{"live", "blue", "depart", "recast", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			g, _, b, obj := spellSourceFixture(t)
			saved := g.CaptureSpellDamageSource(obj.ID)
			binding := BindDamageSource("selected")
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
			ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
			if err := PreventNextDamageFromExactSource(FromDamageSource(binding), You, DamageSourceHasColor(core.Red)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if state == "blue" {
				g.ChangeObjectColor(obj.ID, core.Blue)
			}
			if state == "depart" || state == "recast" {
				g.GetStack().Pop()
				if state == "recast" {
					replacement := *obj
					replacement.ID = uuid.New()
					g.GetStack().Push(&replacement)
					g.DealDamageToPlayerFromSource(b, 1, g.CaptureSpellDamageSource(replacement.ID))
				}
			}
			if state == "cleanup" {
				g.effects.ClearReplacementsEndOfTurn()
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				recipient := game.GetPlayer(b.PlayerID())
				before := recipient.Life()
				game.DealDamageToPlayerFromSource(recipient, 2, saved)
				want := before
				if state == "blue" || state == "cleanup" {
					want -= 2
				}
				if recipient.Life() != want {
					t.Fatalf("life=%d want=%d", recipient.Life(), want)
				}
				if state == "blue" {
					game.ChangeObjectColor(obj.ID, core.Red)
					game.DealDamageToPlayerFromSource(recipient, 2, saved)
					if recipient.Life() != want {
						t.Fatal("property mismatch consumed shield")
					}
				}
				game.DealDamageToPlayerFromSource(recipient, 1, saved)
				if recipient.Life() != want-1 {
					t.Fatal("shield survived matching event")
				}
			}
		})
	}
}

func TestExactSourcePreventionDeclarations(t *testing.T) {
	for _, body := range []Effect{
		PreventNextDamageFromExactSource(nil, You, DamageSourceFilter{}),
		PreventNextDamageFromExactSource(This, nil, DamageSourceFilter{}),
		PreventNextDamageFromExactSource(FromDamageSource(BindDamageSource("unbound")), You, DamageSourceFilter{}),
		PreventNextDamageFromExactSource(This, You, DamageSourceHasColor(core.Colorless)),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestExactSourcePreventionRechecksAndOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 10), a.PlayerID())
		saved := g.CaptureDamageSource(p.ID())
		if err := ApplyEffect(g, PreventNextDamageFromExactSource(This, You, DamageSourceHasColor(core.Red)), p.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, core.Blue)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DealDamageToPlayerFromSource(b, 2, saved)
		if b.Life() != 18 {
			t.Fatal("shield used saved color for a live source")
		}
		if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, core.Red)), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.AddPreventionShield(b.PlayerID(), 1)
		g.DealDamageToPlayerFromSource(b, 1, saved)
		if b.Life() != 18 || b.prompts != 1 {
			t.Fatal("affected player did not choose the competing effect")
		}
		g.DealDamageToPlayerFromSource(b, 2, saved)
		want := 17
		if choice == 1 {
			want = 18
		}
		if b.Life() != want {
			t.Fatalf("life=%d want=%d", b.Life(), want)
		}
	}
}

func TestExactSourcePreventionIndependentBindings(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "{R}", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "{R}", 1, 10), a.PlayerID())
	one, two := BindDamageSource("one"), BindDamageSource("two")
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{one: g.CaptureDamageSource(first.ID()), two: g.CaptureDamageSource(second.ID())}
	for _, binding := range []*DamageSourceBinding{one, two} {
		if err := PreventNextDamageFromExactSource(FromDamageSource(binding), You, DamageSourceFilter{}).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	delete(g.resolution.damageSourceBindings, one)
	delete(g.resolution.damageSourceBindings, two)
	clone := g.Clone()
	g.DealDamageToPlayer(b, 1, first.ID())
	g.DealDamageToPlayer(b, 1, first.ID())
	g.DealDamageToPlayer(b, 1, second.ID())
	if b.Life() != 19 {
		t.Fatal("shield installations shared state")
	}
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 1, first.ID())
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("shield consumption changed clone")
	}
	if err := PreventNextDamageFromExactSource(FromDamageSource(one), You, DamageSourceFilter{}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.effects.replacements) != 2 {
		t.Fatal("missing binding installed a shield")
	}
}

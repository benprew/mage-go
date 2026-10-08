package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingRedirectionReferences(t *testing.T) {
	for _, mode := range []string{"permanent", "card-type"} {
		for _, state := range []string{"live", "tap", "untap", "phase", "phase-cycle", "depart", "return", "remove-abilities", "control", "damage", "cleanup"} {
			t.Run(mode+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				rule := RedirectDamageToPermanentIf(You, This, IsArtifact, PermanentIsUntapped(This))
				if mode == "card-type" {
					rule = RedirectDamageFromCardTypeToPermanentIf(You, This, core.TypeArtifact, PermanentIsUntapped(This))
				}
				card := NewCreature("Subject", "{W}", 1, 20, WithAbility(Static(rule)))
				subject := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(subject.ID())
				baseline := g.Clone()
				want := 1
				switch state {
				case "tap":
					g.TapPermanent(subject)
					want = 0
				case "untap":
					g.TapPermanent(subject)
					g.UntapPermanent(subject)
				case "phase":
					g.PhaseOut(subject.ID())
					want = 0
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(subject.ID()))
				case "depart", "return":
					g.RemoveFromBattlefield(subject)
					g.zones.ClearLKI()
					want = 0
					if state == "return" {
						returned := g.PutOnBattlefield(card, b.PlayerID())
						saved = g.CaptureDamageSource(returned.ID())
						want = 1
					}
				case "remove-abilities":
					if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), subject.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = 0
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: subject.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "damage":
					source := g.PutOnBattlefield(NewArtifact("Damage", ""), b.PlayerID())
					g.DealDamageToPlayer(a, 2, source.ID())
					if a.Life() != 20 || g.FindPermanent(subject.ID()).Damage != 2 {
						t.Fatal("redirection failed")
					}
				case "cleanup":
					g.effects.ClearReplacementsEndOfTurn()
				}
				for _, game := range []*Game{g, g.Clone()} {
					got := game.WaitingRedirectionPermanentSources()
					if len(got) != want {
						t.Fatalf("subjects=%d want=%d", len(got), want)
					}
					if want == 1 && (!got[0].SameObject(saved) || got[0].Controller() != game.CaptureDamageSource(saved.ID()).Controller()) {
						t.Fatal("wrong live subject information")
					}
					if len(game.WaitingPreventionPermanentSources()) != 0 {
						t.Fatal("redirection entered prevention category")
					}
				}
				if len(baseline.WaitingRedirectionPermanentSources()) != 1 {
					t.Fatal("mutation changed clone")
				}
			})
		}
	}
}

func TestWaitingRedirectionHostAndStaleCycle(t *testing.T) {
	for _, state := range []string{"live", "reattach", "tap", "source-phase", "host-phase", "phase-cycle", "source-return", "host-return", "control", "noncreature"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "{R}", 1, 20)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "{U}", 1, 20), b.PlayerID())
			card := NewAura("Rule", "", WithAbility(Static(RedirectDamageToPermanentIf(You, AttachedPermanent, AnyPermanent, PermanentIsUntapped(This)))))
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			var saved ReplacementEffect
			for _, r := range g.effects.cycleReplacements {
				if _, ok := r.(*staticDamageRedirectReplacement); ok {
					saved = r.Clone()
					break
				}
			}
			if saved == nil {
				t.Fatal("missing rule")
			}
			expected := host
			switch state {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
				expected = other
			case "tap":
				g.TapPermanent(aura)
				expected = nil
			case "source-phase":
				g.PhaseOut(aura.ID())
				expected = nil
			case "host-phase":
				g.PhaseOut(host.ID())
				expected = nil
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				expected = nil
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, a.PlayerID())
				expected = nil
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "noncreature":
				g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrIsCreature)
				expected = nil
			}
			g.effects.cycleReplacements = []ReplacementEffect{saved}
			got := g.WaitingRedirectionPermanentSources()
			if expected == nil {
				if len(got) != 0 {
					t.Fatal("stale rule exposed absent destination")
				}
			} else if len(got) != 1 || !got[0].SameObject(g.CaptureDamageSource(expected.ID())) {
				t.Fatal("stale rule lost live destination")
			}
		})
	}
}

func TestWaitingRedirectionChoicesAndBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "{R}", 1, 20), b.PlayerID())
	for range 2 {
		aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(RedirectDamageToPermanentIf(You, AttachedPermanent, AnyPermanent, PermanentIsUntapped(This))))), a.PlayerID())
		g.Attach(aura.ID(), host.ID())
	}
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "{U}", 1, 20), a.PlayerID())
	if err := ApplyEffect(g, PreventNextDamage(This, Fixed(1)), unrelated.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingRedirectionPermanentSources()) != 1 {
		t.Fatal("duplicates or unrelated subjects")
	}
	got := g.WaitingRedirectionPermanentSources()
	got[0].card.SetOwner(uuid.New())
	if g.WaitingRedirectionPermanentSources()[0].card.Owner() == got[0].card.Owner() {
		t.Fatal("query shared card data")
	}
	first, second := BindDamageSource("first"), BindDamageSource("second")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, binding := range []*DamageSourceBinding{first, second} {
		effect := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceHasColor(core.Red), binding)
		card := NewArtifact("Choice", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	if err := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceHasColor(core.Blue), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[first].Valid() || !g.resolution.damageSourceBindings[second].SameObject(g.CaptureDamageSource(host.ID())) || !clone.resolution.damageSourceBindings[first].Valid() {
		t.Fatal("bindings or clones shared state")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.zones.ClearLKI()
	if err := ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceFilter{}, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.damageSourceBindings[second].Valid() {
		t.Fatal("empty choice kept value")
	}
	for _, effect := range []Effect{ChooseDamageSourceFromWaitingRedirectionReferences(nil, DamageSourceFilter{}, first), ChooseDamageSourceFromWaitingRedirectionReferences(You, DamageSourceFilter{}, nil)} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

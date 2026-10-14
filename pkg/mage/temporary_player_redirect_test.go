package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type playerRedirectChooser struct {
	*BasePlayer
	accept  bool
	offers  int
	onOffer func()
}

func (p *playerRedirectChooser) ChooseMayAbility(string) bool {
	p.offers++
	if p.onOffer != nil {
		p.onOffer()
	}
	return p.accept
}

func TestTemporaryPlayerRedirectOptional(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(map[bool]string{false: "decline", true: "accept"}[accept], func(t *testing.T) {
			a := &playerRedirectChooser{BasePlayer: NewBasePlayer("A"), accept: accept}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), b.PlayerID())
			effect := MayRedirectDamageFromPermanentsToPlayerThisTurn(IsCreature, You)
			if err := ValidateCardDefinition("Rule", NewInstant("Rule", "", WithAbility(Spell(effect)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.offers != 0 {
				t.Fatal("installation made a choice")
			}
			g.DealSimultaneousDamage(PermanentDamageAssignment(p, 2, g.CaptureDamageSource(source.ID())), PermanentDamageAssignment(source, 3, g.CaptureDamageSource(source.ID())))
			if a.offers != 2 {
				t.Fatalf("offers=%d", a.offers)
			}
			if accept {
				if a.Life() != 15 || p.Damage != 0 || source.Damage != 0 {
					t.Fatal("batch did not redirect")
				}
			} else if a.Life() != 20 || p.Damage != 2 || source.Damage != 3 {
				t.Fatal("decline changed damage")
			}
			clone := g.Clone()
			clone.GetEffects().ClearReplacementsEndOfTurn()
			g.DealDamageToPermanent(p, 1, source.ID())
			clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 1, source.ID())
			if accept && (a.Life() != 14 || clone.FindPermanent(p.ID()).Damage != 1) {
				t.Fatal("clone cleanup shared state")
			}
		})
	}
}

func TestTemporaryPlayerRedirectSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := &StackObject{ID: uuid.New(), Card: NewSorcery("Damage", ""), Controller: b.PlayerID()}
	g.PushStack(spell)

	target := DeclareSpellTarget("spell")
	maker := g.PutOnBattlefield(NewArtifact("Rule", "", WithAbility(Activate(Costs(), RedirectSpellDamageToSourceControllerThisTurn(target), WithTargets(target)))), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), maker.ID(), 0, []uuid.UUID{spell.ID}); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureSpellDamageSource(spell.ID)
	installation := g.StackPeek()
	g.ResolveStackObject(installation)
	g.GetStack().RemoveByID(installation.ID)
	clone := g.Clone()
	g.GetStack().RemoveByID(spell.ID)
	g.zones.ClearLKI()
	for _, game := range []*Game{g, clone} {
		game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 3, saved)
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 17 {
			t.Fatal("saved spell controller")
		}
		other := cloneDamageSource(saved)
		other.stackID = uuid.New()
		game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, other)
		if game.GetPlayer(a.PlayerID()).Life() != 19 {
			t.Fatal("different spell inherited rule")
		}
		if len(game.WaitingRedirectionSpellSources()) != 1 {
			t.Fatal("waiting spell identity missing")
		}
	}
}

func TestTemporaryPlayerRedirectRecipientState(t *testing.T) {
	for _, state := range []string{"live", "late-entry", "phase", "noncreature", "return", "blocked", "zero", "decline", "control", "player-prevention"} {
		t.Run(state, func(t *testing.T) {
			a := &playerRedirectChooser{BasePlayer: NewBasePlayer("A"), accept: state != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), b.PlayerID())
			if err := ApplyEffect(g, MayRedirectDamageFromPermanentsToPlayerThisTurn(IsCreature, You), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			amount := 2
			switch state {
			case "late-entry":
				p = g.PutOnBattlefield(NewCreature("Later", "", 1, 20), b.PlayerID())
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(p.Card, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "noncreature":
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrIsCreature)
			case "blocked":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			case "zero":
				amount = 0
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "player-prevention":
				g.AddPreventionShield(a.PlayerID(), 1)
			}
			g.DealDamageToPermanent(p, amount, source.ID())
			want := 18
			if state == "phase" || state == "noncreature" || state == "blocked" || state == "zero" || state == "decline" {
				want = 20
			}
			if state == "player-prevention" {
				want = 19
			}
			if a.Life() != want {
				t.Fatalf("life=%d want %d", a.Life(), want)
			}
			if state == "blocked" && a.offers != 0 {
				t.Fatal("prohibited redirect offered a choice")
			}
			if (state == "live" || state == "return" || state == "late-entry" || state == "control") && p.Damage != 0 {
				t.Fatal("live filter failed")
			}
		})
	}
}

func TestTemporaryPlayerRedirectInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		MayRedirectDamageFromPermanentsToPlayerThisTurn(IsCreature, nil),
		MayRedirectDamageFromPermanentsToPlayerThisTurn(PermanentFilter{}, You),
		RedirectSpellDamageToSourceControllerThisTurn(nil),
		RedirectSpellDamageToSourceControllerThisTurn(ThisSpell),
	} {
		if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(effect)))); err == nil {
			t.Fatal("invalid accepted")
		}
	}
}

type orderedPlayerRedirectChooser struct {
	*playerRedirectChooser
	last bool
}

func (p *orderedPlayerRedirectChooser) ChooseMode(options []string, _ string) int {
	if p.last {
		return len(options) - 1
	}
	return 0
}

func TestTemporaryPlayerRedirectReplacementOrder(t *testing.T) {
	for _, last := range []bool{false, true} {
		t.Run(map[bool]string{false: "prevention-first", true: "redirect-first"}[last], func(t *testing.T) {
			a := &orderedPlayerRedirectChooser{playerRedirectChooser: &playerRedirectChooser{BasePlayer: NewBasePlayer("A"), accept: true}, last: last}
			b := &orderedPlayerRedirectChooser{playerRedirectChooser: &playerRedirectChooser{BasePlayer: NewBasePlayer("B"), accept: true}, last: last}
			g := NewGame(a, b)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20, WithKeyword(core.Lifelink)), b.PlayerID())
			g.AddPreventionShield(victim.ID(), 1)
			if err := ApplyEffect(g, MayRedirectDamageFromPermanentsToPlayerThisTurn(IsCreature, You), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.AddPreventionShield(a.PlayerID(), 1)
			g.DealDamageToPermanent(victim, 3, source.ID())
			wantDamage := 1
			if last {
				wantDamage = 2
			}
			if a.Life() != 20-wantDamage || victim.Damage != 0 || b.Life() != 20+wantDamage {
				t.Fatalf("order/source lost: lives=%d/%d damage=%d", a.Life(), b.Life(), victim.Damage)
			}
		})
	}
}

func TestTemporaryPlayerRedirectChoiceVisit(t *testing.T) {
	a := &playerRedirectChooser{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), b.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20), b.PlayerID())
	if err := ApplyEffect(g, MayRedirectDamageFromPermanentsToPlayerThisTurn(IsCreature, You), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.onOffer = func() { g.RemoveFromBattlefield(p); g.PutOnBattlefield(p.Card, b.PlayerID()) }
	g.DealDamageToPermanent(p, 2, source.ID())
	if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != 0 {
		t.Fatal("returned recipient supplied the old damage visit")
	}
}

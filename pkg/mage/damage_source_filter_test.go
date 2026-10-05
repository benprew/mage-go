package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSourceFilterIncarnation(t *testing.T) {
	for _, recipient := range []string{"permanent", "player", "source-only"} {
		for _, state := range []string{"live", "departed", "returned", "returned-departed", "phased", "phase-cycle"} {
			t.Run(recipient+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "{R}", 1, 4)
				p := g.PutOnBattlefield(card, a.PlayerID())
				source := g.CaptureDamageSource(p.ID())
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
				filter := NewPermanentFilter("original source", func(p *Permanent, _ *Game) bool {
					return p.HasType(TypeCreature) && p.HasAttr(AttrIsArtifact) && p.ControllerID() == a.PlayerID() && p.Colors()[0] == Blue
				})
				p.GrantBaseAttr(AttrIsArtifact)
				g.AddContinuousEffect(BecomesColor(p.ID(), Blue, EndOfTurn))
				switch state {
				case "departed", "returned", "returned-departed":
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					if state != "departed" {
						q := g.PutOnBattlefield(card, b.PlayerID())
						if state == "returned-departed" {
							g.RemoveFromBattlefield(q)
						}
					}
					g.zones.ClearLKI()
				case "phased":
					g.PhaseOut(p.ID())
					g.ApplyContinuousEffects()
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				rule := &damagePreventionRuleReplacement{from: filter}
				var action Action
				if recipient == "permanent" {
					rule.to = IsID(victim.ID())
					action = NewDamageToCreatureAction(p.ID(), victim.ID(), 1, false).withDamageSource(source)
				} else {
					action = NewDamageToPlayerAction(p.ID(), b.PlayerID(), 1, false).withDamageSource(source)
					if recipient == "player" {
						rule.toPlayerID = b.PlayerID()
					}
				}
				if !rule.Matches(action, g) {
					t.Fatal("filter lost current or retained original source characteristics")
				}
				if g.effects.ApplyReplacements(action, g) == nil {
					t.Fatal("test rule was installed before explicit installation")
				}
				g.effects.AddReplacement(rule)
				if g.effects.ApplyReplacements(action, g) != nil {
					t.Fatal("matching damage escaped prevention")
				}
			})
		}
	}
}

func TestDamageSourceFilterMissingSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
	rule := &damagePreventionRuleReplacement{from: IsCreature, toPlayerID: b.PlayerID()}
	for _, source := range []DamageSource{{id: p.ID(), incarnation: uuid.New()}, {id: uuid.New(), incarnation: p.incarnationID}, {id: p.ID(), card: NewInstant("Spell", "", nil)}} {
		if rule.Matches(NewDamageToPlayerAction(source.ID(), b.PlayerID(), 1, false).withDamageSource(source), g) {
			t.Fatal("missing captured source used another incarnation")
		}
	}
	if !rule.Matches(NewDamageToPlayerAction(p.ID(), b.PlayerID(), 1, false), g) {
		t.Fatal("UUID-only compatibility lost")
	}
}

func TestDamageSourceFilterCopyClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "{R}", 1, 3)
	p := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(p.ID())
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(card, b.PlayerID())
	action := NewDamageToPlayerAction(p.ID(), b.PlayerID(), 1, false).withDamageSource(source)
	rule := &damagePreventionRuleReplacement{from: And(HasColorFilter(Red), NewPermanentFilter("original controller", func(p *Permanent, _ *Game) bool { return p.ControllerID() == a.PlayerID() })), toPlayerID: b.PlayerID(), oneShot: true}
	g.effects.AddReplacement(rule)
	clone := g.Clone()
	if clone.effects.ApplyReplacements(action.WithAmount(2), clone) != nil || rule.consumed {
		t.Fatal("clone consumed original prevention")
	}
	if g.effects.ApplyReplacements(action, g) != nil || !rule.consumed {
		t.Fatal("original source did not consume prevention")
	}
	if g.effects.ApplyReplacements(action, g) == nil {
		t.Fatal("one-shot rule prevented twice")
	}
}

func TestDamageSourceFilterRedirect(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
	source := g.CaptureDamageSource(p.ID())
	g.RemoveFromBattlefield(p)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
	rule := &damagePreventionRuleReplacement{from: IsCreature, to: IsID(victim.ID())}
	redirect := &playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()}
	action := redirect.Replace(NewDamageToPlayerAction(p.ID(), b.PlayerID(), 2, true).withDamageSource(source), g).(*DamageToCreatureAction).WithAmount(1)
	if !rule.Matches(action, g) || action.DamageSource().Incarnation() != source.Incarnation() {
		t.Fatal("redirection or amount change lost source")
	}
	rule.combatOnly = true
	if !rule.Matches(action, g) {
		t.Fatal("combat flag lost")
	}
	if rule.Matches(NewDamageToCreatureAction(p.ID(), victim.ID(), 1, false).withDamageSource(source), g) {
		t.Fatal("combat-only rule prevented noncombat damage")
	}
}

func TestDamageSourceFilterSimultaneousDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	source := g.CaptureDamageSource(p.ID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
	g.effects.AddReplacement(&damagePreventionRuleReplacement{from: IsCreature, to: IsID(victim.ID())})
	g.DealSimultaneousDamage(PermanentDamageAssignment(p, 1, source), PermanentDamageAssignment(victim, 2, source))
	g.CheckStateBasedActions()
	if g.FindPermanent(p.ID()) != nil || g.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("batch prevention or simultaneous source death")
	}
	g.DealDamageToPermanentFromSource(g.FindPermanent(victim.ID()), 2, source)
	if g.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("departed batch source escaped prevention")
	}
}

func TestDamageSourceFilterReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "", 1, 3), a.PlayerID())
		source := g.CaptureDamageSource(p.ID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), b.PlayerID())
		g.RemoveFromBattlefield(p)
		g.effects.AddReplacement(&damagePreventionRuleReplacement{from: IsCreature, to: IsID(victim.ID()), toPlayerID: b.PlayerID()})
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.DealDamageToPermanentFromSource(victim, 2, source)
		if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 || b.prompts != 1 {
			t.Fatalf("choice %d: life=%d damage=%d prompts=%d", choice, b.Life(), g.FindPermanent(victim.ID()).Damage, b.prompts)
		}
	}
}

type sourceFilterChoicePlayer struct {
	*BasePlayer
	choice, prompts int
}

func (p *sourceFilterChoicePlayer) ChooseMode(_ []string, _ string) int { p.prompts++; return p.choice }

func TestDamageSourceFilterCurrentAndRetainedViews(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 1, 3)
	p := g.PutOnBattlefield(card, a.PlayerID())
	source := g.CaptureDamageSource(p.ID())
	action := NewDamageToPlayerAction(p.ID(), b.PlayerID(), 1, false).withDamageSource(source)
	rule := &damagePreventionRuleReplacement{from: NewPermanentFilter("artifact", func(p *Permanent, _ *Game) bool { return p.HasType(TypeArtifact) }), toPlayerID: b.PlayerID()}
	if rule.Matches(action, g) {
		t.Fatal("nonartifact matched")
	}
	p.GrantBaseAttr(AttrIsArtifact)
	if !rule.Matches(action, g) {
		t.Fatal("live filter used announcement characteristics")
	}
	p.RevokeBaseAttr(AttrIsArtifact)
	if rule.Matches(action, g) {
		t.Fatal("live filter ignored type removal")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	returned.GrantBaseAttr(AttrIsArtifact)
	if rule.Matches(action, g) {
		t.Fatal("original nonartifact used returned artifact")
	}
	g.RemoveFromBattlefield(returned)
	if rule.Matches(action, g) {
		t.Fatal("later departure replaced original snapshot")
	}
	snapshot := g.damageFilterSource(p.ID(), source)
	snapshot.GrantBaseAttr(AttrIsArtifact)
	snapshot.Counters[P1P1] = 9
	if g.damageFilterSource(p.ID(), source).HasType(TypeArtifact) || g.damageFilterSource(p.ID(), source).Counters[P1P1] != 0 {
		t.Fatal("filter view changed retained snapshot")
	}
	clone := g.Clone()
	clone.damageFilterSource(p.ID(), source).Counters[P1P1] = 4
	if g.damageFilterSource(p.ID(), source).Counters[P1P1] != 0 {
		t.Fatal("clone changed original snapshot")
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDividedDamageBatchObservers(t *testing.T) {
	for _, activation := range []bool{false, true} {
		t.Run(map[bool]string{false: "spell", true: "activation"}[activation], func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
			a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
				return map[uuid.UUID]int{victim.ID(): 2, b.PlayerID(): 1}
			}
			check := func() {
				if victim.Damage != 2 || b.Life() != 19 {
					t.Errorf("observer saw partial damage: damage=%d life=%d", victim.Damage, b.Life())
				}
			}
			events, callbacks := 0, 0
			g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(EvtDamageDealt, false, DrawCards(Fixed(0))).SetCondition(func(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool {
				events++
				check()
				return false
			}))), a.PlayerID())
			g.damage.SetOnDamageDealt(func(string, string, int, bool) { callbacks++; check() })
			targets := []uuid.UUID{victim.ID(), b.PlayerID()}
			if activation {
				source := g.PutOnBattlefield(NewArtifact("Division source", "", WithActivatedAbility(DealDividedDamage(Fixed(3)), Tap(), WithTarget(TargetUpToNCreaturesOrPlayers(2)))), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
			} else {
				spell := NewInstant("Division spell", "", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(3))))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByName(a.PlayerID(), spell.Name(), targets); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if events != 2 || callbacks != 2 {
				t.Fatalf("events=%d callbacks=%d, want 2 each", events, callbacks)
			}
		})
	}
}

type dividedDamagePreparationProbe struct {
	replacementBase
	check func()
}

func (*dividedDamagePreparationProbe) Matches(action Action, _ GameReader) bool {
	switch action.(type) {
	case *DamageToPlayerAction, *DamageToCreatureAction:
		return true
	}
	return false
}
func (*dividedDamagePreparationProbe) IsActive(GameReader) bool   { return true }
func (r *dividedDamagePreparationProbe) Clone() ReplacementEffect { c := *r; return &c }
func (r *dividedDamagePreparationProbe) Replace(action Action, _ *Game) Action {
	r.check()
	return action
}

func TestDividedDamageBatchPreparesBeforeExecution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
	preparations := 0
	g.effects.AddReplacement(&dividedDamagePreparationProbe{check: func() {
		preparations++
		if b.Life() != 20 || victim.Damage != 0 {
			t.Error("replacement saw an executed share")
		}
	}})
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.AddPreventionShield(victim.ID(), 1)
	events := 0
	g.damage.SetOnDamageDealt(func(string, string, int, bool) {
		events++
		if victim.Damage != 3 || b.Life() != 20 {
			t.Error("redirected batch was partially visible")
		}
	})
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID(), b.PlayerID()}, DamageDistribution: map[uuid.UUID]int{victim.ID(): 2, b.PlayerID(): 2}}
	if err := DealDividedDamage(Fixed(4)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if preparations != 2 || events != 2 || victim.Damage != 3 {
		t.Fatalf("preparations=%d events=%d damage=%d", preparations, events, victim.Damage)
	}
}

func TestDividedDamageBatchSourceLKIAndIsolation(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	card := NewCreature("Source", "", 2, 2, WithKeyword(Lifelink), WithKeyword(Deathtouch), WithActivatedAbility(DealDividedDamage(Fixed(4)), SacrificeSourceCost(), WithTarget(TargetUpToNCreaturesOrPlayers(2))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
	a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{victim.ID(): 2, b.PlayerID(): 2}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	returned := g.PutOnBattlefield(card, b.PlayerID())
	returned.FaceDown = true
	clone := g.Clone()
	copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.AddPreventionShield(b.PlayerID(), 1)
	clone.ResolveStackObject(copied)
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 19 || clone.FindPermanent(victim.ID()) != nil {
		t.Fatal("copied batch lost departure LKI or prevention")
	}
	if a.Life() != 20 || b.Life() != 20 || victim.Damage != 0 {
		t.Fatal("clone changed original state")
	}
	damageEvents, gainEvents := 0, 0
	for _, eventType := range []EventType{EvtDamageDealt, EvtLifeGained} {
		g.PutOnBattlefield(NewEnchantment("LKI observer", "", WithAbility(NewTriggered(eventType, false, DrawCards(Fixed(0))).SetCondition(func(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			current := g.FindPermanent(victim.ID())
			if a.Life() != 24 || b.Life() != 18 || current == nil || current.Damage != 2 || !current.DeathtouchDamage {
				t.Error("observer saw partial damage or lifelink")
			}
			if event.Type == EvtDamageDealt {
				damageEvents++
				if event.SourceIncarnation != source.incarnationID || event.SourceControllerID != a.PlayerID() {
					t.Error("damage event used the returned source")
				}
			} else {
				gainEvents++
			}
			return false
		}))), a.PlayerID())
	}
	g.ResolveStack()
	if damageEvents != 2 || gainEvents != 2 || a.Life() != 24 || b.Life() != 18 || g.FindPermanent(victim.ID()) != nil || g.FindPermanent(returned.ID()) == nil {
		t.Fatal("original batch lost its source incarnation")
	}
}

func TestDividedDamageBatchEmptyAndLegacySource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceID := uuid.New()
	effect := DealDividedDamage(Fixed(4))
	for _, dist := range []map[uuid.UUID]int{nil, {}, {b.PlayerID(): 0}, {b.PlayerID(): -1}, {uuid.New(): 4}} {
		if err := effect.Apply(&EffectContext{Game: g, SourceID: sourceID, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, DamageDistribution: dist}); err != nil {
			t.Fatal(err)
		}
		if b.Life() != 20 {
			t.Fatal("empty instruction dealt damage")
		}
	}
	if err := effect.Apply(&EffectContext{Game: g, SourceID: sourceID, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, DamageDistribution: map[uuid.UUID]int{b.PlayerID(): 4}}); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 16 || !g.HasDealtDamageToPlayer(sourceID, b.PlayerID()) {
		t.Fatal("legacy source ID or damage was lost")
	}
}

func TestDividedDamageBatchIllegalShares(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased", "protected", "prevented"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			card := NewCreature("Recipient", "", 2, 9)
			victim := g.PutOnBattlefield(card, b.PlayerID())
			spell := NewInstant("Division spell", "{R}", NewMultiTargetSpell([]Target{TargetUpToNCreaturesOrPlayers(2)}, DealDividedDamage(Fixed(4))))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
				return map[uuid.UUID]int{victim.ID(): 3, b.PlayerID(): 1}
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(victim)
			case "returned":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(victim.ID())
			case "protected":
				victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(Red))
			case "prevented":
				g.AddPreventionShield(victim.ID(), 3)
			}
			g.ResolveStack()
			if victim.Damage != 0 || b.Life() != 19 || a.divisions != 1 {
				t.Fatalf("lost share changed allocation: damage=%d life=%d prompts=%d", victim.Damage, b.Life(), a.divisions)
			}
		})
	}
}

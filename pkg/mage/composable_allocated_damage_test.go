package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAllocatedDamageAnnouncementAndBatch(t *testing.T) {
	for _, activation := range []bool{false, true} {
		t.Run(map[bool]string{false: "spell", true: "activation"}[activation], func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			recipients := DeclareDamageTargets("recipients", 1, 3)
			effect := DealAllocatedDamage(This, recipients, 3)
			victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
			response := map[uuid.UUID]int{victim.ID(): 2, b.PlayerID(): 1}
			a.ManaPool().Add(Red, 1)
			a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				if total != 3 || len(ids) != 2 || a.ManaPool().Count(Red) != 1 || g.StackSize() != 0 {
					t.Error("allocation occurred after payment or publication")
				}
				return response
			}
			if activation {
				source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(ManaCostOf("{R}")), effect, WithTargets(recipients)))), a.PlayerID())
				if err := ValidateCardDefinition(source.Card.Name(), source.Card); err != nil {
					t.Fatal(err)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			} else {
				spell := NewInstant("Allocated", "{R}", Spell(effect, WithTargets(recipients)))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
					t.Fatal(err)
				}
				if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			obj := g.StackPeek()
			if a.divisions != 1 || !maps.Equal(obj.DamageDistribution, response) || a.ManaPool().Count(Red) != 0 {
				t.Fatal("allocation was not locked")
			}
			response[b.PlayerID()] = 99
			clone := g.Clone()
			copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			copyObj.DamageDistribution[b.PlayerID()] = 2
			if obj.DamageDistribution[b.PlayerID()] != 1 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] != 1 {
				t.Fatal("allocation maps share state")
			}
			callbacks := 0
			g.damage.SetOnDamageDealt(func(string, string, int, bool) {
				callbacks++
				if g.FindPermanent(victim.ID()).Damage != 2 || b.Life() != 19 {
					t.Errorf("observer saw damage=%d life=%d dist=%v", g.FindPermanent(victim.ID()).Damage, b.Life(), obj.DamageDistribution)
				}
			})
			g.ResolveStack()
			if callbacks != 2 || b.Life() != 19 || g.FindPermanent(victim.ID()).Damage != 2 {
				t.Fatal("allocated shares were not dealt")
			}
		})
	}
}

func TestAllocatedDamageDeclarations(t *testing.T) {
	recipients := DeclareDamageTargets("recipients", 1, 3)
	extra := DeclarePlayerTarget("extra")
	valid := DealAllocatedDamage(This, recipients, 3)
	four := DeclareDamageTargets("four", 4, 4)
	empty := &DamageTargetsBinding{}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "direct spell", Card: NewInstant("Valid", "", Spell(valid, WithTargets(recipients)))},
		{Name: "missing declaration", Card: NewInstant("Invalid", "", Spell(valid)), WantErr: true},
		{Name: "extra group", Card: NewInstant("Invalid", "", Spell(valid, WithTargets(recipients, extra))), WantErr: true},
		{Name: "nested", Card: NewInstant("Invalid", "", Spell(Sequence(valid, GainLifeFor(You, Fixed(1))), WithTargets(recipients))), WantErr: true},
		{Name: "trigger", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(Enters(This), valid, WithTargets(recipients)))), WantErr: true},
		{Name: "modal", Card: NewInstant("Invalid", "", WithAbility(SpellModes(ModeOf("Allocate", valid, WithTargets(recipients)), ModeOf("Gain", GainLifeFor(You, Fixed(1)))))), WantErr: true},
		{Name: "impossible minimum", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamage(This, four, 3), WithTargets(four))), WantErr: true},
		{Name: "zero", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamage(This, recipients, 0), WithTargets(recipients))), WantErr: true},
		{Name: "negative", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamage(This, recipients, -1), WithTargets(recipients))), WantErr: true},
		{Name: "nil source", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamage(nil, recipients, 3), WithTargets(recipients))), WantErr: true},
		{Name: "empty recipient declaration", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamage(This, empty, 3), WithTargets(empty))), WantErr: true},
		{Name: "nil recipients", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamage(This, nil, 3))), WantErr: true},
	})
}

func TestAllocatedDamageSourceLKIAndIsolation(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 1, 2)
	card := NewCreature("Source", "", 2, 2, WithKeyword(Lifelink), WithKeyword(Deathtouch), WithAbility(Activate(Costs(SacrificeSourceCost()), DealAllocatedDamage(This, recipients, 4), WithTargets(recipients))))
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

func TestAllocatedDamageIllegalShares(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased", "protected", "prevented"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			card := NewCreature("Recipient", "", 2, 9)
			victim := g.PutOnBattlefield(card, b.PlayerID())
			recipients := DeclareDamageTargets("recipients", 1, 2)
			spell := NewInstant("Division spell", "{R}", Spell(DealAllocatedDamage(This, recipients, 4), WithTargets(recipients)))
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

func TestAllocatedDamagePaymentAndTargetRollback(t *testing.T) {
	for _, state := range []string{"insufficient shares", "failed payment", "absent", "phased", "duplicate", "wrong type"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			recipients := DeclareDamageTargets("recipients", 1, 3)
			total := 3
			if state == "insufficient shares" {
				total = 1
			}
			spell := NewInstant("Allocated", "{R}", Spell(DealAllocatedDamage(This, recipients, total), WithTargets(recipients)))
			if state == "failed payment" {
				WithAdditionalCost(LifePayCost(11))(spell)
				WithAdditionalCost(LifePayCost(11))(spell)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
			targets := []uuid.UUID{victim.ID(), b.PlayerID()}
			switch state {
			case "absent":
				targets[0] = uuid.New()
			case "phased":
				g.PhaseOut(victim.ID())
			case "duplicate":
				targets[0] = b.PlayerID()
			case "wrong type":
				targets[0] = g.PutOnBattlefield(NewArtifact("Other", ""), b.PlayerID()).ID()
			}
			a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				return sanitizeDamageDistribution(nil, ids, total)
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), targets); err == nil {
				t.Fatal("invalid announcement or payment accepted")
			}
			prompts := 0
			if state == "failed payment" {
				prompts = 1
			}
			if a.divisions != prompts || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed payment or stack")
			}
		})
	}
}

func TestAllocatedDamageEmptyAndAbsentSource(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 0, 3)
	spell := NewInstant("Allocated", "", Spell(DealAllocatedDamage(This, recipients, 3), WithTargets(recipients)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if a.divisions != 0 || len(g.StackPeek().DamageDistribution) != 0 {
		t.Fatal("empty allocation prompted")
	}
	g.ResolveStack()
	effect := DealAllocatedDamage(This, recipients, 3)
	ctx := &EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID(), DamageDistribution: map[uuid.UUID]int{b.PlayerID(): 3}, targetBindings: map[*targetDeclaration][]capturedPermanent{recipients.targetDeclaration: {{id: b.PlayerID()}}}}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if b.Life() != 20 {
		t.Fatal("absent source dealt damage")
	}
}

func TestAllocatedDamageReplacementBatch(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Allocated", "", Spell(DealAllocatedDamage(This, recipients, 4), WithTargets(recipients)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
	a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{victim.ID(): 2, b.PlayerID(): 2}
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	preparations := 0
	g.effects.AddReplacement(&dividedDamagePreparationProbe{check: func() {
		preparations++
		if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
			t.Error("replacement observed an executed share")
		}
	}})
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.AddPreventionShield(victim.ID(), 1)
	callbacks := 0
	g.damage.SetOnDamageDealt(func(string, string, int, bool) {
		callbacks++
		if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 3 {
			t.Error("redirected batch was partially visible")
		}
	})
	g.ResolveStack()
	if preparations != 2 || callbacks != 2 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 3 {
		t.Fatal("replacement or finite prevention did not use one batch")
	}
}

func TestAllocatedDamageCopyRetarget(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Allocated", "", Spell(DealAllocatedDamage(This, recipients, 4), WithTargets(recipients)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 9), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 9), b.PlayerID())
	a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{first.ID(): 3, b.PlayerID(): 1}
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{second.ID(), a.PlayerID()}}
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if copied.DamageDistribution[second.ID()] != 3 || copied.DamageDistribution[a.PlayerID()] != 1 || original.DamageDistribution[first.ID()] != 3 {
		t.Fatal("retargeted copy changed shares")
	}
	g.ResolveStackObject(copied)
	if g.FindPermanent(first.ID()).Damage != 0 || g.FindPermanent(second.ID()).Damage != 3 || a.Life() != 19 || b.Life() != 20 || a.divisions != 1 {
		t.Fatal("copy used original recipients or reannounced shares")
	}
}

func TestAllocatedDamageAdditionalCastRoutes(t *testing.T) {
	for _, route := range []string{"alternate", "prepared", "paradigm"} {
		t.Run(route, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			recipients := DeclareDamageTargets("recipients", 1, 2)
			spell := NewInstant("Allocated route", "", Spell(DealAllocatedDamage(This, recipients, 3), WithTargets(recipients)))
			spell.SetOwner(a.PlayerID())
			a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				if total != 3 || len(ids) != 2 || g.StackSize() != 0 {
					t.Error("allocation occurred after admission")
				}
				return map[uuid.UUID]int{a.PlayerID(): 1, b.PlayerID(): 2}
			}
			if route == "alternate" {
				a.AddToHand(spell)
				if err := g.CastCardFromZoneWithoutPaying(a.PlayerID(), spell.ID(), ZoneHand, []uuid.UUID{a.PlayerID(), b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				cast, _ := setupCopyDivisionRoute(g, a, spell, route)
				a.chooseQueue = [][]uuid.UUID{{a.PlayerID(), b.PlayerID()}}
				if err := cast(); err != nil {
					t.Fatal(err)
				}
			}
			if a.divisions != 1 || g.StackPeek().DamageDistribution[b.PlayerID()] != 2 {
				t.Fatal("cast route omitted allocation")
			}
			g.ResolveStack()
			if a.Life() != 19 || b.Life() != 18 {
				t.Fatal("cast route lost allocation")
			}
		})
	}
}

package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAllocatedDamageXAnnouncement(t *testing.T) {
	for _, route := range []string{"hand", "alternate", "activation"} {
		for _, x := range []int{0, 1, 4} {
			t.Run(route+string(rune('0'+x)), func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				recipients := DeclareUpToXDamageTargets("recipients")
				body := Sequence(DealAllocatedDamageX(This, recipients), GainLifeFor(You, Fixed(1)))
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
				ids := []uuid.UUID{victim.ID(), b.PlayerID()}
				if x < 2 {
					ids = ids[:x]
				}
				response := map[uuid.UUID]int{}
				if x > 0 {
					response[victim.ID()] = x
				}
				if x > 1 {
					response[victim.ID()] = x - 1
					response[b.PlayerID()] = 1
				}
				a.ManaPool().Add(Colorless, x)
				a.choose = func(targets []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
					if total != x || len(targets) != len(ids) || a.ManaPool().TotalMana() != x || g.StackSize() != 0 {
						t.Errorf("wrong announcement: total=%d targets=%v", total, targets)
					}
					return response
				}
				g.SetXValue(99)
				var err error
				if route == "activation" {
					source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(ManaPayment("{X}")), body, WithTargets(recipients)))), a.PlayerID())
					g.SetXValue(x)
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids)
				} else {
					spell := NewInstant("Allocated X", "{X}", Spell(body, WithTargets(recipients)))
					spell.SetOwner(a.PlayerID())
					if route == "alternate" {
						a.AddToGraveyard(spell)
						err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneGraveyard, ParseManaCost("{X}"), ids, x)
					} else {
						a.AddToHand(spell)
						err = g.CastSpellByName(a.PlayerID(), spell.Name(), ids, x)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				if obj.XValue != x || !maps.Equal(obj.DamageDistribution, response) || a.ManaPool().TotalMana() != 0 {
					t.Fatal("X or allocation was not locked")
				}
				response[b.PlayerID()] = 99
				clone := g.Clone()
				copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
				if x > 0 {
					copied.DamageDistribution[b.PlayerID()] = 7
				}
				if obj.DamageDistribution[b.PlayerID()] == 99 || obj.DamageDistribution[b.PlayerID()] == 7 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] == 7 {
					t.Fatal("allocation map is shared")
				}
				g.SetXValue(77)
				callbacks := 0
				g.damage.SetOnDamageDealt(func(string, string, int, bool) {
					callbacks++
					if g.FindPermanent(victim.ID()).Damage != max(0, x-1) && x != 1 {
						t.Error("observer saw partial batch")
					}
					if x > 1 && b.Life() != 19 {
						t.Error("observer saw partial player damage")
					}
				})
				g.ResolveStack()
				wantDamage := x
				if x > 1 {
					wantDamage--
				}
				if g.FindPermanent(victim.ID()).Damage != wantDamage || a.Life() != 21 || callbacks != len(ids) {
					t.Fatalf("wrong resolution: damage=%d life=%d callbacks=%d", g.FindPermanent(victim.ID()).Damage, a.Life(), callbacks)
				}
				wantPrompts := 1
				if x == 0 {
					wantPrompts = 0
				}
				if a.divisions != wantPrompts {
					t.Fatal("wrong prompt count")
				}
			})
		}
	}
}

func TestAllocatedDamageXAdmissionRollback(t *testing.T) {
	for _, activation := range []bool{false, true} {
		for _, state := range []string{"negative", "zero", "insufficient", "unpaid", "duplicate", "phased"} {
			t.Run(state+map[bool]string{false: " spell", true: " activation"}[activation], func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				recipients := DeclareDamageTargets("recipients", 1, 2)
				body := DealAllocatedDamageX(This, recipients)
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
				ids := []uuid.UUID{victim.ID(), b.PlayerID()}
				x := 4
				switch state {
				case "negative":
					x = -1
				case "zero":
					x = 0
				case "insufficient":
					x = 1
				case "duplicate":
					ids[1] = victim.ID()
				case "phased":
					g.PhaseOut(victim.ID())
				}
				a.ManaPool().Add(Red, 1)
				a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
					return sanitizeDamageDistribution(nil, ids, total)
				}
				costs := []Cost{ManaPayment("{R}"), LifePayCost(1)}
				if state == "unpaid" {
					costs = append(costs, LifePayCost(20))
				}
				var source *Permanent
				var err error
				if activation {
					source = g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(append(costs, TapCost(This))...), body, WithTargets(recipients)))), a.PlayerID())
					g.SetXValue(x)
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids)
				} else {
					spell := NewInstant("Failed X", "{R}", Spell(body, WithTargets(recipients)))
					for _, cost := range costs[1:] {
						WithAdditionalCost(cost)(spell)
					}
					spell.SetOwner(a.PlayerID())
					a.AddToHand(spell)
					err = g.CastSpellByName(a.PlayerID(), spell.Name(), ids, x)
				}
				if err == nil {
					t.Fatal("invalid announcement accepted")
				}
				if g.StackSize() != 0 || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || (source != nil && source.Tapped) || (!activation && len(a.Hand()) != 1) {
					t.Fatal("failed announcement spent resources")
				}
				want := 0
				if state == "unpaid" {
					want = 1
				}
				if a.divisions != want {
					t.Fatal("allocation prompted at wrong boundary")
				}
			})
		}
	}
}

func TestAllocatedDamageXDeclaration(t *testing.T) {
	recipients := DeclareUpToXDamageTargets("recipients")
	effect := DealAllocatedDamageX(This, recipients)
	extra := DeclarePlayerTarget("extra")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "spell", Card: NewInstant("Valid", "{X}", Spell(effect, WithTargets(recipients)))},
		{Name: "mode", Card: NewInstant("Valid", "{X}", WithAbility(SpellModes(ModeOf("Damage", effect, WithTargets(recipients)), ModeOf("Life", GainLifeFor(You, Fixed(1))))))},
		{Name: "missing targets", Card: NewInstant("Invalid", "", Spell(effect)), WantErr: true},
		{Name: "nil source", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamageX(nil, recipients), WithTargets(recipients))), WantErr: true},
		{Name: "nil recipients", Card: NewInstant("Invalid", "", Spell(DealAllocatedDamageX(This, nil))), WantErr: true},
		{Name: "extra group", Card: NewInstant("Invalid", "", Spell(effect, WithTargets(recipients, extra))), WantErr: true},
		{Name: "two allocations", Card: NewInstant("Invalid", "", Spell(Sequence(effect, DealAllocatedDamage(This, recipients, 3)), WithTargets(recipients))), WantErr: true},
		{Name: "conditional", Card: NewInstant("Invalid", "", Spell(IfState(Controls(IsCreature), effect), WithTargets(recipients))), WantErr: true},
		{Name: "trigger", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(Enters(This), effect, WithTargets(recipients)))), WantErr: true},
	})
}

func TestAllocatedDamageXLKIAndLostShares(t *testing.T) {
	for _, state := range []string{"present", "returned", "phased", "prevented", "protected"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			recipients := DeclareUpToXDamageTargets("recipients")
			card := NewCreature("Source", "{R}", 2, 2, WithKeyword(Lifelink), WithAbility(Activate(Costs(SacrificeSourceCost()), DealAllocatedDamageX(This, recipients), WithTargets(recipients))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 9)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				if total != 4 {
					t.Fatal("wrong announced X")
				}
				return map[uuid.UUID]int{victim.ID(): 3, b.PlayerID(): 1}
			}
			g.SetXValue(4)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
			switch state {
			case "returned":
				g.RemoveFromBattlefield(victim)
				g.PutOnBattlefield(victimCard, b.PlayerID())
			case "phased":
				g.PhaseOut(victim.ID())
			case "prevented":
				g.AddPreventionShield(victim.ID(), 3)
			case "protected":
				victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(Red))
			}
			clone := g.Clone()
			clone.SetXValue(2)
			copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
			clone.AddPreventionShield(b.PlayerID(), 1)
			clone.ResolveStackObject(copied)
			wantDamage := 0
			if state == "present" {
				wantDamage = 3
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20+wantDamage || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("copy lost source LKI or prevention")
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.SetXValue(99)
			g.ResolveStack()
			if state == "phased" {
				g.PhaseIn([]uuid.UUID{victim.ID()})
			}
			if a.Life() != 21+wantDamage || b.Life() != 19 || g.FindPermanent(victim.ID()).Damage != wantDamage {
				t.Fatalf("wrong retained shares: lifeA=%d lifeB=%d damage=%d", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
			}
		})
	}
}

func TestAllocatedDamageXSelectedMode(t *testing.T) {
	for _, activation := range []bool{false, true} {
		for mode := range 2 {
			a := &allocatedModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}, mode: mode}
			b := newRecordingPlayer("B")
			g := NewGame(a, b)
			recipients := DeclareDamageTargets("recipients", 1, 3)
			modes := []ModeDefinition{ModeOf("Damage", DealAllocatedDamageX(This, recipients), WithTargets(recipients)), ModeOf("Life", GainLifeFor(You, Fixed(2)))}
			ids := []uuid.UUID{b.PlayerID()}
			if mode == 1 {
				ids = nil
			}
			a.chooseQueue = [][]uuid.UUID{ids}
			a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				if total != 3 || mode != 0 {
					t.Fatal("inactive or incorrect allocation")
				}
				return map[uuid.UUID]int{b.PlayerID(): 3}
			}
			a.ManaPool().Add(Colorless, 3)
			var err error
			if activation {
				source := g.PutOnBattlefield(NewArtifact("Modes", "", WithAbility(ActivateModes(Costs(ManaPayment("{X}")), modes[0], modes[1]))), a.PlayerID())
				g.SetXValue(3)
				err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids)
			} else {
				spell := NewInstant("Modes", "{X}", WithAbility(SpellModes(modes...)))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				g.SetXValue(77)
				err = g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 3)
			}
			if err != nil {
				t.Fatal(err)
			}
			g.SetXValue(99)
			g.ResolveStack()
			if (mode == 0 && (b.Life() != 17 || a.divisions != 1)) || (mode == 1 && (a.Life() != 22 || b.Life() != 20 || a.divisions != 0)) {
				t.Fatal("wrong selected mode")
			}
		}
	}
}

func TestAllocatedDamageXCopyRetargetAndInterleave(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareUpToXDamageTargets("recipients")
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 9), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 9), b.PlayerID())
	a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{ids[0]: total - 1, ids[1]: 1}
	}
	var firstSpell Card
	for _, x := range []int{4, 2} {
		spell := NewInstant("Interleaved X", "", Spell(DealAllocatedDamageX(This, recipients), WithTargets(recipients)))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), b.PlayerID()}, x); err != nil {
			t.Fatal(err)
		}
		if x == 4 {
			firstSpell = spell
		}
	}
	original := g.StackObjects()[0]
	a.chooseQueue = [][]uuid.UUID{{second.ID(), a.PlayerID()}}
	copied := g.CopySpellOnStack(firstSpell.ID(), a.PlayerID(), true)
	if copied.XValue != 4 || copied.DamageDistribution[second.ID()] != 3 || copied.DamageDistribution[a.PlayerID()] != 1 || original.DamageDistribution[first.ID()] != 3 {
		t.Fatal("retargeting lost X or shares")
	}
	g.SetXValue(77)
	g.ResolveStackObject(copied)
	if g.FindPermanent(second.ID()).Damage != 3 || g.FindPermanent(first.ID()).Damage != 0 || a.Life() != 19 || b.Life() != 20 || a.divisions != 2 {
		t.Fatal("copy reannounced or used another frame")
	}
}

func TestAllocatedDamageXCopyCastAdmission(t *testing.T) {
	for _, route := range []string{"prepared", "paradigm"} {
		g, a, b := newActivationDivisionGame()
		recipients := DeclareUpToXDamageTargets("recipients")
		spell := NewInstant("X copy route", "", Spell(Sequence(DealAllocatedDamageX(This, recipients), GainLifeFor(You, Fixed(1))), WithTargets(recipients)))
		cast, _ := setupCopyDivisionRoute(g, a, spell, route)
		x := 0
		if route == "prepared" {
			x = 3
		}
		g.SetXValue(3)
		ids := []uuid.UUID{b.PlayerID()}
		if x == 0 {
			ids = nil
		}
		a.chooseQueue = [][]uuid.UUID{ids}
		a.choose = func(_ []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
			if total != x {
				t.Fatal("wrong copy-cast X")
			}
			return map[uuid.UUID]int{b.PlayerID(): total}
		}
		if err := cast(); err != nil {
			t.Fatal(err)
		}
		if g.StackPeek().XValue != x {
			t.Fatal("wrong stored X")
		}
		g.SetXValue(99)
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != 20-x {
			t.Fatal("copy cast lost allocation")
		}
	}
}

func TestAllocatedDamageXReplacementBatch(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 1, 2)
	spell := NewInstant("Allocated", "", Spell(DealAllocatedDamageX(This, recipients), WithTargets(recipients)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
	a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{victim.ID(): 2, b.PlayerID(): 2}
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}, 4); err != nil {
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

func TestAllocatedDamageXEmptyAndAbsent(t *testing.T) {
	for _, x := range []int{0, 4} {
		g, a, _ := newActivationDivisionGame()
		recipients := DeclareUpToXDamageTargets("recipients")
		spell := NewInstant("Empty X", "", Spell(Sequence(DealAllocatedDamageX(This, recipients), GainLifeFor(You, Fixed(1))), WithTargets(recipients)))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		g.SetXValue(99)
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, x); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 21 || a.divisions != 0 {
			t.Fatal("empty allocation did not continue")
		}
	}
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 1, 1)
	effect := DealAllocatedDamageX(This, recipients)
	ctx := &EffectContext{Game: g, SourceID: uuid.New(), Controller: a.PlayerID(), DamageDistribution: map[uuid.UUID]int{b.PlayerID(): 4}, targetBindings: map[*targetDeclaration][]capturedPermanent{recipients.targetDeclaration: {{id: b.PlayerID()}}}}
	if err := effect.Apply(ctx); err != nil || b.Life() != 20 {
		t.Fatal("absent source did not do nothing")
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestEvenDamageLegalRecipients(t *testing.T) {
	for _, state := range []string{"present", "absent", "returned", "phased", "protected", "prevented"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			spell := NewInstant("Even damage", "{R}", Spell(DealDamageEvenly(This, targets, Fixed(7)), WithTargets(targets)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Red, 1)
			victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), b.PlayerID())
			walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 20), b.PlayerID())
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), walker.ID(), b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			wantShare, wantVictim := 2, 2
			switch state {
			case "absent":
				g.RemoveFromBattlefield(victim)
				wantShare, wantVictim = 3, 0
			case "returned":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victim.Card, b.PlayerID())
				wantShare, wantVictim = 3, 0
			case "phased":
				g.PhaseOut(victim.ID())
				wantShare, wantVictim = 3, 0
			case "protected":
				victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(Red))
				wantShare, wantVictim = 3, 0
			case "prevented":
				g.AddPreventionShield(victim.ID(), 20)
				wantVictim = 0
			}
			g.ResolveStack()
			if victim.Damage != wantVictim || int(walker.Counters[Loyalty]) != 20-wantShare || b.Life() != 20-wantShare || a.divisions != 0 {
				t.Fatalf("damage=%d loyalty=%d life=%d prompts=%d", victim.Damage, walker.Counters[Loyalty], b.Life(), a.divisions)
			}
		})
	}
}

func TestEvenDamageEmptyZeroAndAllIllegal(t *testing.T) {
	for _, state := range []string{"empty", "zero", "negative", "too many", "all illegal"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			total := 1
			if state == "zero" {
				total = 0
			}
			if state == "negative" {
				total = -1
			}
			victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), b.PlayerID())
			ids := []uuid.UUID{victim.ID(), b.PlayerID()}
			if state == "empty" {
				ids = nil
			}
			if state == "all illegal" {
				ids = ids[:1]
			}
			spell := NewInstant("No damage", "", Spell(Sequence(DealDamageEvenly(This, targets, Fixed(total)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 0); err != nil {
				t.Fatal(err)
			}
			if state == "all illegal" {
				g.PhaseOut(victim.ID())
			}
			preparations := 0
			g.effects.AddReplacement(&dividedDamagePreparationProbe{check: func() { preparations++ }})
			g.ResolveStack()
			wantLife := 21
			if state == "all illegal" {
				wantLife = 20
			}
			if a.Life() != wantLife || b.Life() != 20 || victim.Damage != 0 || preparations != 0 || a.divisions != 0 {
				t.Fatal("empty or zero division caused damage, or all-illegal body ran")
			}
		})
	}
}

func TestEvenDamageBatchAndResults(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	targets := DeclareAnyNumberDamageTargets("recipients")
	total := BindNumber("actual")
	results := BindDamageResults("results")
	item := BindDamageResult("item")
	body := Sequence(DealDamageEvenly(This, targets, Fixed(5), DamageTotalInto(total), DamageResultsInto(results)),
		GainLifeFor(You, FromNumber(total)), ForEachDamageResult(results, item, GainLifeFor(You, Fixed(1))))
	spell := NewInstant("Batch", "", Spell(body, WithTargets(targets)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), b.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	preparations, events := 0, 0
	g.effects.AddReplacement(&dividedDamagePreparationProbe{check: func() {
		preparations++
		if victim.Damage != 0 || b.Life() != 20 {
			t.Error("replacement observed an executed share")
		}
	}})
	g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
	g.AddPreventionShield(victim.ID(), 1)
	g.damage.SetOnDamageDealt(func(string, string, int, bool) {
		events++
		if victim.Damage != 3 || b.Life() != 20 {
			t.Error("observer saw partial batch")
		}
	})
	g.ResolveStack()
	if victim.Damage != 3 || b.Life() != 20 || a.Life() != 25 || preparations != 2 || events != 2 || a.divisions != 0 {
		t.Fatalf("damage=%d life=%d preparations=%d events=%d", victim.Damage, a.Life(), preparations, events)
	}
}

func TestEvenDamageSourceLKIAndCopies(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareAnyNumberDamageTargets("recipients")
	card := NewCreature("Source", "", 1, 2, WithKeyword(Lifelink), WithKeyword(Deathtouch),
		WithAbility(Activate(Costs(SacrificeSourceCost()), DealDamageEvenly(This, targets, Fixed(5)), WithTargets(targets))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(victim.ID())
	clone.ResolveStackObject(copyObj)
	if clone.GetPlayer(a.PlayerID()).Life() != 25 || clone.GetPlayer(b.PlayerID()).Life() != 15 || clone.FindPermanentIncludingPhased(victim.ID()).Damage != 0 {
		t.Fatal("copy did not recalculate division or retain source LKI")
	}
	if a.Life() != 20 || b.Life() != 20 || victim.Damage != 0 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 18 || g.FindPermanent(victim.ID()) != nil {
		t.Fatal("original lost lifelink, deathtouch, or its division")
	}
}

func TestEvenDamageDeclarations(t *testing.T) {
	targets := DeclareAnyNumberDamageTargets("recipients")
	total := BindNumber("total")
	valid := DealDamageEvenly(This, targets, Fixed(5), DamageTotalInto(total))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "spell", Card: NewInstant("Valid", "", Spell(Sequence(valid, GainLifeFor(You, FromNumber(total))), WithTargets(targets)))},
		{Name: "activation", Card: NewArtifact("Valid", "", WithAbility(Activate(Costs(TapCost(This)), valid, WithTargets(targets))))},
		{Name: "trigger", Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(Enters(This), valid, WithTargets(targets))))},
		{Name: "mode", Card: NewInstant("Valid", "", WithAbility(SpellModes(ModeOf("damage", valid, WithTargets(targets)), ModeOf("life", GainLifeFor(You, Fixed(1))))))},
		{Name: "undeclared", Card: NewInstant("Invalid", "", Spell(valid)), WantErr: true},
		{Name: "nil source", Card: NewInstant("Invalid", "", Spell(DealDamageEvenly(nil, targets, Fixed(1)), WithTargets(targets))), WantErr: true},
		{Name: "nil recipients", Card: NewInstant("Invalid", "", Spell(DealDamageEvenly(This, nil, Fixed(1)))), WantErr: true},
		{Name: "empty recipients", Card: NewInstant("Invalid", "", Spell(DealDamageEvenly(This, &DamageTargetsBinding{}, Fixed(1)))), WantErr: true},
		{Name: "nil amount", Card: NewInstant("Invalid", "", Spell(DealDamageEvenly(This, targets, nil), WithTargets(targets))), WantErr: true},
		{Name: "nil option", Card: NewInstant("Invalid", "", Spell(DealDamageEvenly(This, targets, Fixed(1), nil), WithTargets(targets))), WantErr: true},
		{Name: "duplicate result", Card: NewInstant("Invalid", "", Spell(DealDamageEvenly(This, targets, Fixed(1), DamageTotalInto(total), DamageTotalInto(total)), WithTargets(targets))), WantErr: true},
		{Name: "read before write", Card: NewInstant("Invalid", "", Spell(Sequence(GainLifeFor(You, FromNumber(total)), valid), WithTargets(targets))), WantErr: true},
	})
}

func TestEvenDamageGroupAndAmountCapture(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first := DeclareAnyNumberDamageTargets("first")
	second := DeclareAnyNumberDamageTargets("second")
	calls := 0
	spell := NewInstant("Groups", "", Spell(Sequence(
		DealDamageEvenly(This, first, countedLifeAmount{&calls}),
		DealDamageEvenly(This, second, Fixed(3))), WithTargetGroups(first, second)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 30), b.PlayerID())
	if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{{a.PlayerID(), victim.ID()}, {b.PlayerID()}}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if calls != 1 || a.Life() != 10 || victim.Damage != 10 || b.Life() != 17 {
		t.Fatalf("calls=%d life=%d/%d damage=%d", calls, a.Life(), b.Life(), victim.Damage)
	}
}

func TestEvenDamageClearsResults(t *testing.T) {
	for _, state := range []string{"empty", "zero share", "prevented", "absent source", "phased source"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
			targets := DeclareAnyNumberDamageTargets("recipients")
			total, results := BindNumber("total"), BindDamageResults("results")
			effect := DealDamageEvenly(This, targets, Fixed(1), DamageTotalInto(total), DamageResultsInto(results))
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
				targets.targetDeclaration: {{id: b.PlayerID()}},
			}}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.resolution.numberBindings[total] != 1 || len(g.resolution.damageResultsBindings[results]) != 1 {
				t.Fatal("initial results missing")
			}
			switch state {
			case "empty":
				ctx.targetBindings[targets.targetDeclaration] = nil
			case "zero share":
				ctx.targetBindings[targets.targetDeclaration] = []capturedPermanent{{id: a.PlayerID()}, {id: b.PlayerID()}}
			case "prevented":
				g.AddPreventionShield(b.PlayerID(), 1)
			case "absent source":
				ctx.SourceID = uuid.New()
			case "phased source":
				g.PhaseOut(source.ID())
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.resolution.numberBindings[total] != 0 || len(g.resolution.damageResultsBindings[results]) != 0 || b.Life() != 19 {
				t.Fatal("empty action retained earlier results")
			}
		})
	}
}

func TestEvenDamageRetargetAndPayment(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareAnyNumberDamageTargets("recipients")
	body := DealDamageEvenly(This, targets, Fixed(5))
	spell := NewInstant("Retarget", "{R}", Spell(body, WithTargets(targets)), WithAdditionalCost(LifePayCost(2)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 20), b.PlayerID())
	for _, ids := range [][]uuid.UUID{{victim.ID(), b.PlayerID()}, {b.PlayerID(), b.PlayerID()}} {
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 0); err == nil {
			t.Fatal("invalid or unpaid cast accepted")
		}
		if a.Life() != 20 || g.StackSize() != 0 || len(a.Hand()) != 1 {
			t.Fatal("failed cast paid costs")
		}
	}
	a.ManaPool().Add(Red, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{b.PlayerID(), b.PlayerID()}, 0); err == nil {
		t.Fatal("funded duplicate accepted")
	}
	if a.ManaPool().Count(Red) != 1 || a.Life() != 20 {
		t.Fatal("illegal cast paid costs")
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{other.ID(), a.PlayerID()}}
	copyObj := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	g.ResolveStackObject(copyObj)
	if a.Life() != 16 || b.Life() != 20 || other.Damage != 2 || victim.Damage != 0 || original.Targets[0] != victim.ID() {
		t.Fatal("retargeted copy changed original or paid again")
	}
	g.PhaseOut(victim.ID())
	g.ResolveStackObject(original)
	if b.Life() != 15 || a.Life() != 16 || victim.Damage != 0 {
		t.Fatal("original did not divide among its surviving targets")
	}
}

package mage

import (
	"maps"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type allocatedModePlayer struct {
	*activationDivisionPlayer
	mode int
}

func (p *allocatedModePlayer) ChooseMode([]string, string) int { return p.mode }

func allocatedModes() []ModeDefinition {
	first := DeclareDamageTargets("first", 1, 2)
	second := DeclareDamageTargets("second", 1, 2)
	return []ModeDefinition{
		ModeOf("Three", Sequence(GainLifeFor(You, Fixed(1)), Sequence(DealAllocatedDamage(This, first, 3)), GainLifeFor(You, Fixed(2))), WithTargets(first)),
		ModeOf("Four", DealAllocatedDamage(This, second, 4), WithTargets(second)),
		ModeOf("Life", GainLifeFor(You, Fixed(5))),
	}
}

func TestAllocatedDamageSelectedMode(t *testing.T) {
	for _, activation := range []bool{false, true} {
		for mode := range 3 {
			t.Run(map[bool]string{false: "spell", true: "activation"}[activation]+string(rune('0'+mode)), func(t *testing.T) {
				a := &allocatedModePlayer{activationDivisionPlayer: &activationDivisionPlayer{recordingPlayer: newRecordingPlayer("A")}, mode: mode}
				b := newRecordingPlayer("B")
				g := NewGame(a, b)
				a.ManaPool().Add(Red, 1)
				victim := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 9), b.PlayerID())
				modes := allocatedModes()
				total := 3 + mode
				response := map[uuid.UUID]int{victim.ID(): total - 1, b.PlayerID(): 1}
				a.choose = func(ids []uuid.UUID, amount int, _ *Game) map[uuid.UUID]int {
					if mode == 2 || amount != total || len(ids) != 2 || a.ManaPool().Count(Red) != 1 || g.StackSize() != 0 {
						t.Fatalf("mode=%d total=%d ids=%v mana=%d stack=%d", mode, amount, ids, a.ManaPool().Count(Red), g.StackSize())
					}
					return response
				}
				targets := []uuid.UUID{victim.ID(), b.PlayerID()}
				if mode == 2 {
					targets = nil
				}
				a.chooseQueue = [][]uuid.UUID{targets}
				var source *Permanent
				if activation {
					source = g.PutOnBattlefield(NewArtifact("Modal source", "", WithAbility(ActivateModes(Costs(ManaCostOf("{R}")), modes[0], modes[1], modes[2]))), a.PlayerID())
					if err := ValidateCardDefinition(source.Card.Name(), source.Card); err != nil {
						t.Fatal(err)
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
						t.Fatal(err)
					}
				} else {
					card := NewInstant("Modal allocation", "{R}", WithAbility(SpellModes(modes...)))
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					if err := g.CastSpellByName(a.PlayerID(), card.Name(), targets); err != nil {
						t.Fatal(err)
					}
				}
				obj := g.StackPeek()
				if obj.ModeChoice != mode || a.ManaPool().Count(Red) != 0 {
					t.Fatal("selected mode was not locked")
				}
				if mode == 2 {
					if a.divisions != 0 || len(obj.DamageDistribution) != 0 {
						t.Fatal("inactive allocations were announced")
					}
					g.ResolveStack()
					if a.Life() != 25 || b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
						t.Fatal("inactive damage resolved")
					}
					return
				}
				if a.divisions != 1 || !maps.Equal(obj.DamageDistribution, response) {
					t.Fatal("wrong allocation")
				}
				response[b.PlayerID()] = 99
				clone := g.Clone()
				copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
				copied.DamageDistribution[b.PlayerID()] = 2
				if copied.ModeChoice != mode || obj.DamageDistribution[b.PlayerID()] != 1 || clone.StackObjects()[0].DamageDistribution[b.PlayerID()] != 1 {
					t.Fatal("mode or allocation copy isolation failed")
				}
				callbacks := 0
				g.damage.SetOnDamageDealt(func(string, string, int, bool) {
					callbacks++
					if g.FindPermanent(victim.ID()).Damage != total-1 || b.Life() != 19 || (mode == 0 && a.Life() != 21) {
						t.Errorf("damage=%d lifeB=%d lifeA=%d", g.FindPermanent(victim.ID()).Damage, b.Life(), a.Life())
					}
				})
				g.ResolveStack()
				life := 20
				if mode == 0 {
					life = 23
				}
				if callbacks != 2 || g.FindPermanent(victim.ID()).Damage != total-1 || b.Life() != 19 || a.Life() != life {
					t.Fatal("selected mode resolved wrong effects")
				}
			})
		}
	}
}

func TestAllocatedDamageModeDeclarations(t *testing.T) {
	recipients := DeclareDamageTargets("recipients", 1, 3)
	other := DeclareDamageTargets("other", 1, 3)
	four := DeclareDamageTargets("four", 4, 4)
	valid := DealAllocatedDamage(This, recipients, 3)
	gain := ModeOf("Gain", GainLifeFor(You, Fixed(1)))
	cases := []struct {
		name    string
		body    ResolvingEffect
		targets []TargetBinding
	}{
		{"multiple", Sequence(valid, valid), []TargetBinding{recipients}},
		{"legacy sibling", Sequence(valid, DealDividedDamage(Fixed(3))), []TargetBinding{recipients}},
		{"branch", IfState(Controls(IsCreature), valid), []TargetBinding{recipients}},
		{"extra targets", valid, []TargetBinding{recipients, other}},
		{"other mode binding", valid, []TargetBinding{other}},
		{"missing targets", valid, nil},
		{"zero total", DealAllocatedDamage(This, recipients, 0), []TargetBinding{recipients}},
		{"impossible minimum", DealAllocatedDamage(This, four, 3), []TargetBinding{four}},
		{"nil source", DealAllocatedDamage(nil, recipients, 3), []TargetBinding{recipients}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bad := ModeOf("Bad", c.body, WithTargets(c.targets...))
			for _, card := range []Card{
				NewInstant("Bad spell", "", WithAbility(SpellModes(bad, gain))),
				NewArtifact("Bad activation", "", WithAbility(ActivateModes(Costs(ManaCostOf("{1}")), bad, gain))),
			} {
				if err := ValidateCardDefinition(card.Name(), card); err == nil {
					t.Fatal("invalid modal allocation accepted")
				}
			}
		})
	}
	trigger := NewCreature("Bad trigger", "", 1, 1, WithAbility(WhenEventModes(Enters(This), ModeOf("Allocate", valid, WithTargets(recipients)), gain)))
	if err := ValidateCardDefinition(trigger.Name(), trigger); err == nil {
		t.Fatal("triggered allocation accepted")
	}
}

func TestAllocatedDamageModalIdentityAndReplacements(t *testing.T) {
	for _, state := range []string{"absent", "returned", "phased", "protected", "prevented", "redirected"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			recipients := DeclareDamageTargets("recipients", 1, 2)
			sourceCard := NewCreature("Source", "{G}", 2, 2, WithKeyword(Lifelink), WithKeyword(Deathtouch), WithAbility(ActivateModes(Costs(SacrificeSourceCost()),
				ModeOf("Allocate", Sequence(DealAllocatedDamage(This, recipients, 4)), WithTargets(recipients)), ModeOf("Other", GainLifeFor(You, Fixed(9))))))
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			victimCard := NewCreature("Recipient", "", 2, 9)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			a.chooseQueue = [][]uuid.UUID{{victim.ID(), b.PlayerID()}}
			a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
				return map[uuid.UUID]int{victim.ID(): 3, b.PlayerID(): 1}
			}
			if err := ValidateCardDefinition(sourceCard.Name(), sourceCard); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			returned := g.PutOnBattlefield(sourceCard, b.PlayerID())
			returned.FaceDown = true
			switch state {
			case "absent":
				g.RemoveFromBattlefield(victim)
			case "returned":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "phased":
				g.PhaseOut(victim.ID())
			case "protected":
				victim.RuntimeAbilities = append(victim.RuntimeAbilities, ProtectionFromColor(Green))
			case "prevented":
				g.AddPreventionShield(victim.ID(), 3)
			case "redirected":
				preparations := 0
				g.effects.AddReplacement(&dividedDamagePreparationProbe{check: func() {
					preparations++
					if b.Life() != 20 || g.FindPermanent(victim.ID()).Damage != 0 {
						t.Error("replacement saw a partial batch")
					}
				}})
				g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: b.PlayerID(), redirectPermID: victim.ID()})
				g.AddPreventionShield(victim.ID(), 1)
				defer func() {
					if preparations != 2 {
						t.Error("replacement did not prepare both shares")
					}
				}()
			}
			if state != "redirected" {
				clone := g.Clone()
				copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
				clone.ResolveStackObject(copyObj)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			dealt := 1
			if state == "redirected" {
				dealt = 3
			}
			if a.Life() != 20+dealt || (state != "redirected" && b.Life() != 19) || (state == "redirected" && b.Life() != 20) {
				t.Fatal("mode lost departure lifelink or recipient shares")
			}
			if state == "redirected" {
				if g.FindPermanent(victim.ID()) != nil {
					t.Fatal("departure deathtouch did not apply")
				}
			} else if victim.Damage != 0 {
				t.Fatal("illegal or prevented share moved to another recipient")
			}
			if g.FindPermanent(returned.ID()) == nil {
				t.Fatal("damage used returned source")
			}
		})
	}
}

func TestAllocatedDamageModalRollback(t *testing.T) {
	for _, activation := range []bool{false, true} {
		for _, state := range []string{"insufficient shares", "failed payment", "absent", "phased", "duplicate", "wrong type"} {
			t.Run(map[bool]string{false: "spell/", true: "activation/"}[activation]+state, func(t *testing.T) {
				g, a, b := newActivationDivisionGame()
				recipients := DeclareDamageTargets("recipients", 1, 3)
				total := 3
				if state == "insufficient shares" {
					total = 1
				}
				mode := ModeOf("Allocate", DealAllocatedDamage(This, recipients, total), WithTargets(recipients))
				other := ModeOf("Other", GainLifeFor(You, Fixed(9)))
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
				a.ManaPool().Add(Red, 1)
				a.chooseQueue = [][]uuid.UUID{targets}
				a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
					return sanitizeDamageDistribution(nil, ids, total)
				}
				var err error
				if activation {
					costs := Costs(ManaCostOf("{R}"), Tap())
					if state == "failed payment" {
						costs = Costs(ManaCostOf("{R}"), Tap(), LifePayCost(11), LifePayCost(11))
					}
					source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(ActivateModes(costs, mode, other))), a.PlayerID())
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
					if g.FindPermanent(source.ID()).Tapped {
						t.Fatal("failed activation tapped source")
					}
				} else {
					card := NewInstant("Modal rollback", "{R}", WithAbility(SpellModes(mode, other)))
					if state == "failed payment" {
						WithAdditionalCost(LifePayCost(11))(card)
						WithAdditionalCost(LifePayCost(11))(card)
					}
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					err = g.CastSpellByName(a.PlayerID(), card.Name(), targets)
					if len(a.Hand()) != 1 {
						t.Fatal("failed spell left hand")
					}
				}
				prompts := 0
				if state == "failed payment" {
					prompts = 1
				}
				if err == nil || a.divisions != prompts || a.Life() != 20 || a.ManaPool().Count(Red) != 1 || g.StackSize() != 0 {
					t.Fatalf("rollback failed: err=%v prompts=%d life=%d mana=%d stack=%d", err, a.divisions, a.Life(), a.ManaPool().Count(Red), g.StackSize())
				}
			})
		}
	}
}

func TestAllocatedDamageModalCopyRetarget(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	card := NewInstant("Modal copy", "", WithAbility(SpellModes(allocatedModes()...)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 9), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 9), b.PlayerID())
	a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{first.ID(): 2, b.PlayerID(): 1}
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{first.ID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{second.ID(), a.PlayerID()}}
	copied := g.CopySpellOnStack(card.ID(), a.PlayerID(), true)
	if copied.ModeChoice != 0 || copied.DamageDistribution[second.ID()] != 2 || copied.DamageDistribution[a.PlayerID()] != 1 || original.DamageDistribution[first.ID()] != 2 {
		t.Fatal("retargeting changed mode or shares")
	}
	g.ResolveStackObject(copied)
	if g.FindPermanent(first.ID()).Damage != 0 || g.FindPermanent(second.ID()).Damage != 2 || a.Life() != 22 || b.Life() != 20 || a.divisions != 1 {
		t.Fatal("copied mode used old recipients or reannounced allocation")
	}
}

func TestAllocatedDamageModalCastRoutes(t *testing.T) {
	for _, route := range []string{"alternate", "prepared", "paradigm"} {
		t.Run(route, func(t *testing.T) {
			g, a, b := newActivationDivisionGame()
			card := NewInstant("Modal route", "", WithAbility(SpellModes(allocatedModes()...)))
			card.SetOwner(a.PlayerID())
			a.choose = func(ids []uuid.UUID, total int, _ *Game) map[uuid.UUID]int {
				if total != 3 || len(ids) != 2 || g.StackSize() != 0 {
					t.Error("wrong modal allocation timing")
				}
				return map[uuid.UUID]int{a.PlayerID(): 1, b.PlayerID(): 2}
			}
			if route == "alternate" {
				a.AddToHand(card)
				if err := g.CastCardFromZoneWithoutPaying(a.PlayerID(), card.ID(), ZoneHand, []uuid.UUID{a.PlayerID(), b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				cast, _ := setupCopyDivisionRoute(g, a, card, route)
				a.chooseQueue = [][]uuid.UUID{{a.PlayerID(), b.PlayerID()}}
				if err := cast(); err != nil {
					t.Fatal(err)
				}
			}
			if a.divisions != 1 || g.StackPeek().ModeChoice != 0 || g.StackPeek().DamageDistribution[b.PlayerID()] != 2 {
				t.Fatal("route omitted selected-mode allocation")
			}
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 18 {
				t.Fatal("route lost selected mode")
			}
		})
	}
}

func TestAllocatedDamageModalEmptyGroup(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	recipients := DeclareDamageTargets("recipients", 0, 3)
	card := NewInstant("Empty modal allocation", "", WithAbility(SpellModes(
		ModeOf("Allocate", Sequence(DealAllocatedDamage(This, recipients, 3), GainLifeFor(You, Fixed(1))), WithTargets(recipients)),
		ModeOf("Other", GainLifeFor(You, Fixed(9))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.chooseQueue = [][]uuid.UUID{nil}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if a.divisions != 0 || len(g.StackPeek().DamageDistribution) != 0 {
		t.Fatal("empty mode requested shares")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("empty instruction did not continue")
	}
}

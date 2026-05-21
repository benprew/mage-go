// Package gametest: engine-rules tests for Chapter 5 (Turn Structure, CR 500–514).
// Only gap entries from the ch05_turn_structure.md plan are implemented here;
// entries already covered by turn_*_test.go are skipped.
package gametest

import (
	"slices"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ---------------------------------------------------------------------------
// CR 500.8 — Extra phases (G1)
// ---------------------------------------------------------------------------

// TestCR500_8_ExtraPhaseInsertedAfterSpecified verifies CR 500.8.
func TestCR500_8_ExtraPhaseInsertedAfterSpecified(t *testing.T) {
	var beginCombatFires atomic.Int32
	watchName := "CR500.8 BeginCombat Watcher"
	if !mage.CardRegistered(watchName) {
		mage.Register(watchName, func() mage.Card {
			return mage.NewCreature(watchName, "{1}", 1, 1,
				mage.WithSubTypes("Spirit"),
				mage.WithAbility(mage.NewTriggered(core.EvtBeginCombat, false,
					mage.FuncEffect("count begin-combat", mage.EffectProperties{},
						func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
							beginCombatFires.Add(1)
							return nil
						}))))
		})
	}

	beginCombatFires.Store(0)
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, watchName)
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	tg.Schedule.BuildNextTurn()
	tg.AppendExtraStep(core.BeginCombat)
	tg.AppendExtraStep(core.EndCombat)
	for {
		step, ok := tg.Schedule.PopNextStep()
		if !ok {
			break
		}
		tg.Step = step
		tg.RunStepWithPriority(step)
	}
	got := beginCombatFires.Load()
	if got < 2 {
		t.Errorf("CR 500.8: EvtBeginCombat fired %d time(s); want >= 2", got)
	}
}

// ---------------------------------------------------------------------------
// CR 500.9 — Extra steps (G2)
// ---------------------------------------------------------------------------

// TestCR500_9_ExtraStepInsertedAfterSpecified verifies CR 500.9.
func TestCR500_9_ExtraStepInsertedAfterSpecified(t *testing.T) {
	tg := NewTestGame(t)
	tg.padLibraries()
	tg.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
	tg.Schedule.BuildNextTurn()
	tg.InsertStepAfter(core.Upkeep, core.Upkeep)

	var observed []core.PhaseStep
	for {
		step, ok := tg.Schedule.PopNextStep()
		if !ok {
			break
		}
		observed = append(observed, step)
		tg.Step = step
		tg.RunStepWithPriority(step)
	}

	upkeepCount := 0
	for _, s := range observed {
		if s == core.Upkeep {
			upkeepCount++
		}
	}
	if upkeepCount != 2 {
		t.Errorf("CR 500.9: expected 2 Upkeep steps, got %d in %v", upkeepCount, observed)
	}
	drawIdx := -1
	for i, s := range observed {
		if s == core.Draw {
			drawIdx = i
			break
		}
	}
	if drawIdx == -1 {
		t.Fatalf("CR 500.9: Draw step not found in %v", observed)
	}
	count := 0
	for i, s := range observed {
		if s == core.Upkeep {
			count++
			if count == 2 && i >= drawIdx {
				t.Errorf("CR 500.9: extra Upkeep (idx %d) must precede Draw (idx %d)", i, drawIdx)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// CR 505.1b — Phase-count phrasing (G3)
// ---------------------------------------------------------------------------

// TestCR505_1b_PrecombatPostcombatPhaseCount verifies CR 505.1b.
func TestCR505_1b_PrecombatPostcombatPhaseCount(t *testing.T) {
	if !core.PrecombatMain.IsMainPhase() {
		t.Error("CR 505.1b: PrecombatMain must be classified as a main phase")
	}
	if !core.PostcombatMain.IsMainPhase() {
		t.Error("CR 505.1b: PostcombatMain must be classified as a main phase")
	}
	count := 0
	var first, last core.PhaseStep
	for _, s := range core.AllSteps() {
		if s.IsMainPhase() {
			count++
			if count == 1 {
				first = s
			}
			last = s
		}
	}
	if count != 2 {
		t.Errorf("CR 505.1b: expected 2 main phases in AllSteps(), got %d", count)
	}
	if first != core.PrecombatMain {
		t.Errorf("CR 505.1b: first main phase must be PrecombatMain, got %v", first)
	}
	if last != core.PostcombatMain {
		t.Errorf("CR 505.1b: second main phase must be PostcombatMain, got %v", last)
	}
	preIdx, combatIdx, postIdx := -1, -1, -1
	for i, s := range core.AllSteps() {
		if s == core.PrecombatMain && preIdx == -1 {
			preIdx = i
		}
		if s == core.BeginCombat && combatIdx == -1 {
			combatIdx = i
		}
		if s == core.PostcombatMain && postIdx == -1 {
			postIdx = i
		}
	}
	if preIdx >= combatIdx || combatIdx >= postIdx {
		t.Errorf("CR 505.1b: step order wrong: pre=%d combat=%d post=%d", preIdx, combatIdx, postIdx)
	}
}

// ---------------------------------------------------------------------------
// CR 506.3 — Only creatures can attack (G4)
// ---------------------------------------------------------------------------

// TestCR506_3_OnlyCreaturesAttack verifies CR 506.3: a non-creature
// permanent declared as an attacker is silently ignored.
func TestCR506_3_OnlyCreaturesAttack(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	tg.Attack(1, PlayerA, "Forest")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 20)
}

// ---------------------------------------------------------------------------
// CR 506.6 — "Had to attack" requirement (G5)
// ---------------------------------------------------------------------------

// TestCR506_6_HadToAttackRequirementPresent verifies CR 506.6: AttrMustAttack
// creatures are auto-added to the attacker list without an explicit Attack().
func TestCR506_6_HadToAttackRequirementPresent(t *testing.T) {
	const mustAtkName = "CR506.6 Must Attack Bear"
	if !mage.CardRegistered(mustAtkName) {
		mage.Register(mustAtkName, func() mage.Card {
			return mage.NewCreature(mustAtkName, "{1}{R}", 2, 2,
				mage.WithSubTypes("Bear"),
				mage.WithKeyword(core.MustAttack))
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, mustAtkName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 18)
}

// ---------------------------------------------------------------------------
// CR 507.2 — Priority at beginning of combat (G7)
// ---------------------------------------------------------------------------

// TestCR507_2_PriorityAtBeginningOfCombat verifies CR 507.2: the active player
// may cast instants during the beginning-of-combat step.
func TestCR507_2_PriorityAtBeginningOfCombat(t *testing.T) {
	const healName = "CR507.2 BeginCombat Instant"
	if !mage.CardRegistered(healName) {
		mage.Register(healName, func() mage.Card {
			return mage.NewInstant(healName, "{W}",
				mage.NewSpellAbility(mage.GainLife(3)))
		})
	}
	tg := NewTestGame(t)
	tg.SetLife(PlayerA, 17)
	tg.AddCard(core.ZoneHand, PlayerA, healName)
	tg.CastSpell(1, core.BeginCombat, PlayerA, healName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerA, 20)
}

// ---------------------------------------------------------------------------
// CR 508.1c — Restriction prevents illegal declaration (G8)
// ---------------------------------------------------------------------------

// TestCR508_1c_DeclareAttackersRestrictionIllegal verifies CR 508.1c: a creature with
// AttrCanAttack revoked cannot be declared as an attacker.
func TestCR508_1c_DeclareAttackersRestrictionIllegal(t *testing.T) {
	const restrictedName = "CR508.1c Restricted Bear"
	if !mage.CardRegistered(restrictedName) {
		mage.Register(restrictedName, func() mage.Card {
			return mage.NewCreature(restrictedName, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	tg := NewTestGame(t)
	permID := tg.AddCard(core.ZoneBattlefield, PlayerA, restrictedName)
	perm := tg.FindPermanent(permID)
	if perm == nil {
		t.Fatal("permanent not found")
	}
	perm.RevokeBaseAttr(core.AttrCanAttack)
	tg.Attack(1, PlayerA, restrictedName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 20)
}

// ---------------------------------------------------------------------------
// CR 508.1d — Requirement obeyed to maximum extent (G9)
// ---------------------------------------------------------------------------

// TestCR508_1d_DeclareAttackersRequirementMaxObeyed verifies CR 508.1d.
func TestCR508_1d_DeclareAttackersRequirementMaxObeyed(t *testing.T) {
	const reqName = "CR508.1d Must-Attack Bear"
	if !mage.CardRegistered(reqName) {
		mage.Register(reqName, func() mage.Card {
			return mage.NewCreature(reqName, "{1}{R}", 2, 2,
				mage.WithSubTypes("Bear"),
				mage.WithKeyword(core.MustAttack))
		})
	}
	t.Run("requirement alone forces attack", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, reqName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerB, 18)
	})
	t.Run("restriction overrides requirement", func(t *testing.T) {
		tg := NewTestGame(t)
		permID := tg.AddCard(core.ZoneBattlefield, PlayerA, reqName)
		perm := tg.FindPermanent(permID)
		if perm == nil {
			t.Fatal("permanent not found")
		}
		perm.RevokeBaseAttr(core.AttrCanAttack)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerB, 20)
	})
}

// ---------------------------------------------------------------------------
// CR 508.3a — "Whenever [creature] attacks" trigger (G10)
// ---------------------------------------------------------------------------

// TestCR508_3a_WhenAttacksTrigger verifies CR 508.3a: AttacksTrigger
// fires when and only when the source creature is declared as an attacker.
func TestCR508_3a_WhenAttacksTrigger(t *testing.T) {
	const atkTrigName = "CR508.3 Attacks Trigger Bear"
	if !mage.CardRegistered(atkTrigName) {
		mage.Register(atkTrigName, func() mage.Card {
			return mage.NewCreature(atkTrigName, "{1}{R}", 2, 2,
				mage.WithSubTypes("Bear"),
				mage.WithAbility(mage.AttacksTrigger(mage.GainLife(3), false)))
		})
	}
	t.Run("trigger fires when source attacks", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, atkTrigName)
		tg.Attack(1, PlayerA, atkTrigName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerA, 23)
		tg.AssertLife(PlayerB, 18)
	})
	t.Run("trigger does not fire when creature stays home", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, atkTrigName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerA, 20)
	})
	t.Run("trigger does not fire for a different creature attacking", func(t *testing.T) {
		const otherBear = "CR508.3 Plain Bear"
		if !mage.CardRegistered(otherBear) {
			mage.Register(otherBear, func() mage.Card {
				return mage.NewCreature(otherBear, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, atkTrigName)
		tg.AddCard(core.ZoneBattlefield, PlayerA, otherBear)
		tg.Attack(1, PlayerA, otherBear)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerA, 20)
	})
}

// ---------------------------------------------------------------------------
// CR 508.4 — ETB attacking (G11)
// ---------------------------------------------------------------------------

// TestCR508_4_DeclareAttackersETBAttacking verifies CR 508.4: a creature put onto
// the battlefield attacking is treated as an attacking creature (deals combat
// damage) but per CR 508.3a does not cause AttacksTrigger abilities to fire —
// it never "attacked."
func TestCR508_4_DeclareAttackersETBAttacking(t *testing.T) {
	spellName := "CR508.4 Beckon Token"
	if !mage.CardRegistered(spellName) {
		mage.Register(spellName, func() mage.Card {
			return mage.NewInstant(spellName, "{R}",
				mage.NewSpellAbility(mage.CreateTokenAttacking("CR508.4 Marauder", 3, 3,
					[]core.CardType{core.TypeCreature},
					[]string{"Goblin", "Warrior"})))
		})
	}

	triggerName := "CR508.4 Trigger Beast"
	var attacksFires atomic.Int32
	if !mage.CardRegistered(triggerName) {
		mage.Register(triggerName, func() mage.Card {
			return mage.NewCreature(triggerName, "{2}{R}", 2, 2,
				mage.WithSubTypes("Beast"),
				mage.WithAbility(mage.AttacksTrigger(
					mage.FuncEffect("count attacks fires", mage.EffectProperties{},
						func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
							attacksFires.Add(1)
							return nil
						}), false)))
		})
	}
	putAttackingName := "CR508.4 Put Beast Attacking"
	if !mage.CardRegistered(putAttackingName) {
		mage.Register(putAttackingName, func() mage.Card {
			return mage.NewInstant(putAttackingName, "{R}",
				mage.NewSpellAbility(mage.FuncEffect("put beast attacking", mage.EffectProperties{},
					func(g *mage.Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						beast, err := mage.CreateCard(triggerName)
						if err != nil {
							return err
						}
						var defender uuid.UUID
						for _, p := range g.AllPlayers() {
							if p.PlayerID() != controller {
								defender = p.PlayerID()
								break
							}
						}
						g.PutOnBattlefieldAttacking(beast, controller, defender)
						return nil
					})))
		})
	}

	// Part 1: token put into play attacking deals combat damage.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg.AddCard(core.ZoneHand, PlayerA, spellName)
	tg.CastSpell(1, core.BeginCombat, PlayerA, spellName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertPermanentCount(PlayerA, "CR508.4 Marauder", 1)
	tg.AssertLife(PlayerB, 17) // 3 damage from the put-in-play attacker

	// Part 2: AttacksTrigger must NOT fire when a creature with that trigger
	// is put onto the battlefield attacking (CR 508.3a).
	attacksFires.Store(0)
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	tg2.AddCard(core.ZoneHand, PlayerA, putAttackingName)
	tg2.CastSpell(1, core.BeginCombat, PlayerA, putAttackingName)
	tg2.StopAt(1, core.PostcombatMain)
	tg2.Execute()
	tg2.AssertGraveyardCount(PlayerA, putAttackingName, 1)
	tg2.AssertPermanentCount(PlayerA, triggerName, 1)
	tg2.AssertLife(PlayerB, 18) // 2 damage from the 2/2, no trigger damage
	if got := attacksFires.Load(); got != 0 {
		t.Fatalf("CR 508.3a: AttacksTrigger fired %d time(s); must not fire for a creature put into play attacking", got)
	}
}

// ---------------------------------------------------------------------------
// CR 508.5 — Defending player reference (G12)
// ---------------------------------------------------------------------------

// TestCR508_5_DefendingPlayerRef verifies CR 508.5: in a 2-player
// game, the defending player is the opponent.
func TestCR508_5_DefendingPlayerRef(t *testing.T) {
	const drainName = "CR508.5 Attacks Drains Defender"
	if !mage.CardRegistered(drainName) {
		mage.Register(drainName, func() mage.Card {
			return mage.NewCreature(drainName, "{1}{B}", 2, 2,
				mage.WithSubTypes("Vampire"),
				mage.WithAbility(mage.AttacksTrigger(
					mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectEachOpponent()),
					false)))
		})
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, drainName)
	tg.Attack(1, PlayerA, drainName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 16)
	tg.AssertLife(PlayerA, 20)
}

// ---------------------------------------------------------------------------
// CR 509.1b — Flying evasion block restriction (G13)
// ---------------------------------------------------------------------------

// TestCR509_1b_DeclareBlockersEvasionFlyingRestriction verifies CR 509.1b: a ground
// creature cannot block a flying attacker.
func TestCR509_1b_DeclareBlockersEvasionFlyingRestriction(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Air Elemental")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	tg.Attack(1, PlayerA, "Air Elemental")
	tg.Block(1, PlayerB, "Grizzly Bears", "Air Elemental")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 16)
	tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 1)
}

// TestCR509_1b_DeclareBlockersEvasionFlyingRestrictionReach verifies CR 509.1b
// complement: a creature with reach can legally block a flyer.
// Air Elemental (4/4 flying) vs Giant Spider (2/4 reach):
// Spider dies (4 >= 4 toughness); Air Elemental survives (2 < 4 toughness).
func TestCR509_1b_DeclareBlockersEvasionFlyingRestrictionReach(t *testing.T) {
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Air Elemental")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Giant Spider")
	tg.Attack(1, PlayerA, "Air Elemental")
	tg.Block(1, PlayerB, "Giant Spider", "Air Elemental")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertPermanentCount(PlayerB, "Giant Spider", 0)
	tg.AssertPermanentCount(PlayerA, "Air Elemental", 1)
	tg.AssertLife(PlayerB, 20)
}

// ---------------------------------------------------------------------------
// CR 509.1c — Block-if-able requirement (G14)
// ---------------------------------------------------------------------------

// TestCR509_1c_DeclareBlockersBlockRequirementObeyed verifies CR 509.1c / Lure: the
// engine redirects block declarations to the MustBeBlocked attacker.
func TestCR509_1c_DeclareBlockersBlockRequirementObeyed(t *testing.T) {
	const luredAtk = "CR509.1c Lured Attacker"
	const otherAtk = "CR509.1c Other Attacker"
	const blocker = "CR509.1c Blocker"
	if !mage.CardRegistered(luredAtk) {
		mage.Register(luredAtk, func() mage.Card {
			return mage.NewCreature(luredAtk, "{2}{G}", 3, 3, mage.WithSubTypes("Beast"))
		})
	}
	if !mage.CardRegistered(otherAtk) {
		mage.Register(otherAtk, func() mage.Card {
			return mage.NewCreature(otherAtk, "{1}{R}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}
	if !mage.CardRegistered(blocker) {
		mage.Register(blocker, func() mage.Card {
			return mage.NewCreature(blocker, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
		})
	}
	tg := NewTestGame(t)
	luredID := tg.AddCard(core.ZoneBattlefield, PlayerA, luredAtk)
	tg.AddCard(core.ZoneBattlefield, PlayerA, otherAtk)
	tg.AddCard(core.ZoneBattlefield, PlayerB, blocker)
	luredPerm := tg.FindPermanent(luredID)
	if luredPerm == nil {
		t.Fatal("lured permanent not found")
	}
	luredPerm.GrantBaseAttr(core.AttrMustBeBlocked)
	tg.Attack(1, PlayerA, luredAtk, otherAtk)
	tg.Block(1, PlayerB, blocker, otherAtk)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 19)
	tg.AssertGraveyardCount(PlayerB, blocker, 1)
	tg.AssertPermanentCount(PlayerA, luredAtk, 1)
}

// ---------------------------------------------------------------------------
// CR 509.3a — "Whenever [creature] blocks" fires once per combat (G15)
// ---------------------------------------------------------------------------

// TestCR509_3a_BlocksOnceTriggerMultipleAttackers verifies CR 509.3a.
// XXX: engine fires EvtDeclaredBlocker once per pair; BlocksTrigger fires N
// times for N attackers blocked. This test documents the engine gap.
func TestCR509_3a_BlocksOnceTriggerMultipleAttackers(t *testing.T) {
	const multiBlockName = "CR509.3a Multi-Blocker"
	var blockTriggerCount atomic.Int32

	if !mage.CardRegistered(multiBlockName) {
		mage.Register(multiBlockName, func() mage.Card {
			return mage.NewCreature(multiBlockName, "{3}", 1, 5,
				mage.WithSubTypes("Wall"),
				mage.WithKeyword(core.CanBlockAny),
				mage.WithAbility(mage.BlocksTrigger(
					mage.FuncEffect("count blocks", mage.EffectProperties{},
						func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
							blockTriggerCount.Add(1)
							return nil
						}), false)))
		})
	}
	const atk1 = "CR509.3a Attacker A"
	const atk2 = "CR509.3a Attacker B"
	if !mage.CardRegistered(atk1) {
		mage.Register(atk1, func() mage.Card {
			return mage.NewCreature(atk1, "{1}{R}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}
	if !mage.CardRegistered(atk2) {
		mage.Register(atk2, func() mage.Card {
			return mage.NewCreature(atk2, "{1}{R}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}
	blockTriggerCount.Store(0)
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atk1)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atk2)
	tg.AddCard(core.ZoneBattlefield, PlayerB, multiBlockName)
	tg.Attack(1, PlayerA, atk1, atk2)
	tg.Block(1, PlayerB, multiBlockName, atk1)
	tg.Block(1, PlayerB, multiBlockName, atk2)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	got := blockTriggerCount.Load()
	if got != 1 {
		t.Errorf("CR 509.3a: 'whenever blocks' trigger fired %d time(s); CR requires exactly 1 per combat", got)
	}
}

// ---------------------------------------------------------------------------
// CR 509.3b — "Whenever [creature] blocks a creature" — per attacker (G16)
// ---------------------------------------------------------------------------

// TestCR509_3b_BlocksACreatureTrigger verifies CR 509.3b: fires once
// per creature blocked.
func TestCR509_3b_BlocksACreatureTrigger(t *testing.T) {
	const perBlockName = "CR509.3b Per-Attacker Blocker"
	var perBlockCount atomic.Int32

	if !mage.CardRegistered(perBlockName) {
		mage.Register(perBlockName, func() mage.Card {
			return mage.NewCreature(perBlockName, "{3}", 1, 5,
				mage.WithSubTypes("Wall"),
				mage.WithKeyword(core.CanBlockAny),
				mage.WithAbility(
					mage.NewTriggered(core.EvtDeclaredBlocker, false,
						mage.FuncEffect("count per-attacker block", mage.EffectProperties{},
							func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
								perBlockCount.Add(1)
								return nil
							})).
						SetCondition(func(evt *core.GameEvent, _ mage.GameReader, sourceID, _ uuid.UUID) bool {
							return evt.SourceID == sourceID
						})))
		})
	}
	const batk1 = "CR509.3b Att A"
	const batk2 = "CR509.3b Att B"
	if !mage.CardRegistered(batk1) {
		mage.Register(batk1, func() mage.Card {
			return mage.NewCreature(batk1, "{R}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}
	if !mage.CardRegistered(batk2) {
		mage.Register(batk2, func() mage.Card {
			return mage.NewCreature(batk2, "{R}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}
	perBlockCount.Store(0)
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, batk1)
	tg.AddCard(core.ZoneBattlefield, PlayerA, batk2)
	tg.AddCard(core.ZoneBattlefield, PlayerB, perBlockName)
	tg.Attack(1, PlayerA, batk1, batk2)
	tg.Block(1, PlayerB, perBlockName, batk1)
	tg.Block(1, PlayerB, perBlockName, batk2)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	if got := perBlockCount.Load(); got != 2 {
		t.Errorf("CR 509.3b: per-attacker block trigger fired %d time(s); want 2", got)
	}
}

// ---------------------------------------------------------------------------
// CR 509.3c — "Becomes blocked" fires once per combat (G17)
// ---------------------------------------------------------------------------

// TestCR509_3c_BecomesBlockedOnce verifies CR 509.3c: fires exactly
// once per combat even when two creatures block.
func TestCR509_3c_BecomesBlockedOnce(t *testing.T) {
	const becomeBlockedAtkName = "CR509.3c Becomes-Blocked Attacker"
	var becomeBlockedCount atomic.Int32

	if !mage.CardRegistered(becomeBlockedAtkName) {
		mage.Register(becomeBlockedAtkName, func() mage.Card {
			return mage.NewCreature(becomeBlockedAtkName, "{2}{G}", 4, 4,
				mage.WithSubTypes("Beast"),
				mage.WithAbility(
					mage.NewTriggered(core.EvtBlockersDecl, false,
						mage.FuncEffect("count becomes-blocked", mage.EffectProperties{},
							func(gr *mage.Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
								cg := gr.CombatGroupFor(sourceID)
								if cg != nil && len(cg.BlockerIDs) > 0 {
									becomeBlockedCount.Add(1)
								}
								return nil
							}),
					),
				),
			)
		})
	}
	const blkA = "CR509.3c Blocker A"
	const blkB = "CR509.3c Blocker B"
	if !mage.CardRegistered(blkA) {
		mage.Register(blkA, func() mage.Card {
			return mage.NewCreature(blkA, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(blkB) {
		mage.Register(blkB, func() mage.Card {
			return mage.NewCreature(blkB, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
		})
	}
	becomeBlockedCount.Store(0)
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, becomeBlockedAtkName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, blkA)
	tg.AddCard(core.ZoneBattlefield, PlayerB, blkB)
	tg.Attack(1, PlayerA, becomeBlockedAtkName)
	tg.Block(1, PlayerB, blkA, becomeBlockedAtkName)
	tg.Block(1, PlayerB, blkB, becomeBlockedAtkName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	if got := becomeBlockedCount.Load(); got != 1 {
		t.Errorf("CR 509.3c: 'becomes blocked' trigger fired %d time(s); want 1", got)
	}
}

// ---------------------------------------------------------------------------
// CR 509.3d — "Becomes blocked by a creature" — per blocker (G18)
// ---------------------------------------------------------------------------

// TestCR509_3d_BecomesBlockedByCreatureTrigger verifies CR 509.3d:
// fires once per blocking creature.
func TestCR509_3d_BecomesBlockedByCreatureTrigger(t *testing.T) {
	const blockedByAtkName = "CR509.3d Blocked-By Attacker"
	var blockedByCount atomic.Int32

	if !mage.CardRegistered(blockedByAtkName) {
		mage.Register(blockedByAtkName, func() mage.Card {
			return mage.NewCreature(blockedByAtkName, "{2}{G}", 4, 4,
				mage.WithSubTypes("Beast"),
				mage.WithAbility(
					mage.NewTriggered(core.EvtDeclaredBlocker, false,
						mage.FuncEffect("count blocked-by", mage.EffectProperties{},
							func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
								blockedByCount.Add(1)
								return nil
							})).
						SetCondition(func(evt *core.GameEvent, _ mage.GameReader, sourceID, _ uuid.UUID) bool {
							return evt.TargetID == sourceID
						})))
		})
	}
	const bblkA = "CR509.3d Blocker A"
	const bblkB = "CR509.3d Blocker B"
	if !mage.CardRegistered(bblkA) {
		mage.Register(bblkA, func() mage.Card {
			return mage.NewCreature(bblkA, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(bblkB) {
		mage.Register(bblkB, func() mage.Card {
			return mage.NewCreature(bblkB, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
		})
	}
	blockedByCount.Store(0)
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, blockedByAtkName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, bblkA)
	tg.AddCard(core.ZoneBattlefield, PlayerB, bblkB)
	tg.Attack(1, PlayerA, blockedByAtkName)
	tg.Block(1, PlayerB, bblkA, blockedByAtkName)
	tg.Block(1, PlayerB, bblkB, blockedByAtkName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	if got := blockedByCount.Load(); got != 2 {
		t.Errorf("CR 509.3d: 'blocked by a creature' trigger fired %d time(s); want 2", got)
	}
}

// ---------------------------------------------------------------------------
// CR 509.3e — Blocker-count threshold trigger (G19)
// ---------------------------------------------------------------------------

// TestCR509_3e_BlockerCountThresholdTrigger verifies CR 509.3e.
func TestCR509_3e_BlockerCountThresholdTrigger(t *testing.T) {
	const threshAtkName = "CR509.3e Threshold Attacker"
	var threshFired atomic.Int32

	if !mage.CardRegistered(threshAtkName) {
		mage.Register(threshAtkName, func() mage.Card {
			return mage.NewCreature(threshAtkName, "{3}", 4, 4,
				mage.WithSubTypes("Beast"),
				mage.WithAbility(
					mage.NewTriggered(core.EvtBlockersDecl, false,
						mage.FuncEffect("threshold fires", mage.EffectProperties{},
							func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
								threshFired.Add(1)
								return nil
							})).
						SetCondition(func(_ *core.GameEvent, gr mage.GameReader, sourceID, _ uuid.UUID) bool {
							cg := gr.CombatGroupFor(sourceID)
							return cg != nil && len(cg.BlockerIDs) >= 2
						})))
		})
	}
	const th1 = "CR509.3e Blocker 1"
	const th2 = "CR509.3e Blocker 2"
	if !mage.CardRegistered(th1) {
		mage.Register(th1, func() mage.Card {
			return mage.NewCreature(th1, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
		})
	}
	if !mage.CardRegistered(th2) {
		mage.Register(th2, func() mage.Card {
			return mage.NewCreature(th2, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
		})
	}
	t.Run("fires when blocked by two", func(t *testing.T) {
		threshFired.Store(0)
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, threshAtkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, th1)
		tg.AddCard(core.ZoneBattlefield, PlayerB, th2)
		tg.Attack(1, PlayerA, threshAtkName)
		tg.Block(1, PlayerB, th1, threshAtkName)
		tg.Block(1, PlayerB, th2, threshAtkName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		if threshFired.Load() != 1 {
			t.Errorf("CR 509.3e: threshold trigger did not fire when blocked by 2")
		}
	})
	t.Run("does not fire when blocked by one", func(t *testing.T) {
		const thOnly = "CR509.3e Only Blocker"
		if !mage.CardRegistered(thOnly) {
			mage.Register(thOnly, func() mage.Card {
				return mage.NewCreature(thOnly, "{1}{G}", 1, 2, mage.WithSubTypes("Bear"))
			})
		}
		threshFired.Store(0)
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, threshAtkName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, thOnly)
		tg.Attack(1, PlayerA, threshAtkName)
		tg.Block(1, PlayerB, thOnly, threshAtkName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		if threshFired.Load() != 0 {
			t.Errorf("CR 509.3e: threshold trigger fired with only 1 blocker; want 0")
		}
	})
}

// ---------------------------------------------------------------------------
// CR 509.3f — Characteristic at blocking time; no retro-trigger (G20)
// ---------------------------------------------------------------------------

// TestCR509_3f_CharacteristicChangeNoRetroTrigger verifies CR 509.3f:
// trigger conditions evaluated at declaration time; post-declaration color
// change does not retro-trigger.
func TestCR509_3f_CharacteristicChangeNoRetroTrigger(t *testing.T) {
	const colorWatcherName = "CR509.3f Green Block Watcher"
	const blueBlocker = "CR509.3f Blue Blocker"
	const laceName = "CR509.3f Lace Instant"
	const atkForBlock = "CR509.3f Attacker"

	if !mage.CardRegistered(colorWatcherName) {
		mage.Register(colorWatcherName, func() mage.Card {
			trig := mage.NewTriggered(core.EvtDeclaredBlocker, false,
				mage.GainLife(2)).
				SetCondition(func(evt *core.GameEvent, gr mage.GameReader, _, _ uuid.UUID) bool {
					blk := gr.FindPermanent(evt.SourceID)
					if blk == nil {
						return false
					}
					return slices.Contains(blk.Colors(), core.Green)
				})
			return mage.NewCreature(colorWatcherName, "{2}", 1, 1,
				mage.WithSubTypes("Spirit"),
				mage.WithAbility(trig))
		})
	}
	if !mage.CardRegistered(blueBlocker) {
		mage.Register(blueBlocker, func() mage.Card {
			return mage.NewCreature(blueBlocker, "{1}{U}", 2, 2, mage.WithSubTypes("Merfolk"))
		})
	}
	if !mage.CardRegistered(laceName) {
		mage.Register(laceName, func() mage.Card {
			return mage.NewInstant(laceName, "{G}",
				mage.NewTargetedSpell(mage.TargetPermanent(),
					mage.ChangeColorEffect(core.Green)))
		})
	}
	if !mage.CardRegistered(atkForBlock) {
		mage.Register(atkForBlock, func() mage.Card {
			return mage.NewCreature(atkForBlock, "{1}{R}", 2, 2, mage.WithSubTypes("Goblin"))
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, colorWatcherName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, blueBlocker)
	tg.AddCard(core.ZoneBattlefield, PlayerA, atkForBlock)
	tg.AddCard(core.ZoneHand, PlayerB, laceName)
	tg.Attack(1, PlayerA, atkForBlock)
	tg.Block(1, PlayerB, blueBlocker, atkForBlock)
	tg.CastSpell(1, core.CombatDamage, PlayerB, laceName, blueBlocker)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertLife(PlayerB, 20)
}

// ---------------------------------------------------------------------------
// CR 509.3g — "Attacks and isn't blocked" trigger (G21)
// ---------------------------------------------------------------------------

// TestCR509_3g_AttacksAndIsntBlockedTrigger verifies CR 509.3g.
func TestCR509_3g_AttacksAndIsntBlockedTrigger(t *testing.T) {
	const isntBlockedName = "CR509.3g Isnt Blocked Bear"
	if !mage.CardRegistered(isntBlockedName) {
		mage.Register(isntBlockedName, func() mage.Card {
			return mage.NewCreature(isntBlockedName, "{1}{R}", 2, 2,
				mage.WithSubTypes("Bear"),
				mage.WithAbility(
					mage.NewTriggered(core.EvtBlockersDecl, false,
						mage.GainLife(3)).
						SetCondition(func(_ *core.GameEvent, gr mage.GameReader, sourceID, _ uuid.UUID) bool {
							if !gr.IsAttackingInCombat(sourceID) {
								return false
							}
							cg := gr.CombatGroupFor(sourceID)
							return cg == nil || len(cg.BlockerIDs) == 0
						})))
		})
	}
	t.Run("fires when unblocked", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, isntBlockedName)
		tg.Attack(1, PlayerA, isntBlockedName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerA, 23)
		tg.AssertLife(PlayerB, 18)
	})
	t.Run("does not fire when blocked", func(t *testing.T) {
		const bkr = "CR509.3g Blocker"
		if !mage.CardRegistered(bkr) {
			mage.Register(bkr, func() mage.Card {
				return mage.NewCreature(bkr, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, isntBlockedName)
		tg.AddCard(core.ZoneBattlefield, PlayerB, bkr)
		tg.Attack(1, PlayerA, isntBlockedName)
		tg.Block(1, PlayerB, bkr, isntBlockedName)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerA, 20)
	})
}

// ---------------------------------------------------------------------------
// CR 509.4 — ETB blocking (G22)
// ---------------------------------------------------------------------------

// TestCR509_4_DeclareBlockersETBBlocking verifies CR 509.4: a creature put onto
// the battlefield blocking is treated as a blocking creature (absorbs combat
// damage) but per CR 509.4 does not cause BlocksTrigger abilities to fire —
// it never "blocked."
func TestCR509_4_DeclareBlockersETBBlocking(t *testing.T) {
	blockTokenSpell := "CR509.4 Emergency Blocker"
	if !mage.CardRegistered(blockTokenSpell) {
		mage.Register(blockTokenSpell, func() mage.Card {
			return mage.NewInstant(blockTokenSpell, "{W}",
				mage.NewTargetedSpell(mage.TargetCreature(mage.IsAttacking),
					mage.CreateTokenBlocking("CR509.4 Shield Drone", 0, 4,
						[]core.CardType{core.TypeCreature},
						[]string{"Soldier"})))
		})
	}

	blockTriggerName := "CR509.4 Trigger Wall"
	var blocksFires atomic.Int32
	if !mage.CardRegistered(blockTriggerName) {
		mage.Register(blockTriggerName, func() mage.Card {
			return mage.NewCreature(blockTriggerName, "{1}{W}", 0, 4,
				mage.WithSubTypes("Wall"),
				mage.WithKeyword(core.Defender),
				mage.WithAbility(mage.BlocksTrigger(
					mage.FuncEffect("count blocks fires", mage.EffectProperties{},
						func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
							blocksFires.Add(1)
							return nil
						}), false)))
		})
	}
	putBlockingName := "CR509.4 Put Wall Blocking"
	if !mage.CardRegistered(putBlockingName) {
		mage.Register(putBlockingName, func() mage.Card {
			return mage.NewInstant(putBlockingName, "{W}",
				mage.NewTargetedSpell(mage.TargetCreature(mage.IsAttacking),
					mage.FuncEffect("put wall blocking", mage.EffectProperties{},
						func(g *mage.Game, _, controller uuid.UUID, targets []uuid.UUID) error {
							wall, err := mage.CreateCard(blockTriggerName)
							if err != nil {
								return err
							}
							var atkID uuid.UUID
							if len(targets) > 0 {
								atkID = targets[0]
							}
							g.PutOnBattlefieldBlocking(wall, controller, atkID)
							return nil
						})))
		})
	}

	// Part 1: token put into play blocking absorbs the attacker's damage.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Plains")
	tg.AddCard(core.ZoneHand, PlayerB, blockTokenSpell)
	tg.Attack(1, PlayerA, "Grizzly Bears")
	tg.CastSpell(1, core.DeclareBlockers, PlayerB, blockTokenSpell, "Grizzly Bears")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertPermanentCount(PlayerB, "CR509.4 Shield Drone", 1)
	tg.AssertLife(PlayerB, 20) // Bears' damage absorbed by the 0/4 blocker

	// Part 2: BlocksTrigger must NOT fire when a creature with that trigger
	// is put onto the battlefield blocking (CR 509.4).
	blocksFires.Store(0)
	tg2 := NewTestGame(t)
	tg2.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg2.AddCard(core.ZoneBattlefield, PlayerB, "Plains")
	tg2.AddCard(core.ZoneHand, PlayerB, putBlockingName)
	tg2.Attack(1, PlayerA, "Grizzly Bears")
	tg2.CastSpell(1, core.DeclareBlockers, PlayerB, putBlockingName, "Grizzly Bears")
	tg2.StopAt(1, core.PostcombatMain)
	tg2.Execute()
	tg2.AssertPermanentCount(PlayerB, blockTriggerName, 1)
	tg2.AssertLife(PlayerB, 20)
	if got := blocksFires.Load(); got != 0 {
		t.Fatalf("CR 509.4: BlocksTrigger fired %d time(s); must not fire for a creature put into play blocking", got)
	}
}

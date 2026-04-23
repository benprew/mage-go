package eval

import (
	"math"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// ── permPower / permToughness ───────────────────────────────────────────────

func TestPermPower_BaseStats(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	if got := permPower(p); got != 2 {
		t.Errorf("permPower = %d, want 2", got)
	}
}

func TestPermToughness_BaseStats(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	if got := permToughness(p); got != 2 {
		t.Errorf("permToughness = %d, want 2", got)
	}
}

func TestPermPower_PlusCounters(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	p.AddCounter(core.P1P1, 3)
	if got := permPower(p); got != 5 {
		t.Errorf("permPower with +1/+1 counters = %d, want 5", got)
	}
}

func TestPermToughness_PlusCounters(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	p.AddCounter(core.P1P1, 3)
	if got := permToughness(p); got != 5 {
		t.Errorf("permToughness with +1/+1 counters = %d, want 5", got)
	}
}

func TestPermPower_MinusCounters(t *testing.T) {
	p := makePerm("Giant", "{3}{R}", 4, 4, uuid.New())
	p.AddCounter(core.M1M1, 2)
	if got := permPower(p); got != 2 {
		t.Errorf("permPower with -1/-1 counters = %d, want 2", got)
	}
}

func TestPermToughness_MinusCounters(t *testing.T) {
	p := makePerm("Giant", "{3}{R}", 4, 4, uuid.New())
	p.AddCounter(core.M1M1, 2)
	if got := permToughness(p); got != 2 {
		t.Errorf("permToughness with -1/-1 counters = %d, want 2", got)
	}
}

func TestPermPower_BasePTOverride(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	override := [2]int{5, 5}
	p.BasePTOverride = &override
	if got := permPower(p); got != 5 {
		t.Errorf("permPower with override = %d, want 5", got)
	}
}

func TestPermToughness_BasePTOverride(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	override := [2]int{5, 7}
	p.BasePTOverride = &override
	if got := permToughness(p); got != 7 {
		t.Errorf("permToughness with override = %d, want 7", got)
	}
}

func TestPermPower_OverridePlusCounters(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	override := [2]int{3, 3}
	p.BasePTOverride = &override
	p.AddCounter(core.P1P1, 2)
	if got := permPower(p); got != 5 {
		t.Errorf("permPower with override+counters = %d, want 5", got)
	}
}

// ── EvalCreature ────────────────────────────────────────────────────────────

func TestEvalCreature_Vanilla(t *testing.T) {
	p := makePerm("Hill Giant", "{3}{R}", 3, 3, uuid.New())
	// Expected: 3*PowerWeight + 3*ToughnessWeight = 3*2 + 3*1 = 9
	got := EvalCreature(p)
	if got != 9 {
		t.Errorf("EvalCreature(3/3 vanilla) = %d, want 9", got)
	}
}

func TestEvalCreature_Tapped(t *testing.T) {
	p := makePerm("Hill Giant", "{3}{R}", 3, 3, uuid.New())
	p.Tapped = true
	// Base: 9, tapped: 9 * 2/3 = 6
	got := EvalCreature(p)
	if got != 6 {
		t.Errorf("EvalCreature(tapped 3/3) = %d, want 6", got)
	}
}

func TestEvalCreature_SummonSick(t *testing.T) {
	p := makePerm("Hill Giant", "{3}{R}", 3, 3, uuid.New())
	p.GrantBaseAttr(core.AttrSummonSick)
	// Base: 9, summon sick: 9/2 = 4
	got := EvalCreature(p)
	if got != 4 {
		t.Errorf("EvalCreature(summon-sick 3/3) = %d, want 4", got)
	}
}

func TestEvalCreature_SummonSickWithHaste(t *testing.T) {
	p := makePerm("Hasty", "{3}{R}", 3, 3, uuid.New(), mage.WithKeyword(core.Haste))
	p.GrantBaseAttr(core.AttrSummonSick)
	// Haste negates summon-sick penalty; base 9 + haste bonus 2 = 11
	got := EvalCreature(p)
	if got != 11 {
		t.Errorf("EvalCreature(summon-sick+haste 3/3) = %d, want 11", got)
	}
}

// ── keywordBonus ────────────────────────────────────────────────────────────

func TestKeywordBonus_Flying(t *testing.T) {
	p := makePerm("Bird", "{1}{W}", 1, 1, uuid.New(), mage.WithKeyword(core.Flying))
	if got := keywordBonus(p); got != 4 {
		t.Errorf("keywordBonus(flying) = %d, want 4", got)
	}
}

func TestKeywordBonus_MultiKeyword(t *testing.T) {
	p := makePerm("Angel", "{3}{W}{W}", 4, 4, uuid.New(),
		mage.WithKeyword(core.Flying),
		mage.WithKeyword(core.Vigilance),
		mage.WithKeyword(core.FirstStrike),
	)
	// Flying +4, Vigilance +2, FirstStrike +2 = 8
	if got := keywordBonus(p); got != 8 {
		t.Errorf("keywordBonus(flying+vigilance+firststrike) = %d, want 8", got)
	}
}

func TestKeywordBonus_Drawbacks(t *testing.T) {
	p := makePerm("Wall", "{1}{W}", 0, 4, uuid.New(),
		mage.WithKeyword(core.Defender),
		mage.WithKeyword(core.DoesNotUntapKW),
	)
	// Defender -2, DoesNotUntap -2 = -4
	if got := keywordBonus(p); got != -4 {
		t.Errorf("keywordBonus(defender+doesNotUntap) = %d, want -4", got)
	}
}

func TestKeywordBonus_Unblockable(t *testing.T) {
	p := makePerm("Rogue", "{1}{U}", 1, 1, uuid.New(), mage.WithKeyword(core.UnblockableKW))
	if got := keywordBonus(p); got != 5 {
		t.Errorf("keywordBonus(unblockable) = %d, want 5", got)
	}
}

func TestKeywordBonus_RelativeOrder(t *testing.T) {
	u := makePerm("A", "{U}", 1, 1, uuid.New(), mage.WithKeyword(core.UnblockableKW))
	f := makePerm("B", "{U}", 1, 1, uuid.New(), mage.WithKeyword(core.Flying))
	fe := makePerm("C", "{B}", 1, 1, uuid.New(), mage.WithKeyword(core.Fear))
	uScore := keywordBonus(u)
	fScore := keywordBonus(f)
	feScore := keywordBonus(fe)
	if uScore <= fScore || fScore <= feScore {
		t.Errorf("expected unblockable(%d) > flying(%d) > fear(%d)", uScore, fScore, feScore)
	}
}

func TestKeywordBonus_Indestructible(t *testing.T) {
	p := makePerm("God", "{5}", 5, 5, uuid.New(), mage.WithKeyword(core.Indestructible))
	if got := keywordBonus(p); got != 5 {
		t.Errorf("keywordBonus(indestructible) = %d, want 5", got)
	}
}

func TestKeywordBonus_Deathtouch(t *testing.T) {
	p := makePerm("Snake", "{1}{G}", 1, 1, uuid.New(), mage.WithKeyword(core.Deathtouch))
	if got := keywordBonus(p); got != 3 {
		t.Errorf("keywordBonus(deathtouch) = %d, want 3", got)
	}
}

func TestKeywordBonus_DoubleStrike(t *testing.T) {
	p := makePerm("Knight", "{2}{R}", 2, 2, uuid.New(), mage.WithKeyword(core.DoubleStrike))
	if got := keywordBonus(p); got != 4 {
		t.Errorf("keywordBonus(doubleStrike) = %d, want 4", got)
	}
}

func TestKeywordBonus_Landwalk(t *testing.T) {
	p := makePerm("Merfolk", "{U}", 1, 1, uuid.New(), mage.WithKeyword(core.Islandwalk))
	if got := keywordBonus(p); got != 2 {
		t.Errorf("keywordBonus(islandwalk) = %d, want 2", got)
	}
}

func TestKeywordBonus_NoKeywords(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	if got := keywordBonus(p); got != 0 {
		t.Errorf("keywordBonus(vanilla) = %d, want 0", got)
	}
}

// ── abilityBonus ────────────────────────────────────────────────────────────

func TestAbilityBonus_ManaAbility(t *testing.T) {
	p := makePerm("Elf", "{G}", 1, 1, uuid.New(), mage.WithManaAbility(core.Green))
	if got := abilityBonus(p); got != 2 {
		t.Errorf("abilityBonus(mana) = %d, want 2", got)
	}
}

func TestAbilityBonus_AnyColorMana(t *testing.T) {
	p := makePerm("Bird", "{G}", 0, 1, uuid.New(), mage.WithAnyColorMana())
	if got := abilityBonus(p); got != 3 {
		t.Errorf("abilityBonus(anyColor) = %d, want 3", got)
	}
}

func TestAbilityBonus_ActivatedAbility(t *testing.T) {
	p := makePerm("Pinger", "{1}{R}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.TapSourceCost(),
			mage.WithTarget(mage.TargetAnyTarget())),
	)
	// Phase 0D: tap-to-damage scores 4 (up from flat 1)
	if got := abilityBonus(p); got != 4 {
		t.Errorf("abilityBonus(activated damage) = %d, want 4", got)
	}
}

func TestAbilityBonus_NoAbilities(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	if got := abilityBonus(p); got != 0 {
		t.Errorf("abilityBonus(none) = %d, want 0", got)
	}
}

// ── defaultEvaluate ─────────────────────────────────────────────────────────

func TestDefaultEvaluate_EmptyBoard(t *testing.T) {
	g, pa, _ := makeGame()
	got := defaultEvaluate(g, pa.PlayerID())
	if got != 0 {
		t.Errorf("defaultEvaluate(empty) = %d, want 0", got)
	}
}

func TestDefaultEvaluate_LifeAdvantage(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)
	got := defaultEvaluate(g, pa.PlayerID())
	// (25-15)*LifeWeight = 10*3 = 30
	if got != 30 {
		t.Errorf("defaultEvaluate(life advantage) = %d, want 30", got)
	}
}

func TestDefaultEvaluate_CreatureAdvantage(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(perm)
	got := defaultEvaluate(g, pa.PlayerID())
	// Creature: 2*2 + 2*1 = 6, no keywords/abilities
	if got < 6 {
		t.Errorf("defaultEvaluate with own creature = %d, want >= 6", got)
	}
}

func TestDefaultEvaluate_OpponentCreatureSubtracts(t *testing.T) {
	g, pa, pb := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(perm)
	got := defaultEvaluate(g, pa.PlayerID())
	// Opponent creature subtracts: score should be negative
	if got >= 0 {
		t.Errorf("defaultEvaluate with opponent creature = %d, want < 0", got)
	}
}

func TestDefaultEvaluate_HandAdvantage(t *testing.T) {
	g, pa, _ := makeGame()
	c1 := mage.NewCreature("Card", "{1}", 1, 1)
	c2 := mage.NewCreature("Card2", "{1}", 1, 1)
	pa.AddToHand(c1)
	pa.AddToHand(c2)
	got := defaultEvaluate(g, pa.PlayerID())
	// Hand advantage: 2 * CardWeight = 2 * 2 = 4
	if got != 4 {
		t.Errorf("defaultEvaluate(hand advantage) = %d, want 4", got)
	}
}

func TestDefaultEvaluate_LandAdvantage(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)
	got := defaultEvaluate(g, pa.PlayerID())
	// 1 land * LandWeight(1) + 1 untapped mana source * 2 = 3
	if got != 3 {
		t.Errorf("defaultEvaluate(land) = %d, want 3", got)
	}
}

func TestDefaultEvaluate_NilPlayer(t *testing.T) {
	g, _, _ := makeGame()
	// A random UUID won't match any player
	got := defaultEvaluate(g, uuid.New())
	if got != 0 {
		t.Errorf("defaultEvaluate(nil player) = %d, want 0", got)
	}
}

// ── ThreatPerMana ───────────────────────────────────────────────────────────

func TestThreatPerMana_Normal(t *testing.T) {
	p := makePerm("Bear", "{1}{G}", 2, 2, uuid.New())
	tpm := ThreatPerMana(p)
	if tpm <= 0 {
		t.Errorf("ThreatPerMana(2-CMC creature) = %f, want > 0", tpm)
	}
}

func TestThreatPerMana_ZeroCMC(t *testing.T) {
	// Token-like creature with 0 CMC
	p := makePerm("Token", "{0}", 1, 1, uuid.New())
	tpm := ThreatPerMana(p)
	if tpm != 0 {
		t.Errorf("ThreatPerMana(0-CMC) = %f, want 0", tpm)
	}
}

// ── SpellValue ──────────────────────────────────────────────────────────────

func TestSpellValue_Creature(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewCreature("Hill Giant", "{3}{R}", 3, 3)
	card.SetOwner(pa.PlayerID())
	// Creature value: power*2 + toughness = 3*2 + 3 = 9
	got := SpellValue(card, pa, g)
	if got != 9 {
		t.Errorf("SpellValue(creature 3/3) = %d, want 9", got)
	}
}

func TestSpellValue_DrawSpell(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Divination", "{2}{U}",
		mage.NewSpellAbility(mage.DrawCards(mage.Fixed(2))),
	)
	card.SetOwner(pa.PlayerID())
	// DrawCount=2: 2*3=6 base + 2*2=4 empty-hand bonus = 10
	got := SpellValue(card, pa, g)
	if got != 10 {
		t.Errorf("SpellValue(draw 2) = %d, want 10", got)
	}
}

func TestSpellValue_DamageSpell(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	// Base damage: 3. Bear (2/2) EvalCreature=6, lethal bonus=6/2=3. Total=6.
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	got := SpellValue(card, pa, g)
	if got != 6 {
		t.Errorf("SpellValue(bolt with lethal target) = %d, want 6", got)
	}
}

func TestSpellValue_DamageSpellNoLethal(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewInstant("Shock", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(2))),
	)
	card.SetOwner(pa.PlayerID())
	// Opponent has a 5/5 — not lethal
	oppCreature := makePerm("Giant", "{3}{G}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	got := SpellValue(card, pa, g)
	// Just the damage: 2
	if got != 2 {
		t.Errorf("SpellValue(shock, no lethal) = %d, want 2", got)
	}
}

func TestSpellValue_PureRemoval(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewSorcery("Doom Blade", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	// Opponent has a 3/3 creature = EvalCreature score
	oppCreature := makePerm("Hill Giant", "{3}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	got := SpellValue(card, pa, g)
	expected := EvalCreature(oppCreature)
	if got != expected {
		t.Errorf("SpellValue(removal) = %d, want %d", got, expected)
	}
}

func TestSpellValue_FallbackToCMC(t *testing.T) {
	g, pa, _ := makeGame()
	// A sorcery with no recognizable effect properties
	card := mage.NewSorcery("Mystery", "{2}{U}", mage.NewSpellAbility())
	card.SetOwner(pa.PlayerID())
	got := SpellValue(card, pa, g)
	// Should fallback to CMC = 3
	if got != 3 {
		t.Errorf("SpellValue(fallback) = %d, want 3 (CMC)", got)
	}
}

// ── SpellIsWorthless ────────────────────────────────────────────────────────

func TestSpellIsWorthless_TargetedNoTargets(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Doom Blade", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	// No creatures on battlefield → no valid targets
	if !SpellIsWorthless(card, pa, g) {
		t.Error("SpellIsWorthless should be true when no valid targets exist")
	}
}

func TestSpellIsWorthless_TargetedWithTargets(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewSorcery("Doom Blade", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	if SpellIsWorthless(card, pa, g) {
		t.Error("SpellIsWorthless should be false when valid targets exist")
	}
}

func TestSpellIsWorthless_Untargeted(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Divination", "{2}{U}",
		mage.NewSpellAbility(mage.DrawCards(mage.Fixed(2))),
	)
	card.SetOwner(pa.PlayerID())
	if SpellIsWorthless(card, pa, g) {
		t.Error("SpellIsWorthless should be false for untargeted spells")
	}
}

// ── CalculateLethal (Phase 0A) ──────────────────────────────────────────────

func TestCalculateLethal_EvasiveLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(5)
	// 3/3 flyer with no opposing flyers = 3 evasive damage
	flyer := makePerm("Bird", "{2}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Flying))
	// 2/2 unblockable = 2 evasive damage
	rogue := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	// Opponent has a ground blocker (can't block flyer or unblockable)
	wall := makePerm("Wall", "{1}{W}", 0, 5, pb.PlayerID())
	g.AddToBattlefield(flyer, rogue, wall)

	info := CalculateLethal(g, pa.PlayerID())
	if !info.IHaveLethal {
		t.Error("should detect lethal with 3+2 = 5 evasive damage vs 5 life")
	}
	if info.MyBoardDamage < 5 {
		t.Errorf("MyBoardDamage = %d, want >= 5", info.MyBoardDamage)
	}
	if len(info.LethalAttackers) == 0 {
		t.Error("LethalAttackers should not be empty")
	}
}

func TestCalculateLethal_NoLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(20)
	// Only 2 evasive damage, not enough for lethal against 20 life
	rogue := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	g.AddToBattlefield(rogue)

	info := CalculateLethal(g, pa.PlayerID())
	if info.IHaveLethal {
		t.Error("should not detect lethal with only 2 evasive damage vs 20 life")
	}
}

func TestCalculateLethal_TrampleLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	// 6/6 trampler vs 2/3 blocker = 3 trample damage
	trampler := makePerm("Wurm", "{4}{G}{G}", 6, 6, pa.PlayerID(), mage.WithKeyword(core.Trample))
	blocker := makePerm("Bear", "{1}{G}", 2, 3, pb.PlayerID())
	g.AddToBattlefield(trampler, blocker)

	info := CalculateLethal(g, pa.PlayerID())
	if !info.IHaveLethal {
		t.Error("should detect lethal with 6 trample - 3 toughness = 3 vs 3 life")
	}
}

func TestCalculateLethal_OpponentHasLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(4)
	// Opponent has a 4/4 flyer, we have no flyers to block
	flyer := makePerm("Dragon", "{3}{R}", 4, 4, pb.PlayerID(), mage.WithKeyword(core.Flying))
	bear := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(flyer, bear)

	info := CalculateLethal(g, pa.PlayerID())
	if !info.TheyHaveLethal {
		t.Error("should detect opponent has lethal with 4/4 flyer vs 4 life")
	}
}

func TestCalculateLethal_EmptyBoard(t *testing.T) {
	g, pa, _ := makeGame()
	info := CalculateLethal(g, pa.PlayerID())
	if info.IHaveLethal {
		t.Error("should not have lethal on empty board")
	}
	if info.TheyHaveLethal {
		t.Error("opponent should not have lethal on empty board")
	}
}

func TestCalculateLethal_MinimalSet(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	// Two unblockable creatures: 2/2 and 3/3. Only need the 3/3 for lethal.
	small := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	big := makePerm("Assassin", "{2}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	g.AddToBattlefield(small, big)

	info := CalculateLethal(g, pa.PlayerID())
	if !info.IHaveLethal {
		t.Error("should detect lethal")
	}
	// Minimal set should be just the 3/3 (enough for lethal against 3 life)
	if len(info.LethalAttackers) != 1 {
		t.Errorf("LethalAttackers = %d, want 1 (minimal set)", len(info.LethalAttackers))
	}
	if len(info.LethalAttackers) == 1 && info.LethalAttackers[0] != big.ID() {
		t.Error("should pick the 3/3 as the minimal lethal attacker")
	}
}

func TestEstimatePushThrough_FirstStrikeKillsBlocker(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(5)
	// 4/2 first striker vs 3/3 blocker: first strike kills 3-toughness blocker
	// before it deals damage. The attacker's full power pushes through because
	// the blocker is dead before it can block effectively.
	fs := makePerm("Knight", "{2}{W}{W}", 4, 2, pa.PlayerID(),
		mage.WithKeyword(core.FirstStrike))
	blocker := makePerm("Bear", "{2}{G}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(fs, blocker)

	info := CalculateLethal(g, pa.PlayerID())
	// First striker with power >= blocker toughness kills the blocker before
	// it can deal damage. In push-through estimation, this should count as
	// getting damage through (the blocker is dead before it matters).
	if info.MyBoardDamage < 4 {
		t.Errorf("MyBoardDamage = %d, want >= 4 (first striker kills blocker)", info.MyBoardDamage)
	}
}

func TestEstimatePushThrough_DeathtouchKillsAnyBlocker(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	// 2/2 deathtouch + trample vs 6/6 blocker: deathtouch needs 1 to kill,
	// remaining 1 tramples through. Without deathtouch awareness, the 6/6
	// would fully absorb the 2/2's damage.
	dt := makePerm("Snake", "{1}{B}{G}", 2, 2, pa.PlayerID(),
		mage.WithKeyword(core.Deathtouch), mage.WithKeyword(core.Trample))
	blocker := makePerm("Wurm", "{4}{G}{G}", 6, 6, pb.PlayerID())
	g.AddToBattlefield(dt, blocker)

	info := CalculateLethal(g, pa.PlayerID())
	// Deathtouch + trample: only 1 damage needed to kill blocker, rest tramples.
	// 2 power - 1 lethal = 1 trample damage.
	if info.MyBoardDamage < 1 {
		t.Errorf("MyBoardDamage = %d, want >= 1 (deathtouch trample through 6/6)", info.MyBoardDamage)
	}
}

// ── CalculateRace (Phase 0B) ────────────────────────────────────────────────

func TestCalculateRace_FavorableRace(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(20)
	pb.SetLife(6)
	// We have a 3/3 unblockable = 3 evasive damage per turn
	unblockable := makePerm("Rogue", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	// Opponent has a 1/1 = 1 damage per turn (no evasion but no blockers)
	opp := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(unblockable, opp)

	race := CalculateRace(g, pa.PlayerID())
	if race.MyClock >= race.TheirClock {
		t.Errorf("MyClock=%d should be < TheirClock=%d", race.MyClock, race.TheirClock)
	}
}

func TestCalculateRace_NoDamage(t *testing.T) {
	g, pa, _ := makeGame()
	race := CalculateRace(g, pa.PlayerID())
	if race.MyClock != math.MaxInt32 {
		t.Errorf("MyClock = %d, want MaxInt32 with no creatures", race.MyClock)
	}
}

func TestCalculateRace_RacingBothLow(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(6)
	pb.SetLife(6)
	// Both have 3/3 unblockables: 2-turn clocks each
	myAttacker := makePerm("Rogue A", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	theirAttacker := makePerm("Rogue B", "{1}{U}", 3, 3, pb.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	g.AddToBattlefield(myAttacker, theirAttacker)

	race := CalculateRace(g, pa.PlayerID())
	if !race.Racing {
		t.Error("both clocks should be < 5, Racing should be true")
	}
	if race.MyClock != 2 || race.TheirClock != 2 {
		t.Errorf("MyClock=%d TheirClock=%d, want both 2", race.MyClock, race.TheirClock)
	}
}

// ── ManaCurveBonus (Phase 0C) ───────────────────────────────────────────────

func TestManaCurveBonus_PreferExpensive(t *testing.T) {
	// 5 mana available, hand has 2-drop and 5-drop
	cmcs := []int{2, 5}
	bonusFor5 := ManaCurveBonus(5, 5, cmcs)
	bonusFor2 := ManaCurveBonus(2, 5, cmcs)
	if bonusFor5 <= bonusFor2 {
		t.Errorf("5-drop bonus (%.1f) should be > 2-drop bonus (%.1f) with 5 mana", bonusFor5, bonusFor2)
	}
}

func TestManaCurveBonus_OnlyLowCMC(t *testing.T) {
	// Only 2-drops in hand, 5 mana available
	cmcs := []int{2, 2}
	bonus := ManaCurveBonus(2, 5, cmcs)
	// No penalty because there's nothing better to cast
	if bonus < 0 {
		t.Errorf("bonus for 2-drop when only 2-drops available = %.1f, should not be negative", bonus)
	}
}

func TestManaCurveBonus_NoMana(t *testing.T) {
	cmcs := []int{2, 5}
	bonus := ManaCurveBonus(2, 0, cmcs)
	if bonus != 0 {
		t.Errorf("bonus with 0 mana = %.1f, want 0", bonus)
	}
}

// ── AbilityQuality (Phase 0D) ───────────────────────────────────────────────

func TestAbilityQuality_DrawAbility(t *testing.T) {
	p := makePerm("Sage", "{1}{U}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DrawCards(mage.Fixed(1)), mage.TapSourceCost()),
	)
	// The draw ability should score 5
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if ab, ok := inner.(mage.ActivatedAbility); ok {
			got := AbilityQuality(ab)
			if got != 5 {
				t.Errorf("AbilityQuality(draw) = %d, want 5", got)
			}
			return
		}
	}
	t.Fatal("no activated ability found on permanent")
}

func TestAbilityQuality_DamageAbility(t *testing.T) {
	p := makePerm("Pinger", "{1}{R}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.TapSourceCost(),
			mage.WithTarget(mage.TargetAnyTarget())),
	)
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if ab, ok := inner.(mage.ActivatedAbility); ok {
			got := AbilityQuality(ab)
			if got != 4 {
				t.Errorf("AbilityQuality(damage) = %d, want 4", got)
			}
			return
		}
	}
	t.Fatal("no activated ability found on permanent")
}

func TestAbilityQuality_PingerHigherThanPump(t *testing.T) {
	pinger := makePerm("Pinger", "{1}{R}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.TapSourceCost(),
			mage.WithTarget(mage.TargetAnyTarget())),
	)
	pumper := makePerm("Pump", "{2}{G}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.Boost(mage.Fixed(1), mage.Fixed(1)).Targeting(mage.ToSource()), mage.TapSourceCost()),
	)

	pingerScore := abilityBonus(pinger)
	pumperScore := abilityBonus(pumper)
	if pingerScore <= pumperScore {
		t.Errorf("pinger score (%d) should be > pumper score (%d)", pingerScore, pumperScore)
	}
}

// ── CountAvailableMana ──────────────────────────────────────────────────────

func TestCountAvailableMana_UntappedLands(t *testing.T) {
	g, pa, _ := makeGame()
	for i := 0; i < 3; i++ {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
	}
	// Add one tapped land
	tappedLand := mage.NewLand("Mountain")
	tappedLand.SetOwner(pa.PlayerID())
	tp := mage.NewPermanent(tappedLand, pa.PlayerID())
	tp.RevokeBaseAttr(core.AttrSummonSick)
	tp.Tapped = true
	g.AddToBattlefield(tp)

	got := CountAvailableMana(g, pa.PlayerID())
	if got != 3 {
		t.Errorf("CountAvailableMana = %d, want 3 (3 untapped lands)", got)
	}
}

func TestCountAvailableMana_IncludesManaCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	// Mana creature
	elf := makePerm("Llanowar Elves", "{G}", 1, 1, pa.PlayerID(), mage.WithManaAbility(core.Green))
	g.AddToBattlefield(elf)

	got := CountAvailableMana(g, pa.PlayerID())
	if got != 2 {
		t.Errorf("CountAvailableMana = %d, want 2 (1 land + 1 mana creature)", got)
	}
}

// ── clock helper ────────────────────────────────────────────────────────────

func TestClock_Basic(t *testing.T) {
	if got := clock(20, 3); got != 7 {
		t.Errorf("clock(20, 3) = %d, want 7 (ceil(20/3))", got)
	}
	if got := clock(6, 3); got != 2 {
		t.Errorf("clock(6, 3) = %d, want 2", got)
	}
	if got := clock(1, 3); got != 1 {
		t.Errorf("clock(1, 3) = %d, want 1", got)
	}
}

func TestClock_ZeroDamage(t *testing.T) {
	if got := clock(20, 0); got != math.MaxInt32 {
		t.Errorf("clock(20, 0) = %d, want MaxInt32", got)
	}
}

// ── handQuality ─────────────────────────────────────────────────────────────

func TestHandQuality_CastableSpells(t *testing.T) {
	g, pa, _ := makeGame()
	// 3 untapped lands
	for i := 0; i < 3; i++ {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
	}

	c1 := mage.NewCreature("Bear", "{1}{G}", 2, 2) // CMC 2, castable
	c2 := mage.NewCreature("Elf", "{G}", 1, 1)     // CMC 1, castable
	pa.AddToHand(c1)
	pa.AddToHand(c2)

	got := handQuality(pa, g)
	// Both spells castable (CMC <= 3 mana): 2 points each = 4
	if got != 4 {
		t.Errorf("handQuality(castable spells) = %d, want 4", got)
	}
}

func TestHandQuality_UncastableExpensiveSpell(t *testing.T) {
	g, pa, _ := makeGame()
	// 2 untapped lands
	for i := 0; i < 2; i++ {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
	}

	expensive := mage.NewCreature("Wurm", "{5}{G}{G}", 7, 7) // CMC 7, not castable with 2 mana
	pa.AddToHand(expensive)

	got := handQuality(pa, g)
	// CMC 7, available mana 2: CMC > availMana+3 (5) → dead card = 0
	if got != 0 {
		t.Errorf("handQuality(uncastable 7-drop with 2 mana) = %d, want 0", got)
	}
}

func TestHandQuality_NearCastableSpell(t *testing.T) {
	g, pa, _ := makeGame()
	// 2 untapped lands
	for i := 0; i < 2; i++ {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
	}

	// CMC 4, available mana 2: CMC <= availMana+2 (4) → partial value = 1
	spell := mage.NewCreature("Giant", "{3}{R}", 3, 3)
	pa.AddToHand(spell)

	got := handQuality(pa, g)
	if got != 1 {
		t.Errorf("handQuality(near-castable 4-drop with 2 mana) = %d, want 1", got)
	}
}

func TestHandQuality_ExcessLands(t *testing.T) {
	g, pa, _ := makeGame()
	// 6 lands in hand
	for i := 0; i < 6; i++ {
		land := mage.NewLand("Forest")
		pa.AddToHand(land)
	}
	// Lands: first 5 get 1 point each, 6th gets -1 (flood penalty)
	got := handQuality(pa, g)
	if got != 4 {
		t.Errorf("handQuality(6 lands) = %d, want 4 (5*1 + 1*-1)", got)
	}
}

func TestHandQuality_EmptyHand(t *testing.T) {
	g, pa, _ := makeGame()
	got := handQuality(pa, g)
	if got != 0 {
		t.Errorf("handQuality(empty) = %d, want 0", got)
	}
}

// ── Lifelink in Race ────────────────────────────────────────────────────────

func TestCalculateRace_LifelinkBetterClock(t *testing.T) {
	// A 3/3 lifelink creature should have a better effective clock than a vanilla 3/3.
	// Lifelink deals 3 to opponent AND gains 3, effectively a 6-point swing per turn.
	// Against the same opponent board, the lifelink creature's clock should be shorter.
	g1, pa1, pb1 := makeGame()
	pa1.SetLife(10)
	pb1.SetLife(10)
	llCreature := makePerm("Lifelinker", "{1}{W}{W}", 3, 3, pa1.PlayerID(),
		mage.WithKeyword(core.Lifelink), mage.WithKeyword(core.UnblockableKW))
	oppAtk1 := makePerm("Opp Bear", "{1}{R}", 3, 3, pb1.PlayerID(),
		mage.WithKeyword(core.UnblockableKW))
	g1.AddToBattlefield(llCreature, oppAtk1)

	g2, pa2, pb2 := makeGame()
	pa2.SetLife(10)
	pb2.SetLife(10)
	vanillaCreature := makePerm("Vanilla", "{2}{G}", 3, 3, pa2.PlayerID(),
		mage.WithKeyword(core.UnblockableKW))
	oppAtk2 := makePerm("Opp Bear", "{1}{R}", 3, 3, pb2.PlayerID(),
		mage.WithKeyword(core.UnblockableKW))
	g2.AddToBattlefield(vanillaCreature, oppAtk2)

	raceLL := CalculateRace(g1, pa1.PlayerID())
	raceVanilla := CalculateRace(g2, pa2.PlayerID())

	// With lifelink, our clock should be the same (3 damage per turn, same)
	// but their clock should be longer (we gain life, so they need more turns)
	if raceLL.TheirClock <= raceVanilla.TheirClock {
		t.Errorf("lifelink TheirClock=%d should be > vanilla TheirClock=%d (we gain life)",
			raceLL.TheirClock, raceVanilla.TheirClock)
	}
}

func TestCalculateRace_LifelinkIncreasesEffectiveLife(t *testing.T) {
	// Opponent has lifelink creatures — their effective life goes up each turn,
	// meaning our clock should be longer.
	g1, pa1, pb1 := makeGame()
	pa1.SetLife(10)
	pb1.SetLife(10)
	myAtk1 := makePerm("My Rogue", "{1}{U}", 3, 3, pa1.PlayerID(),
		mage.WithKeyword(core.UnblockableKW))
	oppLL := makePerm("Opp Lifelinker", "{1}{W}{W}", 3, 3, pb1.PlayerID(),
		mage.WithKeyword(core.Lifelink), mage.WithKeyword(core.UnblockableKW))
	g1.AddToBattlefield(myAtk1, oppLL)

	g2, pa2, pb2 := makeGame()
	pa2.SetLife(10)
	pb2.SetLife(10)
	myAtk2 := makePerm("My Rogue", "{1}{U}", 3, 3, pa2.PlayerID(),
		mage.WithKeyword(core.UnblockableKW))
	oppVanilla := makePerm("Opp Vanilla", "{2}{R}", 3, 3, pb2.PlayerID(),
		mage.WithKeyword(core.UnblockableKW))
	g2.AddToBattlefield(myAtk2, oppVanilla)

	raceOppLL := CalculateRace(g1, pa1.PlayerID())
	raceOppVanilla := CalculateRace(g2, pa2.PlayerID())

	// When opponent has lifelink, our clock to kill them should be longer
	if raceOppLL.MyClock <= raceOppVanilla.MyClock {
		t.Errorf("MyClock vs lifelink (%d) should be > MyClock vs vanilla (%d)",
			raceOppLL.MyClock, raceOppVanilla.MyClock)
	}
}

// ── Damage-aware creature scoring ───────────────────────────────────────────

func TestEvalCreatureInGame_DamagedReducesScore(t *testing.T) {
	g, pa, _ := makeGame()
	fresh := makePerm("Hill Giant", "{3}{R}", 3, 3, pa.PlayerID())
	damaged := makePerm("Hill Giant", "{3}{R}", 3, 3, pa.PlayerID())
	damaged.Damage = 2 // one toughness from death

	freshScore := EvalCreatureInGame(fresh, g)
	damagedScore := EvalCreatureInGame(damaged, g)

	if damagedScore >= freshScore {
		t.Fatalf("damaged creature should score less than fresh: damaged=%d fresh=%d",
			damagedScore, freshScore)
	}
	if damagedScore <= 0 {
		t.Fatalf("1 effective toughness should still have positive value, got %d", damagedScore)
	}
}

// ── handQuality in defaultEvaluate ──────────────────────────────────────────

func TestDefaultEvaluate_HandQualityPrefersCastable(t *testing.T) {
	// Two games, identical except the hand. Game 1 has a castable spell;
	// game 2 has an uncastable spell. handQuality should make game 1 better.
	g1, pa1, _ := makeGame()
	addLands(g1, pa1, "Mountain", 3)
	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))))
	bolt.SetOwner(pa1.PlayerID())
	pa1.AddToHand(bolt)

	g2, pa2, _ := makeGame()
	addLands(g2, pa2, "Mountain", 3)
	fireball := mage.NewInstant("Fireball", "{8}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(8))))
	fireball.SetOwner(pa2.PlayerID())
	pa2.AddToHand(fireball)

	s1 := defaultEvaluate(g1, pa1.PlayerID())
	s2 := defaultEvaluate(g2, pa2.PlayerID())
	if s1 <= s2 {
		t.Fatalf("castable hand (%d) should score above dead hand (%d)", s1, s2)
	}
}

// ── Race clock integration in defaultEvaluate ───────────────────────────────

func TestDefaultEvaluate_RaceFavorsShorterClock(t *testing.T) {
	// Both players at low life with equal board size, except I have a bigger
	// threat — my clock should be shorter, and defaultEvaluate should reward that.
	g, pa, pb := makeGame()
	pa.SetLife(5)
	pb.SetLife(5)
	g.AddToBattlefield(
		makePerm("Shivan Dragon", "{4}{R}{R}", 5, 5, pa.PlayerID()))
	g.AddToBattlefield(
		makePerm("Grizzly Bears", "{1}{G}", 2, 2, pb.PlayerID()))
	g.SetStep(core.PrecombatMain)

	fromA := defaultEvaluate(g, pa.PlayerID())
	fromB := defaultEvaluate(g, pb.PlayerID())
	if fromA <= fromB {
		t.Fatalf("shorter-clock player should score higher: A=%d B=%d", fromA, fromB)
	}
}

// addLands mirrors the helper in pkg/mage/interactive/ai/search_test.go.
// Kept local to avoid a cross-package test-helper import.
func addLands(g *mage.Game, p *mage.BasePlayer, name string, count int) {
	colorMap := map[string]core.Color{
		"Forest": core.Green, "Mountain": core.Red, "Plains": core.White,
		"Island": core.Blue, "Swamp": core.Black,
	}
	color := colorMap[name]
	for i := 0; i < count; i++ {
		land := mage.NewLand(name, mage.WithManaAbility(color))
		land.SetOwner(p.PlayerID())
		perm := mage.NewPermanent(land, p.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(perm)
	}
}

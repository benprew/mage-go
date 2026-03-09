package interactive

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// makePerm creates a creature permanent with cleared summoning sickness.
func makePerm(name, cost string, power, toughness int, owner uuid.UUID, opts ...mage.CardOption) *mage.Permanent {
	card := mage.NewCreature(name, cost, power, toughness, opts...)
	card.SetOwner(owner)
	perm := mage.NewPermanent(card, owner)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	return perm
}

// makeGame creates a two-player game with BasePlayer instances.
func makeGame() (*mage.Game, *mage.BasePlayer, *mage.BasePlayer) {
	pa := mage.NewBasePlayer("Alice")
	pb := mage.NewBasePlayer("Bob")
	g := mage.NewGame(pa, pb)
	return g, pa, pb
}

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

// ── evalCreature ────────────────────────────────────────────────────────────

func TestEvalCreature_Vanilla(t *testing.T) {
	p := makePerm("Hill Giant", "{3}{R}", 3, 3, uuid.New())
	// Expected: 3*PowerWeight + 3*ToughnessWeight = 3*2 + 3*1 = 9
	got := evalCreature(p)
	if got != 9 {
		t.Errorf("evalCreature(3/3 vanilla) = %d, want 9", got)
	}
}

func TestEvalCreature_Tapped(t *testing.T) {
	p := makePerm("Hill Giant", "{3}{R}", 3, 3, uuid.New())
	p.Tapped = true
	// Base: 9, tapped: 9 * 2/3 = 6
	got := evalCreature(p)
	if got != 6 {
		t.Errorf("evalCreature(tapped 3/3) = %d, want 6", got)
	}
}

func TestEvalCreature_SummonSick(t *testing.T) {
	p := makePerm("Hill Giant", "{3}{R}", 3, 3, uuid.New())
	p.GrantBaseAttr(core.AttrSummonSick)
	// Base: 9, summon sick: 9/2 = 4
	got := evalCreature(p)
	if got != 4 {
		t.Errorf("evalCreature(summon-sick 3/3) = %d, want 4", got)
	}
}

func TestEvalCreature_SummonSickWithHaste(t *testing.T) {
	p := makePerm("Hasty", "{3}{R}", 3, 3, uuid.New(), mage.WithKeyword(core.Haste))
	p.GrantBaseAttr(core.AttrSummonSick)
	// Haste negates summon-sick penalty; base 9 + haste bonus 2 = 11
	got := evalCreature(p)
	if got != 11 {
		t.Errorf("evalCreature(summon-sick+haste 3/3) = %d, want 11", got)
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
	if !(uScore > fScore && fScore > feScore) {
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
	g.Battlefield = append(g.Battlefield, perm)
	got := defaultEvaluate(g, pa.PlayerID())
	// Creature: 2*2 + 2*1 = 6, no keywords/abilities
	if got < 6 {
		t.Errorf("defaultEvaluate with own creature = %d, want >= 6", got)
	}
}

func TestDefaultEvaluate_OpponentCreatureSubtracts(t *testing.T) {
	g, pa, pb := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, perm)
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
	g.Battlefield = append(g.Battlefield, lp)
	got := defaultEvaluate(g, pa.PlayerID())
	// 1 land * LandWeight + 1 untapped land bonus = 1 + 1 = 2
	if got != 2 {
		t.Errorf("defaultEvaluate(land) = %d, want 2", got)
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

// ── spellValue ──────────────────────────────────────────────────────────────

func TestSpellValue_Creature(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewCreature("Hill Giant", "{3}{R}", 3, 3)
	card.SetOwner(pa.PlayerID())
	// Creature value: power*2 + toughness = 3*2 + 3 = 9
	got := spellValue(card, pa, g)
	if got != 9 {
		t.Errorf("spellValue(creature 3/3) = %d, want 9", got)
	}
}

func TestSpellValue_DrawSpell(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Divination", "{2}{U}",
		mage.NewSpellAbility(mage.DrawCards(mage.Fixed(2))),
	)
	card.SetOwner(pa.PlayerID())
	// DrawCount=2, so 2*3 = 6
	got := spellValue(card, pa, g)
	if got != 6 {
		t.Errorf("spellValue(draw 2) = %d, want 6", got)
	}
}

func TestSpellValue_DamageSpell(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	// Base damage: 3. With an opponent creature of toughness <= 3, +2 lethal bonus.
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)
	got := spellValue(card, pa, g)
	// 3 (damage) + 2 (lethal) = 5
	if got != 5 {
		t.Errorf("spellValue(bolt with lethal target) = %d, want 5", got)
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
	g.Battlefield = append(g.Battlefield, oppCreature)
	got := spellValue(card, pa, g)
	// Just the damage: 2
	if got != 2 {
		t.Errorf("spellValue(shock, no lethal) = %d, want 2", got)
	}
}

func TestSpellValue_PureRemoval(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewSorcery("Doom Blade", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	// Opponent has a 3/3 creature = evalCreature score
	oppCreature := makePerm("Hill Giant", "{3}{R}", 3, 3, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)
	got := spellValue(card, pa, g)
	expected := evalCreature(oppCreature)
	if got != expected {
		t.Errorf("spellValue(removal) = %d, want %d", got, expected)
	}
}

func TestSpellValue_FallbackToCMC(t *testing.T) {
	g, pa, _ := makeGame()
	// A sorcery with no recognizable effect properties
	card := mage.NewSorcery("Mystery", "{2}{U}", mage.NewSpellAbility())
	card.SetOwner(pa.PlayerID())
	got := spellValue(card, pa, g)
	// Should fallback to CMC = 3
	if got != 3 {
		t.Errorf("spellValue(fallback) = %d, want 3 (CMC)", got)
	}
}

// ── spellIsWorthless ────────────────────────────────────────────────────────

func TestSpellIsWorthless_TargetedNoTargets(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Doom Blade", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	// No creatures on battlefield → no valid targets
	if !spellIsWorthless(card, pa, g) {
		t.Error("spellIsWorthless should be true when no valid targets exist")
	}
}

func TestSpellIsWorthless_TargetedWithTargets(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewSorcery("Doom Blade", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)
	if spellIsWorthless(card, pa, g) {
		t.Error("spellIsWorthless should be false when valid targets exist")
	}
}

func TestSpellIsWorthless_Untargeted(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Divination", "{2}{U}",
		mage.NewSpellAbility(mage.DrawCards(mage.Fixed(2))),
	)
	card.SetOwner(pa.PlayerID())
	if spellIsWorthless(card, pa, g) {
		t.Error("spellIsWorthless should be false for untargeted spells")
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
	g.Battlefield = append(g.Battlefield, flyer, rogue, wall)

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
	g.Battlefield = append(g.Battlefield, rogue)

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
	g.Battlefield = append(g.Battlefield, trampler, blocker)

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
	g.Battlefield = append(g.Battlefield, flyer, bear)

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
	g.Battlefield = append(g.Battlefield, small, big)

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

// ── CalculateRace (Phase 0B) ────────────────────────────────────────────────

func TestCalculateRace_FavorableRace(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(20)
	pb.SetLife(6)
	// We have a 3/3 unblockable = 3 evasive damage per turn
	unblockable := makePerm("Rogue", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	// Opponent has a 1/1 = 1 damage per turn (no evasion but no blockers)
	opp := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, unblockable, opp)

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
	g.Battlefield = append(g.Battlefield, myAttacker, theirAttacker)

	race := CalculateRace(g, pa.PlayerID())
	if !race.Racing {
		t.Error("both clocks should be < 5, Racing should be true")
	}
	if race.MyClock != 2 || race.TheirClock != 2 {
		t.Errorf("MyClock=%d TheirClock=%d, want both 2", race.MyClock, race.TheirClock)
	}
}

// ── manaCurveBonus (Phase 0C) ───────────────────────────────────────────────

func TestManaCurveBonus_PreferExpensive(t *testing.T) {
	// 5 mana available, hand has 2-drop and 5-drop
	cmcs := []int{2, 5}
	bonusFor5 := manaCurveBonus(5, 5, cmcs)
	bonusFor2 := manaCurveBonus(2, 5, cmcs)
	if bonusFor5 <= bonusFor2 {
		t.Errorf("5-drop bonus (%.1f) should be > 2-drop bonus (%.1f) with 5 mana", bonusFor5, bonusFor2)
	}
}

func TestManaCurveBonus_OnlyLowCMC(t *testing.T) {
	// Only 2-drops in hand, 5 mana available
	cmcs := []int{2, 2}
	bonus := manaCurveBonus(2, 5, cmcs)
	// No penalty because there's nothing better to cast
	if bonus < 0 {
		t.Errorf("bonus for 2-drop when only 2-drops available = %.1f, should not be negative", bonus)
	}
}

func TestManaCurveBonus_NoMana(t *testing.T) {
	cmcs := []int{2, 5}
	bonus := manaCurveBonus(2, 0, cmcs)
	if bonus != 0 {
		t.Errorf("bonus with 0 mana = %.1f, want 0", bonus)
	}
}

// ── abilityQuality (Phase 0D) ───────────────────────────────────────────────

func TestAbilityQuality_DrawAbility(t *testing.T) {
	p := makePerm("Sage", "{1}{U}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DrawCards(mage.Fixed(1)), mage.TapSourceCost()),
	)
	// The draw ability should score 5
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if ab, ok := inner.(mage.ActivatedAbility); ok {
			got := abilityQuality(ab)
			if got != 5 {
				t.Errorf("abilityQuality(draw) = %d, want 5", got)
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
			got := abilityQuality(ab)
			if got != 4 {
				t.Errorf("abilityQuality(damage) = %d, want 4", got)
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
		mage.WithActivatedAbility(mage.BoostUntilEndOfTurn(mage.Fixed(1), mage.Fixed(1), mage.SelectSource), mage.TapSourceCost()),
	)

	pingerScore := abilityBonus(pinger)
	pumperScore := abilityBonus(pumper)
	if pingerScore <= pumperScore {
		t.Errorf("pinger score (%d) should be > pumper score (%d)", pingerScore, pumperScore)
	}
}

// ── countAvailableMana ──────────────────────────────────────────────────────

func TestCountAvailableMana_UntappedLands(t *testing.T) {
	g, pa, _ := makeGame()
	for i := 0; i < 3; i++ {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.Battlefield = append(g.Battlefield, lp)
	}
	// Add one tapped land
	tappedLand := mage.NewLand("Mountain")
	tappedLand.SetOwner(pa.PlayerID())
	tp := mage.NewPermanent(tappedLand, pa.PlayerID())
	tp.RevokeBaseAttr(core.AttrSummonSick)
	tp.Tapped = true
	g.Battlefield = append(g.Battlefield, tp)

	got := countAvailableMana(g, pa.PlayerID())
	if got != 3 {
		t.Errorf("countAvailableMana = %d, want 3 (3 untapped lands)", got)
	}
}

func TestCountAvailableMana_IncludesManaCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.Battlefield = append(g.Battlefield, lp)

	// Mana creature
	elf := makePerm("Llanowar Elves", "{G}", 1, 1, pa.PlayerID(), mage.WithManaAbility(core.Green))
	g.Battlefield = append(g.Battlefield, elf)

	got := countAvailableMana(g, pa.PlayerID())
	if got != 2 {
		t.Errorf("countAvailableMana = %d, want 2 (1 land + 1 mana creature)", got)
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

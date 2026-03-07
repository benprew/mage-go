package interactive

import (
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
	if got := abilityBonus(p); got != 1 {
		t.Errorf("abilityBonus(activated) = %d, want 1", got)
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

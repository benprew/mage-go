package main

import "fmt"

// Color represents the five Magic colors plus colorless.
type Color int

const (
	White Color = iota
	Blue
	Black
	Red
	Green
	Colorless
)

var AllColors = []Color{White, Blue, Black, Red, Green}

func (c Color) Symbol() string {
	switch c {
	case White:
		return "W"
	case Blue:
		return "U"
	case Black:
		return "B"
	case Red:
		return "R"
	case Green:
		return "G"
	case Colorless:
		return "C"
	}
	return "?"
}

func (c Color) Label() string {
	switch c {
	case White:
		return "WHITE"
	case Blue:
		return "BLUE"
	case Black:
		return "BLACK"
	case Red:
		return "RED"
	case Green:
		return "GREEN"
	case Colorless:
		return "COLORLESS"
	}
	return "UNKNOWN"
}

// Rarity levels.
type Rarity int

const (
	Common Rarity = iota
	Uncommon
	Rare
)

var AllRarities = []Rarity{Common, Uncommon, Rare}

func (r Rarity) String() string {
	switch r {
	case Common:
		return "Common"
	case Uncommon:
		return "Uncommon"
	case Rare:
		return "Rare"
	}
	return "?"
}

func (r Rarity) Char() string {
	switch r {
	case Common:
		return "C"
	case Uncommon:
		return "U"
	case Rare:
		return "R"
	}
	return "?"
}

// Keyword abilities.
type Keyword int

const (
	Flying Keyword = iota
	FirstStrike
	DoubleStrike
	Trample
	Vigilance
	Haste
	Lifelink
	Deathtouch
	Reach
	Defender
	Menace
	Hexproof
	Indestructible
	Flash
)

var AllKeywords = []Keyword{
	Flying, FirstStrike, DoubleStrike, Trample, Vigilance,
	Haste, Lifelink, Deathtouch, Reach, Defender,
	Menace, Hexproof, Indestructible, Flash,
}

func (k Keyword) String() string {
	switch k {
	case Flying:
		return "flying"
	case FirstStrike:
		return "first strike"
	case DoubleStrike:
		return "double strike"
	case Trample:
		return "trample"
	case Vigilance:
		return "vigilance"
	case Haste:
		return "haste"
	case Lifelink:
		return "lifelink"
	case Deathtouch:
		return "deathtouch"
	case Reach:
		return "reach"
	case Defender:
		return "defender"
	case Menace:
		return "menace"
	case Hexproof:
		return "hexproof"
	case Indestructible:
		return "indestructible"
	case Flash:
		return "flash"
	}
	return "?"
}

// EffectType — the kinds of effects the generator can produce.
type EffectType int

const (
	DirectDamage EffectType = iota
	DamageAllCreatures
	DestroyCreature
	DestroyArtifact
	DestroyEnchantment
	DestroyLand
	Bounce
	CounterSpell
	DrawCards
	Discard
	GainLife
	DrainLife
	PumpSelf
	PumpTarget
	TapTarget
	UntapTarget
	AddMana
	GlobalBuff
	AuraBuff
	ExileCreature
	DestroyAllCreatures
	MassReturn
	GrantKeyword
	DamagePlayer
	AuraDebuff
)

// StatShape controls power/toughness distribution.
type StatShape int

const (
	Balanced StatShape = iota
	Aggressive
	Defensive
	Extreme
)

// CardType for the generated card.
type CardType int

const (
	CreatureType CardType = iota
	InstantType
	SorceryType
	EnchantmentType
)

func (ct CardType) String() string {
	switch ct {
	case CreatureType:
		return "creature"
	case InstantType:
		return "instant"
	case SorceryType:
		return "sorcery"
	case EnchantmentType:
		return "enchantment"
	}
	return "?"
}

// ManaCost represents a card's mana cost.
type ManaCost struct {
	Generic int
	White   int
	Blue    int
	Black   int
	Red     int
	Green   int
}

func (m ManaCost) CMC() int {
	return m.Generic + m.White + m.Blue + m.Black + m.Red + m.Green
}

func (m ManaCost) String() string {
	s := ""
	if m.Generic > 0 || m.CMC() == 0 {
		s += fmt.Sprintf("{%d}", m.Generic)
	}
	for i := 0; i < m.White; i++ {
		s += "{W}"
	}
	for i := 0; i < m.Blue; i++ {
		s += "{U}"
	}
	for i := 0; i < m.Black; i++ {
		s += "{B}"
	}
	for i := 0; i < m.Red; i++ {
		s += "{R}"
	}
	for i := 0; i < m.Green; i++ {
		s += "{G}"
	}
	return s
}

// Card is a generated card — data only, no engine integration.
type Card struct {
	Name      string
	ManaCost  ManaCost
	CardType  CardType
	Subtypes  []string
	Power     int
	Toughness int
	Keywords  []Keyword
	Abilities []string // text descriptions of abilities
}

func (c Card) CMC() int {
	return c.ManaCost.CMC()
}

func (c Card) IsCreature() bool    { return c.CardType == CreatureType }
func (c Card) IsInstant() bool     { return c.CardType == InstantType }
func (c Card) IsSorcery() bool     { return c.CardType == SorceryType }
func (c Card) IsEnchantment() bool { return c.CardType == EnchantmentType }

// CardTemplate determines the structural skeleton of a generated card.
type CardTemplate int

const (
	// Creatures
	Vanilla CardTemplate = iota
	FrenchVanilla
	ETBCreature
	ActivatedCreature
	Lord
	DrawbackCreature
	WallCreature
	ManaCreature
	EvasionCreature
	MultiKeywordCreature
	CantripCreature
	TapCreature
	// Spells
	Burn
	Removal
	CombatTrick
	DrawSpell
	Counterspell
	BounceSpell
	DiscardSpell
	LifeGainSpell
	DisenchantSpell
	MassDamage
	LandDestruction
	DrainSpell
	Wrath
	ExileRemoval
	PumpSwarm
	KeywordGrant
	DamagePlayerSpell
	MassBounce
	Cantrip
	// Enchantments
	GlobalBuffEnch
	AuraBuffEnch
	DebuffAura
	GlobalDebuff
	KeywordAura
)

var AllTemplates = []CardTemplate{
	Vanilla, FrenchVanilla, ETBCreature, ActivatedCreature,
	Lord, DrawbackCreature, WallCreature, ManaCreature,
	EvasionCreature, MultiKeywordCreature, CantripCreature, TapCreature,
	Burn, Removal, CombatTrick, DrawSpell, Counterspell,
	BounceSpell, DiscardSpell, LifeGainSpell, DisenchantSpell,
	MassDamage, LandDestruction, DrainSpell, Wrath, ExileRemoval,
	PumpSwarm, KeywordGrant, DamagePlayerSpell, MassBounce, Cantrip,
	GlobalBuffEnch, AuraBuffEnch, DebuffAura, GlobalDebuff, KeywordAura,
}

func (t CardTemplate) IsCreature() bool {
	return t >= Vanilla && t <= TapCreature
}

func (t CardTemplate) IsSpell() bool {
	return t >= Burn && t <= Cantrip
}

func (t CardTemplate) IsEnchantment() bool {
	return t >= GlobalBuffEnch && t <= KeywordAura
}

func (t CardTemplate) String() string {
	switch t {
	case Vanilla:
		return "vanilla"
	case FrenchVanilla:
		return "frenchVanilla"
	case ETBCreature:
		return "etbCreature"
	case ActivatedCreature:
		return "activatedCreature"
	case Lord:
		return "lord"
	case DrawbackCreature:
		return "drawbackCreature"
	case WallCreature:
		return "wallCreature"
	case ManaCreature:
		return "manaCreature"
	case EvasionCreature:
		return "evasionCreature"
	case MultiKeywordCreature:
		return "multiKeywordCreature"
	case CantripCreature:
		return "cantripCreature"
	case TapCreature:
		return "tapCreature"
	case Burn:
		return "burn"
	case Removal:
		return "removal"
	case CombatTrick:
		return "combatTrick"
	case DrawSpell:
		return "drawSpell"
	case Counterspell:
		return "counterspell"
	case BounceSpell:
		return "bounceSpell"
	case DiscardSpell:
		return "discardSpell"
	case LifeGainSpell:
		return "lifeGainSpell"
	case DisenchantSpell:
		return "disenchantSpell"
	case MassDamage:
		return "massDamage"
	case LandDestruction:
		return "landDestruction"
	case DrainSpell:
		return "drainSpell"
	case Wrath:
		return "wrath"
	case ExileRemoval:
		return "exileRemoval"
	case PumpSwarm:
		return "pumpSwarm"
	case KeywordGrant:
		return "keywordGrant"
	case DamagePlayerSpell:
		return "damagePlayer"
	case MassBounce:
		return "massBounce"
	case Cantrip:
		return "cantrip"
	case GlobalBuffEnch:
		return "globalBuff"
	case AuraBuffEnch:
		return "auraBuff"
	case DebuffAura:
		return "debuffAura"
	case GlobalDebuff:
		return "globalDebuff"
	case KeywordAura:
		return "keywordAura"
	}
	return "?"
}

// PrimaryEffectType returns the effect type this template uses, for color access checks.
func (t CardTemplate) PrimaryEffectType() (EffectType, bool) {
	switch t {
	case Burn:
		return DirectDamage, true
	case Removal:
		return DestroyCreature, true
	case CombatTrick:
		return PumpTarget, true
	case DrawSpell:
		return DrawCards, true
	case Counterspell:
		return CounterSpell, true
	case BounceSpell:
		return Bounce, true
	case DiscardSpell:
		return Discard, true
	case LifeGainSpell:
		return GainLife, true
	case DisenchantSpell:
		return DestroyEnchantment, true
	case MassDamage:
		return DamageAllCreatures, true
	case LandDestruction:
		return DestroyLand, true
	case DrainSpell:
		return DrainLife, true
	case GlobalBuffEnch:
		return GlobalBuff, true
	case AuraBuffEnch:
		return AuraBuff, true
	case Wrath:
		return DestroyAllCreatures, true
	case ExileRemoval:
		return ExileCreature, true
	case PumpSwarm:
		return GlobalBuff, true
	case KeywordGrant:
		return GrantKeyword, true
	case DamagePlayerSpell:
		return DamagePlayer, true
	case MassBounce:
		return MassReturn, true
	case Cantrip:
		return DrawCards, true
	case DebuffAura:
		return AuraDebuff, true
	case GlobalDebuff:
		return AuraDebuff, true
	case KeywordAura:
		return AuraBuff, true
	case ManaCreature:
		return AddMana, true
	case TapCreature:
		return TapTarget, true
	default:
		return 0, false
	}
}

// KeywordSet is a set of keywords.
type KeywordSet map[Keyword]bool

func NewKeywordSet() KeywordSet { return make(KeywordSet) }

func (ks KeywordSet) Add(k Keyword)      { ks[k] = true }
func (ks KeywordSet) Has(k Keyword) bool { return ks[k] }

func (ks KeywordSet) Slice() []Keyword {
	var out []Keyword
	for _, k := range AllKeywords {
		if ks[k] {
			out = append(out, k)
		}
	}
	return out
}

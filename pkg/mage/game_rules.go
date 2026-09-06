package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/catalog"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

// GameRules manages all game-rule modifier state: spell costs, mana conversion,
// untap limits, hand size, land plays, and per-player rule flags like Lich,
// Channel, Sanctuary, and minimum life. It is owned by EffectManager and
// exposed as a public field.
type GameRules struct {
	ManaConversion           map[Color]Color             // from color -> to color (Sunglasses of Urza)
	SpellCostIncreases       map[Color]int               // color -> additional generic cost for spells of that color
	SpellCostReductions      map[Color]int               // color -> generic cost reduction for spells of that color
	SpellTypeCostReductions  map[CardType]int            // type -> generic cost reduction for spells of that type
	LandUntapMax             int                         // -1 = no limit; >= 0 = max lands that may untap per turn
	ArtifactUntapMax         int                         // -1 = no limit; >= 0 = max artifacts that may untap per turn
	CreatureUntapMax         int                         // -1 = no limit; >= 0 = max creatures that may untap per turn
	UnlimitedLandPlays       bool                        // true if a player can play unlimited lands (Fastbond)
	sanctuaryActive          map[uuid.UUID]bool          // player -> if true, only flying/islandwalk can attack them
	lichActive               map[uuid.UUID]uuid.UUID     // player -> source permanent ID of active Lich
	skipNextDraw             map[uuid.UUID]bool          // player -> if true, skip normal draw in draw step
	channelActive            map[uuid.UUID]bool          // players with Channel active this turn
	minimumLife              map[uuid.UUID]bool          // players whose life can't go below 1 (Ali from Cairo)
	maxHandSize              map[uuid.UUID]int           // player -> max hand size override (Cursed Rack)
	expansionCastBlock       []string                    // set codes blocked from casting/playing
	NullifiedLandwalks       map[Attr]bool               // landwalk attrs that are nullified (Great Wall, etc.)
	ActivationCostReductions map[uuid.UUID]int           // permanent ID → generic mana reduction for activated abilities
	entersTappedRules        []func(*Permanent) bool     // filters registered by continuous effects (Kismet, etc.)
	SpellCostReducers        []SpellCostReducer          // conditional generic-cost reducers registered each Apply() cycle
	UncounterableFilters     []uncounterableEntry        // static "can't be countered" filters (Allosaurus Shepherd, Vexing Shusher)
	FlashGrants              []flashGrantEntry           // continuous "you may cast X spells as though they had flash" grants (Rattlechains, Vedalken Orrery, Leyline of Anticipation)
	cantCastSpells           map[uuid.UUID]bool          // players forbidden from casting spells this Apply() cycle (Angelic Arbiter, etc.)
	revealedTopCard          map[uuid.UUID]bool          // players playing with top card of library revealed (Future Sight, Oracle of Mul Daya). Per Apply() cycle.
	playLandsFromZones       map[uuid.UUID]map[Zone]bool // player -> zones (other than hand) from which lands may be played. Per Apply() cycle.
	additionalLandPlays      map[uuid.UUID]int           // per-cycle additional land-play allowance from static abilities (Azusa, Oracle of Mul Daya).
	cantPlayLands            bool                        // true if players can't play lands (Worms of the Earth)
	landsCantEnter           bool                        // true if lands can't enter the battlefield (Worms of the Earth)
	deepWaterActive          map[uuid.UUID]bool          // player -> if true, lands they control produce {U} instead of other types (Deep Water)
}

// flashGrantEntry holds a continuous flash-permission grant. Player is the
// player whose spells are affected; if Player == uuid.Nil the grant applies
// to every player. Filter restricts which cards in that player's hand are
// granted flash; a nil filter grants flash to every nonland card.
type flashGrantEntry struct {
	SourceID uuid.UUID
	Player   uuid.UUID
	Filter   CardFilter
}

// NewGameRules creates a GameRules with all maps initialized.
func NewGameRules() *GameRules {
	return &GameRules{
		LandUntapMax:             -1,
		ArtifactUntapMax:         -1,
		CreatureUntapMax:         -1,
		ManaConversion:           make(map[Color]Color),
		SpellCostIncreases:       make(map[Color]int),
		SpellCostReductions:      make(map[Color]int),
		SpellTypeCostReductions:  make(map[CardType]int),
		sanctuaryActive:          make(map[uuid.UUID]bool),
		lichActive:               make(map[uuid.UUID]uuid.UUID),
		skipNextDraw:             make(map[uuid.UUID]bool),
		channelActive:            make(map[uuid.UUID]bool),
		minimumLife:              make(map[uuid.UUID]bool),
		maxHandSize:              make(map[uuid.UUID]int),
		NullifiedLandwalks:       make(map[Attr]bool),
		ActivationCostReductions: make(map[uuid.UUID]int),
		deepWaterActive:          make(map[uuid.UUID]bool),
	}
}

// ResetPerCycle resets all state that is recomputed each Apply() cycle.
func (r *GameRules) ResetPerCycle() {
	r.LandUntapMax = -1
	r.ArtifactUntapMax = -1
	r.CreatureUntapMax = -1
	r.UnlimitedLandPlays = false
	clear(r.maxHandSize)
	clear(r.SpellCostIncreases)
	clear(r.SpellCostReductions)
	clear(r.SpellTypeCostReductions)
	clear(r.ManaConversion)
	clear(r.minimumLife)
	r.expansionCastBlock = r.expansionCastBlock[:0]
	clear(r.NullifiedLandwalks)
	clear(r.ActivationCostReductions)
	r.entersTappedRules = r.entersTappedRules[:0]
	r.SpellCostReducers = r.SpellCostReducers[:0]
	r.UncounterableFilters = r.UncounterableFilters[:0]
	r.FlashGrants = r.FlashGrants[:0]
	clear(r.cantCastSpells)
	clear(r.revealedTopCard)
	clear(r.playLandsFromZones)
	clear(r.additionalLandPlays)
	r.cantPlayLands = false
	r.landsCantEnter = false
}

// AddRevealedTopCard marks playerID as playing with the top card of their
// library revealed for this Apply() cycle. Cleared by ResetPerCycle.
func (r *GameRules) AddRevealedTopCard(playerID uuid.UUID) {
	if r.revealedTopCard == nil {
		r.revealedTopCard = make(map[uuid.UUID]bool)
	}
	r.revealedTopCard[playerID] = true
}

// IsTopCardRevealed reports whether the given player is currently playing
// with the top card of their library revealed.
func (r *GameRules) IsTopCardRevealed(playerID uuid.UUID) bool {
	return r.revealedTopCard[playerID]
}

// AddPlayLandsFromZone permits playerID to play lands from the given zone
// (in addition to their hand) for this Apply() cycle. Cleared by
// ResetPerCycle.
func (r *GameRules) AddPlayLandsFromZone(playerID uuid.UUID, zone Zone) {
	if r.playLandsFromZones == nil {
		r.playLandsFromZones = make(map[uuid.UUID]map[Zone]bool)
	}
	zones := r.playLandsFromZones[playerID]
	if zones == nil {
		zones = make(map[Zone]bool)
		r.playLandsFromZones[playerID] = zones
	}
	zones[zone] = true
}

// CanPlayLandsFromZone reports whether playerID may currently play lands
// from the given zone. Hand is always permitted; other zones require an
// active continuous-effect grant.
func (r *GameRules) CanPlayLandsFromZone(playerID uuid.UUID, zone Zone) bool {
	if zone == ZoneHand {
		return true
	}
	zones := r.playLandsFromZones[playerID]
	if zones == nil {
		return false
	}
	return zones[zone]
}

// AddAdditionalLandPlay registers an additional land play allowance for the
// player from a static ability for this Apply() cycle. This stacks with
// per-turn grants from spells (GrantExtraLandPlay) and is recomputed each
// Apply() so it auto-clears with the source.
func (r *GameRules) AddAdditionalLandPlay(playerID uuid.UUID, n int) {
	if n <= 0 {
		return
	}
	if r.additionalLandPlays == nil {
		r.additionalLandPlays = make(map[uuid.UUID]int)
	}
	r.additionalLandPlays[playerID] += n
}

// AdditionalLandPlays returns the per-cycle static additional land-play
// allowance for the given player.
func (r *GameRules) AdditionalLandPlays(playerID uuid.UUID) int {
	return r.additionalLandPlays[playerID]
}

// AddCantCastSpells registers a continuous "this player can't cast spells"
// rule for this Apply() cycle. Used by Angelic Arbiter and similar global
// restrictions. Cleared by ResetPerCycle.
func (r *GameRules) AddCantCastSpells(playerID uuid.UUID) {
	if r.cantCastSpells == nil {
		r.cantCastSpells = make(map[uuid.UUID]bool)
	}
	r.cantCastSpells[playerID] = true
}

// PlayerCantCastSpells reports whether the given player is currently
// forbidden from casting spells by an active continuous effect.
func (r *GameRules) PlayerCantCastSpells(playerID uuid.UUID) bool {
	return r.cantCastSpells[playerID]
}

// AddFlashGrant registers a continuous "may cast as though it had flash"
// permission for this Apply() cycle. If player == uuid.Nil the grant
// applies globally; if filter == nil it applies to every nonland card. The
// grant is keyed by sourceID so it is dropped automatically when the source
// permanent leaves the battlefield (continuous-effect framework drops the
// effect, ResetPerCycle clears the slice, and Apply re-registers it only if
// the source is still active).
func (r *GameRules) AddFlashGrant(sourceID, player uuid.UUID, filter CardFilter) {
	r.FlashGrants = append(r.FlashGrants, flashGrantEntry{SourceID: sourceID, Player: player, Filter: filter})
}

// HasFlashGrant returns true if any active grant lets the given player cast
// the given card as though it had flash. A grant whose Filter is the zero
// value matches all cards (CR semantics for "may cast as though it had
// flash" when no card-type restriction is named).
func (r *GameRules) HasFlashGrant(player uuid.UUID, card Card) bool {
	for _, g := range r.FlashGrants {
		if g.Player != uuid.Nil && g.Player != player {
			continue
		}
		if g.Filter.Match(card) {
			return true
		}
	}
	return false
}

// AddUncounterableFilter registers a "can't be countered" filter for this
// Apply() cycle. Cleared by ResetPerCycle.
func (r *GameRules) AddUncounterableFilter(sourceID uuid.UUID, filter UncounterableFilter) {
	r.UncounterableFilters = append(r.UncounterableFilters, uncounterableEntry{SourceID: sourceID, Filter: filter})
}

// AddSpellCostReducer registers a conditional spell-cost reducer for this
// Apply() cycle. Cleared by ResetPerCycle.
func (r *GameRules) AddSpellCostReducer(red SpellCostReducer) {
	r.SpellCostReducers = append(r.SpellCostReducers, red)
}

// AddEntersTappedRule registers a filter that causes matching permanents to enter tapped.
func (r *GameRules) AddEntersTappedRule(f func(*Permanent) bool) {
	r.entersTappedRules = append(r.entersTappedRules, f)
}

// ShouldEnterTapped returns true if any registered rule says this permanent enters tapped.
func (r *GameRules) ShouldEnterTapped(perm *Permanent) bool {
	for _, f := range r.entersTappedRules {
		if f(perm) {
			return true
		}
	}
	return false
}

// ClearEndOfTurn resets all turn-scoped game rule state.
func (r *GameRules) ClearEndOfTurn() {
	r.channelActive = make(map[uuid.UUID]bool)
	r.deepWaterActive = make(map[uuid.UUID]bool)
}

// SyncManaConversions syncs mana conversion state to all player mana pools.
func (r *GameRules) SyncManaConversions(players []Player) {
	for _, p := range players {
		if len(r.ManaConversion) > 0 {
			p.ManaPool().ManaConversions = r.ManaConversion
		} else {
			p.ManaPool().ManaConversions = nil
		}
	}
}

// ---------------------------------------------------------------------------
// Sanctuary (Island Sanctuary)
// ---------------------------------------------------------------------------

// SetSanctuaryActive marks a player as protected by Island Sanctuary.
func (r *GameRules) SetSanctuaryActive(playerID uuid.UUID) {
	r.sanctuaryActive[playerID] = true
}

// IsSanctuaryActive returns true if the player is protected by Island Sanctuary.
func (r *GameRules) IsSanctuaryActive(playerID uuid.UUID) bool {
	return r.sanctuaryActive[playerID]
}

// ClearSanctuary clears Island Sanctuary protection for a player.
func (r *GameRules) ClearSanctuary(playerID uuid.UUID) {
	delete(r.sanctuaryActive, playerID)
}

// ---------------------------------------------------------------------------
// Lich
// ---------------------------------------------------------------------------

// SetLichActive records the source permanent ID of a Lich controlled by playerID.
func (r *GameRules) SetLichActive(playerID, sourceID uuid.UUID) {
	r.lichActive[playerID] = sourceID
}

// IsLichActive returns true if the player has an active Lich still on the battlefield.
func (r *GameRules) IsLichActive(g *Game, playerID uuid.UUID) bool {
	sourceID, ok := r.lichActive[playerID]
	if !ok {
		return false
	}
	return g.FindPermanent(sourceID) != nil
}

// ClearLich clears Lich replacement effects for a player.
func (r *GameRules) ClearLich(playerID uuid.UUID) {
	delete(r.lichActive, playerID)
}

// ---------------------------------------------------------------------------
// Skip Draw
// ---------------------------------------------------------------------------

// SetSkipNextDraw marks a player to skip their next draw step draw.
func (r *GameRules) SetSkipNextDraw(playerID uuid.UUID) {
	r.skipNextDraw[playerID] = true
}

// ShouldSkipDraw returns true and clears the flag if the player should skip their draw.
func (r *GameRules) ShouldSkipDraw(playerID uuid.UUID) bool {
	if r.skipNextDraw[playerID] {
		delete(r.skipNextDraw, playerID)
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Mana Conversion
// ---------------------------------------------------------------------------

// SetManaConversion sets a mana color conversion (e.g. Red→White for Sunglasses of Urza).
func (r *GameRules) SetManaConversion(from, to Color) {
	r.ManaConversion[from] = to
}

// GetManaConversion returns the converted color for a given color, if any.
func (r *GameRules) GetManaConversion(from Color) (Color, bool) {
	to, ok := r.ManaConversion[from]
	return to, ok
}

// ClearManaConversion clears all mana conversions.
func (r *GameRules) ClearManaConversion() {
	r.ManaConversion = make(map[Color]Color)
}

// ---------------------------------------------------------------------------
// Channel
// ---------------------------------------------------------------------------

// SetChannelActive marks a player as having Channel active this turn.
func (r *GameRules) SetChannelActive(playerID uuid.UUID) {
	r.channelActive[playerID] = true
}

// IsChannelActive returns true if the player has Channel active.
func (r *GameRules) IsChannelActive(playerID uuid.UUID) bool {
	return r.channelActive[playerID]
}

// ClearChannelActive clears Channel state (called at end of turn).
func (r *GameRules) ClearChannelActive() {
	r.channelActive = make(map[uuid.UUID]bool)
}

// ---------------------------------------------------------------------------
// Minimum Life (Ali from Cairo)
// ---------------------------------------------------------------------------

// SetMinimumLife marks a player as having minimum-life protection (Ali from Cairo).
func (r *GameRules) SetMinimumLife(playerID uuid.UUID) {
	r.minimumLife[playerID] = true
}

// IsMinimumLifeActive returns true if the player's life can't go below 1.
func (r *GameRules) IsMinimumLifeActive(playerID uuid.UUID) bool {
	return r.minimumLife[playerID]
}

// ClearMinimumLife resets minimum-life state (called when effect source leaves).
func (r *GameRules) ClearMinimumLife() {
	r.minimumLife = make(map[uuid.UUID]bool)
}

// ---------------------------------------------------------------------------
// Expansion Blocking (City in a Bottle, Golgothian Sylex)
// ---------------------------------------------------------------------------

// AddExpansionCastBlock registers a set code as blocked from casting/playing.
func (r *GameRules) AddExpansionCastBlock(setCode string) {
	r.expansionCastBlock = append(r.expansionCastBlock, setCode)
}

// IsCardExpansionBlocked returns true if the named card belongs to any blocked set.
func (r *GameRules) IsCardExpansionBlocked(cardName string) bool {
	for _, setCode := range r.expansionCastBlock {
		if catalog.Global().CardInSet(setCode, cardName) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Hand Size
// ---------------------------------------------------------------------------

// SetMaxHandSize sets the max hand size override for a player.
func (r *GameRules) SetMaxHandSize(playerID uuid.UUID, size int) {
	r.maxHandSize[playerID] = size
}

// SetNoMaximumHandSize removes the player's maximum hand size while the
// continuous effect that calls it remains active.
func (r *GameRules) SetNoMaximumHandSize(playerID uuid.UUID) {
	r.maxHandSize[playerID] = -1
}

// MaxHandSize returns the max hand size for a player (default 7).
func (r *GameRules) MaxHandSize(playerID uuid.UUID) int {
	if size, ok := r.maxHandSize[playerID]; ok {
		return size
	}
	return 7
}

// ---------------------------------------------------------------------------
// Spell Cost Queries
// ---------------------------------------------------------------------------

// SpellCostIncrease returns the additional generic cost for spells of the given color.
func (r *GameRules) SpellCostIncrease(c Color) int {
	return r.SpellCostIncreases[c]
}

// SpellCostReduction returns the generic cost reduction for spells of the given color.
func (r *GameRules) SpellCostReduction(c Color) int {
	return r.SpellCostReductions[c]
}

// SpellTypeCostReduction returns the generic cost reduction for spells of the given type.
func (r *GameRules) SpellTypeCostReduction(t CardType) int {
	return r.SpellTypeCostReductions[t]
}

// ---------------------------------------------------------------------------
// Landwalk Nullification (Great Wall, Crevasse, Deadfall, Quagmire, Undertow)
// ---------------------------------------------------------------------------

// NullifyLandwalk marks a landwalk attr as nullified (can be blocked as though
// the creature didn't have it).
func (r *GameRules) NullifyLandwalk(kw Attr) {
	r.NullifiedLandwalks[kw] = true
}

// IsLandwalkNullified returns true if the given landwalk attr is nullified.
func (r *GameRules) IsLandwalkNullified(kw Attr) bool {
	return r.NullifiedLandwalks[kw]
}

// ---------------------------------------------------------------------------
// Land Entry Restrictions (Worms of the Earth)
// ---------------------------------------------------------------------------

// SetCantPlayLands sets whether players can play lands during this Apply() cycle.
func (r *GameRules) SetCantPlayLands(val bool) {
	r.cantPlayLands = val
}

// CantPlayLands reports whether players are prohibited from playing lands.
func (r *GameRules) CantPlayLands() bool {
	return r.cantPlayLands
}

// SetLandsCantEnter sets whether lands cannot enter the battlefield during this Apply() cycle.
func (r *GameRules) SetLandsCantEnter(val bool) {
	r.landsCantEnter = val
}

// LandsCantEnter reports whether lands are prohibited from entering the battlefield.
func (r *GameRules) LandsCantEnter() bool {
	return r.landsCantEnter
}

// ---------------------------------------------------------------------------
// Deep Water
// ---------------------------------------------------------------------------

// SetDeepWaterActive sets whether lands controlled by playerID produce {U} this turn.
func (r *GameRules) SetDeepWaterActive(playerID uuid.UUID) {
	if r.deepWaterActive == nil {
		r.deepWaterActive = make(map[uuid.UUID]bool)
	}
	r.deepWaterActive[playerID] = true
}

// IsDeepWaterActive reports whether Deep Water is active for playerID.
func (r *GameRules) IsDeepWaterActive(playerID uuid.UUID) bool {
	return r.deepWaterActive != nil && r.deepWaterActive[playerID]
}

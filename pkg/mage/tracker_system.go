package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TurnTrackers encapsulates observations and bookkeeping of actions completed within a turn.
// All turn trackers are reset between turns during cleanup (with selected history such as
// attackedLastTurn preserved across turns).
type TurnTrackers struct {
	discardCount            map[uuid.UUID]int
	lifeGained              map[uuid.UUID]int
	permDamageReceived      map[uuid.UUID]int
	attackedOrBlocked       map[uuid.UUID]bool
	playerCastSpell         map[uuid.UUID]bool
	playerAttacked          map[uuid.UUID]bool
	cardsDrawn              map[uuid.UUID]int
	cardsLeftGraveyard      map[uuid.UUID]int
	cardsPutIntoExile       int
	exileZoneChangesPending map[uuid.UUID]int

	landsPlayed      int
	extraLandPlays   map[uuid.UUID]int
	optionalCostPaid map[uuid.UUID]bool

	damageTaken         map[uuid.UUID]int
	artifactDamageTaken map[uuid.UUID]int

	attacked         map[uuid.UUID]bool
	attackedLastTurn map[uuid.UUID]map[uuid.UUID]bool
	blocked          map[uuid.UUID][]uuid.UUID

	instantsCast  map[uuid.UUID]int
	sorceriesCast map[uuid.UUID]int

	creatureDeaths           int
	untappedLandsAtTurnStart map[uuid.UUID]int
	timesTargeted            map[uuid.UUID]int
	cleanupPriorityRounds    int
}

// NewTurnTrackers creates initialized TurnTrackers.
func NewTurnTrackers() TurnTrackers {
	return TurnTrackers{
		damageTaken:         make(map[uuid.UUID]int),
		artifactDamageTaken: make(map[uuid.UUID]int),
		attacked:            make(map[uuid.UUID]bool),
		blocked:             make(map[uuid.UUID][]uuid.UUID),
		instantsCast:        make(map[uuid.UUID]int),
		sorceriesCast:       make(map[uuid.UUID]int),
		timesTargeted:       make(map[uuid.UUID]int),
	}
}

// RecordEvent updates turn trackers from an emitted GameEvent.
func (t *TurnTrackers) RecordEvent(evt *GameEvent) {
	switch evt.Type {
	case EvtZoneChange:
		if evt.ToZone != ZoneExile || evt.SourceID == uuid.Nil {
			return
		}
		t.cardsPutIntoExile++
		if t.exileZoneChangesPending == nil {
			t.exileZoneChangesPending = make(map[uuid.UUID]int)
		}
		t.exileZoneChangesPending[evt.SourceID]++
	case EvtDiscard:
		if t.discardCount == nil {
			t.discardCount = make(map[uuid.UUID]int)
		}
		t.discardCount[evt.PlayerID]++
	case EvtLifeGained:
		if t.lifeGained == nil {
			t.lifeGained = make(map[uuid.UUID]int)
		}
		amt := max(evt.Amount, 0)
		t.lifeGained[evt.PlayerID] += amt
	case EvtDamageDealt:
		if evt.TargetID == uuid.Nil {
			return
		}
		if t.permDamageReceived == nil {
			t.permDamageReceived = make(map[uuid.UUID]int)
		}
		amt := max(evt.Amount, 0)
		t.permDamageReceived[evt.TargetID] += amt
	case EvtDeclaredBlocker:
		if !evt.Flag {
			return
		}
		if t.attackedOrBlocked == nil {
			t.attackedOrBlocked = make(map[uuid.UUID]bool)
		}
		t.attackedOrBlocked[evt.SourceID] = true
	case EvtDeclaredAttacker:
		if t.attackedOrBlocked == nil {
			t.attackedOrBlocked = make(map[uuid.UUID]bool)
		}
		t.attackedOrBlocked[evt.SourceID] = true
		if t.playerAttacked == nil {
			t.playerAttacked = make(map[uuid.UUID]bool)
		}
		t.playerAttacked[evt.PlayerID] = true
	case EvtCardDrawn:
		if t.cardsDrawn == nil {
			t.cardsDrawn = make(map[uuid.UUID]int)
		}
		t.cardsDrawn[evt.PlayerID]++
	case EvtCardsLeftGraveyard:
		if t.cardsLeftGraveyard == nil {
			t.cardsLeftGraveyard = make(map[uuid.UUID]int)
		}
		amt := evt.Amount
		if amt <= 0 {
			amt = 1
		}
		t.cardsLeftGraveyard[evt.PlayerID] += amt
	case EvtSpellCast:
		if t.playerCastSpell == nil {
			t.playerCastSpell = make(map[uuid.UUID]bool)
		}
		t.playerCastSpell[evt.PlayerID] = true
	}
}

// RecordCardPutIntoExile updates the exile tracker.
func (t *TurnTrackers) RecordCardPutIntoExile(card Card) {
	if card == nil {
		return
	}
	if t.consumePendingExileZoneChange(card.ID()) {
		return
	}
	t.cardsPutIntoExile++
}

func (t *TurnTrackers) consumePendingExileZoneChange(cardID uuid.UUID) bool {
	if cardID == uuid.Nil || t.exileZoneChangesPending == nil {
		return false
	}
	n := t.exileZoneChangesPending[cardID]
	if n <= 0 {
		return false
	}
	if n == 1 {
		delete(t.exileZoneChangesPending, cardID)
	} else {
		t.exileZoneChangesPending[cardID] = n - 1
	}
	return true
}

// Recording helpers

func (t *TurnTrackers) RecordDamageTaken(playerID uuid.UUID, amt int, isArtifact bool) {
	if t.damageTaken == nil {
		t.damageTaken = make(map[uuid.UUID]int)
	}
	t.damageTaken[playerID] += amt
	if isArtifact {
		if t.artifactDamageTaken == nil {
			t.artifactDamageTaken = make(map[uuid.UUID]int)
		}
		t.artifactDamageTaken[playerID] += amt
	}
}

func (t *TurnTrackers) RecordAttacked(permID uuid.UUID) {
	if t.attacked == nil {
		t.attacked = make(map[uuid.UUID]bool)
	}
	t.attacked[permID] = true
}

func (t *TurnTrackers) RecordBlocked(blockerID, attackerID uuid.UUID) {
	if t.blocked == nil {
		t.blocked = make(map[uuid.UUID][]uuid.UUID)
	}
	t.blocked[blockerID] = append(t.blocked[blockerID], attackerID)
}

func (t *TurnTrackers) RemoveBlockedAttacker(blockerID, attackerID uuid.UUID) {
	if t.blocked == nil {
		return
	}
	rest := t.blocked[blockerID][:0]
	for _, aid := range t.blocked[blockerID] {
		if aid != attackerID {
			rest = append(rest, aid)
		}
	}
	t.blocked[blockerID] = rest
}

func (t *TurnTrackers) RecordInstantCast(playerID uuid.UUID) {
	if t.instantsCast == nil {
		t.instantsCast = make(map[uuid.UUID]int)
	}
	t.instantsCast[playerID]++
}

func (t *TurnTrackers) RecordSorceryCast(playerID uuid.UUID) {
	if t.sorceriesCast == nil {
		t.sorceriesCast = make(map[uuid.UUID]int)
	}
	t.sorceriesCast[playerID]++
}

func (t *TurnTrackers) RecordTimesTargeted(id uuid.UUID) {
	if t.timesTargeted == nil {
		t.timesTargeted = make(map[uuid.UUID]int)
	}
	t.timesTargeted[id]++
}

func (t *TurnTrackers) RecordCreatureDeath() {
	t.creatureDeaths++
}

func (t *TurnTrackers) RecordLandPlayed() {
	t.landsPlayed++
}

func (t *TurnTrackers) SetLandsPlayed(n int) {
	t.landsPlayed = n
}

func (t *TurnTrackers) ResetLandsPlayed() {
	t.landsPlayed = 0
	t.extraLandPlays = nil
}

func (t *TurnTrackers) AddExtraLandPlays(playerID uuid.UUID, n int) {
	if t.extraLandPlays == nil {
		t.extraLandPlays = make(map[uuid.UUID]int)
	}
	t.extraLandPlays[playerID] += n
}

func (t *TurnTrackers) SetOptionalCostPaid(sourceID uuid.UUID, paid bool) {
	if t.optionalCostPaid == nil {
		t.optionalCostPaid = make(map[uuid.UUID]bool)
	}
	t.optionalCostPaid[sourceID] = paid
}

func (t *TurnTrackers) ClearOptionalCostPaid(sourceID uuid.UUID) {
	if t.optionalCostPaid == nil {
		return
	}
	delete(t.optionalCostPaid, sourceID)
}

func (t *TurnTrackers) SetUntappedLandsAtTurnStart(playerID uuid.UUID, count int) {
	if t.untappedLandsAtTurnStart == nil {
		t.untappedLandsAtTurnStart = make(map[uuid.UUID]int)
	}
	t.untappedLandsAtTurnStart[playerID] = count
}

func (t *TurnTrackers) IncrementCleanupPriorityRounds() {
	t.cleanupPriorityRounds++
}

// ResetForNewTurn clears per-turn trackers and archives the active player's attacker history.
func (t *TurnTrackers) ResetForNewTurn(activePlayerID uuid.UUID) {
	if t.attackedLastTurn == nil {
		t.attackedLastTurn = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	lastMap := make(map[uuid.UUID]bool, len(t.attacked))
	maps.Copy(lastMap, t.attacked)
	t.attackedLastTurn[activePlayerID] = lastMap

	t.discardCount = nil
	t.lifeGained = nil
	t.permDamageReceived = nil
	t.attackedOrBlocked = nil
	t.playerCastSpell = nil
	t.playerAttacked = nil
	t.cardsDrawn = nil
	t.cardsLeftGraveyard = nil
	t.cardsPutIntoExile = 0
	t.exileZoneChangesPending = nil

	t.landsPlayed = 0
	t.extraLandPlays = nil
	t.optionalCostPaid = nil

	t.damageTaken = make(map[uuid.UUID]int)
	t.artifactDamageTaken = make(map[uuid.UUID]int)

	t.attacked = make(map[uuid.UUID]bool)
	t.blocked = make(map[uuid.UUID][]uuid.UUID)
	t.instantsCast = make(map[uuid.UUID]int)
	t.sorceriesCast = make(map[uuid.UUID]int)
	t.timesTargeted = make(map[uuid.UUID]int)
	t.creatureDeaths = 0
}

// Queries

func (t *TurnTrackers) PlayerCardsDrawn(playerID uuid.UUID) int {
	return t.cardsDrawn[playerID]
}

func (t *TurnTrackers) PlayerCardsLeftGraveyard(playerID uuid.UUID) int {
	return t.cardsLeftGraveyard[playerID]
}

func (t *TurnTrackers) PlayerHadCardLeaveGraveyard(playerID uuid.UUID) bool {
	return t.PlayerCardsLeftGraveyard(playerID) > 0
}

func (t *TurnTrackers) CardsPutIntoExile() int {
	return t.cardsPutIntoExile
}

func (t *TurnTrackers) PlayerDiscardCount(playerID uuid.UUID) int {
	return t.discardCount[playerID]
}

func (t *TurnTrackers) PlayerLifeGained(playerID uuid.UUID) int {
	return t.lifeGained[playerID]
}

func (t *TurnTrackers) PermanentDamageReceived(permID uuid.UUID) int {
	return t.permDamageReceived[permID]
}

func (t *TurnTrackers) PermanentAttackedOrBlocked(permID uuid.UUID) bool {
	return t.attackedOrBlocked[permID]
}

func (t *TurnTrackers) PlayerCastSpell(playerID uuid.UUID) bool {
	return t.playerCastSpell[playerID]
}

func (t *TurnTrackers) PlayerAttacked(playerID uuid.UUID) bool {
	return t.playerAttacked[playerID]
}

func (t *TurnTrackers) LandsPlayed() int {
	return t.landsPlayed
}

func (t *TurnTrackers) ExtraLandPlays(playerID uuid.UUID) int {
	return t.extraLandPlays[playerID]
}

func (t *TurnTrackers) OptionalCostPaid(sourceID uuid.UUID) bool {
	return t.optionalCostPaid[sourceID]
}

func (t *TurnTrackers) DamageTaken(playerID uuid.UUID) int {
	return t.damageTaken[playerID]
}

func (t *TurnTrackers) ArtifactDamageTaken(playerID uuid.UUID) int {
	return t.artifactDamageTaken[playerID]
}

func (t *TurnTrackers) Attacked(permID uuid.UUID) bool {
	return t.attacked[permID]
}

func (t *TurnTrackers) AttackedMap() map[uuid.UUID]bool {
	return t.attacked
}

func (t *TurnTrackers) AttackedLastTurn(playerID uuid.UUID) map[uuid.UUID]bool {
	return t.attackedLastTurn[playerID]
}

func (t *TurnTrackers) Blocked(blockerID uuid.UUID) []uuid.UUID {
	return t.blocked[blockerID]
}

func (t *TurnTrackers) BlockedMap() map[uuid.UUID][]uuid.UUID {
	return t.blocked
}

func (t *TurnTrackers) InstantsCast(playerID uuid.UUID) int {
	return t.instantsCast[playerID]
}

func (t *TurnTrackers) SorceriesCast(playerID uuid.UUID) int {
	return t.sorceriesCast[playerID]
}

func (t *TurnTrackers) CreatureDeaths() int {
	return t.creatureDeaths
}

func (t *TurnTrackers) UntappedLandsAtTurnStart(playerID uuid.UUID) int {
	return t.untappedLandsAtTurnStart[playerID]
}

func (t *TurnTrackers) TimesTargeted(id uuid.UUID) int {
	return t.timesTargeted[id]
}

func (t *TurnTrackers) CleanupPriorityRounds() int {
	return t.cleanupPriorityRounds
}

// Clone creates an independent deep copy of TurnTrackers.
func (t *TurnTrackers) Clone() TurnTrackers {
	c := TurnTrackers{
		cardsPutIntoExile:        t.cardsPutIntoExile,
		landsPlayed:              t.landsPlayed,
		creatureDeaths:           t.creatureDeaths,
		cleanupPriorityRounds:    t.cleanupPriorityRounds,
		discardCount:             cloneUUIDMap(t.discardCount),
		lifeGained:               cloneUUIDMap(t.lifeGained),
		permDamageReceived:       cloneUUIDMap(t.permDamageReceived),
		attackedOrBlocked:        cloneUUIDMap(t.attackedOrBlocked),
		playerCastSpell:          cloneUUIDMap(t.playerCastSpell),
		playerAttacked:           cloneUUIDMap(t.playerAttacked),
		cardsDrawn:               cloneUUIDMap(t.cardsDrawn),
		cardsLeftGraveyard:       cloneUUIDMap(t.cardsLeftGraveyard),
		exileZoneChangesPending:  cloneUUIDMap(t.exileZoneChangesPending),
		extraLandPlays:           cloneUUIDMap(t.extraLandPlays),
		optionalCostPaid:         cloneUUIDMap(t.optionalCostPaid),
		damageTaken:              cloneUUIDIntMap(t.damageTaken),
		artifactDamageTaken:      cloneUUIDIntMap(t.artifactDamageTaken),
		attacked:                 cloneUUIDBoolMap(t.attacked),
		blocked:                  cloneBlockedThisTurn(t.blocked),
		instantsCast:             cloneUUIDIntMap(t.instantsCast),
		sorceriesCast:            cloneUUIDIntMap(t.sorceriesCast),
		untappedLandsAtTurnStart: cloneUUIDMap(t.untappedLandsAtTurnStart),
		timesTargeted:            cloneUUIDMap(t.timesTargeted),
	}
	if len(t.attackedLastTurn) > 0 {
		c.attackedLastTurn = cloneNestedUUIDMap(t.attackedLastTurn)
	}
	return c
}

// DuelTrackers encapsulates objective tracking accumulated across the entire duel.
type DuelTrackers struct {
	spellsCastByColor map[uuid.UUID]map[Color]int
	spellsCastByType  map[uuid.UUID]map[CardType]int
	landsPlayed       map[uuid.UUID]int
	attackersDeclared map[uuid.UUID]int
	creatureDeaths    map[uuid.UUID]int
	nonCombatDamage   map[uuid.UUID]int
}

// NewDuelTrackers creates initialized DuelTrackers.
func NewDuelTrackers() DuelTrackers {
	return DuelTrackers{}
}

// RecordEvent updates duel objective trackers from an emitted GameEvent.
func (d *DuelTrackers) RecordEvent(evt *GameEvent, cardLookup func(uuid.UUID) Card, getOpponent func(uuid.UUID) uuid.UUID) {
	switch evt.Type {
	case EvtSpellCast:
		card := cardLookup(evt.SourceID)
		if card == nil {
			return
		}
		if d.spellsCastByColor == nil {
			d.spellsCastByColor = make(map[uuid.UUID]map[Color]int)
		}
		byColor := d.spellsCastByColor[evt.PlayerID]
		if byColor == nil {
			byColor = make(map[Color]int)
			d.spellsCastByColor[evt.PlayerID] = byColor
		}
		for _, c := range card.ManaCost().Colors() {
			byColor[c]++
		}

		if d.spellsCastByType == nil {
			d.spellsCastByType = make(map[uuid.UUID]map[CardType]int)
		}
		byType := d.spellsCastByType[evt.PlayerID]
		if byType == nil {
			byType = make(map[CardType]int)
			d.spellsCastByType[evt.PlayerID] = byType
		}
		for _, t := range []CardType{TypeCreature, TypeInstant, TypeSorcery, TypeArtifact, TypeEnchantment, TypePlaneswalker} {
			if card.HasType(t) {
				byType[t]++
			}
		}
	case EvtLandPlayed:
		if d.landsPlayed == nil {
			d.landsPlayed = make(map[uuid.UUID]int)
		}
		d.landsPlayed[evt.PlayerID]++
	case EvtDeclaredAttacker:
		if d.attackersDeclared == nil {
			d.attackersDeclared = make(map[uuid.UUID]int)
		}
		d.attackersDeclared[evt.PlayerID]++
	case EvtDamageDealt:
		if evt.Flag || evt.Amount <= 0 {
			return
		}
		dealerID := getOpponent(evt.TargetID)
		if dealerID == uuid.Nil {
			return
		}
		if d.nonCombatDamage == nil {
			d.nonCombatDamage = make(map[uuid.UUID]int)
		}
		d.nonCombatDamage[dealerID] += evt.Amount
	}
}

// RecordCreatureDeath records a creature death in the duel.
func (d *DuelTrackers) RecordCreatureDeath(controllerID uuid.UUID) {
	if d.creatureDeaths == nil {
		d.creatureDeaths = make(map[uuid.UUID]int)
	}
	d.creatureDeaths[controllerID]++
}

// ObjectivesFor returns the DuelObjectives snapshot for the player.
func (d *DuelTrackers) ObjectivesFor(playerID, opponentID uuid.UUID) DuelObjectives {
	obj := DuelObjectives{
		SpellsByColor:        make(map[Color]int),
		SpellsByType:         make(map[CardType]int),
		LandsPlayed:          d.landsPlayed[playerID],
		AttackersDeclared:    d.attackersDeclared[playerID],
		NonCombatDamageDealt: d.nonCombatDamage[playerID],
	}
	if d.spellsCastByColor != nil {
		maps.Copy(obj.SpellsByColor, d.spellsCastByColor[playerID])
	}
	if d.spellsCastByType != nil {
		maps.Copy(obj.SpellsByType, d.spellsCastByType[playerID])
	}
	if opponentID != uuid.Nil && d.creatureDeaths != nil {
		obj.OpponentCreaturesDestroyed = d.creatureDeaths[opponentID]
	}
	return obj
}

// Clone creates an independent deep copy of DuelTrackers.
func (d *DuelTrackers) Clone() DuelTrackers {
	return DuelTrackers{
		landsPlayed:       cloneUUIDMap(d.landsPlayed),
		attackersDeclared: cloneUUIDMap(d.attackersDeclared),
		creatureDeaths:    cloneUUIDMap(d.creatureDeaths),
		nonCombatDamage:   cloneUUIDMap(d.nonCombatDamage),
		spellsCastByColor: cloneUUIDColorMap(d.spellsCastByColor),
		spellsCastByType:  cloneUUIDCardTypeMap(d.spellsCastByType),
	}
}

// TrackerSystem groups TurnTrackers and DuelTrackers.
type TrackerSystem struct {
	Turn TurnTrackers
	Duel DuelTrackers
}

// NewTrackerSystem creates an initialized TrackerSystem.
func NewTrackerSystem() TrackerSystem {
	return TrackerSystem{
		Turn: NewTurnTrackers(),
		Duel: NewDuelTrackers(),
	}
}

// Clone creates an independent deep copy of TrackerSystem.
func (ts *TrackerSystem) Clone() TrackerSystem {
	return TrackerSystem{
		Turn: ts.Turn.Clone(),
		Duel: ts.Duel.Clone(),
	}
}

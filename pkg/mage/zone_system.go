package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

// ZoneSystem manages the battlefield, exile zone, Last-Known Information (LKI),
// copy-on-write battlefield sharing, and zone transition bookkeeping.
type ZoneSystem struct {
	battlefield              []*Permanent
	battlefieldShared        bool
	battlefieldSliceShared   bool
	ownedPermanents          map[uuid.UUID]struct{}
	exile                    []ExiledCard
	enteringPermanent        *Permanent
	lki                      map[uuid.UUID]*PermanentLKI
	castFromExilePermissions []CastableFromExilePermission
	exileInsteadCards        map[uuid.UUID]uuid.UUID
}

// NewZoneSystem creates an initialized ZoneSystem.
func NewZoneSystem() ZoneSystem {
	return ZoneSystem{
		exileInsteadCards: make(map[uuid.UUID]uuid.UUID),
	}
}

// Clone creates a search clone with copy-on-write battlefield sharing and deep-copied exile.
func (zs *ZoneSystem) Clone() ZoneSystem {
	clone := ZoneSystem{}

	// Share battlefield permanents copy-on-write. Cap-limiting forces appends in
	// either branch to allocate a distinct slice header/backing array.
	if len(zs.battlefield) > 0 {
		clone.battlefield = zs.battlefield[:len(zs.battlefield):len(zs.battlefield)]
		zs.battlefieldSliceShared = true
		clone.battlefieldSliceShared = true
	}
	zs.battlefieldShared = true
	zs.ownedPermanents = nil
	clone.battlefieldShared = true

	// Deep copy exile zone.
	if len(zs.exile) > 0 {
		clone.exile = make([]ExiledCard, len(zs.exile))
		for i, ec := range zs.exile {
			clone.exile[i] = ExiledCard{
				Card:       ec.Card, // shared Card ref
				ExiledBy:   ec.ExiledBy,
				FaceDown:   ec.FaceDown,
				RevealedTo: append([]uuid.UUID(nil), ec.RevealedTo...),
			}
		}
	}

	if zs.enteringPermanent != nil {
		clone.enteringPermanent = zs.enteringPermanent
	}

	// Deep copy LKI snapshots.
	if len(zs.lki) > 0 {
		clone.lki = make(map[uuid.UUID]*PermanentLKI, len(zs.lki))
		maps.Copy(clone.lki, zs.lki)
	}

	// Deep copy cast-from-exile permissions.
	if len(zs.castFromExilePermissions) > 0 {
		clone.castFromExilePermissions = make([]CastableFromExilePermission, len(zs.castFromExilePermissions))
		copy(clone.castFromExilePermissions, zs.castFromExilePermissions)
	}

	// Deep copy exile-instead tags.
	if len(zs.exileInsteadCards) > 0 {
		clone.exileInsteadCards = make(map[uuid.UUID]uuid.UUID, len(zs.exileInsteadCards))
		maps.Copy(clone.exileInsteadCards, zs.exileInsteadCards)
	} else {
		clone.exileInsteadCards = make(map[uuid.UUID]uuid.UUID)
	}

	return clone
}

// Battlefield returns the raw battlefield permanents slice.
func (zs *ZoneSystem) Battlefield() []*Permanent {
	return zs.battlefield
}

// FindPermanent finds a permanent by ID on the battlefield.
// Phased-out permanents are invisible.
func (zs *ZoneSystem) FindPermanent(id uuid.UUID) *Permanent {
	for _, p := range zs.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.ID() == id {
			return p
		}
	}
	if zs.enteringPermanent != nil && zs.enteringPermanent.ID() == id {
		return zs.enteringPermanent
	}
	return nil
}

// FindPermanentIncludingPhased finds a permanent by ID even if phased out.
func (zs *ZoneSystem) FindPermanentIncludingPhased(id uuid.UUID) *Permanent {
	for _, p := range zs.battlefield {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
// Phased-out permanents are invisible.
func (zs *ZoneSystem) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	for _, p := range zs.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.Name() == name && p.ControllerID() == controller {
			return p
		}
	}
	return nil
}

// AnyBattlefield returns true if any permanent on the battlefield matches f.
// Phased-out permanents are invisible.
func (zs *ZoneSystem) AnyBattlefield(f PermanentFilter, g *Game) bool {
	for _, p := range zs.battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			return true
		}
	}
	return false
}

// FilterBattlefield returns all permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (zs *ZoneSystem) FilterBattlefield(f PermanentFilter, g *Game) []*Permanent {
	var result []*Permanent
	for _, p := range zs.battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			result = append(result, p)
		}
	}
	return result
}

// CountBattlefield returns the number of permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (zs *ZoneSystem) CountBattlefield(f PermanentFilter, g *Game) int {
	n := 0
	for _, p := range zs.battlefield {
		if p.PhasedOut {
			continue
		}
		if f.Match(p, g) {
			n++
		}
	}
	return n
}

// AllBattlefield returns all permanents on the battlefield.
func (zs *ZoneSystem) AllBattlefield() []*Permanent {
	return zs.battlefield
}

// MutablePermanent returns an owned battlefield permanent pointer suitable for
// mutation. Search clones initially share permanent pointers; the first write
// to a shared permanent clones that permanent and replaces the battlefield
// entry in this ZoneSystem only. Phased-out permanents are invisible.
func (zs *ZoneSystem) MutablePermanent(id uuid.UUID) *Permanent {
	for i, p := range zs.battlefield {
		if p.PhasedOut {
			continue
		}
		if p.ID() == id {
			return zs.mutablePermanentAt(i)
		}
	}
	if zs.enteringPermanent != nil && zs.enteringPermanent.ID() == id {
		return zs.enteringPermanent
	}
	return nil
}

// MutablePermanentIncludingPhased is the mutable counterpart to
// FindPermanentIncludingPhased.
func (zs *ZoneSystem) MutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	for i, p := range zs.battlefield {
		if p.ID() == id {
			return zs.mutablePermanentAt(i)
		}
	}
	return nil
}

func (zs *ZoneSystem) mutablePermanentAt(i int) *Permanent {
	p := zs.battlefield[i]
	if !zs.battlefieldShared {
		return p
	}
	if zs.ownedPermanents != nil {
		if _, ok := zs.ownedPermanents[p.ID()]; ok {
			return p
		}
	}
	zs.ensureBattlefieldSliceOwned()
	cp := new(Permanent)
	clonePermanentInto(cp, p)
	zs.battlefield[i] = cp
	zs.addOwnedPermanent(cp)
	return cp
}

func (zs *ZoneSystem) ensureBattlefieldSliceOwned() {
	if !zs.battlefieldSliceShared {
		return
	}
	if len(zs.battlefield) == 0 {
		zs.battlefield = nil
		zs.battlefieldSliceShared = false
		return
	}
	cp := make([]*Permanent, len(zs.battlefield))
	copy(cp, zs.battlefield)
	zs.battlefield = cp
	zs.battlefieldSliceShared = false
}

func (zs *ZoneSystem) addOwnedPermanent(p *Permanent) {
	if p == nil || !zs.battlefieldShared {
		return
	}
	if zs.ownedPermanents == nil {
		zs.ownedPermanents = make(map[uuid.UUID]struct{})
	}
	zs.ownedPermanents[p.ID()] = struct{}{}
}

// AddToBattlefield appends permanents directly to the battlefield without ETB processing.
func (zs *ZoneSystem) AddToBattlefield(perms ...*Permanent) {
	zs.ensureBattlefieldSliceOwned()
	for _, p := range perms {
		zs.addOwnedPermanent(p)
		zs.battlefield = append(zs.battlefield, p)
	}
}

// TruncateBattlefield truncates the battlefield to the given length (for undo snapshots).
func (zs *ZoneSystem) TruncateBattlefield(n int) {
	zs.ensureBattlefieldSliceOwned()
	if zs.ownedPermanents != nil {
		for i := n; i < len(zs.battlefield); i++ {
			delete(zs.ownedPermanents, zs.battlefield[i].ID())
		}
	}
	if n < len(zs.battlefield) {
		zs.battlefield = zs.battlefield[:n]
	}
}

// AddPermanent adds a permanent to the battlefield during ETB resolution.
func (zs *ZoneSystem) AddPermanent(perm *Permanent) {
	zs.ensureBattlefieldSliceOwned()
	zs.addOwnedPermanent(perm)
	zs.battlefield = append(zs.battlefield, perm)
}

// RemovePermanent removes a permanent from the battlefield by ID.
func (zs *ZoneSystem) RemovePermanent(id uuid.UUID) (*Permanent, bool) {
	zs.ensureBattlefieldSliceOwned()
	for i, p := range zs.battlefield {
		if p.ID() == id {
			zs.battlefield = append(zs.battlefield[:i], zs.battlefield[i+1:]...)
			if zs.ownedPermanents != nil {
				delete(zs.ownedPermanents, id)
			}
			return p, true
		}
	}
	return nil, false
}

// PhaseOut phases the permanent with the given ID out of the battlefield, along
// with any Auras and Equipment attached to it (CR 702.26f, "phasing out
// indirectly").
func (zs *ZoneSystem) PhaseOut(id uuid.UUID) []uuid.UUID {
	perm := zs.MutablePermanent(id)
	if perm == nil {
		return nil
	}
	perm.PhasedOut = true
	phased := []uuid.UUID{id}
	for _, p := range zs.battlefield {
		if p.PhasedOut || p.AttachedTo != id {
			continue
		}
		if !p.HasSubType("Aura") && !p.HasSubType("Equipment") {
			continue
		}
		att := zs.MutablePermanent(p.ID())
		if att == nil {
			continue
		}
		att.PhasedOut = true
		phased = append(phased, att.ID())
	}
	return phased
}

// PhaseIn phases the given permanents back onto the battlefield. IDs that are
// not currently phased out are skipped.
func (zs *ZoneSystem) PhaseIn(ids []uuid.UUID) {
	for _, id := range ids {
		perm := zs.MutablePermanentIncludingPhased(id)
		if perm != nil && perm.PhasedOut {
			perm.PhasedOut = false
		}
	}
}

// EnteringPermanent returns the permanent currently being put onto the battlefield.
func (zs *ZoneSystem) EnteringPermanent() *Permanent {
	return zs.enteringPermanent
}

// SetEnteringPermanent sets the permanent currently being put onto the battlefield.
func (zs *ZoneSystem) SetEnteringPermanent(p *Permanent) {
	zs.enteringPermanent = p
}

// ClearEnteringPermanent clears the entering permanent reference.
func (zs *ZoneSystem) ClearEnteringPermanent() {
	zs.enteringPermanent = nil
}

// Exile returns the full exile zone slice.
func (zs *ZoneSystem) Exile() []ExiledCard {
	return zs.exile
}

// ExileCard moves a card to the exile zone face up.
func (zs *ZoneSystem) ExileCard(card Card, exiledBy uuid.UUID) {
	zs.exile = append(zs.exile, ExiledCard{Card: card, ExiledBy: exiledBy})
}

// ExileCardFaceDown moves a card to the exile zone face down.
func (zs *ZoneSystem) ExileCardFaceDown(card Card, exiledBy uuid.UUID, revealedTo ...uuid.UUID) {
	rev := append([]uuid.UUID(nil), revealedTo...)
	zs.exile = append(zs.exile, ExiledCard{
		Card:       card,
		ExiledBy:   exiledBy,
		FaceDown:   true,
		RevealedTo: rev,
	})
}

// RevealExiledCardTo grants player permission to inspect a face-down exiled card.
func (zs *ZoneSystem) RevealExiledCardTo(cardID, playerID uuid.UUID) {
	for i := range zs.exile {
		if zs.exile[i].Card.ID() != cardID {
			continue
		}
		ec := &zs.exile[i]
		if !ec.FaceDown {
			return
		}
		if slices.Contains(ec.RevealedTo, playerID) {
			return
		}
		ec.RevealedTo = append(ec.RevealedTo, playerID)
		return
	}
}

// FindExiledCard finds an exiled card by its ID.
func (zs *ZoneSystem) FindExiledCard(cardID uuid.UUID) *ExiledCard {
	for i := range zs.exile {
		if zs.exile[i].Card.ID() == cardID {
			return &zs.exile[i]
		}
	}
	return nil
}

// RemoveFromExile removes a card from exile by ID and returns it.
func (zs *ZoneSystem) RemoveFromExile(cardID uuid.UUID) (Card, bool) {
	for i, ec := range zs.exile {
		if ec.Card.ID() == cardID {
			zs.exile = append(zs.exile[:i], zs.exile[i+1:]...)
			return ec.Card, true
		}
	}
	return nil, false
}

// RemoveExiledCardBySource removes all exiled cards with the given ExiledBy ID and returns them.
func (zs *ZoneSystem) RemoveExiledCardBySource(exiledBy uuid.UUID) []ExiledCard {
	var found []ExiledCard
	remaining := zs.exile[:0]
	for _, ec := range zs.exile {
		if ec.ExiledBy == exiledBy {
			found = append(found, ec)
		} else {
			remaining = append(remaining, ec)
		}
	}
	zs.exile = remaining
	return found
}

// ReplaceExiledCard replaces an exiled card matching cardID with replacement.
func (zs *ZoneSystem) ReplaceExiledCard(cardID uuid.UUID, replacement Card) bool {
	for i := range zs.exile {
		if zs.exile[i].Card.ID() == cardID {
			zs.exile = append([]ExiledCard(nil), zs.exile...)
			zs.exile[i].Card = replacement
			return true
		}
	}
	return false
}

// GrantCastFromExile records permission for the given player to cast the given exiled card.
func (zs *ZoneSystem) GrantCastFromExile(playerID, cardID uuid.UUID, anyColorMana bool) {
	zs.castFromExilePermissions = append(zs.castFromExilePermissions, CastableFromExilePermission{
		CardID:       cardID,
		PlayerID:     playerID,
		AnyColorMana: anyColorMana,
	})
}

// CastFromExilePermissionFor returns the permission record for the player and card.
func (zs *ZoneSystem) CastFromExilePermissionFor(playerID, cardID uuid.UUID) *CastableFromExilePermission {
	for i := range zs.castFromExilePermissions {
		perm := &zs.castFromExilePermissions[i]
		if perm.PlayerID == playerID && perm.CardID == cardID {
			return perm
		}
	}
	return nil
}

// ClearCastFromExilePermission removes any permission tied to the card.
func (zs *ZoneSystem) ClearCastFromExilePermission(cardID uuid.UUID) {
	kept := zs.castFromExilePermissions[:0]
	for _, perm := range zs.castFromExilePermissions {
		if perm.CardID != cardID {
			kept = append(kept, perm)
		}
	}
	zs.castFromExilePermissions = kept
}

// CastFromExilePermissions returns all active permissions.
func (zs *ZoneSystem) CastFromExilePermissions() []CastableFromExilePermission {
	return zs.castFromExilePermissions
}

// AddExileInstead registers a card ID that should be exiled instead of put into graveyard.
func (zs *ZoneSystem) AddExileInstead(cardID, sourceID uuid.UUID) {
	if zs.exileInsteadCards == nil {
		zs.exileInsteadCards = make(map[uuid.UUID]uuid.UUID)
	}
	zs.exileInsteadCards[cardID] = sourceID
}

// IsExileInstead reports whether the card ID is marked to be exiled instead of going to graveyard.
func (zs *ZoneSystem) IsExileInstead(cardID uuid.UUID) bool {
	if zs.exileInsteadCards == nil {
		return false
	}
	_, ok := zs.exileInsteadCards[cardID]
	return ok
}

// ClearExileInstead clears all exile-instead markers.
func (zs *ZoneSystem) ClearExileInstead() {
	zs.exileInsteadCards = make(map[uuid.UUID]uuid.UUID)
}

// CaptureLKI records an LKI snapshot for a permanent about to leave the battlefield.
func (zs *ZoneSystem) CaptureLKI(p *Permanent, g GameReader) {
	if p == nil {
		return
	}
	if zs.lki == nil {
		zs.lki = make(map[uuid.UUID]*PermanentLKI)
	}
	isToken := p.IsToken
	owner := p.Card.Owner()
	if owner == uuid.Nil {
		owner = p.ControllerID()
	}
	snap := &Permanent{}
	clonePermanentInto(snap, p)
	zs.lki[p.ID()] = &PermanentLKI{
		ID:        p.ID(),
		Name:      p.Name(),
		Owner:     owner,
		Power:     p.CurrentPower(g),
		Toughness: p.CurrentToughness(g),
		IsToken:   isToken,
		Snapshot:  snap,
	}
}

// LKI returns the last-known-information snapshot for a permanent.
func (zs *ZoneSystem) LKI(id uuid.UUID) *PermanentLKI {
	if zs.lki == nil {
		return nil
	}
	return zs.lki[id]
}

// LKIAbilities returns the captured runtime-abilities of an object from LKI.
func (zs *ZoneSystem) LKIAbilities(id uuid.UUID) []Ability {
	if lki := zs.LKI(id); lki != nil {
		return lki.ViewAbilities()
	}
	return nil
}

// ClearLKI clears all LKI snapshots.
func (zs *ZoneSystem) ClearLKI() {
	zs.lki = nil
}

// LookupObject returns an LKIView for the given object ID, preferring live permanent.
func (zs *ZoneSystem) LookupObject(id uuid.UUID, g GameReader) LKIView {
	if p := zs.FindPermanent(id); p != nil {
		return livePermanentView{p: p, g: g}
	}
	if lki := zs.LKI(id); lki != nil {
		return lki
	}
	return nil
}

// ClearEndOfTurn clears turn-scoped zone state (LKI and exile-instead riders).
func (zs *ZoneSystem) ClearEndOfTurn() {
	zs.ClearExileInstead()
	zs.ClearLKI()
}

package mage

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// BandFormer is an optional interface for players that declare attacking bands.
// The game engine calls GetBandFormations after attackers are declared.
type BandFormer interface {
	GetBandFormations(turn int, g *Game) [][]uuid.UUID
}

// BandingDamageDistributor is an optional interface for players that control
// how incoming combat damage is distributed across banded creatures.
// The attacking player uses this for an attacking band; the defending player
// uses this for a blocking band.
type BandingDamageDistributor interface {
	GetBandingDamageDistribution(members []*Permanent) map[uuid.UUID]int
}

// BandingDamageAssigner permits separate source assignments in band combat.
// Incoming assignments must use the full amount. Outgoing trample assignments
// can leave damage for the defender after shared lethal damage checks.
// A nil or invalid incoming assignment uses bounded number choices instead.
type BandingDamageAssigner interface {
	GetBandingDamageAssignment(g *Game, source *Permanent, recipients []*Permanent, amount int) map[uuid.UUID]int
}

// CombatDamageAssigner lets the attacker controller divide combat damage.
// A normal nontrample source must assign its complete amount among its blockers.
// It has no lethal order. Invalid maps use the default assignment.
// GetBlockerOrder remains available for the existing normal trample path.
// An attacking band asks for each source separately. Only a source with trample
// can leave damage for the defender after the complete plan supplies lethal damage.
type CombatDamageAssigner interface {
	GetBlockerOrder(g *Game, attacker *Permanent, blockers []*Permanent, totalPower int) []uuid.UUID
	GetCombatDamageAssignment(g *Game, attacker *Permanent, blockers []*Permanent, totalPower int) map[uuid.UUID]int
}

// BlockAssignment represents a single blocker-attacker pair.
type BlockAssignment struct {
	BlockerID  uuid.UUID
	AttackerID uuid.UUID
}

// TriggerOrderInfo describes a pending trigger for player ordering choices under CR 603.3b.
type TriggerOrderInfo struct {
	SourceID    uuid.UUID
	SourceName  string
	AbilityText string
}

// TriggerOrderChooser is an optional interface for players to choose the order
// their simultaneous triggers are placed on the stack (CR 603.3b).
// The returned slice contains indices into triggers, indicating the order to put triggers on the stack
// (the first trigger put on the stack is placed first and resolves last).
// If nil or invalid, the engine falls back to the deterministic default order.
type TriggerOrderChooser interface {
	ChooseTriggerOrder(triggers []TriggerOrderInfo) []int
}

// Player interface for game players.
type Player interface {
	PlayerID() uuid.UUID
	Name() string
	Life() int
	SetLife(int)
	StartingLife() int
	SetStartingLife(int)
	GainLife(int)
	LoseLife(int)
	IsAlive() bool
	DrewFromEmpty() bool
	ClearDrewFromEmpty()
	SetLost()
	Hand() []Card
	AddToHand(Card)
	SetHand([]Card)
	RemoveFromHand(uuid.UUID) (Card, bool)
	DiscardCard(uuid.UUID) (Card, bool)
	Graveyard() []Card
	SetGraveyard([]Card)
	AddToGraveyard(Card)
	RemoveFromGraveyard(uuid.UUID) (Card, bool)
	Library() []Card
	SetLibrary([]Card)
	DrawCard() (Card, bool)
	ManaPool() *ManaPool

	AddToLibrary(Card)
	ShuffleLibrary()
	ClearGraveyard()

	// Ante zone
	Ante() []Card
	SetAnte([]Card)
	AddToAnte(Card)
	RemoveFromAnte(uuid.UUID) (Card, bool)

	// Decision-making (overridden by TestPlayer)
	ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *Game) []uuid.UUID
	DeclareAttackers(g *Game) []uuid.UUID
	DeclareBlockers(g *Game) []BlockAssignment
	ChooseMayAbility(description string) bool

	// Draw history remains available until turn cleanup.
	LastDrawnCardID() uuid.UUID
	ClearLastDrawnCard()

	// Poison counters
	PoisonCounters() int
	AddPoisonCounters(int)

	// Player choice methods (overridden by TestPlayer for scripted choices)
	ChooseMode(modes []string, reason string) int
	ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent
	ChooseCardsFromHand(amount int, reason string, g GameReader) []Card
	// ChooseCardFromHand picks one card from a pre-filtered candidate list
	// drawn from the player's hand. Used when an effect or cost limits the
	// choice to a subset of hand (e.g. "discard an artifact card"). Callers
	// must pre-filter candidates; implementations should return a card from
	// `candidates` (or nil if empty).
	ChooseCardFromHand(candidates []Card, reason string, g GameReader) Card
	ChooseCardFromGraveyard(candidates []Card, reason string, g GameReader) Card
	ChooseManaColor(reason string) Color
	ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card
	ChooseCardFromOutsideGame(candidates []Card, reason string, g GameReader) Card
	ChooseNumber(minimum, maximum int, reason string) int

	// ChooseString asks the player to pick one option from a string list,
	// e.g. a creature type for "as ~ enters, choose a creature type"
	// (Herald's Horn). Implementations must return a value from `options`;
	// callers pre-filter the list (e.g. exclude the source's own colors).
	// If `options` is empty, implementations should return "".
	ChooseString(options []string, reason string) string

	// Each chosen target must receive at least one damage (CR 601.2d).
	// The response must use only the chosen targets and sum to total.
	// The engine uses a legal fallback if the response is invalid.
	ChooseDamageDistribution(possible []uuid.UUID, total int, reason string, g *Game) map[uuid.UUID]int

	// ChooseScryPlacement implements the controller's choice for a scry (CR 701.18).
	// `top` is the top N cards of the library in their current order (top first).
	// The implementation returns:
	//   - bottom: the subset of `top` (by ID) that go to the bottom of the library,
	//     in the order they will be placed (last ID becomes the new bottom card).
	//   - topOrder: the remaining cards (by ID) in the order they will be placed
	//     back on top of the library (first ID becomes the new top card).
	// The union of bottom and topOrder must equal the IDs in `top` exactly once each;
	// the engine validates this and falls back to the original order on any mismatch.
	ChooseScryPlacement(top []Card, reason string, g GameReader) (bottom []uuid.UUID, topOrder []uuid.UUID)

	// ChooseSurveilPlacement implements the controller's choice for a surveil
	// (CR 701.42). `top` is the top N cards of the library in their current
	// order (top first). The implementation returns:
	//   - graveyard: the subset of `top` (by ID) that go to the graveyard,
	//     in the order they will be placed.
	//   - topOrder: the remaining cards (by ID) in the order they will be
	//     placed back on top of the library (first ID becomes the new top card).
	// The union must equal the IDs in `top` exactly once each; the engine
	// validates this and falls back to the original order on any mismatch.
	ChooseSurveilPlacement(top []Card, reason string, g GameReader) (graveyard []uuid.UUID, topOrder []uuid.UUID)
}

// BasePlayer implements Player with basic functionality.
type BasePlayer struct {
	id                       uuid.UUID
	name                     string
	life                     int
	startingLife             int
	lost                     bool // true if the player has lost the game (e.g. deck-out)
	drewFromEmpty            bool // set when a draw is attempted from an empty library
	hand                     []Card
	handVisits               map[uuid.UUID]uuid.UUID
	handPublic               bool
	handRevealedVisits       map[uuid.UUID]uuid.UUID
	handStack                *Stack
	departedHandSources      map[uuid.UUID]handDeparture
	graveyard                []Card
	graveyardVisits          map[uuid.UUID]uuid.UUID
	departedGraveyardSources map[uuid.UUID]DamageSource
	library                  []Card
	libraryVisits            map[uuid.UUID]uuid.UUID
	libraryOrder             []uuid.UUID
	libraryTopPublic         bool
	ante                     []Card
	anteVisits               map[uuid.UUID]uuid.UUID
	manaPool                 *ManaPool
	lastDrawnCardID          uuid.UUID
	lastDrawnCardVisit       uuid.UUID
	poisonCounters           int
}

func NewBasePlayer(name string) *BasePlayer {
	return &BasePlayer{
		id:       uuid.New(),
		name:     name,
		life:     20,
		manaPool: NewManaPool(),
	}
}

// NewBasePlayerWithID creates a BasePlayer with a specific ID (used for game cloning).
func NewBasePlayerWithID(id uuid.UUID, name string) *BasePlayer {
	return &BasePlayer{
		id:       id,
		name:     name,
		life:     20,
		manaPool: NewManaPool(),
	}
}

func (p *BasePlayer) PlayerID() uuid.UUID { return p.id }
func (p *BasePlayer) Name() string        { return p.name }
func (p *BasePlayer) Life() int           { return p.life }
func (p *BasePlayer) SetLife(n int)       { p.life = n }
func (p *BasePlayer) StartingLife() int {
	if p.startingLife == 0 {
		return 20
	}
	return p.startingLife
}
func (p *BasePlayer) SetStartingLife(n int) { p.startingLife = n }
func (p *BasePlayer) IsAlive() bool         { return !p.lost }
func (p *BasePlayer) DrewFromEmpty() bool   { return p.drewFromEmpty }
func (p *BasePlayer) ClearDrewFromEmpty()   { p.drewFromEmpty = false }
func (p *BasePlayer) SetLost()              { p.lost = true }
func (p *BasePlayer) ManaPool() *ManaPool   { return p.manaPool }

func (p *BasePlayer) GainLife(n int) {
	p.life += n
}

func (p *BasePlayer) LoseLife(n int) {
	p.life -= n
}

// removeAt returns a fresh slice with the element at index i removed.
// Used by RemoveFromHand/Graveyard/Ante to avoid mutating a backing array
// that may be shared with a parent game state (see cloneCardSlice).
func removeAt(s []Card, i int) []Card {
	out := make([]Card, len(s)-1)
	copy(out, s[:i])
	copy(out[i:], s[i+1:])
	return out
}

func (p *BasePlayer) Hand() []Card { return p.hand }

func (p *BasePlayer) AddToHand(c Card) {
	p.assertOwner(c)
	p.hand = append(p.hand, c)
	p.recordHandVisit(c.ID())
}

func (p *BasePlayer) SetHand(cards []Card) {
	previous := p.handVisits
	for _, card := range p.hand {
		if !slices.ContainsFunc(cards, func(current Card) bool { return current.ID() == card.ID() }) {
			p.saveHandDeparture(card)
		}
	}
	p.hand = cards
	p.handVisits = make(map[uuid.UUID]uuid.UUID, len(cards))
	for _, card := range cards {
		visit := previous[card.ID()]
		if visit == uuid.Nil {
			visit = uuid.New()
		}
		p.handVisits[card.ID()] = visit
	}
}

func (p *BasePlayer) recordHandVisit(id uuid.UUID) {
	p.handVisits = maps.Clone(p.handVisits)
	if p.handVisits == nil {
		p.handVisits = make(map[uuid.UUID]uuid.UUID)
	}
	p.handVisits[id] = uuid.New()
}

func (p *BasePlayer) handVisit(id uuid.UUID) uuid.UUID { return p.handVisits[id] }

func (p *BasePlayer) RemoveFromHand(id uuid.UUID) (Card, bool) {
	for i, c := range p.hand {
		if c.ID() != id {
			continue
		}
		p.saveHandDeparture(c)
		p.hand = removeAt(p.hand, i)
		p.handVisits = maps.Clone(p.handVisits)
		delete(p.handVisits, id)
		return c, true
	}
	return nil, false
}

func (p *BasePlayer) DiscardCard(id uuid.UUID) (Card, bool) {
	c, ok := p.RemoveFromHand(id)
	if !ok {
		return nil, false
	}
	p.AddToGraveyard(c)
	return c, true
}

func (p *BasePlayer) Graveyard() []Card { return p.graveyard }

func (p *BasePlayer) SetGraveyard(cards []Card) {
	for _, card := range p.graveyard {
		p.saveGraveyardDeparture(card)
	}
	p.graveyardVisits = nil
	p.graveyard = cards
	for _, card := range cards {
		p.recordGraveyardVisit(card.ID())
	}
}

func (p *BasePlayer) recordGraveyardVisit(id uuid.UUID) {
	if p.graveyardVisits == nil {
		p.graveyardVisits = make(map[uuid.UUID]uuid.UUID)
	} else {
		p.graveyardVisits = maps.Clone(p.graveyardVisits)
	}
	p.graveyardVisits[id] = uuid.New()
}

func (p *BasePlayer) graveyardVisit(id uuid.UUID) uuid.UUID { return p.graveyardVisits[id] }

func (p *BasePlayer) replaceGraveyardCard(id uuid.UUID, replacement Card) bool {
	cards, ok := replaceCardInSlice(p.graveyard, id, replacement)
	if ok {
		p.graveyard = cards
	}
	return ok
}

func (p *BasePlayer) AddToGraveyard(c Card) {
	p.assertOwner(c)
	p.graveyard = append(p.graveyard, c)
	p.recordGraveyardVisit(c.ID())
}

func (p *BasePlayer) RemoveFromGraveyard(id uuid.UUID) (Card, bool) {
	for i, c := range p.graveyard {
		if c.ID() != id {
			continue
		}
		p.saveGraveyardDeparture(c)
		p.graveyardVisits = maps.Clone(p.graveyardVisits)
		delete(p.graveyardVisits, id)
		p.graveyard = removeAt(p.graveyard, i)
		return c, true
	}
	return nil, false
}

func (p *BasePlayer) Library() []Card { return p.library }

// SetLibrary keeps retained visits outside reordered regions.
// A revealed card becomes a new object when its region is reordered.
func (p *BasePlayer) SetLibrary(cards []Card) {
	previous := p.libraryVisits
	oldOrder := make([]uuid.UUID, 0, len(p.libraryOrder))
	newOrder := make([]uuid.UUID, 0, len(cards))
	present := make(map[uuid.UUID]bool, len(cards))
	for _, card := range cards {
		present[card.ID()] = true
	}
	for _, id := range p.libraryOrder {
		if present[id] {
			oldOrder = append(oldOrder, id)
		}
	}
	for _, card := range cards {
		if previous[card.ID()] != uuid.Nil {
			newOrder = append(newOrder, card.ID())
		}
	}
	reordered := make(map[uuid.UUID]bool)
	balance := make(map[uuid.UUID]int)
	unmatched := 0
	update := func(id uuid.UUID, change int) {
		if balance[id] != 0 {
			unmatched--
		}
		balance[id] += change
		if balance[id] != 0 {
			unmatched++
		}
	}
	for i, id := range newOrder {
		changed := unmatched != 0 || oldOrder[i] != id
		update(oldOrder[i], 1)
		update(id, -1)
		reordered[id] = changed
	}
	p.libraryVisits = make(map[uuid.UUID]uuid.UUID, len(cards))
	for _, card := range cards {
		visit := previous[card.ID()]
		if visit == uuid.Nil || reordered[card.ID()] {
			visit = uuid.New()
		}
		p.libraryVisits[card.ID()] = visit
	}
	if p.libraryTopPublic && len(p.libraryOrder) != 0 {
		oldTop := p.libraryOrder[0]
		if present[oldTop] && (len(cards) == 0 || cards[0].ID() != oldTop) {
			p.libraryVisits[oldTop] = uuid.New()
		}
	}
	p.libraryOrder = make([]uuid.UUID, len(cards))
	for i, card := range cards {
		p.libraryOrder[i] = card.ID()
	}
	p.library = cards
}

func (p *BasePlayer) libraryVisit(id uuid.UUID) uuid.UUID { return p.libraryVisits[id] }

func (p *BasePlayer) recordLibraryVisit(id uuid.UUID) {
	p.libraryVisits = maps.Clone(p.libraryVisits)
	if p.libraryVisits == nil {
		p.libraryVisits = make(map[uuid.UUID]uuid.UUID)
	}
	p.libraryVisits[id] = uuid.New()
}

func (p *BasePlayer) AddToLibrary(c Card) {
	p.assertOwner(c)
	p.library = append(p.library, c)
	p.libraryOrder = append(slices.Clone(p.libraryOrder), c.ID())
	p.recordLibraryVisit(c.ID())
}

func (p *BasePlayer) ClearGraveyard() {
	p.SetGraveyard(nil)
}

func (p *BasePlayer) Ante() []Card { return p.ante }

func (p *BasePlayer) SetAnte(cards []Card) {
	visits := make(map[uuid.UUID]uuid.UUID, len(cards))
	for _, card := range cards {
		visit := p.anteVisits[card.ID()]
		if visit == uuid.Nil {
			visit = uuid.New()
		}
		visits[card.ID()] = visit
	}
	p.ante = cards
	p.anteVisits = visits
}

func (p *BasePlayer) anteVisit(id uuid.UUID) uuid.UUID { return p.anteVisits[id] }

func (p *BasePlayer) AddToAnte(c Card) {
	p.ante = append(p.ante, c)
	p.anteVisits = maps.Clone(p.anteVisits)
	if p.anteVisits == nil {
		p.anteVisits = make(map[uuid.UUID]uuid.UUID)
	}
	p.anteVisits[c.ID()] = uuid.New()
}

func (p *BasePlayer) RemoveFromAnte(id uuid.UUID) (Card, bool) {
	for i, c := range p.ante {
		if c.ID() == id {
			p.ante = removeAt(p.ante, i)
			p.anteVisits = maps.Clone(p.anteVisits)
			delete(p.anteVisits, id)
			return c, true
		}
	}
	return nil, false
}

func (p *BasePlayer) ShuffleLibrary() {
	// Library backing arrays are shared across game clones (see cloneCardSlice).
	// Copy into a fresh array before shuffling so we don't mutate the parent's view.
	fresh := make([]Card, len(p.library))
	copy(fresh, p.library)
	p.library = fresh
	p.libraryVisits = make(map[uuid.UUID]uuid.UUID, len(p.library))
	for _, card := range p.library {
		p.libraryVisits[card.ID()] = uuid.New()
	}
	rand.Shuffle(len(p.library), func(i, j int) {
		p.library[i], p.library[j] = p.library[j], p.library[i]
	})
	p.libraryOrder = make([]uuid.UUID, len(p.library))
	for i, card := range p.library {
		p.libraryOrder[i] = card.ID()
	}
}

func (p *BasePlayer) DrawCard() (Card, bool) {
	if len(p.library) == 0 {
		p.drewFromEmpty = true
		return nil, false
	}
	c := p.library[0]
	p.library = p.library[1:]
	p.libraryOrder = p.libraryOrder[1:]
	p.libraryVisits = maps.Clone(p.libraryVisits)
	delete(p.libraryVisits, c.ID())
	p.AddToHand(c)
	p.lastDrawnCardID = c.ID()
	p.lastDrawnCardVisit = p.handVisit(c.ID())
	return c, true
}

// LastDrawnCardID returns the ID of the last card drawn this turn.
func (p *BasePlayer) LastDrawnCardID() uuid.UUID { return p.lastDrawnCardID }

// ClearLastDrawnCard prevents a prior turn's draw from paying a cost.
func (p *BasePlayer) ClearLastDrawnCard() {
	p.lastDrawnCardID = uuid.Nil
	p.lastDrawnCardVisit = uuid.Nil
}

func (p *BasePlayer) lastDrawnHandCard() capturedCard {
	return capturedCard{id: p.lastDrawnCardID, visit: p.lastDrawnCardVisit, zone: ZoneHand}
}

// PoisonCounters returns the number of poison counters this player has.
func (p *BasePlayer) PoisonCounters() int { return p.poisonCounters }

// AddPoisonCounters adds n poison counters to this player.
func (p *BasePlayer) AddPoisonCounters(n int) { p.poisonCounters += n }

// Default decision implementations (overridden by TestPlayer).
func (p *BasePlayer) ChooseTargets(possible []uuid.UUID, minimum, maximum int, g *Game) []uuid.UUID {
	if len(possible) >= minimum {
		return possible[:minimum]
	}
	return nil
}

func (p *BasePlayer) DeclareAttackers(g *Game) []uuid.UUID {
	return nil
}

func (p *BasePlayer) DeclareBlockers(g *Game) []BlockAssignment {
	return nil
}

func (p *BasePlayer) ChooseMayAbility(description string) bool {
	return false
}

// Each target needs a positive share before the remainder can go to the first.
func (p *BasePlayer) ChooseDamageDistribution(possible []uuid.UUID, total int, reason string, g *Game) map[uuid.UUID]int {
	return sanitizeDamageDistribution(nil, possible, total)
}

// Default choice implementations (pick first available option).

func (p *BasePlayer) ChooseMode(modes []string, reason string) int {
	return 0
}

func (p *BasePlayer) ChoosePermanent(candidates []*Permanent, reason string, g GameReader) *Permanent {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}

func (p *BasePlayer) ChooseCardsFromHand(amount int, reason string, g GameReader) []Card {
	hand := p.Hand()
	if amount > len(hand) {
		amount = len(hand)
	}
	result := make([]Card, amount)
	copy(result, hand[:amount])
	return result
}

func (p *BasePlayer) ChooseCardFromHand(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

func (p *BasePlayer) ChooseCardFromGraveyard(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) == 0 {
		return nil
	}
	return candidates[0]
}

func (p *BasePlayer) ChooseManaColor(reason string) Color {
	return White // default to White
}

func (p *BasePlayer) ChooseCardFromLibrary(candidates []Card, reason string, g GameReader) Card {
	if len(candidates) > 0 {
		return candidates[0]
	}
	return nil
}

func (p *BasePlayer) ChooseNumber(minimum, maximum int, reason string) int {
	return maximum // default: choose maximum
}

func (p *BasePlayer) ChooseString(options []string, reason string) string {
	if len(options) == 0 {
		return ""
	}
	return options[0]
}

// ChooseScryPlacement: deterministic default keeps every revealed card on top
// in its original order. Card implementations and AI players may override this.
func (p *BasePlayer) ChooseScryPlacement(top []Card, reason string, g GameReader) (bottom, topOrder []uuid.UUID) {
	topOrder = make([]uuid.UUID, len(top))
	for i, c := range top {
		topOrder[i] = c.ID()
	}
	return nil, topOrder
}

// ChooseSurveilPlacement: deterministic default keeps every revealed card on
// top in its original order (no cards milled to the graveyard). Card
// implementations and AI players may override this.
func (p *BasePlayer) ChooseSurveilPlacement(top []Card, reason string, g GameReader) (graveyard, topOrder []uuid.UUID) {
	topOrder = make([]uuid.UUID, len(top))
	for i, c := range top {
		topOrder[i] = c.ID()
	}
	return nil, topOrder
}

func (p *BasePlayer) assertOwner(c Card) {
	owner := c.Owner()
	if owner == uuid.Nil {
		c.SetOwner(p.id)
	} else if owner != p.id {
		panic(fmt.Sprintf("card %s (ID %s) has owner %s, expected %s (%s)",
			c.Name(), c.ID(), owner, p.id, p.name))
	}
}

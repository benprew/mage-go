package mage

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// Sentinel errors for common failure conditions.
var (
	ErrPlayerNotFound    = errors.New("player not found")
	ErrSourceNotFound    = errors.New("source not found on battlefield")
	ErrPermanentNotFound = errors.New("permanent not found")
	ErrCardNotInHand     = errors.New("card not found in hand")
	ErrSourceTapped      = errors.New("source is already tapped")
	ErrNoCreature        = errors.New("no creature to sacrifice")
	ErrSorcerySpeed      = errors.New("can only activate at sorcery speed")
	ErrAnteDisabled      = errors.New("ante is not enabled for this game")
	ErrAnteSettled       = errors.New("ante result has already been settled")
)

// ExiledCard records a card in exile and the reason for the exile.
//
// FaceDown: When true, the card is in exile face down (CR 707, 406.3).
// The engine hides the card characteristics from unauthorized players.
// Cards such as Gonti, Lord of Luxury use this field to hide cards from opponents.
//
// RevealedTo: Contains the IDs of players permitted to inspect the card (CR 408).
// This list usually includes the exiling player and the card owner.
// Other players see only that a face-down card is in exile.
// Unauthorized players cannot inspect card characteristics.
type ExiledCard struct {
	spellCopy    bool
	link         *ExileLink
	linkedSource capturedPermanent
	Card         Card
	ExiledBy     uuid.UUID // ID of the permanent/spell that caused the exile
	Visit        uuid.UUID
	Counters     [NumCounters]int
	FaceDown     bool
	RevealedTo   []uuid.UUID
}

// VisibleTo reports whether the given player may inspect this exiled card's
// identity. Face-up exiled cards are visible to everyone; face-down exiled
// cards are visible only to players in RevealedTo.
func (ec *ExiledCard) VisibleTo(playerID uuid.UUID) bool {
	if !ec.FaceDown {
		return true
	}
	return slices.Contains(ec.RevealedTo, playerID)
}

// Game is the central game state and engine.
type Game struct {
	drawn               bool
	players             []Player
	playerManaAbilities []installedPlayerManaAbility
	playerPayments      []installedPlayerPayment
	publicReveals       []PublicCardReveal
	libraryLooks        map[uuid.UUID][]PublicCardReveal
	handLooks           map[uuid.UUID][]PublicCardReveal
	anteEnabled         bool
	originalOwners      map[uuid.UUID]uuid.UUID
	anteResult          []OwnershipChange
	anteSettled         bool
	zones               ZoneSystem
	stack               *Stack
	combat              *Combat
	effects             *EffectManager
	layer2Controllers   map[uuid.UUID]uuid.UUID
	mana                ManaSystem

	// TurnSystem owns turn, step, active player, extra turns, scheduling, and untap skips.
	turns TurnSystem

	// TriggerSystem owns pending, delayed, and state-triggered abilities.
	triggers TriggerSystem

	// ResolutionState owns transient resolution and cost scratch state.
	resolution ResolutionState

	// TrackerSystem owns turn-scoped and duel-scoped observations and counters.
	trackers TrackerSystem

	// DamageSystem owns damage execution, history, aggregation, and reflection.
	damage            DamageSystem
	damageBatch       *damageBatch
	damageEventSource DamageSource

	// Mill amount modifiers (CR 614 replacement-style) keyed by source permanent ID.
	// Each entry maps milled-player ID -> proposed amount -> new amount; modifiers
	// stack and are dropped when the source leaves the battlefield (cleared in Apply).
	millModifiers []millModifierEntry

	// Life-gain amount modifiers (CR 614 replacement-style) keyed by source
	// permanent ID. Used by Rhox Faithmender ("If you would gain life, you
	// gain twice that much life instead.") and similar effects.
	lifeGainModifiers []lifeGainModifierEntry

	// RandomSource encapsulates random number and coin flip outcomes
	random RandomSource

	// Priority handler — called when a player receives priority.
	// If nil, the engine drains the stack atomically (legacy behavior).
	onPriority PriorityHandler

	// AfterPriorityAction is called after a non-pass priority action is executed.
	// Used by the interactive layer for logging and display.
	afterPriorityAction func(g *Game, playerIdx int, action PriorityAction)

	// BeforeStackResolve is called before the top of the stack is resolved
	// during a priority round. Used by the interactive layer for logging.
	beforeStackResolve func(g *Game)

	// Control flags
	stopped bool

	// customState is a per-game string-keyed bag for set-specific keyword
	// support to stash auxiliary state (e.g. Paradigm "have I resolved a
	// spell with this name yet?" tracking). Populate via paradigmStateOf
	// and similar accessors in keyword_sos.go. Survives the lifetime of
	// the game; cleared per-game via NewGame.
	customState map[string]any
}

func (g *Game) ActivePlayer() int {
	return g.turns.ActivePlayerIndex()
}

// NewGame creates a new 2-player game.
func NewGame(playerA, playerB Player) *Game {
	return newGame(playerA, playerB, false)
}

func newGame(playerA, playerB Player, anteEnabled bool) *Game {
	if !anteEnabled {
		for _, player := range []Player{playerA, playerB} {
			if player == nil {
				continue
			}
			retained, excluded := PrepareCardsForAnte(player.Library(), false)
			if len(excluded) != 0 {
				player.SetLibrary(retained)
			}
		}
	}
	g := &Game{
		players:        []Player{playerA, playerB},
		anteEnabled:    anteEnabled,
		originalOwners: make(map[uuid.UUID]uuid.UUID),
		zones:          NewZoneSystem(),
		stack:          NewStack(),
		combat:         NewCombat(),
		effects:        NewEffectManager(),
		resolution:     NewResolutionState(),
		trackers:       NewTrackerSystem(),
		random:         NewRandomSource(),
		triggers:       NewTriggerSystem(),
		mana:           NewManaSystem(),
		damage:         NewDamageSystem(),
		turns:          NewTurnSystem(),
		customState:    make(map[string]any),
	}
	g.bindHandStack()
	return g
}

// GetPlayer returns the player with the given ID.
func (g *Game) GetPlayer(id uuid.UUID) Player {
	for _, p := range g.players {
		if p.PlayerID() == id {
			return p
		}
	}
	return nil
}

// GetOpponent returns the other player.
func (g *Game) GetOpponent(id uuid.UUID) Player {
	for _, p := range g.players {
		if p.PlayerID() != id {
			return p
		}
	}
	return nil
}

// ActivePlayerObj returns the currently active player.
func (g *Game) ActivePlayerObj() Player {
	if len(g.players) == 0 {
		return nil
	}
	idx := g.turns.ActivePlayerIndex()
	if idx < 0 || idx >= len(g.players) {
		return nil
	}
	return g.players[idx]
}

// NonActivePlayerObj returns the non-active player.
func (g *Game) NonActivePlayerObj() Player {
	if len(g.players) < 2 {
		return nil
	}
	idx := (g.turns.ActivePlayerIndex() + 1) % len(g.players)
	return g.players[idx]
}

// FindPermanent finds a permanent by ID on the battlefield.
// Phased-out permanents are invisible.
func (g *Game) FindPermanent(id uuid.UUID) *Permanent {
	return g.zones.FindPermanent(id)
}

// PhaseOut phases the permanent with the given ID out of the battlefield, along
// with any Auras and Equipment attached to it (CR 702.26f, "phasing out
// indirectly"). Counters, attachments, and other on-permanent state are
// retained while phased out (CR 702.26d). It returns the IDs that were phased
// out — the permanent followed by its attachments — so the caller can later
// phase exactly that set back in with PhaseIn.
func (g *Game) PhaseOut(id uuid.UUID) []uuid.UUID {
	// Information queries need the state before the group becomes absent.
	if source := g.FindPermanent(id); source != nil {
		g.zones.capturePhaseInformation(source, g)
		for _, p := range g.zones.battlefield {
			if !p.PhasedOut && p.AttachedTo == id && (p.HasSubType("Aura") || p.HasSubType("Equipment")) {
				g.zones.capturePhaseInformation(p, g)
			}
		}
	}
	phased := g.zones.PhaseOut(id)
	expired := false
	for _, permanentID := range phased {
		g.combat.RemoveFromCombat(permanentID)
		expired = g.effects.expireSourceTapped(permanentID) || expired
		expired = g.effects.expireSourcePresence(permanentID) || expired
		expired = g.effects.expireCapturedControlSource(permanentID, false) || expired
		expired = g.effects.expireControlPowerSubject(permanentID) || expired
	}
	expired = g.effects.expireCounterDurations(g) || expired
	if expired {
		g.ApplyContinuousEffects()
	}
	g.syncManaSpendingPermissions()
	return phased
}

// PhaseIn phases the given permanents back onto the battlefield. IDs that are
// not currently phased out are skipped.
func (g *Game) PhaseIn(ids []uuid.UUID) {
	g.zones.PhaseIn(ids)
	g.syncManaSpendingPermissions()
}

// FindPermanentIncludingPhased finds a permanent by ID even if phased out.
func (g *Game) FindPermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.zones.FindPermanentIncludingPhased(id)
}

// MutablePermanent returns an owned battlefield permanent pointer suitable for
// mutation. Search clones initially share permanent pointers; the first write
// to a shared permanent clones that permanent and replaces the battlefield
// entry in this Game only. Phased-out permanents are invisible.
func (g *Game) MutablePermanent(id uuid.UUID) *Permanent {
	return g.zones.MutablePermanent(id)
}

// mutablePermanentIncludingPhased is the mutable counterpart to
// FindPermanentIncludingPhased.
func (g *Game) mutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.zones.MutablePermanentIncludingPhased(id)
}

// MutablePermanentIncludingPhased is the exported mutable counterpart to
// FindPermanentIncludingPhased. Returns an owned battlefield permanent
// pointer suitable for mutation (triggers copy-on-write under search).
func (g *Game) MutablePermanentIncludingPhased(id uuid.UUID) *Permanent {
	return g.zones.MutablePermanentIncludingPhased(id)
}

// FindPermanentByName finds a permanent by name on the battlefield (first match).
// Phased-out permanents are invisible.
func (g *Game) FindPermanentByName(name string, controller uuid.UUID) *Permanent {
	return g.zones.FindPermanentByName(name, controller)
}

// AnyBattlefield returns true if any permanent on the battlefield matches f.
// Phased-out permanents are invisible.
func (g *Game) AnyBattlefield(f PermanentFilter) bool {
	return g.zones.AnyBattlefield(f, g)
}

// FilterBattlefield returns all permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) FilterBattlefield(f PermanentFilter) []*Permanent {
	return g.zones.FilterBattlefield(f, g)
}

// CountBattlefield returns the number of permanents on the battlefield matching f.
// Phased-out permanents are invisible.
func (g *Game) CountBattlefield(f PermanentFilter) int {
	return g.zones.CountBattlefield(f, g)
}

// FindCardAnywhere finds a card by ID anywhere in the game.
// Also checks the currently resolving card (which may be in limbo between
// stack pop and graveyard placement during resolution).
func (g *Game) FindCardAnywhere(id uuid.UUID) Card {
	for _, p := range g.zones.battlefield {
		if p.ID() == id {
			return p.Card
		}
	}
	// Search the stack (spells that have been cast but not yet resolved)
	for _, obj := range g.stack.Objects() {
		if obj.Card != nil && obj.Card.ID() == id {
			return obj.Card
		}
	}
	// Check the currently resolving card (popped from stack, not yet in graveyard)
	if rc := g.resolution.ResolvingCard(); rc != nil && rc.ID() == id {
		return rc
	}
	for _, pl := range g.players {
		for _, c := range pl.Library() {
			if c.ID() == id {
				return c
			}
		}
		for _, c := range pl.Hand() {
			if c.ID() == id {
				return c
			}
		}
		for _, c := range pl.Graveyard() {
			if c.ID() == id {
				return c
			}
		}
		for _, c := range pl.Ante() {
			if c.ID() == id {
				return c
			}
		}
	}
	for _, ec := range g.zones.exile {
		if ec.Card.ID() == id {
			return ec.Card
		}
	}
	for _, object := range g.zones.commandObjects {
		if object.card.ID() == id {
			card := object.card.Copy()
			card.SetID(id)
			return card
		}
	}
	return nil
}

// findCardForDamageSource finds the Card associated with a damage source ID.
// This looks at permanents, graveyard, and the currently resolving card.
func (g *Game) findCardForDamageSource(sourceID uuid.UUID) Card {
	perm := g.FindPermanent(sourceID)
	if perm != nil {
		return perm.Card
	}
	if rc := g.resolution.ResolvingCard(); rc != nil && rc.ID() == sourceID {
		return rc
	}
	return g.FindCardAnywhere(sourceID)
}

// FlipCoin simulates a coin flip. Returns true for "win" (heads).
// If CoinFlipResults is non-empty, pops from the front (for test determinism).
func (g *Game) FlipCoin(playerID uuid.UUID) bool {
	return g.random.FlipCoin(playerID)
}

// TryPayMana attempts to pay a mana cost using floating mana and any untapped
// mana sources the player may activate. It is suitable for resolving effects
// that ask a player to pay mana outside the spell-casting pipeline.
func (g *Game) TryPayMana(playerID uuid.UUID, manaCostStr string) bool {
	cost := ParseManaCost(manaCostStr)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: playerID,
		Costs:      []Cost{&ManaCostPayment{MC: cost}},
	})
	if err != nil {
		return false
	}
	return tx.Commit() == nil
}

// PutOnBattlefield puts a card onto the battlefield under the given controller.
func (g *Game) PutOnBattlefield(card Card, controller uuid.UUID) *Permanent {
	return g.putOnBattlefield(card, controller, uuid.Nil, nil, uuid.Nil, 0)
}

// PutOnBattlefieldBy records the player who puts the card onto the battlefield.
func (g *Game) PutOnBattlefieldBy(card Card, controller, actor uuid.UUID) *Permanent {
	return g.putOnBattlefield(card, controller, actor, nil, uuid.Nil, 0)
}

func (g *Game) putOnBattlefield(card Card, controller, actor uuid.UUID, colors *[]Color, colorSourceID uuid.UUID, castColorsSpent int) *Permanent {
	return g.putOnBattlefieldWithEntry(card, controller, actor, colors, colorSourceID, castColorsSpent, false)
}

func (g *Game) putOnBattlefieldWithEntry(card Card, controller, actor uuid.UUID, colors *[]Color, colorSourceID uuid.UUID, castColorsSpent int, tapped bool) *Permanent {
	perm := g.prepareBattlefieldEntryWithContext(card, controller, colors, battlefieldEntryContext{actor: actor, castColorsSpent: castColorsSpent}, tapped)
	if perm == nil {
		return nil
	}
	g.registerBattlefieldEntry(perm, colors, colorSourceID)
	g.completeBattlefieldEntry(perm, controller, actor, castColorsSpent)
	return perm
}

func (g *Game) putTokensOnBattlefieldTogether(cards []Card, controller uuid.UUID) []*Permanent {
	return g.putTokensOnBattlefieldTogetherWithLink(cards, controller, nil, capturedPermanent{})
}

func (g *Game) putTokensOnBattlefieldTogetherWithLink(cards []Card, controller uuid.UUID, link *CreatedTokenLink, source capturedPermanent) []*Permanent {
	permanents := make([]*Permanent, 0, len(cards))
	for _, card := range cards {
		if perm := g.prepareBattlefieldEntry(card, controller, nil, 0, false); perm != nil {
			perm.createdTokenLink = link
			perm.createdTokenSource = source
			permanents = append(permanents, perm)
		}
	}
	for _, perm := range permanents {
		g.registerBattlefieldEntry(perm, nil, uuid.Nil)
	}
	for _, perm := range permanents {
		g.observeBattlefieldEntryFrom(perm, controller, controller, 0, ZoneAny)
	}
	for _, perm := range permanents {
		g.finishRetainedEntry(perm)
	}
	return permanents
}

func (g *Game) prepareBattlefieldEntry(card Card, controller uuid.UUID, colors *[]Color, castColorsSpent int, tapped bool) *Permanent {
	return g.prepareBattlefieldEntryWithContext(card, controller, colors, battlefieldEntryContext{castColorsSpent: castColorsSpent}, tapped)
}

func (g *Game) prepareBattlefieldEntryWithContext(card Card, controller uuid.UUID, colors *[]Color, context battlefieldEntryContext, tapped bool) *Permanent {
	perm, failed := g.prepareBattlefieldEntryResultWithContext(card, controller, colors, context, tapped)
	if failed != nil {
		owner := failed.Owner()
		if owner == uuid.Nil {
			owner = controller
		}
		if player := g.GetPlayer(owner); player != nil {
			player.AddToGraveyard(failed)
		}
	}
	return perm
}

// Entry batches must finish choices before a failed entry moves to the graveyard.
func (g *Game) prepareBattlefieldEntryResultWithContext(card Card, controller uuid.UUID, colors *[]Color, context battlefieldEntryContext, tapped bool) (*Permanent, Card) {
	if g.IsLandEntryProhibited(card) && (context.entry.withoutSubtype == nil || !context.entry.withoutSubtype.asEnchantment) {
		return nil, nil
	}
	if context.actor == uuid.Nil {
		context.actor = controller
	}
	perm := NewPermanent(card, controller)
	if colors != nil {
		override := append([]Color(nil), (*colors)...)
		perm.ColorOverride = &override
	}
	g.zones.addOwnedPermanent(perm)
	perm.turnControlGained = g.turns.Turn()
	perm.controlGainedTurns = map[uuid.UUID]int{controller: g.turns.Turn()}

	g.syncAbilityContext(perm)
	g.zones.SetEnteringPermanent(perm)
	defer g.zones.ClearEnteringPermanent()
	cancelSubtype := context.entry.prepareSubtype(g, perm)
	prepared := false
	defer func() {
		if !prepared {
			cancelSubtype()
		}
	}()
	entry := &battlefieldEntryAction{context: context, permanent: perm, copies: make(map[entryCopyKey]*entryCopyReplacement), textChanges: entryCopyTextChanges(card)}
	if g.effects.ApplyReplacements(entry, g) == nil {
		if g.isLandEntryProhibited(perm.HasType(TypeLand)) {
			return nil, nil
		}
		return nil, cardForBattlefieldDeparture(perm.Card)
	}
	if g.isLandEntryProhibited(perm.HasType(TypeLand)) {
		return nil, nil
	}

	// Set tapped status before entry effects and triggers inspect this permanent.
	if tapped || perm.HasKeyword(EntersTapped) || g.effects.Rules.ShouldEnterTapped(perm) {
		perm.Tapped = true
		perm.RevokeBaseAttr(EntersTapped)
	}

	// Each counter kind has one entry event for counter replacements.
	counterAmounts := entry.context.entry.counterAmounts
	for _, a := range perm.RuntimeAbilities {
		if xc, ok := a.(*EntersWithXCountersAbility); ok {
			counterAmounts[xc.CounterType] += max(0, entry.context.announcedX)
		}
	}

	// Add fixed N counters if configured (replacement effect, not a trigger).
	baseCtrTypes := map[CounterType]bool{}
	for _, a := range perm.RuntimeAbilities {
		if nc, ok := a.(*EntersWithNCountersAbility); ok {
			counterAmounts[nc.CounterType] += max(0, nc.Count)
			baseCtrTypes[nc.CounterType] = true
		}
	}
	for _, a := range perm.RuntimeAbilities {
		if xc, ok := a.(*EntersWithXCountersAbility); ok && entry.context.announcedX > 0 {
			baseCtrTypes[xc.CounterType] = true
		}
	}

	// Add computed counters if configured (CR 614.1c self-replacement whose
	// count depends on board state, e.g. Towering Titan: X = total toughness
	// of other creatures you control).
	for _, a := range perm.RuntimeAbilities {
		if cc, ok := a.(*EntersWithComputedCountersAbility); ok && cc.Compute != nil {
			n := cc.Compute(g, perm)
			if n > 0 {
				counterAmounts[cc.CounterType] += n
			}
			baseCtrTypes[cc.CounterType] = true
		}
	}
	for _, ability := range perm.RuntimeAbilities {
		if counters, ok := ability.(*entryCountersAbility); ok && !nilDefinition(counters.amount) {
			n := counters.amount.entryCounterAmount(entry.context)
			if n > 0 {
				counterAmounts[counters.counter] += n
				baseCtrTypes[counters.counter] = true
			}
		}
	}

	for counter, amount := range counterAmounts {
		if amount > 0 {
			ct := CounterType(counter)
			g.AddCountersWithReplacement(perm, ct, amount, perm.ID(), true)
			baseCtrTypes[ct] = true
		}
	}

	// CR 614.1c: "enters with N counters" effects from other sources (Oona's
	// Blackguard, Winding Constrictor) are self-replacements applied during the
	// ETB process even when the entering permanent has no native "enters with"
	// clause for that counter type. Synthesize a 0-amount AddCountersAction so
	// the registered etbAdditionalCountersReplacement effects can intercept it
	// (CR 614.5: multiple such effects combine into a single application).
	for _, ct := range g.etbAdditionalCounterTypesFor(perm) {
		if baseCtrTypes[ct] {
			continue
		}
		action := NewAddCountersAction(perm.ID(), perm.ID(), ct, 0, true)
		result := g.effects.ApplyReplacements(action, g)
		if aca, ok := result.(*AddCountersAction); ok && aca.Amount() > 0 {
			perm.AddCounter(aca.CounterType(), cappedCounterAmount(perm, aca.CounterType(), aca.Amount()))
		}
	}

	// Copy creature on ETB (Vesuvan Doppelganger): copy target creature's P/T and keywords
	for _, a := range perm.RuntimeAbilities {
		if _, ok := a.(*CopyCreatureOnETBAbility); ok && len(g.resolution.ResolvingTargets()) > 0 {
			target := g.FindPermanent(g.resolution.ResolvingTargets()[0])
			if target != nil {
				g.effects.AddCopyEffect(perm.ID(), target)
			}
			break
		}
	}

	for _, a := range perm.RuntimeAbilities {
		if asEnter, ok := UnwrapAbility(a).(AsEntersBattlefieldAbility); ok {
			if !asEnter.OnEnter(g, perm) {
				return nil, cardForBattlefieldDeparture(perm.Card)
			}
		}
	}

	if !g.prepareGraveyardEnchant(perm, context) {
		return nil, nil
	}
	prepared = true
	return perm, nil
}

func (g *Game) registerBattlefieldEntry(perm *Permanent, colors *[]Color, colorSourceID uuid.UUID) {
	g.zones.AddPermanent(perm)
	if colors != nil {
		g.effects.Add(permanentColorEffect(perm, *colors, colorSourceID))
	}

	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				e = g.bindCharacteristicEffect(e, perm)
				// Set the source ID on the continuous effect
				g.setEffectSource(e, perm.ID())
				g.effects.Add(printedStaticEffect(e, perm, sa))
			}
		}
	}
}

func (g *Game) completeBattlefieldEntry(perm *Permanent, controller, actor uuid.UUID, castColorsSpent int) {
	g.completeBattlefieldEntryFrom(perm, controller, actor, castColorsSpent, ZoneAny)
}

func (g *Game) completeBattlefieldEntryFrom(perm *Permanent, controller, actor uuid.UUID, castColorsSpent int, origin Zone) {
	g.observeBattlefieldEntryFrom(perm, controller, actor, castColorsSpent, origin)
	g.finishRetainedEntry(perm)
}

func (g *Game) observeBattlefieldEntryFrom(perm *Permanent, controller, actor uuid.UUID, castColorsSpent int, origin Zone) {
	// Run unconditional ETB effects (e.g. Primal Clay mode choice)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBEffectAbility); ok {
			_ = ApplyEffect(g, etb.Effect, perm.ID(), controller, nil)
		}
	}

	// Run ETB-with-targets effects (e.g. Oubliette exile on entry)
	for _, a := range perm.RuntimeAbilities {
		if etb, ok := a.(*ETBWithTargetsAbility); ok && len(g.resolution.ResolvingTargets()) > 0 {
			_ = ApplyEffect(g, etb.Effect, perm.ID(), controller, g.resolution.ResolvingTargets())
			break
		}
	}

	g.effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:            EvtZoneChange,
		SourceID:        perm.ID(),
		PlayerID:        controller,
		ActorID:         actor,
		Amount:          g.resolution.X(), // preserve X from resolving spell for ETB triggers
		CastColorsSpent: castColorsSpent,
		FromZone:        origin,
		ToZone:          ZoneBattlefield,
	})
}

// PutOnBattlefieldAttacking (CR 508.4) puts a creature onto the battlefield and
// marks it as attacking the given defender (player or planeswalker). For the
// purpose of trigger events and effects, such a creature is "attacking" but
// never "attacked" — AttacksTrigger does not fire. Per CR 508.4a, if the
// specified defender is no longer in the game (zero UUID or unknown player),
// the creature still enters but never becomes an attacking creature.
func (g *Game) PutOnBattlefieldAttacking(card Card, controller, defenderID uuid.UUID) *Permanent {
	if card.Owner() == uuid.Nil {
		card.SetOwner(controller)
	}
	perm := g.PutOnBattlefield(card, controller)
	if perm == nil || defenderID == uuid.Nil || !g.isValidDefender(defenderID) {
		return perm
	}
	g.combat.AddAttackerIncarnation(perm.ID(), perm.incarnationID, defenderID)
	return perm
}

// PutOnBattlefieldBlocking (CR 509.4) puts a creature onto the battlefield and
// marks it as blocking the given attacker. Per CR 509.4, such a creature is
// "blocking" but never "blocked" — BlocksTrigger does not fire. Per CR 509.4a,
// if the specified attacker is no longer attacking, the creature still enters
// but never becomes a blocking creature.
func (g *Game) PutOnBattlefieldBlocking(card Card, controller, attackerID uuid.UUID) *Permanent {
	if card.Owner() == uuid.Nil {
		card.SetOwner(controller)
	}
	perm := g.PutOnBattlefield(card, controller)
	if perm == nil || attackerID == uuid.Nil || !g.combat.IsAttacking(attackerID) {
		return perm
	}
	wasBlocked := g.attackerIsBlocked(attackerID)
	g.combat.AddBlockerIncarnation(perm.ID(), perm.incarnationID, attackerID)
	g.trackers.Turn.RecordBlocked(perm.ID(), attackerID)
	if attacker := g.FindPermanent(attackerID); attacker != nil {
		g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: attackerID, TargetID: perm.ID(), PlayerID: attacker.ControllerID()})
		g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: attackerID, TargetID: perm.ID(), PlayerID: attacker.ControllerID()})
		g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: attackerID, RelatedIDs: []uuid.UUID{perm.ID()}, PlayerID: attacker.ControllerID()})
	}
	if !wasBlocked {
		g.fireCreatureBecomesBlocked(attackerID)
	}
	return perm
}

func (g *Game) attackerIsBlocked(attackerID uuid.UUID) bool {
	for _, group := range g.combat.Groups {
		if group.AttackerID == attackerID {
			return group.Blocked
		}
	}
	return false
}

func (g *Game) fireUnblockedAttackers() {
	for _, group := range g.combat.Groups {
		if group.Blocked || !g.combat.IsAttacking(group.AttackerID) {
			continue
		}
		attacker := g.FindPermanent(group.AttackerID)
		if attacker == nil {
			continue
		}
		g.FireEvent(GameEvent{Type: EvtAttacksUnblocked, SourceID: attacker.ID(), SourceIncarnation: attacker.incarnationID, PlayerID: attacker.ControllerID(), TargetID: group.DefenderID})
	}
}

func (g *Game) blockedAttackersBeforeDeclaration() map[uuid.UUID]bool {
	before := make(map[uuid.UUID]bool, len(g.combat.Groups))
	for _, group := range g.combat.Groups {
		before[group.AttackerID] = group.Blocked
	}
	return before
}

type combatBlockPair struct {
	attacker uuid.UUID
	blocker  uuid.UUID
	object   uuid.UUID
}

func (g *Game) blockerPairsBeforeDeclaration() map[combatBlockPair]bool {
	before := make(map[combatBlockPair]bool)
	for _, group := range g.combat.Groups {
		for i, blockerID := range group.BlockerIDs {
			if i < len(group.BlockerIncarnations) {
				before[combatBlockPair{group.AttackerID, blockerID, group.BlockerIncarnations[i]}] = true
			}
		}
	}
	return before
}

func (g *Game) fireNewBlockerPairs(before map[combatBlockPair]bool) {
	for _, group := range g.combat.Groups {
		attacker := g.FindPermanent(group.AttackerID)
		if attacker == nil {
			continue
		}
		for i, blockerID := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{group.AttackerID, blockerID, group.BlockerIncarnations[i]}
			blocker := g.FindPermanent(blockerID)
			if before[pair] || blocker == nil || blocker.incarnationID != pair.object {
				continue
			}
			g.FireEvent(GameEvent{Type: EvtCreatureBlockedBy, SourceID: attacker.ID(), TargetID: blockerID, PlayerID: attacker.ControllerID()})
			g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: attacker.ID(), TargetID: blockerID, PlayerID: attacker.ControllerID()})
		}
	}
}

func (g *Game) fireNewBlockerGroups(before map[combatBlockPair]bool) {
	type group struct {
		source     uuid.UUID
		blocking   bool
		controller uuid.UUID
		related    []uuid.UUID
	}
	var groups []group
	type groupKey struct {
		source   uuid.UUID
		blocking bool
	}
	indices := make(map[groupKey]int)
	add := func(source, other, controller uuid.UUID, blocking bool) {
		key := groupKey{source, blocking}
		index, ok := indices[key]
		if !ok {
			index = len(groups)
			indices[key] = index
			groups = append(groups, group{source: source, blocking: blocking, controller: controller})
		}
		groups[index].related = append(groups[index].related, other)
	}
	for _, combatGroup := range g.combat.Groups {
		attacker := g.FindPermanent(combatGroup.AttackerID)
		if attacker == nil || attacker.PhasedOut {
			continue
		}
		for i, blockerID := range combatGroup.BlockerIDs {
			if i >= len(combatGroup.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{combatGroup.AttackerID, blockerID, combatGroup.BlockerIncarnations[i]}
			blocker := g.FindPermanent(blockerID)
			if before[pair] || blocker == nil || blocker.PhasedOut || blocker.incarnationID != pair.object {
				continue
			}
			add(attacker.ID(), blockerID, attacker.ControllerID(), false)
			add(blockerID, attacker.ID(), blocker.ControllerID(), true)
		}
	}
	for _, grouped := range groups {
		g.FireEvent(GameEvent{Type: EvtBlockerGroup, SourceID: grouped.source, PlayerID: grouped.controller, Flag: grouped.blocking, RelatedIDs: grouped.related})
	}
}

func (g *Game) fireDeclaredBlockerPairs(before map[combatBlockPair]bool) {
	seen := make(map[uuid.UUID]bool)
	for pair := range before {
		seen[pair.blocker] = true
	}
	for _, group := range g.combat.Groups {
		for i, blockerID := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{group.AttackerID, blockerID, group.BlockerIncarnations[i]}
			blocker := g.FindPermanent(blockerID)
			if before[pair] || blocker == nil || blocker.incarnationID != pair.object {
				continue
			}
			g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: blockerID, TargetID: group.AttackerID, PlayerID: blocker.ControllerID(), Flag: !seen[blockerID]})
			g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: blockerID, TargetID: group.AttackerID, PlayerID: blocker.ControllerID(), Flag: true})
			seen[blockerID] = true
		}
	}
}

func (g *Game) fireCreatureBecomesBlocked(attackerID uuid.UUID) {
	attacker := g.FindPermanent(attackerID)
	if attacker != nil {
		defender := uuid.Nil
		for _, group := range g.combat.Groups {
			if group.AttackerID != attackerID {
				continue
			}
			defender = group.DefenderID
			if permanent := g.FindPermanent(defender); permanent != nil {
				defender = permanent.ControllerID()
			}
			break
		}
		g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: attackerID, PlayerID: attacker.ControllerID(), TargetID: defender})
	}
}

func (g *Game) fireNewlyBlockedAttackers(before map[uuid.UUID]bool) {
	for _, group := range g.combat.Groups {
		if !before[group.AttackerID] && group.Blocked && len(group.BlockerIDs) > 0 {
			g.fireCreatureBecomesBlocked(group.AttackerID)
		}
	}
}

func (g *Game) isValidDefender(id uuid.UUID) bool {
	for _, p := range g.players {
		if p.PlayerID() == id {
			return true
		}
	}
	return g.FindPermanent(id) != nil
}

// setEffectSource sets the source ID on a continuous effect.
func (g *Game) setEffectSource(e ContinuousEffect, id uuid.UUID) {
	e.SetSourceID(id)
}

// turnFaceUp flips a face-down permanent face up, restoring its original characteristics.
func (g *Game) turnFaceUp(perm *Permanent) {
	perm = g.MutablePermanent(perm.ID())
	if perm == nil {
		return
	}
	if !perm.FaceDown {
		return
	}
	perm.FaceDown = false
	perm.BasePTOverride = nil
	// Restore original abilities from the card
	perm.RuntimeAbilities = nil
	for _, a := range perm.Card.Abilities() {
		cp := a
		perm.RuntimeAbilities = append(perm.RuntimeAbilities, cp)
	}
	g.syncAbilityContext(perm)
	// Register continuous effects from static abilities
	for _, a := range perm.RuntimeAbilities {
		if sa, ok := a.(*StaticAbilityHolder); ok {
			for _, e := range sa.Effects {
				e = g.bindCharacteristicEffect(e, perm)
				g.setEffectSource(e, perm.ID())
				g.effects.Add(printedStaticEffect(e, perm, sa))
			}
		}
	}
	g.effects.Apply(g)
}

// RemoveFromBattlefield removes a permanent and handles cleanup.
func (g *Game) RemoveFromBattlefield(perm *Permanent) {
	if perm == nil {
		return
	}
	g.turns.RemoveSkipNextUntap(perm.ID())
	perm = g.mutablePermanentIncludingPhased(perm.ID())
	if perm == nil {
		return
	}
	// Snapshot LKI before any state mutation so death/leave-triggers can
	// read the dying permanent's controller, types, and P/T after the move.
	g.captureLKI(perm)
	g.RemoveFromCombat(perm.ID())

	// Remove continuous effects sourced from this permanent
	g.effects.Remove(perm.ID())

	// If this was attached to something, remove it from that thing's attachments
	if perm.IsAttached() {
		host := g.MutablePermanent(perm.AttachedTo)
		if host != nil {
			filtered := host.Attachments[:0]
			for _, id := range host.Attachments {
				if id != perm.ID() {
					filtered = append(filtered, id)
				}
			}
			host.Attachments = filtered
		}
	}

	// Snapshot attachments before removal so we can clean them up after
	attachments := make([]uuid.UUID, len(perm.Attachments))
	copy(attachments, perm.Attachments)

	// Remove from battlefield
	g.zones.RemovePermanent(perm.ID())

	if g.triggers.zoneBatch == nil {
		g.effects.Apply(g)
	}
	// Leave-zone trigger dispatch (CR 603.6c) happens in the destination-
	// specific path (Destroy / Sacrifice / Exile / Bounce /
	// PutPermanentIntoGraveyard) via EvtZoneChange + LKIAbilities. The
	// permanent's runtime abilities are captured into LKI by captureLKI
	// above.

	// Detach equipment immediately. Auras are left for SBAs to put into the
	// graveyard so that "when enchanted creature dies" triggers can still see
	// the attachment relationship when the host's death events fire.
	for _, attID := range attachments {
		att := g.MutablePermanent(attID)
		if att == nil {
			continue
		}
		if !att.HasSubType("Aura") {
			att.AttachedTo = uuid.Nil
			att.attachedIncarnation = uuid.Nil
		}
	}
}

// DestroyPermanent destroys a permanent (sends to graveyard).
func (g *Game) DestroyPermanent(perm *Permanent) {
	if perm == nil || perm.PhasedOut {
		return
	}
	current := g.FindPermanent(perm.ID())
	if current == nil || current.incarnationID != perm.incarnationID {
		return
	}
	perm = current
	if perm.HasKeyword(Indestructible) {
		return
	}
	// Run through the replacement pipeline (regeneration is now a replacement)
	action := NewDestroyPermanentAction(uuid.Nil, perm.ID())
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return // regenerated or otherwise replaced
	}
	if move, ok := result.(*permanentGraveyardAction); ok {
		g.performPermanentGraveyardAction(move)
		return
	}
	controller := perm.ControllerID()
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		panic("DestroyPermanent: Pemanent has no owner")
	}

	isCreature := perm.HasType(TypeCreature)
	permID := perm.ID()
	card := cardForBattlefieldDeparture(perm.Card)

	isToken := perm.IsToken

	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)

	if !isToken {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(card)
		}
	}

	if isCreature {
		g.trackers.Turn.RecordCreatureDeath()
		g.recordCreatureDeath(controller)
	}

	graveyardEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)
}

// TapPermanent taps a permanent and fires the EvtTapped event.
func (g *Game) TapPermanent(perm *Permanent) {
	if perm == nil {
		return
	}
	perm = g.MutablePermanent(perm.ID())
	if perm == nil || perm.Tapped {
		return
	}
	perm.Tapped = true
	g.FireEvent(GameEvent{
		Type:     EvtTapped,
		SourceID: perm.ID(),
		PlayerID: perm.ControllerID(),
	})
}

// tapPermanentsTogether makes each tap visible before it sends tap events.
func (g *Game) tapPermanentsTogether(permanents []*Permanent) {
	for _, permanent := range permanents {
		g.MutablePermanent(permanent.ID()).Tapped = true
	}
	for _, permanent := range permanents {
		g.FireEvent(GameEvent{
			Type:     EvtTapped,
			SourceID: permanent.ID(),
			PlayerID: permanent.ControllerID(),
		})
	}
}

// checkAbilitiesForEvent checks a set of abilities (from a removed permanent) for triggers.
func (g *Game) checkAbilitiesForEvent(abilities []Ability, evt *GameEvent, sourceID, controller uuid.UUID) {
	if g.triggers.zoneBatch != nil && isBattlefieldLeaveEvent(evt) {
		return
	}
	if isBattlefieldLeaveEvent(evt) && evt.SourceIncarnation == uuid.Nil {
		if lki := g.zones.LKI(evt.SourceID); lki != nil && lki.Snapshot != nil {
			captured := *evt
			captured.SourceIncarnation = lki.Snapshot.incarnationID
			evt = &captured
		}
	}
	for _, a := range abilities {
		ta, ok := UnwrapAbility(a).(TriggeredAbility)
		if !ok {
			continue
		}
		ta.SetSource(sourceID)
		ta.SetController(controller)
		if ta.IsStateTrigger() {
			continue
		}
		if !ta.CheckEventType(evt.Type) {
			continue
		}
		if ta.CheckTrigger(evt, g) {
			g.queuePendingTrigger(&pendingTrigger{
				eventCaptures: captureDetectedTriggerEvent(ta, evt, g, g, sourceID),
				ability:       ta,
				event:         evt,
				sourceID:      sourceID,
				controller:    controller,
			})
		}
	}
}

func (g *Game) syncAbilityContext(perm *Permanent) {
	if perm == nil {
		return
	}
	perm = g.isolatePrintedLandwalkContext(perm)
	perm = g.isolatePrintedProtectionContext(perm)
	perm = g.isolatePrintedStaticAbilityContext(perm)
	perm = g.isolatePrintedManaTapContext(perm)
	perm = g.isolatePrintedSpellCastContext(perm)
	perm = g.isolatePrintedUpkeepContext(perm)
	perm = g.isolatePrintedCombatPairContext(perm)
	perm = g.isolatePrintedLandTapContext(perm)
	perm = g.isolatePrintedActivatedContext(perm)
	for _, ability := range perm.RuntimeAbilities {
		unwrapped := UnwrapAbility(ability)
		unwrapped.SetSource(perm.ID())
		unwrapped.SetController(perm.ControllerID())
	}
}

// PutPermanentIntoGraveyard puts a permanent into its owner's graveyard without
// destroying it. This bypasses indestructible and regeneration. Used by SBAs
// (e.g., 0-toughness creatures) and other rules that move permanents to the
// graveyard without destruction.
func (g *Game) PutPermanentIntoGraveyard(perm *Permanent) {
	if perm == nil || perm.PhasedOut {
		return
	}
	current := g.FindPermanent(perm.ID())
	if current == nil || current.incarnationID != perm.incarnationID {
		return
	}
	proposal := &permanentGraveyardAction{subject: perm.ID(), incarnation: perm.incarnationID, destination: ZoneGraveyard}
	if result, ok := g.effects.ApplyReplacements(proposal, g).(*permanentGraveyardAction); ok {
		g.performPermanentGraveyardAction(result)
	}
}

func (g *Game) putPermanentIntoGraveyard(perm *Permanent) {
	controller := perm.ControllerID()
	owner := perm.Card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}

	isCreature := perm.HasType(TypeCreature)
	isToken := perm.IsToken
	permID := perm.ID()
	card := cardForBattlefieldDeparture(perm.Card)

	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)

	if !isToken {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(card)
		}
	}

	if isCreature {
		g.trackers.Turn.RecordCreatureDeath()
		g.recordCreatureDeath(controller)
	}

	graveyardEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneGraveyard,
	}
	g.FireEvent(graveyardEvt)
	g.checkAbilitiesForEvent(selfAbilities, &graveyardEvt, permID, controller)
}

// MoveFromGraveyard removes a card from the graveyard of a player.
// The function sends an EvtZoneChange event for the card.
// The function also sends an EvtCardsLeftGraveyard event with Amount set to 1.
// The function does not put the card into the destination zone.
// The caller must put the card into the destination zone.
//
// The function returns the removed card, or nil and false if the card was not present.
//
// When multiple cards leave a graveyard together (CR 603.10), use MoveCardsFromGraveyard.
func (g *Game) MoveFromGraveyard(playerID, cardID uuid.UUID, to Zone) (Card, bool) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil, false
	}
	card, ok := p.RemoveFromGraveyard(cardID)
	if !ok {
		return nil, false
	}
	g.FireEvent(GameEvent{
		Type:     EvtZoneChange,
		SourceID: cardID,
		PlayerID: playerID,
		FromZone: ZoneGraveyard,
		ToZone:   to,
	})
	g.FireEvent(GameEvent{
		Type:     EvtCardsLeftGraveyard,
		PlayerID: playerID,
		Amount:   1,
	})
	return card, true
}

// MoveFromAnyGraveyard removes a card from whichever player's graveyard
// contains it and returns that player's ID for zone-change attribution.
func (g *Game) MoveFromAnyGraveyard(cardID uuid.UUID, to Zone) (Card, uuid.UUID, bool) {
	for _, player := range g.players {
		if card, ok := g.MoveFromGraveyard(player.PlayerID(), cardID, to); ok {
			return card, player.PlayerID(), true
		}
	}
	return nil, uuid.Nil, false
}

// MoveCardsFromGraveyard removes the listed cards from playerID's graveyard,
// firing per-card EvtZoneChange events and a SINGLE EvtCardsLeftGraveyard
// event with Amount = number of cards actually removed (CR 603.10 — multiple
// cards moving via the same effect form one zone-change event for the
// purposes of "one or more cards leave your graveyard" triggers). Returns
// the removed cards in input order, skipping any that weren't found.
func (g *Game) MoveCardsFromGraveyard(playerID uuid.UUID, cardIDs []uuid.UUID, to Zone) []Card {
	return g.moveCardsFromGraveyard(playerID, cardIDs, to, nil)
}

func (g *Game) moveCardsFromGraveyard(playerID uuid.UUID, cardIDs []uuid.UUID, to Zone, destination func(Card)) []Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	removed := make([]Card, 0, len(cardIDs))
	var events []GameEvent
	for _, id := range cardIDs {
		c, ok := p.RemoveFromGraveyard(id)
		if !ok {
			continue
		}
		removed = append(removed, c)
		event := GameEvent{Type: EvtZoneChange, SourceID: id, PlayerID: playerID, FromZone: ZoneGraveyard, ToZone: to}
		if destination == nil {
			g.FireEvent(event)
		} else {
			destination(c)
			events = append(events, event)
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
	if len(removed) > 0 {
		g.FireEvent(GameEvent{Type: EvtCardsLeftGraveyard, PlayerID: playerID, Amount: len(removed)})
	}
	return removed
}

// PlayerDiscard removes a card from the player's hand into their graveyard and
// fires EvtDiscard. SourceID = card ID, PlayerID = discarding player.
// Returns the card and true on success.
func (g *Game) PlayerDiscard(p Player, cardID uuid.UUID) (Card, bool) {
	c, ok := p.DiscardCard(cardID)
	if !ok {
		return nil, false
	}
	g.FireEvent(GameEvent{
		Type:               EvtDiscard,
		SourceID:           cardID,
		PlayerID:           p.PlayerID(),
		DiscardedCardVisit: g.graveyardVisit(cardID),
	})
	return c, true
}

// PlayerDiscardByEffect discards a card through the replacement pipeline.
// Use PlayerDiscardByCost for cost discards.
func (g *Game) PlayerDiscardByEffect(p Player, cardID, sourceID uuid.UUID) (Card, bool) {
	if p == nil {
		return nil, false
	}
	return g.playerDiscardAction(NewDiscardAction(sourceID, p.PlayerID(), cardID))
}

func (g *Game) PlayerDiscardByCost(p Player, cardID, sourceID uuid.UUID) (Card, bool) {
	if p == nil {
		return nil, false
	}
	return g.playerDiscardAction(newCostDiscardAction(sourceID, p.PlayerID(), cardID))
}

func (g *Game) playerDiscardAction(action *DiscardAction) (Card, bool) {
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return nil, false
	}
	discard, ok := result.(*DiscardAction)
	if !ok {
		return nil, false
	}
	return g.executeDiscard(discard)
}

func (g *Game) executeDiscard(action *DiscardAction) (Card, bool) {
	c, ok := g.moveDiscardedCard(action)
	if !ok {
		return nil, false
	}
	g.FireEvent(GameEvent{Type: EvtDiscard, SourceID: c.ID(), PlayerID: action.PlayerID(), DiscardedCardVisit: g.graveyardVisit(c.ID())})
	return c, true
}

func (g *Game) moveDiscardedCard(action *DiscardAction) (Card, bool) {
	if action.handVisit != uuid.Nil && g.handVisit(action.CardID()) != action.handVisit {
		return nil, false
	}
	p := g.GetPlayer(action.PlayerID())
	if p == nil {
		return nil, false
	}
	c, ok := p.RemoveFromHand(action.CardID())
	if !ok {
		return nil, false
	}
	if action.Destination() == ZoneLibrary {
		p.SetLibrary(append([]Card{c}, p.Library()...))
	} else {
		p.AddToGraveyard(c)
	}
	return c, true
}

// PlayerLoseLife reduces a player's life total and fires EvtLifeLost. Used by
// effects that cause direct life loss (CR 119.3). Damage that causes life loss
// fires EvtLifeLost separately from executeDamageToPlayer.
func (g *Game) PlayerLoseLife(p Player, amount int) {
	g.playersLoseLife([]Player{p}, amount)
}

// playersLoseLife changes all life totals before events can observe them.
func (g *Game) playersLoseLife(players []Player, amount int) int {
	if amount <= 0 {
		return 0
	}
	lost := 0
	for _, p := range players {
		before := p.Life()
		p.LoseLife(amount)
		lost += before - p.Life()
	}
	for _, p := range players {
		g.FireEvent(GameEvent{
			Type:     EvtLifeLost,
			PlayerID: p.PlayerID(),
			Amount:   amount,
		})
	}
	return lost
}

// PlayerGainLife applies modifiers and replacements without event dispatch.
func (g *Game) PlayerGainLife(p Player, amount int) {
	g.applyLifeGain(p, amount)
}

func (g *Game) gainLifeWithEvent(p Player, amount int) {
	if gained := g.applyLifeGain(p, amount); gained > 0 {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: p.PlayerID(), Amount: gained})
	}
}

// playersGainLife applies each gain before observers read life totals.
func (g *Game) playersGainLife(players []Player, amount int) int {
	if amount <= 0 {
		return 0
	}
	type gain struct {
		player Player
		amount int
	}
	var gains []gain
	seen := make(map[uuid.UUID]bool, len(players))
	for _, player := range players {
		if player == nil || seen[player.PlayerID()] {
			continue
		}
		seen[player.PlayerID()] = true
		if actual := g.prepareLifeGain(player, amount); actual > 0 {
			gains = append(gains, gain{player, actual})
		}
	}
	total := 0
	for _, result := range gains {
		result.player.GainLife(result.amount)
		total += result.amount
	}
	for _, result := range gains {
		g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: result.player.PlayerID(), Amount: result.amount})
	}
	return total
}

// applyLifeGain returns the amount gained after replacement processing.
func (g *Game) applyLifeGain(p Player, amount int) int {
	gained := g.prepareLifeGain(p, amount)
	if gained > 0 {
		p.GainLife(gained)
	}
	return gained
}

func (g *Game) prepareLifeGain(p Player, amount int) int {
	if amount <= 0 {
		return 0
	}
	amount = g.ApplyLifeGainModifiers(p.PlayerID(), amount)
	if amount <= 0 {
		return 0
	}
	action := NewLifeGainAction(uuid.Nil, p.PlayerID(), amount)
	result := g.effects.ApplyReplacements(action, g)
	if gain, ok := result.(*LifeGainAction); ok && gain.Amount() > 0 {
		return gain.Amount()
	}
	return 0
}

// BouncePermanentToHand removes a permanent from the battlefield and adds the
// underlying card to its owner's hand, firing an EvtZoneChange{From: BF,
// To: Hand}. Self-referencing leave triggers fire via the LKI snapshot's
// captured abilities per CR 603.6c.
func (g *Game) BouncePermanentToHand(perm *Permanent) {
	if perm == nil {
		return
	}
	controller := perm.ControllerID()
	permID := perm.ID()
	card := cardForBattlefieldDeparture(perm.Card)
	isToken := perm.IsToken
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = controller
	}
	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)
	if !isToken {
		// CR 111.7 / 111.10g: tokens cease to exist when they leave the
		// battlefield. The zone-change event still fires, but the card is
		// not added to a hand.
		if p := g.GetPlayer(owner); p != nil {
			p.AddToHand(card)
		}
	}
	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneHand,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfAbilities, &zoneEvt, permID, controller)
}

// CounterSpellOnStack removes a spell from the stack by its source ID.
// The countered spell's card goes to its owner's graveyard.
//
// If the targeted spell is currently uncounterable — either because the
// card was registered with [WithUncounterable] or a static filter
// installed via [RegisterUncounterableStatic] matches — the counter has
// no effect: the spell stays on the stack.
func (g *Game) CounterSpellOnStack(spellID uuid.UUID) {
	g.counterStackObject(g.stack.FindBySourceID(spellID))
}

// CounterStackObject counters a spell or ability by its stack object ID (CR 701.5a).
func (g *Game) CounterStackObject(id uuid.UUID) {
	obj := g.stack.FindByID(id)
	if obj == nil {
		return
	}
	g.counterStackObject(obj)
}

func (g *Game) counterStackObject(obj *StackObject) {
	if obj == nil || g.IsSpellUncounterable(obj) {
		return
	}
	g.stack.RemoveByID(obj.ID)
	if obj.ExileOnLeaveStack || g.spellMarkedExileInstead(obj) {
		g.cleanupSpell(obj)
		return
	}
	if obj.Card != nil && !obj.IsCopy {
		if owner := g.GetPlayer(obj.Card.Owner()); owner != nil {
			owner.AddToGraveyard(obj.Card)
		}
	}
}

// PlayerDrawCard draws a card for the player, running the replacement
// pipeline first so that "if you would draw a card" replacement effects
// (Aladdin's Lamp, Ormos's empty-library counters, etc.) intercept the
// draw. When the action is fully replaced, no card is drawn and (false,
// nil) is returned; otherwise the next library card moves to hand and
// EvtCardDrawn fires. The draw is marked isNormalDraw=false because this
// entry point is used by effect-driven draws; the turn-based draw step
// builds its own action with isNormalDraw=true.
func (g *Game) PlayerDrawCard(p Player) (Card, bool) {
	if p == nil {
		return nil, false
	}
	action := g.drawAction(p.PlayerID(), false)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return nil, false
	}
	return g.drawCardRaw(p)
}

// drawCardRaw performs the underlying library-to-hand transfer and fires
// EvtCardDrawn without running the replacement pipeline. Used by the
// normal draw step (which applies replacements itself) and by
// replacement implementations that need to draw after rearranging the
// library (Aladdin's Lamp).
func (g *Game) drawCardRaw(p Player) (Card, bool) {
	c, ok := p.DrawCard()
	if ok {
		if g.turns.Step() == Draw && g.ActivePlayerObj().PlayerID() == p.PlayerID() {
			g.turns.drawStepDraws++
		}
		var visit uuid.UUID
		if visitor, hasVisit := p.(interface{ handVisit(uuid.UUID) uuid.UUID }); hasVisit {
			visit = visitor.handVisit(c.ID())
		}
		g.FireEvent(GameEvent{
			Type:           EvtCardDrawn,
			PlayerID:       p.PlayerID(),
			DrawnCardID:    c.ID(),
			DrawnCardVisit: visit,
		})
	}
	return c, ok
}

// PerformScry executes the scry action for N cards (CR 701.22).
// The player examines the top N cards of the library.
// The player can put any number of these cards on the bottom of the library.
// The player puts the remaining cards on top of the library in any order.
// If the library contains fewer than N cards, the player examines all available cards.
// The function returns the quantity of cards examined.
//
// The method delegates card placement to Player.ChooseScryPlacement.
// The engine validates the returned IDs.
// If an ID mismatch occurs, the engine puts all cards on top in the original order.
func (g *Game) PerformScry(p Player, n int) int {
	if p == nil || n <= 0 {
		return 0
	}
	lib := p.Library()
	if len(lib) == 0 {
		g.FireEvent(GameEvent{Type: EvtScry, PlayerID: p.PlayerID()})
		return 0
	}
	count := min(n, len(lib))
	top := make([]Card, count)
	copy(top, lib[:count])

	bottom, topOrder := p.ChooseScryPlacement(top, "scry", g)
	bottom, topOrder = validateScryPlacement(top, bottom, topOrder)

	idToCard := make(map[uuid.UUID]Card, count)
	for _, c := range top {
		idToCard[c.ID()] = c
	}

	rest := lib[count:]
	newLib := make([]Card, 0, len(lib))
	for _, id := range topOrder {
		newLib = append(newLib, idToCard[id])
	}
	newLib = append(newLib, rest...)
	for _, id := range bottom {
		newLib = append(newLib, idToCard[id])
	}
	p.SetLibrary(newLib)

	g.FireEvent(GameEvent{
		Type:     EvtScry,
		PlayerID: p.PlayerID(),
		Amount:   count,
	})
	return count
}

// PerformSurveil implements surveil N (CR 701.25): the player looks at the top
// N cards of their library, then puts any number of them into their graveyard
// and the rest on top of their library in any order. If the library has fewer
// than N cards, the player surveils however many are present. Returns the
// number of cards actually surveiled.
//
// The placement decision is delegated to Player.ChooseSurveilPlacement; the
// engine validates the returned IDs and falls back to "all on top, original
// order" on any mismatch so a buggy player implementation cannot lose cards.
func (g *Game) PerformSurveil(p Player, n int) int {
	if p == nil || n <= 0 {
		return 0
	}
	lib := p.Library()
	if len(lib) == 0 {
		g.FireEvent(GameEvent{Type: EvtSurveil, PlayerID: p.PlayerID()})
		return 0
	}
	count := min(n, len(lib))
	top := make([]Card, count)
	copy(top, lib[:count])

	graveyard, topOrder := p.ChooseSurveilPlacement(top, "surveil", g)
	graveyard, topOrder = validateScryPlacement(top, graveyard, topOrder)

	idToCard := make(map[uuid.UUID]Card, count)
	for _, c := range top {
		idToCard[c.ID()] = c
	}

	rest := lib[count:]
	newLib := make([]Card, 0, len(lib))
	for _, id := range topOrder {
		newLib = append(newLib, idToCard[id])
	}
	newLib = append(newLib, rest...)
	p.SetLibrary(newLib)
	for _, id := range graveyard {
		if c, ok := idToCard[id]; ok {
			p.AddToGraveyard(c)
		}
	}
	g.FireEvent(GameEvent{Type: EvtSurveil, PlayerID: p.PlayerID(), Amount: count})
	return count
}

// validateScryPlacement ensures the player's choice is a valid partition of
// `top`. On any inconsistency it returns the safe default (all on top in the
// original order) so cards are never dropped.
func validateScryPlacement(top []Card, bottom, topOrder []uuid.UUID) (validBottom, validTopOrder []uuid.UUID) {
	want := make(map[uuid.UUID]bool, len(top))
	for _, c := range top {
		want[c.ID()] = true
	}
	if len(bottom)+len(topOrder) != len(top) {
		return nil, defaultScryTopOrder(top)
	}
	seen := make(map[uuid.UUID]bool, len(top))
	for _, id := range topOrder {
		if !want[id] || seen[id] {
			return nil, defaultScryTopOrder(top)
		}
		seen[id] = true
	}
	for _, id := range bottom {
		if !want[id] || seen[id] {
			return nil, defaultScryTopOrder(top)
		}
		seen[id] = true
	}
	return bottom, topOrder
}

func defaultScryTopOrder(top []Card) []uuid.UUID {
	out := make([]uuid.UUID, len(top))
	for i, c := range top {
		out[i] = c.ID()
	}
	return out
}

// Attach attaches source to target (for auras and equipment).
func (g *Game) Attach(sourceID, targetID uuid.UUID) {
	src := g.MutablePermanent(sourceID)
	target := g.MutablePermanent(targetID)
	if !g.canAttach(src, target) {
		return
	}

	// Detach from current host if any
	if src.IsAttached() {
		oldHost := g.MutablePermanent(src.AttachedTo)
		if oldHost != nil {
			filtered := oldHost.Attachments[:0]
			for _, id := range oldHost.Attachments {
				if id != sourceID {
					filtered = append(filtered, id)
				}
			}
			oldHost.Attachments = filtered
		}
	}

	src.enchantedCard = capturedCard{}
	src.AttachedTo = targetID
	src.attachedIncarnation = target.incarnationID
	target.Attachments = append(target.Attachments, sourceID)
	g.effects.retimestampAttachment(sourceID)

	g.effects.Apply(g)

	g.FireEvent(GameEvent{
		Type:     EvtAttach,
		SourceID: sourceID,
		TargetID: targetID,
	})
}

func (g *Game) canAttach(src, target *Permanent) bool {
	if src == nil || target == nil || src.AttachedTo == target.ID() && src.attachedIncarnation == target.incarnationID {
		return false
	}
	return g.attachmentHostIsLegal(src, target)
}

func (g *Game) attachmentHostIsLegal(src, target *Permanent) bool {
	if src == nil || target == nil {
		return false
	}
	if src.HasSubType("Aura") {
		if src.HasType(TypeCreature) || src.ID() == target.ID() ||
			target.HasAttr(AttrCantBeEnchanted) || target.HasProtectionFromInGame(src.Card, g) ||
			!auraHostIsLegalForController(src.Card, target, g, src.ControllerID()) {
			return false
		}
	} else if src.HasSubType("Equipment") {
		if src.HasType(TypeCreature) || !target.HasType(TypeCreature) || src.ID() == target.ID() || target.HasProtectionFromInGame(src.Card, g) {
			return false
		}
	} else if !src.HasSubType("Fortification") || !target.HasType(TypeLand) || src.ID() == target.ID() || target.HasProtectionFromInGame(src.Card, g) {
		return false
	}
	return true
}

// auraHostIsLegal checks the Aura's enchant limits for its current controller.
// Shroud and hexproof do not restrict an existing attachment (CR 303.4c).
func auraHostIsLegal(auraCard Card, host *Permanent, g *Game) bool {
	controller := auraCard.Owner()
	if aura := g.FindPermanent(auraCard.ID()); aura != nil {
		controller = aura.ControllerID()
	}
	return auraHostIsLegalForController(auraCard, host, g, controller)
}

func auraHostIsLegalForController(auraCard Card, host *Permanent, g *Game, controller uuid.UUID) bool {
	if g.restrictedAuraAttachment(host, auraCard) {
		return false
	}
	targets := auraCard.CastTargets()
	if aura := g.FindPermanent(auraCard.ID()); aura != nil {
		kept := make([]Target, 0, len(targets))
		for _, target := range targets {
			if aura.printedAbilitiesSuppressed || aura.creatureEnchantRemoved && isFixedCreatureEnchant(target) || aura.graveyardEnchantRemoved && isGraveyardCreatureEnchant(target) {
				continue
			}
			kept = append(kept, target)
		}
		targets = kept
		ctx := &EffectContext{Game: g, SourceID: aura.ID(), Controller: controller, sourceCaptured: true, sourceIncarnation: aura.incarnationID}
		for _, hosts := range aura.grantedEnchants {
			matched := false
			for _, member := range hosts.resolvePermanents(ctx) {
				if member.ID() == host.ID() && member.incarnationID == host.incarnationID {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	for _, target := range targets {
		if !enchantTargetMatchesHost(target, host, g, controller) {
			return false
		}
	}
	return true
}

func enchantTargetMatchesHost(target Target, host *Permanent, g *Game, controller uuid.UUID) bool {
	if binding, ok := target.(TargetBinding); ok {
		target = binding.targetSpec()
	}
	switch t := UnwrapTarget(target).(type) {
	case *AnyGraveyardCardTarget:
		return !t.graveyardEnchant
	case *CreatureTarget:
		return host.HasType(TypeCreature) && filtersMatch(t.Filters, host, g)
	case *PermanentTarget:
		if t.controllerOnly && host.ControllerID() != controller || t.opponentOnly && host.ControllerID() == controller {
			return false
		}
		return t.Filter().Match(host, g)
	default:
		return true
	}
}

func filtersMatch(filters []PermanentFilter, p *Permanent, g *Game) bool {
	for _, f := range filters {
		if !f.Match(p, g) {
			return false
		}
	}
	return true
}

// fireBecomesTargetEvents sends an EvtBecomesTarget event for each declared target.
// The function runs after a player puts an object on the stack and chooses targets (CR 603.6c, 119.5).
// The engine sends one event for each distinct target permanent or player.
// Field evt.SourceID contains the ID of the spell or ability source.
// Field evt.TargetID contains the ID of the targeted object.
// Field evt.PlayerID contains the ID of the spell or ability controller.
// Field evt.Flag is true for activated abilities and false for spells.
//
// CR 603.6c states that abilities trigger only once for each event when a spell has multiple targets.
// However, each targeted permanent or player receives the event independently.
// The engine sends one event for each distinct target.
// This ensures that triggers on different permanents receive separate events.
func (g *Game) fireBecomesTargetEvents(obj *StackObject, isAbility bool) {
	if obj == nil {
		return
	}
	seen := make(map[uuid.UUID]bool)
	for _, tid := range obj.Targets {
		if tid == uuid.Nil || seen[tid] {
			continue
		}
		seen[tid] = true
		g.trackers.Turn.RecordTimesTargeted(tid)
		g.FireEvent(GameEvent{
			Type:     EvtBecomesTarget,
			SourceID: obj.SourceID,
			TargetID: tid,
			PlayerID: obj.Controller,
			Flag:     isAbility,
		})
	}
}

// TimesTargetedThisTurn returns how many times the given object (permanent or
// player) has become the target of a spell or activated ability this turn.
// Counter resets at the cleanup step.
func (g *Game) TimesTargetedThisTurn(id uuid.UUID) int {
	return g.trackers.Turn.TimesTargeted(id)
}

// RegisterDelayedTrigger registers a one-shot delayed trigger that will fire
// when the specified event type occurs.
func (g *Game) RegisterDelayedTrigger(dt *DelayedTrigger) {
	if dt != nil {
		captured := *dt
		dt = &captured
		if p := g.FindPermanentIncludingPhased(dt.SourceID); !dt.sourceCaptured && p != nil {
			dt.sourceIncarnation = p.incarnationID
		}
	}
	g.triggers.RegisterDelayed(dt)
}

// DelayedTriggers returns all active delayed triggers.
func (g *Game) DelayedTriggers() []*DelayedTrigger {
	return g.triggers.Delayed()
}

// FireEvent dispatches an event and checks triggered abilities.
func (g *Game) FireEvent(evt GameEvent) {
	if g.drawn {
		return
	}
	if evt.Type == EvtSpellCast {
		if obj := g.stack.FindByID(evt.TargetID); obj != nil && !obj.IsAbility && obj.SourceID == evt.SourceID {
			evt.CastManaSpent = ManaSpentToCast(obj)
			evt.CastColorsSpent = obj.CastContext.DistinctColorsSpent()
			evt.CastAnnouncedX = obj.XValue
		}
	}
	if isBattlefieldLeaveEvent(&evt) && evt.SourceIncarnation == uuid.Nil {
		if lki := g.zones.LKI(evt.SourceID); lki != nil && lki.Snapshot != nil {
			evt.SourceIncarnation = lki.Snapshot.incarnationID
		}
	}
	if g.damageBatch != nil {
		g.damageBatch.events = append(g.damageBatch.events, evt)
		g.damageBatch.sources = append(g.damageBatch.sources, cloneDamageSource(g.damageEventSource))
		return
	}
	if batch := g.triggers.zoneBatch; batch != nil && isBattlefieldLeaveEvent(&evt) {
		batch.events = append(batch.events, evt)
		return
	}
	g.fireEvent(evt, nil)
}

func (g *Game) fireEvent(evt GameEvent, before *Game) {
	defer g.CheckStateTriggers()
	pendingBefore := len(g.triggers.Pending())
	g.recordPerTurnEvent(&evt)
	g.recordPerDuelEvent(&evt)
	observer := g
	if before != nil {
		observer = before
	}
	for _, perm := range observer.zones.battlefield {
		if perm.PhasedOut {
			continue
		}
		observer.syncAbilityContext(perm)
		for _, a := range perm.RuntimeAbilities {
			ta, ok := UnwrapAbility(a).(TriggeredAbility)
			if !ok {
				continue
			}
			if ta.IsStateTrigger() {
				continue
			}
			if ta.TriggerSourceZone() != ZoneBattlefield {
				continue
			}
			if !ta.CheckEventType(evt.Type) {
				continue
			}
			// Default-zone triggers are battlefield-only; explicit
			// non-battlefield zones are scanned in the cross-zone loop below.
			if gt, ok := ta.(*GenericTriggered); ok && !gt.FunctionsInZone(ZoneBattlefield) {
				continue
			}
			if ta.CheckTrigger(&evt, observer) {
				g.queuePendingTrigger(&pendingTrigger{
					eventCaptures: captureDetectedTriggerEvent(ta, &evt, observer, g, perm.ID()),
					ability:       ta,
					event:         &evt,
					sourceID:      perm.ID(),
					controller:    perm.ControllerID(),
				})
			}
		}
	}

	for _, pl := range g.players {
		for _, c := range pl.Graveyard() {
			g.detectCardZoneTriggers(&evt, c, pl.PlayerID(), ZoneGraveyard)
		}
	}
	for _, object := range g.zones.exile {
		if !object.FaceDown && g.GetPlayer(object.Card.Owner()) != nil {
			g.detectCardZoneTriggers(&evt, object.Card, object.Card.Owner(), ZoneExile)
		}
	}

	// Scan spell abilities that function while their source spell is on the
	// stack (CR 113.6i), such as "When you cast this spell, copy it...".
	for _, obj := range g.stack.Objects() {
		if obj == nil || obj.IsAbility || obj.Card == nil {
			continue
		}
		for _, a := range obj.Card.Abilities() {
			gt, ok := UnwrapAbility(a).(*GenericTriggered)
			if !ok {
				continue
			}
			if !gt.FunctionsInZone(ZoneStack) {
				continue
			}
			if !gt.CheckEventType(evt.Type) {
				continue
			}
			if pattern, ok := gt.eventDefinition.(spellCastPattern); ok && pattern.sourceSpell && obj.ID != evt.TargetID {
				continue
			}
			gt.SetSource(obj.SourceID)
			gt.SetController(obj.Controller)
			if !gt.CheckTrigger(&evt, g) {
				continue
			}
			g.queuePendingTrigger(&pendingTrigger{
				ability:       gt,
				event:         &evt,
				eventCaptures: captureDetectedTriggerEvent(gt, &evt, g, g, obj.SourceID),
				sourceID:      obj.SourceID,
				controller:    obj.Controller,
			})
		}
	}

	// Check delayed triggers (one-shot unless Persistent, removed after matching)
	var remaining []*DelayedTrigger
	for _, dt := range g.triggers.Delayed() {
		if dt.EventType == evt.Type || dt.alternateEventType != 0 && dt.alternateEventType == evt.Type {
			if dt.damageReceived != nil && !dt.damageReceived.matches(&evt, g) {
				remaining = append(remaining, dt)
				continue
			}
			if dt.combatPair != nil && !dt.combatPair.matchesSubject(&evt, g, g.FindPermanent(dt.MatchEventID)) {
				remaining = append(remaining, dt)
				continue
			}
			if dt.matchIncarnation != uuid.Nil {
				matched := observer.FindPermanent(evt.SourceID)
				if dt.EventType == EvtAttacksUnblocked && (matched == nil || !matched.HasType(TypeCreature) || evt.SourceIncarnation != uuid.Nil && evt.SourceIncarnation != dt.matchIncarnation) {
					remaining = append(remaining, dt)
					continue
				}
				if matched == nil {
					if lki := observer.zones.LKI(evt.SourceID); lki != nil {
						matched = lki.Snapshot
					}
				}
				if matched == nil || matched.incarnationID != dt.matchIncarnation || matched.PhasedOut {
					remaining = append(remaining, dt)
					continue
				}
			}
			if dt.MatchEventID != uuid.Nil && evt.SourceID != dt.MatchEventID {
				remaining = append(remaining, dt)
				continue
			}
			if dt.MatchPlayerID != uuid.Nil && evt.PlayerID != dt.MatchPlayerID {
				remaining = append(remaining, dt)
				continue
			}
			if dt.MatchTargetID != uuid.Nil && evt.TargetID != dt.MatchTargetID {
				remaining = append(remaining, dt)
				continue
			}
			if (dt.MatchFlag && !evt.Flag) || (dt.matchFlagFalse && evt.Flag) {
				remaining = append(remaining, dt)
				continue
			}
			if evt.Type == EvtZoneChange {
				// Default unset zone matchers to ZoneAny so callers that
				// don't care about the from/to don't have to set them.
				wantFrom := dt.MatchFromZone
				if wantFrom == 0 {
					wantFrom = ZoneAny
				}
				wantTo := dt.MatchToZone
				if wantTo == 0 {
					wantTo = ZoneAny
				}
				if wantFrom != ZoneAny && evt.FromZone != wantFrom {
					remaining = append(remaining, dt)
					continue
				}
				if wantTo != ZoneAny && evt.ToZone != wantTo {
					remaining = append(remaining, dt)
					continue
				}
			}
			if dt.interveningCondition != nil && !dt.interveningCondition.checkState(g, StateContext{SourceID: dt.SourceID, ControllerID: dt.Controller}) {
				remaining = append(remaining, dt)
				continue
			}
			var captures []Effect
			if dt.damageReceived != nil {
				captures = append(captures, &eventAmountCapture{binding: dt.damageReceived.amount, amount: evt.Amount})
			}
			if dt.combatPair != nil {
				captures = dt.combatPair.captureDetectedEvent(&evt, g, dt.SourceID)
			}
			if dt.deathCapture != nil {
				captures = dt.deathCapture.captureDetectedEvent(&evt, observer, dt.SourceID)
				captures = append(captures, &eventPlayerCapture{binding: dt.deathCapture.controller, player: evt.PlayerID})
			}
			g.triggers.AddPending(&pendingTrigger{
				eventCaptures:     captures,
				delayed:           dt,
				sourceID:          dt.SourceID,
				sourceIncarnation: dt.sourceIncarnation,
				controller:        dt.Controller,
				event:             &evt,
			})
			if dt.Persistent {
				remaining = append(remaining, dt)
			}
		} else {
			remaining = append(remaining, dt)
		}
	}
	g.triggers.SetDelayed(remaining)
	g.queueParadigmRecurringTriggers(&evt)
	if evt.Type == EvtTappedForMana {
		g.resolveTappedForManaTriggers(pendingBefore)
	}
}

func (g *Game) resolveTappedForManaTriggers(pendingBefore int) {
	pending := g.triggers.Pending()
	if pendingBefore >= len(pending) {
		return
	}
	ordinary := append([]*pendingTrigger(nil), pending[:pendingBefore]...)
	var immediate []*pendingTrigger
	for _, pt := range pending[pendingBefore:] {
		gt, ok := pt.ability.(*GenericTriggered)
		if ok && isTriggeredManaAbility(gt) {
			immediate = append(immediate, pt)
		} else {
			ordinary = append(ordinary, pt)
		}
	}
	if len(immediate) == 0 {
		return
	}
	g.triggers.SetPending(immediate)
	stackBefore := g.StackSize()
	g.PutTriggersOnStack()
	var resolving []*StackObject
	for g.StackSize() > stackBefore {
		resolving = append(resolving, g.stack.Pop())
	}
	g.triggers.immediateManaDepth++
	for _, obj := range resolving {
		g.ResolveStackObject(obj)
	}
	g.triggers.immediateManaDepth--
	g.triggers.SetPending(append(ordinary, g.triggers.Pending()...))
}

// CheckStateTriggers records state triggers that have no pending or active instance.
// A condition can trigger again after its ability leaves the stack (CR 603.8).
func (g *Game) CheckStateTriggers() {
	if g.drawn {
		return
	}
	if g.triggers.zoneBatch != nil || g.triggers.stateChecksPaused {
		return
	}
	if g.effects.expireCounterDurations(g) {
		g.ApplyContinuousEffects()
		return
	}
	seen := make(map[stateTriggerKey]bool)
	for _, perm := range g.zones.battlefield {
		if perm.PhasedOut {
			continue
		}
		g.syncAbilityContext(perm)
		for _, a := range perm.RuntimeAbilities {
			ta, ok := UnwrapAbility(a).(TriggeredAbility)
			if !ok || !ta.IsStateTrigger() {
				continue
			}
			key := stateTriggerKey{sourceID: perm.ID(), abilityID: ta.AbilityID(), incarnationID: perm.incarnationID}
			seen[key] = true
			if g.stateTriggerActive(key) {
				g.triggers.ArmStateTrigger(key)
				continue
			}
			g.triggers.DisarmStateTrigger(key)
			var captures []Effect
			var matched bool
			if trigger, ok := ta.(*GenericTriggered); ok {
				if condition, ok := trigger.stateDefinition.(interface {
					detectState(*Game, uuid.UUID, uuid.UUID) ([]Effect, bool)
				}); ok {
					captures, matched = condition.detectState(g, perm.ID(), perm.ControllerID())
				} else {
					matched = ta.CheckTrigger(nil, g)
				}
			} else {
				matched = ta.CheckTrigger(nil, g)
			}
			if !matched {
				continue
			}
			g.triggers.ArmStateTrigger(key)
			g.queuePendingTrigger(&pendingTrigger{
				stateKey:      key,
				ability:       ta,
				sourceID:      perm.ID(),
				controller:    perm.ControllerID(),
				eventCaptures: captures,
			})
		}
	}
	// Drop entries for sources no longer on the battlefield so a re-entered
	// instance starts fresh.
	g.triggers.ClearUnseenArmed(seen)
}

// PutTriggersOnStack puts all pending triggers onto the stack.
//
// CR 603.3b specifies the trigger order when players receive priority.
// The active player puts triggered abilities on the stack first in any chosen order.
// Each non-active player in turn order then puts triggered abilities on the stack.
// The last ability put on the stack resolves first.
//
// The engine divides pending triggers into an active group and a non-active group.
// The engine pushes the active group first.
// The engine then pushes the non-active group.
// Thus, the triggers of the non-active player resolve first.
//
// Within each group, the engine reverses the event source order.
// Therefore, triggers from older permanents resolve first.
// CR 603.3b permits any order.
// The engine uses the XMage order to maintain deterministic test results.
func (g *Game) PutTriggersOnStack() {
	if g.drawn {
		return
	}
	pending := g.triggers.Pending()
	if len(pending) > 1 {
		activeIdx := g.turns.ActivePlayerIndex()
		var ordered []*pendingTrigger
		seen := make(map[*pendingTrigger]bool)
		for i := 0; i < len(g.players); i++ {
			pIdx := (activeIdx + i) % len(g.players)
			p := g.players[pIdx]
			if p == nil {
				continue
			}
			var playerTriggers []*pendingTrigger
			for _, pt := range pending {
				if pt.controller == p.PlayerID() {
					playerTriggers = append(playerTriggers, pt)
					seen[pt] = true
				}
			}
			ordered = append(ordered, g.orderPlayerTriggers(p, playerTriggers)...)
		}
		for _, pt := range pending {
			if !seen[pt] {
				ordered = append(ordered, pt)
			}
		}
		g.triggers.SetPending(ordered)
	}
	for _, pt := range g.triggers.Pending() {
		targetSource := g.triggerTargetSource(pt)
		players, permanents := capturedTriggerTargetFacts(pt)
		obj := &StackObject{
			stateKey:           pt.stateKey,
			capturedPlayers:    players,
			capturedPermanents: permanents,
			sourceIncarnation:  pt.sourceIncarnation,
			ID:                 uuid.New(),
			Controller:         pt.controller,
			SourceID:           pt.sourceID,
			IsAbility:          true,
		}
		if dt := pt.delayed; dt != nil && len(dt.modes) == 0 {
			obj.TargetSource = g.triggerTargetSourceWithDeclarations(pt, dt.targets)
			obj.Effects = append([]Effect(nil), dt.Effects...)
			obj.Targets = []uuid.UUID{dt.TargetID}
			if len(dt.targets) > 0 {
				var legal bool
				obj.Targets, obj.TargetSpecs, legal = g.chooseTriggerTargets(pt, dt.targets)
				if !legal {
					continue
				}
			}
			obj.EventAmount = pt.event.Amount
			obj.EventSourceID = pt.event.SourceID
			captureTriggerEventEffects(pt, obj)
			g.pushStack(obj)
			continue
		}
		// CR 603.3c permits only modes with available required targets.
		var modes []Mode
		if dt := pt.delayed; dt != nil {
			modes = dt.modes
			obj.EventAmount = pt.event.Amount
			obj.EventSourceID = pt.event.SourceID
		} else if gt, ok := pt.ability.(*GenericTriggered); ok {
			modes = gt.Modes()
		}
		if len(modes) > 0 {
			ctrl := g.GetPlayer(pt.controller)
			var labels []string
			var indices []int
			for i, mode := range modes {
				legal := g.targetEventQuery(obj.capturedPlayers, obj.capturedPermanents).hasLegalTargetSelection(pt.controller, g.triggerTargetSourceWithDeclarations(pt, mode.Targets), mode.Targets, 0)
				if legal {
					labels = append(labels, mode.Label)
					indices = append(indices, i)
				}
			}
			if len(indices) == 0 || ctrl == nil {
				continue
			}
			reason := "modal trigger"
			if targetSource != nil {
				reason = targetSource.Name()
			}
			choice := ctrl.ChooseMode(labels, reason)
			if choice < 0 || choice >= len(indices) {
				continue
			}
			idx := indices[choice]
			chosen := modes[idx]
			obj.Effects = append(obj.Effects, chosen.Effects...)
			obj.TargetSource = g.triggerTargetSourceWithDeclarations(pt, chosen.Targets)
			obj.ModeChoice = idx
			if len(chosen.Targets) > 0 {
				var legal bool
				obj.Targets, obj.TargetSpecs, legal = g.chooseTriggerTargets(pt, chosen.Targets)
				if !legal {
					continue
				}
			}
			captureTriggerEventEffects(pt, obj)
			g.pushStack(obj)
			continue
		}
		obj.TargetSource = g.triggerTargetSourceWithDeclarations(pt, pt.ability.Targets())
		obj.Effects = append(obj.Effects, pt.ability.Effects()...)
		// CR 603.3d: when a triggered ability with targets is put on the stack,
		// its controller chooses the targets. Declared AddTarget(...) entries
		// take precedence over the legacy event-derived auto-binding below;
		// triggers without declared targets fall through to the auto-bind so
		// existing card behavior is preserved.
		if declared := pt.ability.Targets(); len(declared) > 0 {
			var legal bool
			obj.Targets, obj.TargetSpecs, legal = g.chooseTriggerTargets(pt, declared)
			if !legal {
				continue
			}
			if pt.event != nil && pt.event.Amount != 0 {
				if gt, ok := pt.ability.(*GenericTriggered); ok {
					switch gt.eventType {
					case EvtZoneChange:
						if pt.event.ToZone == ZoneBattlefield {
							obj.XValue = pt.event.Amount
						}
					case EvtDamageDealt:
						obj.EventAmount = pt.event.Amount
						obj.EventSourceID = pt.event.SourceID
					}
				}
			}
			captureTriggerEventEffects(pt, obj)
			g.pushStack(obj)
			continue
		}
		// Bind event context needed by resolving triggered effects. Some events
		// expose their player through targets[0]; zone changes instead expose the
		// moved object so effects can inspect it or its controller.
		if pt.event != nil {
			if gt, ok := pt.ability.(*GenericTriggered); ok {
				switch gt.eventType {
				case EvtZoneChange:
					// Route by (FromZone, ToZone) for stack-object context.
					switch {
					case pt.event.ToZone == ZoneBattlefield:
						// ETB: pass the entering permanent's ID so effects can
						// tap/modify it; preserve X for X-cost ETB triggers.
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = []uuid.UUID{pt.event.SourceID}
						}
						obj.XValue = pt.event.Amount
					case pt.event.FromZone == ZoneBattlefield:
						// Battlefield -> elsewhere (graveyard / exile / hand /
						// library). Pass the leaving permanent's ID first
						// (the dies/leaves convention - Creature Bond reading
						// the dead creature's toughness, Sengir Vampire
						// finding it in the graveyard) and the controller's
						// ID second (Dingus Egg "deal damage to its
						// controller", post-LKI lookups).
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = []uuid.UUID{pt.event.SourceID}
							if pt.event.PlayerID != uuid.Nil {
								obj.Targets = append(obj.Targets, pt.event.PlayerID)
							}
						}
					}
				case EvtDrawStep, EvtCardDrawn:
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				case EvtSpellCast:
					// Pass the spell's ID and caster's player ID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
						if pt.event.PlayerID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.PlayerID)
						}
					}
				case EvtDamageDealt:
					if pt.event.TargetID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.TargetID}
					}
					obj.EventAmount = pt.event.Amount
					obj.EventSourceID = pt.event.SourceID
				case EvtScry, EvtSurveil:
					obj.EventAmount = pt.event.Amount
				case EvtTappedForMana:
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
					obj.EventAmount = pt.event.Amount
					obj.EventSourceID = pt.event.SourceID
				case EvtTapped, EvtAbilityActivated:
					// Pass the permanent's ID so effects can identify it
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtDeclaredAttacker:
					// Pass the declared attacker's ID so effects can identify which
					// creature attacked (Hellrider's "deal 1 damage to the player
					// or planeswalker it's attacking").
					obj.EventSourceID = pt.event.SourceID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
					}
				case EvtCombatDamageDealt:
					// Pass the damaging controller and recipient via the dedicated
					// event-context fields (EventSourceID = recipient, EventAmount
					// = total damage). We deliberately do NOT auto-bind Targets[0]
					// here — DrawCards-style effects fall back to controller when
					// Targets is empty, which is the correct behavior for "draw a
					// card" triggers (Keeper of Fables). Per-step triggers that
					// need the recipient (Oona's Blackguard) read it via
					// g.EventSourceID().
					obj.EventSourceID = pt.event.TargetID
					obj.EventAmount = pt.event.Amount
				case EvtBecomesTarget:
					// Pass the targeted object's ID and the spell/ability
					// source so effects can either identify "this" (the target,
					// e.g. Departed Deckhand sacrificing itself) or the
					// spell/ability that did the targeting (e.g. Kira
					// countering it). Targets[0] is the targeted object;
					// Targets[1] is the offending spell/ability source.
					if pt.event.TargetID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.TargetID}
						if pt.event.SourceID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.SourceID)
						}
					}
				case EvtDeclaredBlocker:
					// Pass the blocker's ID and attacker's ID
					if pt.event.SourceID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.SourceID}
						if pt.event.TargetID != uuid.Nil {
							obj.Targets = append(obj.Targets, pt.event.TargetID)
						}
					}
				case EvtLifeGained, EvtLifeLost:
					// Preserve the life delta for "gain/lose that much" triggers.
					// We deliberately do NOT auto-bind PlayerID as a target; the
					// affected player is rarely the same as the trigger's
					// "you" (e.g. Exquisite Blood's "you gain that much life"
					// targets the controller, not the opponent who lost life).
					obj.EventAmount = pt.event.Amount
				case EvtAttackersDeclared:
					// Preserve the attacker count for "gain that much life" /
					// "draw that many cards" attack-aggregate triggers (Path of
					// Bravery's gain-life clause).
					obj.EventAmount = pt.event.Amount
				case EvtDiscard:
					// Pass the discarding player's ID so effects like
					// "that player loses 2 life" target the discarder.
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				case EvtSacrifice:
					// Pass the sacrificing player's ID for "you sacrifice"
					// effects that need to identify the controller.
					if pt.event.PlayerID != uuid.Nil {
						obj.Targets = []uuid.UUID{pt.event.PlayerID}
					}
				}
				// Typed event subjects are not targets (CR 115.1d).
				if gt.eventDefinition != nil {
					obj.Targets = nil
				}
			}
		}
		captureTriggerEventEffects(pt, obj)
		g.pushStack(obj)
	}
	g.triggers.ClearPending()
}

// chooseTriggerTargets returns false when a required target group cannot be chosen.
// CR 603.3d removes such a trigger before a player receives priority.
func (g *Game) chooseTriggerTargets(pt *pendingTrigger, declared []Target) ([]uuid.UUID, []Target, bool) {
	query := g.triggerTargetQuery(pt)
	controller := g.GetPlayer(pt.controller)
	sourceCard := g.triggerTargetSourceWithDeclarations(pt, declared)
	var out []uuid.UUID
	var aligned []Target
	for i, t := range declared {
		if pair, ok := t.(*randomActivePlayerExchangePairTarget); ok {
			chosen := pair.chooseForTrigger(pt.controller, sourceCard, g)
			if len(chosen) < t.Min() {
				return nil, nil, false
			}
			out = append(out, chosen...)
			for range chosen {
				aligned = append(aligned, t)
			}
			continue
		}
		if random, ok := randomTargetSpec(t); ok {
			chosen := g.chooseRandomTargetsWithQuery(pt.controller, sourceCard, random, 0, query)
			if len(chosen) < t.Min() {
				return nil, nil, false
			}
			out = append(out, chosen...)
			for range chosen {
				aligned = append(aligned, t)
			}
			continue
		}
		t.Reset()
		possible := query.targetCandidatesForList(pt.controller, sourceCard, declared, i, out, 0)
		if len(possible) < t.Min() {
			return nil, nil, false
		}
		if len(possible) == 0 {
			continue
		}
		chooser := controller
		if ref := targetChooserReference(t); ref != nil {
			id, ok := ref.resolvePlayer(&EffectContext{Game: query, Controller: pt.controller})
			if !ok {
				return nil, nil, false
			}
			chooser = g.GetPlayer(id)
		} else if isOpponentChosenTarget(t) {
			if opp := g.GetOpponent(pt.controller); opp != nil {
				chooser = opp
			}
		}
		var chosen []uuid.UUID
		if chooser != nil {
			chosen = chooser.ChooseTargets(possible, t.Min(), t.Max(), g)
		}
		if len(chosen) == 0 && t.Min() > 0 {
			chosen = possible[:t.Min()]
		}
		if _, typed := t.(*targetDeclaration); typed {
			if len(chosen) < t.Min() || len(chosen) > t.Max() {
				return nil, nil, false
			}
			seen := make(map[uuid.UUID]bool, len(chosen))
			for _, id := range chosen {
				if seen[id] || !slices.Contains(possible, id) {
					return nil, nil, false
				}
				seen[id] = true
			}
		}
		if err := t.Choose(pt.controller, sourceCard, g, chosen); err != nil {
			return nil, nil, false
		}
		out = append(out, chosen...)
		for range chosen {
			aligned = append(aligned, t)
		}
	}
	return out, aligned, true
}

func (g *Game) orderPlayerTriggers(p Player, triggers []*pendingTrigger) []*pendingTrigger {
	if len(triggers) <= 1 {
		return triggers
	}
	if chooser, ok := p.(TriggerOrderChooser); ok {
		info := make([]TriggerOrderInfo, len(triggers))
		for i, pt := range triggers {
			info[i].SourceID = pt.sourceID
			if card := g.FindCardAnywhere(pt.sourceID); card != nil {
				info[i].SourceName = card.Name()
			}
			if desc, ok := pt.ability.(interface{ Text() string }); ok {
				info[i].AbilityText = desc.Text()
			}
		}
		chosen := chooser.ChooseTriggerOrder(info)
		if isValidPermutation(chosen, len(triggers)) {
			ordered := make([]*pendingTrigger, len(triggers))
			for i, idx := range chosen {
				ordered[i] = triggers[idx]
			}
			return ordered
		}
	}
	reverseTriggers(triggers)
	return triggers
}

func isValidPermutation(indices []int, n int) bool {
	if len(indices) != n {
		return false
	}
	seen := make([]bool, n)
	for _, idx := range indices {
		if idx < 0 || idx >= n || seen[idx] {
			return false
		}
		seen[idx] = true
	}
	return true
}

func reverseTriggers(s []*pendingTrigger) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Each chosen target must receive damage under CR 601.2d.
// An impossible target list has no valid fallback.
func sanitizeDamageDistribution(in map[uuid.UUID]int, targets []uuid.UUID, total int) map[uuid.UUID]int {
	if len(targets) == 0 || total < len(targets) {
		return nil
	}
	allowed := make(map[uuid.UUID]bool, len(targets))
	for _, id := range targets {
		if id == uuid.Nil || allowed[id] {
			return nil
		}
		allowed[id] = true
	}
	out := make(map[uuid.UUID]int, len(targets))
	remaining := total
	valid := len(in) == len(targets)
	for id, amount := range in {
		if !allowed[id] || amount < 1 || amount > remaining {
			valid = false
			break
		}
		out[id] = amount
		remaining -= amount
	}
	if valid && remaining == 0 {
		return out
	}
	out = make(map[uuid.UUID]int, len(targets))
	for _, id := range targets {
		out[id] = 1
	}
	out[targets[0]] += total - len(targets)
	return out
}

func (g *Game) validateActionTargets(controller uuid.UUID, sourceCard Card, specs []Target, chosen []uuid.UUID, x int, label string) error {
	if len(specs) == 0 {
		return nil
	}
	offset := 0
	for i, spec := range specs {
		minimum, maximum := TargetBounds(spec, x)
		remainingRequired := 0
		for _, later := range specs[i+1:] {
			laterMinimum, _ := TargetBounds(later, x)
			remainingRequired += laterMinimum
		}
		available := max(0, len(chosen)-offset-remainingRequired)
		count := min(maximum, available)
		if count < minimum {
			return fmt.Errorf("not enough targets for %s", label)
		}
		possible := g.targetBindingsQuery(specs[:i], chosen[:offset], nil, x).possibleTargetsWithX(spec, controller, sourceCard, x)
		seen := make(map[uuid.UUID]bool, count)
		for _, id := range chosen[offset : offset+count] {
			if seen[id] || !slices.Contains(possible, id) {
				return fmt.Errorf("invalid target for %s", label)
			}
			seen[id] = true
		}
		offset += count
	}
	if offset != len(chosen) {
		return fmt.Errorf("too many targets for %s", label)
	}
	return nil
}

func (g *Game) validateVariableSpellTargets(controller uuid.UUID, sourceCard Card, chosen []uuid.UUID, x int) error {
	specs := sourceCard.CastTargets()
	if len(specs) != 1 {
		return nil
	}
	spec := specs[0]
	minTargets, maxTargets := TargetBounds(spec, x)
	if _, variable := spec.(VariableTarget); !variable {
		return nil
	}
	if len(chosen) < minTargets || len(chosen) > maxTargets {
		return fmt.Errorf("%s requires exactly %d targets", sourceCard.Name(), minTargets)
	}
	possible := g.possibleTargetsWithX(spec, controller, SpellTargetSource(sourceCard), x)
	seen := make(map[uuid.UUID]bool, len(chosen))
	for _, id := range chosen {
		if seen[id] || !slices.Contains(possible, id) {
			return fmt.Errorf("invalid target for %s", sourceCard.Name())
		}
		seen[id] = true
	}
	return nil
}

func addActionManaCost(total *ManaCost, add ManaCost) {
	total.Generic += add.Generic
	total.White += add.White
	total.Blue += add.Blue
	total.Black += add.Black
	total.Red += add.Red
	total.Green += add.Green
	total.Hybrid = append(total.Hybrid, add.Hybrid...)
	if add.HasX {
		total.HasX = true
		total.XCount += add.XCount
	}
}

func (g *Game) prepareActionCosts(costs []Cost, targets []uuid.UUID, x int) []Cost {
	ctx := actionCostContext{Targets: append([]uuid.UUID(nil), targets...), XValue: x}
	resolved := make([]Cost, 0, len(costs))
	var mana ManaCost
	for _, cost := range costs {
		if contextual, ok := cost.(contextualActionCost); ok {
			cost = contextual.resolveActionCost(ctx)
		}
		if payment, ok := cost.(*ManaCostPayment); ok {
			addActionManaCost(&mana, payment.MC)
			continue
		}
		resolved = append(resolved, cost)
	}
	if !mana.IsZero() {
		resolved = append([]Cost{&ManaCostPayment{MC: mana}}, resolved...)
	}
	return resolved
}

func newStackObject(controller, sourceID uuid.UUID, card Card, effects []Effect, targets []uuid.UUID, xValue int, isAbility bool) *StackObject {
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       card,
		Controller: controller,
		SourceID:   sourceID,
		IsAbility:  isAbility,
		Targets:    targets,
		XValue:     xValue,
	}
	obj.Effects = append(obj.Effects, effects...)
	return obj
}

func chooseModeForStackObject(obj *StackObject, modes []string, chooser Player, sourceName string) {
	if len(modes) == 0 || chooser == nil {
		return
	}
	obj.ModeChoice = chooser.ChooseMode(modes, sourceName)
}

// CastSpellByName finds a card in player's hand, puts it on the stack.
func (g *Game) CastSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, xValues ...int) error {
	return g.castSpellByName(playerID, name, targets, nil, xValues...)
}

// CastSpellByNameWithTargetGroups casts a spell with one choice slice per declared target group.
func (g *Game) CastSpellByNameWithTargetGroups(playerID uuid.UUID, name string, groups [][]uuid.UUID, xValues ...int) error {
	return g.castSpellByName(playerID, name, nil, groups, xValues...)
}

func (g *Game) castSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, groups [][]uuid.UUID, xValues ...int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	// Find card in hand
	var card Card
	for _, c := range p.Hand() {
		if c.Name() == name {
			card = c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("card %s not found in hand", name)
	}

	if err := g.checkSpellTiming(card, playerID); err != nil {
		return err
	}

	// CR 307.1, 302.1, 303.1, 301.1 — sorcery-speed timing. Any spell that
	// is not an instant may be cast only during its controller's main phase,
	// when the stack is empty, and when that player is the active player
	// (i.e. could cast a sorcery).
	// CR 702.8 — Flash: "You may cast this spell any time you could cast an
	// instant." A card with Flash bypasses the sorcery-speed gate entirely.
	if !card.HasType(TypeInstant) && !cardHasKeyword(card, Flash) && !g.effects.Rules.HasFlashGrant(playerID, card) {
		if !g.turns.Step().IsMainPhase() {
			return ErrSorcerySpeed
		}
		if g.ActivePlayerObj().PlayerID() != playerID {
			return ErrSorcerySpeed
		}
		if !g.stack.IsEmpty() {
			return ErrSorcerySpeed
		}
	}

	// Check expansion block (City in a Bottle)
	if g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		return fmt.Errorf("can't cast %s: card is from a blocked expansion", card.Name())
	}
	// Check player-level cast prohibition (Angelic Arbiter, etc.)
	if g.effects.Rules.PlayerCantCastSpells(playerID) {
		return fmt.Errorf("can't cast %s: a continuous effect prevents this player from casting spells", card.Name())
	}
	// Determine X value
	xValue := 0
	if len(xValues) > 0 {
		xValue = xValues[0]
	}
	if xValue < 0 {
		return fmt.Errorf("x cannot be negative")
	}
	grouped := false
	for _, ability := range card.Abilities() {
		if action, ok := ability.(*ActionDefinition); ok && action.kind == ActionSpell && action.groupedTargets {
			grouped = true
			break
		}
	}
	if grouped != (groups != nil) {
		return fmt.Errorf("spell target group announcement does not match declaration")
	}
	var aligned []Target
	if grouped {
		var err error
		targets, aligned, err = g.prepareSpellTargetGroups(playerID, card, groups, xValue)
		if err != nil {
			return err
		}
	}
	selection, err := g.prepareModalSpellSelection(p, card, targets, xValue, "")
	if err != nil {
		return err
	}
	if selection != nil {
		targets = selection.targets
	} else if !grouped {
		targets = g.acquireRandomTargets(playerID, SpellTargetSource(card), card.CastTargets(), targets, xValue)
		targets = g.acquireOpponentChosenTargets(playerID, SpellTargetSource(card), card.CastTargets(), targets)
		if err := g.validateActionTargets(playerID, SpellTargetSource(card), card.CastTargets(), targets, xValue, "spell"); err != nil {
			return err
		}
		if err := g.validateVariableSpellTargets(playerID, card, targets, xValue); err != nil {
			return err
		}
	}

	effects, _ := g.prepareSpellStackPayload(p, card, targets, false)
	if selection != nil {
		effects = selection.effects
	}
	division, err := g.prepareDamageDivision(playerID, card, effects, targets, xValue)
	if err != nil {
		return err
	}

	mc := card.ManaCost()

	var totalCosts []Cost
	if bc, ok := card.(*BaseCard); ok {
		totalCosts = append(totalCosts, bc.AdditionalCosts()...)
	}

	var actionCosts []Cost
	for _, a := range card.Abilities() {
		switch spell := a.(type) {
		case *SpellAbility:
			if spell.Kind() == ActionSpell {
				actionCosts = append(actionCosts, spell.Costs()...)
			}
		case *ModalSpellAbility:
			actionCosts = append(actionCosts, spell.Costs()...)
		}
	}
	actionCosts = g.prepareActionCosts(actionCosts, targets, xValue)
	totalCosts = append(totalCosts, actionCosts...)

	payment, err := g.prepareSpellPaymentTransaction(spellPaymentSpec{
		Card:           card,
		Controller:     playerID,
		Zone:           ZoneHand,
		Mana:           mc,
		ChannelGeneric: g.effects.Rules.IsChannelActive(playerID),
		Costs:          totalCosts,
		Targets:        targets,
		XValue:         xValue,
	})
	if err != nil {
		return fmt.Errorf("cannot pay total cost for %s: %w", name, err)
	}
	handTargets := g.handTargetCards(targets)
	exileTargets := g.exileTargetCards(targets)
	anteTargets := g.anteTargetCards(targets)
	if err := payment.Commit(); err != nil {
		return err
	}

	// If an additional cost set g.resolution.X (e.g. sacrifice-capture-CMC), use it
	if g.resolution.X() != 0 && xValue == 0 && !hasAnnouncedXAllocation(effects) {
		xValue = g.resolution.X()
		g.resolution.SetX(0)
	}

	_, err = g.pushCastSpellObject(castStackObjectOptions{
		DamageDivisionPrepared: true,
		DamageDistribution:     division,
		ModalSelection:         selection,
		Card:                   card,
		Controller:             playerID,
		Targets:                targets,
		TargetSpecs:            aligned,
		XValue:                 xValue,
		CastZone:               ZoneHand,
		SnapshotCast:           true,
		CapturedNumbers:        payment.payment.paidNumbers,
		CapturedCards:          payment.payment.paidCards,
		RevealedHandCards:      payment.payment.revealedHandCards,
		HandTargetCards:        handTargets,
		ExileTargetCards:       exileTargets,
		AnteTargetCards:        anteTargets,
		CapturedCardSets:       payment.payment.paidCardSets,
		CapturedPermanents:     payment.payment.paidPermanents,
	})
	return err
}

// TapForMana taps a permanent for mana using its first mana ability.
// Equivalent to TapForManaWithColor with no color preference. Recognizes both
// *ManaAbility (built via WithManaAbility/WithMultiManaAbility) and
// *SimpleActivatedAbility whose only cost is tapping and whose effects are
// mana-producing (built via WithActivatedAbility(AddMana(...), Tap())
// — e.g. Mana Vault).
func (g *Game) TapForMana(playerID, permanentID uuid.UUID) error {
	return g.mana.TapForMana(g, playerID, permanentID, Colorless)
}

func (g *Game) TapForManaWithColor(playerID, permanentID uuid.UUID, preferredColor Color) error {
	return g.mana.TapForMana(g, playerID, permanentID, preferredColor)
}

// productionsMatchColor reports whether the production list can satisfy a
// request for preferredColor. A specific-color production matches its color;
// A color choice matches only its available options.
func productionsMatchColor(productions []ManaProduction, preferredColor Color) bool {
	for _, p := range productions {
		if slices.Contains(manaProductionColors(p), preferredColor) {
			return true
		}
	}
	return false
}

// abilityManaProductions returns the mana productions for an ability that acts
// as a tap-for-mana mana source, or nil if the ability is not such a source.
// Recognizes both *ManaAbility and *SimpleActivatedAbility whose only cost is
// tapping and whose effects all produce mana (AddMana / AddAnyMana).
func abilityManaProductions(a Ability, g GameReader, sourceID uuid.UUID) []ManaProduction {
	a = UnwrapAbility(a)
	switch ab := a.(type) {
	case *ManaAbility:
		return ab.currentProductions(g, sourceID)
	case *SimpleActivatedAbility:
		return activatedManaProductions(ab)
	}
	return nil
}

// activatedManaProductions extracts mana productions from a SimpleActivatedAbility
// that behaves as a tap-for-mana ability — exactly one cost (tapping the source)
// and all effects are AddMana / AddAnyMana. Returns nil if the shape doesn't match.
func activatedManaProductions(a *SimpleActivatedAbility) []ManaProduction {
	if len(a.costs) != 1 || len(a.targets) != 0 {
		return nil
	}
	if _, ok := a.costs[0].(*tap); !ok {
		return nil
	}
	productions, ok := activatedManaEffects(a.effects)
	if !ok {
		return nil
	}
	return productions
}

// manaSourceInfo describes one permanent or player mana permission.
// Colors is the union used only by preservation
// scoring; payment planning operates on Abilities without merging their output.
type manaSourceInfo struct {
	PlayerManaAbilityID uuid.UUID
	Incarnation         uuid.UUID
	PermanentID         uuid.UUID
	Colors              []Color
	Abilities           []manaSourceAbility
	Bonuses             []ManaBonusColor
}

// AutoTapHint informs the smart auto-tap algorithm about the action being paid
// for. Callers pass a hint so the algorithm can deprioritize tapping the
// permanent that is activating an ability and weight color preferences by
// other spells the player still wants to cast.
type AutoTapHint struct {
	// ReservedSources lists permanents that the surrounding action must keep
	// untapped while mana is produced. Attack declarations use this for chosen
	// attackers; multi-source total costs may reserve more than one permanent.
	ReservedSources []uuid.UUID
	// ActivationSource is the permanent whose activated ability is being paid
	// for (zero UUID if not applicable, e.g. when casting a spell). Sources
	// matching this id are penalized so we only tap them as a last resort.
	ActivationSource uuid.UUID
	// ActivationTapsSource is true when the ability being activated includes
	// a {T} cost on its source. In that case the source will be tapped during
	// cost payment anyway, so it must be hard-excluded from auto-tap to avoid
	// a "source already tapped" failure later.
	ActivationTapsSource bool
	// CastingCard is the card currently being cast (zero UUID if not casting).
	// Its colored pips are excluded from the hand-demand calculation so we
	// don't count the cost we're paying for against itself.
	CastingCard uuid.UUID
}

var manaPoolColors = [...]Color{White, Blue, Black, Red, Green, Colorless}

// AutoTapForCost taps untapped lands/mana sources to pay a mana cost,
// accounting for mana already in the player's pool.
func (g *Game) AutoTapForCost(playerID uuid.UUID, mc ManaCost) error {
	return g.mana.AutoTapForCost(g, playerID, mc, AutoTapHint{}, nil)
}

// AutoTapForCostWithHint taps untapped lands/mana sources to pay a mana cost with a hint.
func (g *Game) AutoTapForCostWithHint(playerID uuid.UUID, mc ManaCost, hint AutoTapHint) error {
	return g.mana.AutoTapForCost(g, playerID, mc, hint, nil)
}

func (g *Game) HypotheticalMana(playerID uuid.UUID) int {
	return g.mana.HypotheticalMana(g, playerID)
}

func (g *Game) CanAfford(playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) bool {
	return g.mana.CanAfford(g, playerID, mc, spellCtx)
}

func (g *Game) MaxXValue(playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) int {
	return g.mana.MaxXValue(g, playerID, mc, spellCtx)
}

// ActivatableInfo describes an activated ability on a permanent that can currently be used.
type ActivatableInfo struct {
	PermanentID   uuid.UUID
	PermanentName string
	AbilityIndex  int
	Description   string
}

// GetCastableSpells returns cards in a player's hand they can currently cast.
func (g *Game) GetCastableSpells(playerID uuid.UUID) []Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	isMainPhase := g.turns.Step().IsMainPhase()
	isActive := g.ActivePlayerObj().PlayerID() == playerID

	var castable []Card
	for _, card := range p.Hand() {
		if card.HasType(TypeLand) {
			continue
		}
		if g.checkSpellTiming(card, playerID) != nil {
			continue
		}
		// CR 702.8 — Flash lets a spell be cast any time you could cast an instant,
		// bypassing the sorcery-speed gate below.
		hasFlash := cardHasKeyword(card, Flash) || g.effects.Rules.HasFlashGrant(p.PlayerID(), card)
		// Sorceries can only be cast at sorcery speed (main phase, active player, empty stack)
		if card.HasType(TypeSorcery) && !hasFlash {
			if !isMainPhase || !isActive || !g.stack.IsEmpty() {
				continue
			}
		}
		// Creatures/artifacts/enchantments are sorcery speed
		if (card.HasType(TypeCreature) || card.HasType(TypeArtifact) || card.HasType(TypeEnchantment)) && !hasFlash {
			if !isMainPhase || !isActive || !g.stack.IsEmpty() {
				continue
			}
		}
		// Check mana (ignore X costs for now - those are always "castable" if base cost met)
		mc := card.ManaCost()
		checkMC := mc
		if mc.HasX {
			// For X spells, check if we can pay the non-X portion
			checkMC = ManaCost{
				Generic: mc.Generic,
				White:   mc.White,
				Blue:    mc.Blue,
				Black:   mc.Black,
				Red:     mc.Red,
				Green:   mc.Green,
			}
		}
		if !g.CanAfford(playerID, checkMC, SpellContextForCard(card)) {
			continue
		}
		// Spells with targets (e.g. auras) can't be cast if no legal targets exist
		if ct := card.CastTargets(); len(ct) > 0 {
			if hasTargetDependencies(ct) {
				if !g.hasLegalTargetSelection(playerID, SpellTargetSource(card), ct, 0) {
					continue
				}
				castable = append(castable, card)
				continue
			}
			hasLegalTarget := false
			for _, t := range ct {
				if len(t.Possible(playerID, SpellTargetSource(card), g)) > 0 {
					hasLegalTarget = true
					break
				}
			}
			if !hasLegalTarget {
				continue
			}
		}
		castable = append(castable, card)
	}
	return castable
}

// GetPlayableLands returns the lands a player can legally play right now.
// Checks: main phase, active player, land-play limit (respects Fastbond etc.),
// and expansion blocks.
func (g *Game) GetPlayableLands(playerID uuid.UUID) []Card {
	if g.StackSize() != 0 {
		return nil
	}
	if !g.turns.Step().IsMainPhase() {
		return nil
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return nil
	}
	if g.trackers.Turn.LandsPlayed() >= g.MaxLandPlays() {
		return nil
	}
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	var lands []Card
	for _, c := range p.Hand() {
		if c.HasType(TypeLand) && !g.IsLandPlayProhibited(c) {
			lands = append(lands, c)
		}
	}
	return lands
}

// GetActivatableAbilities returns activated abilities the player can currently use.
func (g *Game) GetActivatableAbilities(playerID uuid.UUID) []ActivatableInfo {
	var result []ActivatableInfo
	for _, perm := range g.zones.battlefield {
		isOwner := perm.ControllerID() == playerID
		for i, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			aa, ok := inner.(ActivatedAbility)
			if !ok {
				continue
			}
			// An ActionDefinition implements ActivatedAbility regardless of kind.
			// Spell-kind actions are the resolution effect of a cast spell, not
			// something a player activates from the battlefield — surfacing them
			// would let the search re-fire an aura's ETB effect indefinitely.
			if def, ok := inner.(*ActionDefinition); ok && def.Kind() != ActionActivated {
				continue
			}
			// Check if this ability can be used by non-controllers
			if !isOwner {
				saa, isSAA := inner.(*SimpleActivatedAbility)
				if !isSAA || !saa.IsAnyPlayerAbility() {
					continue
				}
			}
			// If opponent-only, the controller cannot activate it
			if isOwner {
				saa, isSAA := inner.(*SimpleActivatedAbility)
				if isSAA && saa.IsOpponentOnlyAbility() {
					continue
				}
			}
			// Automatic payment handles these mana abilities.
			if _, isMana := inner.(*ManaAbility); isMana || abilityManaProductions(inner, g, perm.ID()) != nil {
				continue
			}
			if !aa.CanActivate(playerID, g) {
				continue
			}
			if aa.SorcerySpeed() && !g.turns.Step().IsMainPhase() {
				continue
			}
			desc := ""
			for _, e := range aa.Effects() {
				if desc != "" {
					desc += ", "
				}
				desc += e.Text()
			}
			result = append(result, ActivatableInfo{
				PermanentID:   perm.ID(),
				PermanentName: perm.Name(),
				AbilityIndex:  i,
				Description:   desc,
			})
		}
	}
	return result
}

// CastSpellByID casts a spell from a player's hand by card ID. It resolves
// the ID to the card's name and delegates to CastSpellByName, which owns the
// full cost-modification and additional-cost pipeline. Using a thin adapter
// here means the priority loop and scripted tests go through the same code
// path as the harness / interactive layer.
func (g *Game) CastSpellByID(playerID, cardID uuid.UUID, targets []uuid.UUID, xValue int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}
	var card Card
	for _, c := range p.Hand() {
		if c.ID() == cardID {
			card = c
			break
		}
	}
	if card == nil {
		return ErrCardNotInHand
	}

	// Check expansion block (City in a Bottle)
	if g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		return fmt.Errorf("can't cast %s: card is from a blocked expansion", card.Name())
	}

	return g.CastSpellByName(playerID, card.Name(), targets, xValue)
}

// ActivateAbilityByIndex activates an ability on a permanent by index.
func (g *Game) ActivateAbilityByIndex(playerID, permanentID uuid.UUID, abilityIndex int, targets []uuid.UUID) error {
	return g.activateAbilityByIndex(playerID, permanentID, abilityIndex, targets, nil)
}

// ActivateAbilityByIndexWithTargetGroups announces one choice list per target declaration.
func (g *Game) ActivateAbilityByIndexWithTargetGroups(playerID, permanentID uuid.UUID, abilityIndex int, groups [][]uuid.UUID) error {
	if groups == nil {
		return fmt.Errorf("activation target groups are required")
	}
	return g.activateAbilityByIndex(playerID, permanentID, abilityIndex, nil, groups)
}

func (g *Game) activateAbilityByIndex(playerID, permanentID uuid.UUID, abilityIndex int, targets []uuid.UUID, groups [][]uuid.UUID) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if abilityIndex < 0 || abilityIndex >= len(perm.RuntimeAbilities) {
		return fmt.Errorf("invalid ability index")
	}

	inner := UnwrapAbility(perm.RuntimeAbilities[abilityIndex])
	if groups != nil {
		if _, ok := inner.(*ManaAbility); ok {
			return fmt.Errorf("mana ability does not declare target groups")
		}
	}

	// Handle mana abilities (don't use the stack)
	if ma, ok := inner.(*ManaAbility); ok {
		if perm.ControllerID() != playerID {
			return fmt.Errorf("only the controller may activate mana abilities")
		}
		if perm.HasAttr(AttrCantActivate) {
			return fmt.Errorf("cannot activate mana ability of %s", perm.Name())
		}
		if perm.Tapped || !perm.CanTapForEffect(g) {
			return fmt.Errorf("cannot tap %s for mana", perm.Name())
		}
		payment, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: playerID, SourceID: perm.ID(), Costs: []Cost{Tap()}, Hint: AutoTapHint{ActivationSource: perm.ID(), ActivationTapsSource: true}, ApplyActivationReduction: true})
		if err != nil {
			return err
		}
		manaSourceIncarnation := perm.incarnationID
		if err := payment.Commit(); err != nil {
			return err
		}
		payment.recordAbilityTapPayments(capturedPermanent{permanentID, manaSourceIncarnation})
		perm = g.FindPermanent(permanentID)
		p := g.GetPlayer(playerID)
		produced := 0
		var types []Color
		if p != nil {
			var err error
			produced, types, err = g.mana.addManaFromAbilityWithTypes(g, ma, p, perm)
			if err != nil {
				return err
			}
		}
		g.FireEvent(GameEvent{
			Type:              EvtAbilityActivated,
			SourceID:          perm.ID(),
			SourceIncarnation: perm.incarnationID,
			PlayerID:          playerID,
			Flag:              true,
		})
		g.mana.fireTappedForManaWithTypes(g, perm.ID(), manaSourceIncarnation, playerID, produced, types)
		return nil
	}

	aa, ok := inner.(ActivatedAbility)
	if !ok {
		return fmt.Errorf("not an activated ability")
	}
	if def, ok := inner.(*ActionDefinition); ok && def.Kind() != ActionActivated {
		return fmt.Errorf("not an activated ability")
	}
	grouped := false
	if def, ok := inner.(*ActionDefinition); ok {
		grouped = def.groupedTargets
	}
	if grouped != (groups != nil) {
		return fmt.Errorf("activation target group announcement does not match declaration")
	}
	saa, isSAA := inner.(*SimpleActivatedAbility)
	if isSAA && saa.activationManaX != nil {
		x, valid := saa.activationManaX.value(g, permanentID)
		if !valid {
			return fmt.Errorf("activation X requires a present source")
		}
		previousX := g.resolution.X()
		g.resolution.SetX(x)
		defer g.resolution.SetX(previousX)
	}
	if perm.ControllerID() != playerID {
		if !isSAA || !saa.IsAnyPlayerAbility() {
			return fmt.Errorf("only the controller may activate this ability")
		}
	}
	if perm.ControllerID() == playerID && isSAA && saa.IsOpponentOnlyAbility() {
		return fmt.Errorf("only opponents may activate this ability")
	}
	if perm.HasAttr(AttrCantActivateNonManaAbilities) && !canProduceMana(aa.Effects(), aa.Targets()) {
		return fmt.Errorf("non-mana activated abilities of %s are prevented", perm.Name())
	}
	if !aa.CanActivate(playerID, g) {
		return fmt.Errorf("cannot activate ability")
	}

	actionTargets := aa.Targets()
	actionEffects := aa.Effects()
	targetSource := g.AbilityTargetSource(perm.Card, aa, actionTargets)
	var alignedTargets []Target
	modeChoice := -1
	var modalTargets [][]uuid.UUID
	if modal, ok := aa.(interface{ Modes() []Mode }); ok && len(modal.Modes()) > 0 {
		modes := modal.Modes()
		var available []Mode
		var indices []int
		for i, mode := range modes {
			legal := g.hasLegalTargetSelection(playerID, g.AbilityTargetSource(perm.Card, aa, mode.Targets), mode.Targets, g.resolution.X())
			if legal {
				available = append(available, mode)
				indices = append(indices, i)
			}
		}
		if len(available) == 0 {
			return fmt.Errorf("no legal activation mode")
		}
		chooser := g.GetPlayer(playerID)
		if effectChooser, ok := chooser.(EffectModeChooser); ok {
			modeChoice = effectChooser.ChooseModeWithEffects(available, perm.Card.Name(), g)
		} else if chooser != nil {
			labels := make([]string, len(available))
			for i, mode := range available {
				labels[i] = mode.Label
			}
			modeChoice = chooser.ChooseMode(labels, perm.Card.Name())
		}
		if modeChoice < 0 || modeChoice >= len(available) {
			return fmt.Errorf("invalid activation mode choice")
		}
		modeChoice = indices[modeChoice]
		mode := modes[modeChoice]
		actionTargets = mode.Targets
		actionEffects = mode.Effects
		targetSource = g.AbilityTargetSource(perm.Card, aa, actionTargets)
		targets = g.promptTargetsForList(playerID, targetSource, actionTargets)
		modalTargets = make([][]uuid.UUID, len(modes))
		modalTargets[modeChoice] = append([]uuid.UUID(nil), targets...)
	}

	if grouped {
		var err error
		targets, alignedTargets, err = g.prepareTargetGroups(playerID, targetSource, actionTargets, groups, g.resolution.X(), "ability target group")
		if err != nil {
			return err
		}
	}
	// When the caller supplied no targets but the ability's targeting is
	// forced (e.g. "you", or a single legal target), fill them in here so the
	// ability doesn't fizzle. The interactive layer relies on this to skip
	// prompting for targets that offer no real choice.
	if !grouped && len(targets) == 0 {
		if forced := ForcedActivationTargets(playerID, targetSource, actionTargets, g); forced != nil {
			targets = forced
		}
	}
	if !grouped {
		targets = g.acquireRandomTargets(playerID, targetSource, actionTargets, targets, g.resolution.X())
		if modalTargets == nil {
			// The mode choice already includes the opponent's target choice.
			targets = g.acquireOpponentChosenTargets(playerID, targetSource, actionTargets, targets)
		}
	}

	if !grouped {
		if err := g.validateActionTargets(playerID, targetSource, actionTargets, targets, g.resolution.X(), "ability"); err != nil {
			return err
		}
	}

	damageDistribution, err := g.prepareActivationDamageDivision(playerID, perm.Card, actionEffects, targets)
	if err != nil {
		return err
	}

	// Lock and validate the total activation cost before changing live state.
	// If the ability includes {T}, reserve its source from mana planning.
	hasTapCost := false
	for _, c := range aa.Costs() {
		if _, ok := c.(*tap); ok {
			hasTapCost = true
			break
		}
	}
	hint := AutoTapHint{ActivationSource: perm.ID(), ActivationTapsSource: hasTapCost}
	preparedCosts := g.prepareActionCosts(aa.Costs(), targets, g.resolution.X())
	payment, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller:               playerID,
		SourceID:                 perm.ID(),
		Costs:                    preparedCosts,
		Targets:                  targets,
		XValue:                   g.resolution.X(),
		Hint:                     hint,
		ApplyActivationReduction: true,
	})
	if err != nil {
		return fmt.Errorf("cannot pay activation costs: %w", err)
	}
	var capturedNumbers map[*NumberBinding]int
	var capturedPermanents map[*PermanentBinding]capturedPermanent
	if definition, ok := aa.(*ActionDefinition); ok {
		captureSpecs := actionTargets
		if grouped {
			captureSpecs = alignedTargets
		}
		capturedNumbers, capturedPermanents = definition.captureValues(g, perm, playerID, captureSpecs, targets)
	}
	handTargets := g.handTargetCards(targets)
	exileTargets := g.exileTargetCards(targets)
	anteTargets := g.anteTargetCards(targets)
	if err := payment.Commit(); err != nil {
		return err
	}
	payment.recordAbilityTapPayments(capturedPermanent{perm.ID(), perm.incarnationID})
	if len(payment.paidNumbers) > 0 {
		if capturedNumbers == nil {
			capturedNumbers = make(map[*NumberBinding]int)
		}
		maps.Copy(capturedNumbers, payment.paidNumbers)
	}

	// Mark once-per-turn abilities as used
	if saa, ok := aa.(*SimpleActivatedAbility); ok {
		saa.markActivatedInGame(g, perm.incarnationID)
	}

	obj := newStackObject(playerID, perm.ID(), nil, actionEffects, targets, g.resolution.X(), true)
	obj.activatedAbility = true
	obj.targetHandCards = handTargets
	obj.targetExileCards = exileTargets
	obj.targetAnteCards = anteTargets
	obj.DamageDistribution = damageDistribution
	obj.capturedNumbers = capturedNumbers
	obj.capturedCards = maps.Clone(payment.paidCards)
	obj.revealedHandCards = slices.Clone(payment.revealedHandCards)
	obj.ownsHandReveals = len(obj.revealedHandCards) > 0
	obj.capturedCardSets = cloneCardsBindings(payment.paidCardSets)
	if len(payment.paidPermanents) > 0 {
		if capturedPermanents == nil {
			capturedPermanents = make(map[*PermanentBinding]capturedPermanent)
		}
		maps.Copy(capturedPermanents, payment.paidPermanents)
	}
	obj.capturedPermanents = capturedPermanents
	obj.TargetSpecs = expandTargetSpecs(actionTargets, targets, g.resolution.X())
	if grouped {
		obj.TargetSpecs = append([]Target(nil), alignedTargets...)
	}
	obj.sourceIncarnation = perm.incarnationID
	obj.TargetSource = targetSource
	obj.ModeChoice = modeChoice
	obj.ModalTargets = modalTargets

	if canProduceMana(actionEffects, actionTargets) {
		g.FireEvent(GameEvent{
			Type:              EvtAbilityActivated,
			SourceID:          perm.ID(),
			SourceIncarnation: perm.incarnationID,
			PlayerID:          playerID,
			Flag:              hasTapCost,
		})
		obj.tappedForMana = hasTapCost
		produced, colors := g.resolveStackObject(obj)
		if hasTapCost && produced > 0 {
			g.mana.ApplyManaBonuses(g, perm, colors, g.GetPlayer(playerID))
			g.mana.fireTappedForManaWithTypes(g, perm.ID(), obj.sourceIncarnation, playerID, produced, colors)
		}
		return nil
	}

	if modeChoice < 0 {
		chooseModeForStackObject(obj, perm.Card.Modes(), g.GetPlayer(playerID), perm.Card.Name())
	}

	g.pushStack(obj)

	g.FireEvent(GameEvent{
		Type:               EvtAbilityActivated,
		ActivatedAbilityID: obj.ID,
		SourceID:           perm.ID(),
		SourceIncarnation:  perm.incarnationID,
		PlayerID:           playerID,
		Flag:               hasTapCost,
	})

	g.fireBecomesTargetEvents(obj, true)

	return nil
}

// IsGameOver reports whether the game has ended.
func (g *Game) IsGameOver() bool {
	if g.drawn {
		return true
	}
	for _, p := range g.players {
		if !p.IsAlive() {
			return true
		}
	}
	return false
}

// Winner returns the winner's name, or an empty string if there is no winner.
func (g *Game) Winner() string {
	if g.drawn {
		return ""
	}
	var alive []Player
	for _, p := range g.players {
		if p.IsAlive() {
			alive = append(alive, p)
		}
	}
	if len(alive) == 1 {
		return alive[0].Name()
	}
	return ""
}

// IsDraw reports whether the game has ended in a draw (CR 104.4).
func (g *Game) IsDraw() bool {
	if g.drawn {
		return true
	}
	if !g.IsGameOver() {
		return false
	}
	for _, p := range g.players {
		if p.IsAlive() {
			return false
		}
	}
	return true
}

func (g *Game) Stack() *Stack {
	return g.stack
}

func (g *Game) stateTriggerActive(key stateTriggerKey) bool {
	if g.triggers.resolvingState == key {
		return true
	}
	for _, pt := range g.triggers.Pending() {
		if pt.stateKey == key {
			return true
		}
	}
	for _, obj := range g.stack.Objects() {
		if obj.stateKey == key {
			return true
		}
	}
	return false
}

func (g *Game) queuePendingTrigger(pt *pendingTrigger) {
	if trigger, ok := pt.ability.(*GenericTriggered); ok && trigger.sourceCardZoneOptions > 0 {
		g.triggers.AddPending(pt)
		return
	}
	if p := g.FindPermanent(pt.sourceID); p != nil {
		pt.sourceIncarnation = p.incarnationID
	} else if lki := g.LKI(pt.sourceID); lki != nil && lki.Snapshot != nil {
		pt.sourceIncarnation = lki.Snapshot.incarnationID
	}
	g.triggers.AddPending(pt)
}

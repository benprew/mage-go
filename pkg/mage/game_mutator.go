package mage

import (
	. "github.com/mage/mage/pkg/mage/core"

	"github.com/google/uuid"
)

// GameReader is the read-only view of Game used by ValueSource and PlayerSelector.
// It exposes query methods but no mutation. *Game satisfies this interface.
type GameReader interface {
	GetPlayer(uuid.UUID) Player
	GetOpponent(uuid.UUID) Player
	ActivePlayerObj() Player
	NonActivePlayerObj() Player
	FindPermanent(uuid.UUID) *Permanent
	FindPermanentByName(string, uuid.UUID) *Permanent
	FindCardAnywhere(uuid.UUID) Card
	AnyBattlefield(PermanentFilter) bool
	FilterBattlefield(PermanentFilter) []*Permanent
	CountBattlefield(PermanentFilter) int
	AllPlayers() []Player
	XValue() int
	ModeValue() int
	GetResolvingCard() Card
	FindStackObject(uuid.UUID) *StackObject
	CombatGroups() []*CombatGroup
	DamageTakenByPlayer(uuid.UUID) int
	HasAttackedThisTurn(uuid.UUID) bool
	CreatureDeaths() int
}

// GameMutator is the mutation surface passed to Effect.Apply. It embeds GameReader
// for all read access and adds high-level mutation verbs. Using this interface
// instead of raw *Game prevents effects from accidentally reaching into unexposed
// fields and clarifies what effects are permitted to do.
type GameMutator interface {
	GameReader

	// Player/life mutations
	PlayerGainLife(Player, int)
	FireEvent(GameEvent)

	// Battlefield mutations
	PutOnBattlefield(Card, uuid.UUID) *Permanent
	RemoveFromBattlefield(*Permanent)
	DestroyPermanent(*Permanent)
	ExilePermanent(*Permanent)
	Sacrifice(*Permanent)

	// Damage
	DealDamageToPlayer(Player, int, uuid.UUID)
	DealDamageToPermanent(*Permanent, int, uuid.UUID)

	// Stack
	CounterSpellOnStack(uuid.UUID)
	PushStack(*StackObject)

	// Other game actions
	Attach(sourceID, targetID uuid.UUID)
	RegisterDelayedTrigger(*DelayedTrigger)
	GrantExtraTurn(uuid.UUID)
	RemoveFromCombat(uuid.UUID)

	// Continuous effect management
	AddContinuousEffect(ContinuousEffect)
	ApplyContinuousEffects()

	// EffectManager delegators
	SetPreventCombatDamage()
	AddRegenerationShield(uuid.UUID)
	AddPreventionShield(uuid.UUID, int)
	AddForcefieldShield(uuid.UUID)
	IsLichActive(uuid.UUID) bool
	SetLichActive(uuid.UUID, uuid.UUID)
	ClearLich(uuid.UUID)
	AddColorPrevention(uuid.UUID, Color)
	AddReverseDamageShield(uuid.UUID)
	SetChannelActive(uuid.UUID)
	SetCreatureDamageRedirect(uuid.UUID, uuid.UUID)
	SetSkipNextDraw(uuid.UUID)
	SetSanctuaryActive(uuid.UUID)
	SetMinimumLife(uuid.UUID)
	CopyEffectCurrentName(uuid.UUID) string
	UpdateCopyEffect(uuid.UUID, *Permanent)

	// Damage prevention / redirection / reflection
	AddTypePrevention(uuid.UUID, CardType)
	SetArtifactDamageRedirect(controllerID, permID uuid.UUID)
	SetDamageReflection(playerID, eyeSourceID, chosenSourceID uuid.UUID)

	// Draw replacement (Aladdin's Lamp)
	SetDrawReplacement(playerID uuid.UUID, count int)

	// Mana restriction
	SetArtifactManaOnly(uuid.UUID)
	SetCreatureManaOnly(uuid.UUID)

	// Artifact damage tracking
	GetArtifactDamageTaken(uuid.UUID) int

	// Coin flip
	FlipCoin(playerID uuid.UUID) bool

	// Mana payment
	TryPayCostFromLands(playerID uuid.UUID, manaCostStr string) bool

	// Spell casting (e.g. Shahrazad)
	CastSpellByName(playerID uuid.UUID, name string, targets []uuid.UUID, xValues ...int) error
}

// Compile-time checks that *Game satisfies both interfaces.
var _ GameReader = (*Game)(nil)
var _ GameMutator = (*Game)(nil)

// --- GameReader proxy methods on *Game ---

// AllPlayers returns all players in the game.
func (g *Game) AllPlayers() []Player { return g.Players }

// XValue returns the current X value for the resolving spell/ability.
func (g *Game) XValue() int { return g.CurrentX }

// ModeValue returns the current chosen mode for the resolving modal spell.
func (g *Game) ModeValue() int { return g.CurrentMode }

// GetResolvingCard returns the card currently being resolved from the stack.
func (g *Game) GetResolvingCard() Card { return g.ResolvingCard }

// FindStackObject finds a stack object by its source card ID.
func (g *Game) FindStackObject(id uuid.UUID) *StackObject {
	return g.Stack.FindBySourceID(id)
}

// CombatGroups returns the current combat groups (attacker/blocker pairings).
// Returns nil if combat has not been initialized.
func (g *Game) CombatGroups() []*CombatGroup {
	if g.Combat == nil {
		return nil
	}
	return g.Combat.Groups
}

// DamageTakenByPlayer returns the total damage the given player has taken this turn.
func (g *Game) DamageTakenByPlayer(playerID uuid.UUID) int {
	return g.DamageTakenThisTurn[playerID]
}

// HasAttackedThisTurn reports whether the permanent with the given ID attacked this turn.
func (g *Game) HasAttackedThisTurn(permID uuid.UUID) bool {
	return g.AttackedThisTurn[permID]
}

// CreatureDeaths returns the number of creatures that died this turn.
func (g *Game) CreatureDeaths() int {
	return g.CreatureDeathsThisTurn
}

// --- GameMutator proxy methods on *Game ---

// GrantExtraTurn gives the specified player an extra turn after the current one.
func (g *Game) GrantExtraTurn(playerID uuid.UUID) {
	g.ExtraTurns = append(g.ExtraTurns, playerID)
}

// RemoveFromCombat removes a permanent from combat by ID.
func (g *Game) RemoveFromCombat(id uuid.UUID) {
	g.Combat.RemoveFromCombat(id)
}

// PushStack pushes a stack object onto the stack.
func (g *Game) PushStack(obj *StackObject) {
	g.Stack.Push(obj)
}

// AddContinuousEffect registers a continuous effect with the effect manager.
func (g *Game) AddContinuousEffect(e ContinuousEffect) {
	g.Effects.Add(e)
}

// ApplyContinuousEffects re-applies all continuous effects to current permanents.
func (g *Game) ApplyContinuousEffects() {
	g.Effects.Apply(g)
}

// SetPreventCombatDamage flags that all combat damage is prevented this turn.
func (g *Game) SetPreventCombatDamage() {
	g.Effects.Damage.SetPreventCombatDamage()
}

// AddRegenerationShield adds a regeneration shield to the specified permanent.
func (g *Game) AddRegenerationShield(id uuid.UUID) {
	g.Effects.Damage.AddRegenerationShield(id)
}

// AddPreventionShield adds a damage prevention shield to the specified permanent.
func (g *Game) AddPreventionShield(id uuid.UUID, amount int) {
	g.Effects.Damage.AddPreventionShield(id, amount)
}

// AddForcefieldShield adds a Forcefield shield for the specified player.
func (g *Game) AddForcefieldShield(id uuid.UUID) {
	g.Effects.Damage.AddForcefieldShield(id)
}

// IsLichActive reports whether the Lich enchantment is active for the player.
func (g *Game) IsLichActive(playerID uuid.UUID) bool {
	return g.Effects.IsLichActive(g, playerID)
}

// SetLichActive marks the Lich enchantment as active for the player.
func (g *Game) SetLichActive(playerID, sourceID uuid.UUID) {
	g.Effects.SetLichActive(playerID, sourceID)
}

// ClearLich removes the Lich enchantment state for the player.
func (g *Game) ClearLich(playerID uuid.UUID) {
	g.Effects.ClearLich(playerID)
}

// AddColorPrevention adds a color-based damage prevention rule for the player.
func (g *Game) AddColorPrevention(playerID uuid.UUID, color Color) {
	g.Effects.Damage.AddColorPrevention(playerID, color)
}

// AddReverseDamageShield adds a reverse-damage shield for the player.
func (g *Game) AddReverseDamageShield(playerID uuid.UUID) {
	g.Effects.Damage.AddReverseDamageShield(playerID)
}

// SetChannelActive marks the Channel ability as active for the player.
func (g *Game) SetChannelActive(playerID uuid.UUID) {
	g.Effects.SetChannelActive(playerID)
}

// SetCreatureDamageRedirect redirects damage dealt to a creature to a player.
func (g *Game) SetCreatureDamageRedirect(creatureID, playerID uuid.UUID) {
	g.Effects.Damage.SetCreatureDamageRedirect(creatureID, playerID)
}

// SetSkipNextDraw sets a flag to skip the next draw step for the player.
func (g *Game) SetSkipNextDraw(playerID uuid.UUID) {
	g.Effects.SetSkipNextDraw(playerID)
}

// SetSanctuaryActive marks the Ivory Tower sanctuary effect as active.
func (g *Game) SetSanctuaryActive(playerID uuid.UUID) {
	g.Effects.SetSanctuaryActive(playerID)
}

// SetMinimumLife marks a player as having minimum-life protection (Ali from Cairo).
func (g *Game) SetMinimumLife(playerID uuid.UUID) {
	g.Effects.SetMinimumLife(playerID)
}

// AddTypePrevention adds a card-type damage prevention rule for the player.
func (g *Game) AddTypePrevention(playerID uuid.UUID, ct CardType) {
	g.Effects.Damage.AddTypePrevention(playerID, ct)
}

// SetArtifactDamageRedirect sets a creature that absorbs artifact damage dealt to a player.
func (g *Game) SetArtifactDamageRedirect(controllerID, permID uuid.UUID) {
	g.Effects.Damage.SetArtifactDamageRedirect(controllerID, permID)
}

// SetDamageReflection sets a one-shot damage reflection for a player (Eye for an Eye).
func (g *Game) SetDamageReflection(playerID, eyeSourceID, chosenSourceID uuid.UUID) {
	g.Effects.Damage.SetDamageReflection(playerID, eyeSourceID, chosenSourceID)
}

// SetDrawReplacement stores a pending draw replacement for a player (Aladdin's Lamp).
func (g *Game) SetDrawReplacement(playerID uuid.UUID, count int) {
	g.Effects.Damage.SetDrawReplacement(playerID, count)
}

// SetArtifactManaOnly marks a player as having artifact-only mana restriction active.
func (g *Game) SetArtifactManaOnly(playerID uuid.UUID) {
	if g.ArtifactManaOnly == nil {
		g.ArtifactManaOnly = make(map[uuid.UUID]bool)
	}
	g.ArtifactManaOnly[playerID] = true
}

// SetCreatureManaOnly marks a player as having creature-only mana restriction active.
func (g *Game) SetCreatureManaOnly(playerID uuid.UUID) {
	if g.CreatureManaOnly == nil {
		g.CreatureManaOnly = make(map[uuid.UUID]bool)
	}
	g.CreatureManaOnly[playerID] = true
}

// GetArtifactDamageTaken returns the artifact damage the player has taken this turn.
func (g *Game) GetArtifactDamageTaken(playerID uuid.UUID) int {
	return g.ArtifactDamageTakenThisTurn[playerID]
}

// CopyEffectCurrentName returns the name of the creature currently being copied
// by the Doppelganger copy effect for the specified permanent.
func (g *Game) CopyEffectCurrentName(permID uuid.UUID) string {
	return g.Effects.CopyEffectCurrentName(permID)
}

// UpdateCopyEffect updates the Doppelganger copy effect to copy a new target.
func (g *Game) UpdateCopyEffect(permID uuid.UUID, target *Permanent) {
	g.Effects.UpdateCopyEffect(permID, target)
}

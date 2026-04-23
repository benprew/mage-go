package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Pipeline: sequence of steps sharing an EffectContext
// ---------------------------------------------------------------------------

// PipelineData executes steps in sequence, sharing an EffectContext so
// intermediate values flow between steps without closures.
type PipelineData struct {
	Steps []EffectData
	Txt   string
	Props EffectProperties
}

// Pipeline creates a pipeline effect from a sequence of steps.
func Pipeline(text string, props EffectProperties, steps ...EffectData) Effect {
	return DataEffect(&PipelineData{Steps: steps, Txt: text, Props: props})
}

func (e *PipelineData) EffectText() string          { return e.Txt }
func (e *PipelineData) EffectProps() EffectProperties { return e.Props }

func execPipeline(ctx *EffectContext, e *PipelineData) error {
	for _, step := range e.Steps {
		if err := ExecuteEffect(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Snapshot: read a permanent's properties into context variables
// ---------------------------------------------------------------------------

// SnapshotPermanentData reads a permanent (from target or source) and stores
// its properties as named variables in the EffectContext:
//
//	"{name}"            → uuid.UUID (the permanent's ID)
//	"{name}.power"      → int (current power)
//	"{name}.toughness"  → int (current toughness)
//	"{name}.controller" → uuid.UUID
//	"{name}.cmc"        → int (converted mana cost)
//	"{name}.name"       → string
type SnapshotPermanentData struct {
	Selector PermanentSelector
	StoreAs  string
}

// SnapshotPermanent creates a pipeline step that reads a permanent's properties.
func SnapshotPermanent(sel PermanentSelector, storeAs string) EffectData {
	return &SnapshotPermanentData{Selector: sel, StoreAs: storeAs}
}

func (e *SnapshotPermanentData) EffectText() string          { return "" }
func (e *SnapshotPermanentData) EffectProps() EffectProperties { return EffectProperties{} }

func execSnapshotPermanent(ctx *EffectContext, e *SnapshotPermanentData) error {
	var perm *Permanent
	if e.Selector == SelectSource {
		perm = ctx.Game.FindPermanent(ctx.SourceID)
	} else {
		if len(ctx.Targets) == 0 {
			return nil
		}
		perm = ctx.Game.FindPermanent(ctx.Targets[0])
	}
	if perm == nil {
		ctx.SetBool(e.StoreAs+".missing", true)
		return nil
	}
	ctx.SetUUID(e.StoreAs, perm.ID())
	ctx.SetInt(e.StoreAs+".power", perm.CurrentPower(ctx.Game))
	ctx.SetInt(e.StoreAs+".toughness", perm.CurrentToughness(ctx.Game))
	ctx.SetUUID(e.StoreAs+".controller", perm.Controller)
	ctx.SetInt(e.StoreAs+".cmc", perm.Card.ManaCost().CMC())
	ctx.Vars[e.StoreAs+".name"] = perm.Name()
	return nil
}

// ---------------------------------------------------------------------------
// SnapshotSource: read the source permanent's counters into context
// ---------------------------------------------------------------------------

// SnapshotSourceCounterData reads a counter value from the source permanent.
type SnapshotSourceCounterData struct {
	CounterType CounterType
	StoreAs     string
}

// SnapshotSourceCounter reads a counter count from the source permanent.
func SnapshotSourceCounter(ct CounterType, storeAs string) EffectData {
	return &SnapshotSourceCounterData{CounterType: ct, StoreAs: storeAs}
}

func (e *SnapshotSourceCounterData) EffectText() string          { return "" }
func (e *SnapshotSourceCounterData) EffectProps() EffectProperties { return EffectProperties{} }

func execSnapshotSourceCounter(ctx *EffectContext, e *SnapshotSourceCounterData) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		ctx.SetInt(e.StoreAs, 0)
		return nil
	}
	ctx.SetInt(e.StoreAs, int(perm.Counters[e.CounterType]))
	return nil
}

// ---------------------------------------------------------------------------
// Gathered-permanent operations: act on a variable-bound permanent
// ---------------------------------------------------------------------------

// ExileGatheredData exiles the permanent stored in a context variable.
type ExileGatheredData struct{ VarName string }

func ExileGathered(varName string) EffectData { return &ExileGatheredData{VarName: varName} }

func (e *ExileGatheredData) EffectText() string            { return "exile" }
func (e *ExileGatheredData) EffectProps() EffectProperties { return EffectProperties{Outcome: OutcomeDetriment} }

func execExileGathered(ctx *EffectContext, e *ExileGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm == nil {
		return nil
	}
	ctx.Game.ExilePermanent(perm)
	return nil
}

// DestroyGatheredData destroys the permanent stored in a context variable.
type DestroyGatheredData struct {
	VarName    string
	NoRegen    bool
}

func DestroyGathered(varName string) EffectData {
	return &DestroyGatheredData{VarName: varName}
}
func DestroyGatheredNoRegen(varName string) EffectData {
	return &DestroyGatheredData{VarName: varName, NoRegen: true}
}

func (e *DestroyGatheredData) EffectText() string            { return "destroy" }
func (e *DestroyGatheredData) EffectProps() EffectProperties { return EffectProperties{Outcome: OutcomeDetriment} }

func execDestroyGathered(ctx *EffectContext, e *DestroyGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm == nil {
		return nil
	}
	if perm.HasKeyword(Indestructible) {
		return nil
	}
	if e.NoRegen {
		perm.GrantBaseAttr(CantRegenerate)
	}
	ctx.Game.DestroyPermanent(perm)
	return nil
}

// SacrificeGatheredData sacrifices the permanent stored in a context variable.
type SacrificeGatheredData struct{ VarName string }

func SacrificeGathered(varName string) EffectData { return &SacrificeGatheredData{VarName: varName} }

func (e *SacrificeGatheredData) EffectText() string            { return "sacrifice" }
func (e *SacrificeGatheredData) EffectProps() EffectProperties { return EffectProperties{} }

func execSacrificeGathered(ctx *EffectContext, e *SacrificeGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm == nil {
		return nil
	}
	ctx.Game.Sacrifice(perm)
	return nil
}

// BounceGatheredData returns the permanent stored in a context variable to its owner's hand.
type BounceGatheredData struct{ VarName string }

func BounceGathered(varName string) EffectData { return &BounceGatheredData{VarName: varName} }

func (e *BounceGatheredData) EffectText() string            { return "return to hand" }
func (e *BounceGatheredData) EffectProps() EffectProperties { return EffectProperties{Outcome: OutcomeDetriment, IsBounce: true} }

func execBounceGathered(ctx *EffectContext, e *BounceGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm == nil {
		return nil
	}
	card := perm.Card
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = perm.Controller
	}
	ctx.Game.RemoveFromBattlefield(perm)
	p := ctx.Game.GetPlayer(owner)
	if p != nil {
		p.AddToHand(card)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Variable-based ValueSource and PlayerSelector
// ---------------------------------------------------------------------------

// VarValue is a ValueSource that reads an int from EffectContext.Vars.
// It can only be used inside a Pipeline (where an EffectContext exists).
// For standalone use it returns 0.
type VarValue struct {
	Name    string
	context *EffectContext // set by the executor before resolving
}

func VarInt(name string) *VarValue { return &VarValue{Name: name} }

func (v *VarValue) Resolve(_ GameReader, _, _ uuid.UUID, _ []uuid.UUID) int {
	if v.context == nil {
		return 0
	}
	return v.context.GetInt(v.Name)
}

func (v *VarValue) Text() string { return v.Name }

// VarPlayerSelector is a PlayerSelector that reads a UUID from EffectContext.Vars.
type VarPlayerSelector struct {
	Name    string
	context *EffectContext
}

func VarPlayer(name string) *VarPlayerSelector { return &VarPlayerSelector{Name: name} }

func (s *VarPlayerSelector) Select(_ GameReader, _, _ uuid.UUID, _ []uuid.UUID) []uuid.UUID {
	if s.context == nil {
		return nil
	}
	id := s.context.TryGetUUID(s.Name)
	if id == uuid.Nil {
		return nil
	}
	return []uuid.UUID{id}
}

func (s *VarPlayerSelector) Text() string { return s.Name }

// GainLifeVarData gains life for a player identified by a context variable.
type GainLifeVarData struct {
	PlayerVar string
	AmountVar string
}

func GainLifeFromVar(playerVar, amountVar string) EffectData {
	return &GainLifeVarData{PlayerVar: playerVar, AmountVar: amountVar}
}

func (e *GainLifeVarData) EffectText() string            { return "gain life" }
func (e *GainLifeVarData) EffectProps() EffectProperties { return EffectProperties{Outcome: OutcomeBenefit} }

func execGainLifeVar(ctx *EffectContext, e *GainLifeVarData) error {
	playerID := ctx.TryGetUUID(e.PlayerVar)
	if playerID == uuid.Nil {
		return nil
	}
	amount := ctx.GetInt(e.AmountVar)
	if amount <= 0 {
		return nil
	}
	p := ctx.Game.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	ctx.Game.PlayerGainLife(p, amount)
	if !ctx.Game.IsLichActive(playerID) {
		ctx.Game.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: playerID, Amount: amount})
	}
	return nil
}

// DealDamageVarData deals damage read from a context variable to a target.
type DealDamageVarData struct {
	AmountVar string
}

func DealDamageFromVar(amountVar string) EffectData {
	return &DealDamageVarData{AmountVar: amountVar}
}

func (e *DealDamageVarData) EffectText() string            { return "deal damage" }
func (e *DealDamageVarData) EffectProps() EffectProperties { return EffectProperties{Outcome: OutcomeDetriment} }

func execDealDamageVar(ctx *EffectContext, e *DealDamageVarData) error {
	amount := ctx.GetInt(e.AmountVar)
	if amount <= 0 {
		return nil
	}
	if len(ctx.Targets) == 0 {
		return nil
	}
	targetID := ctx.Targets[0]
	for _, pl := range ctx.Game.AllPlayers() {
		if pl.PlayerID() == targetID {
			ctx.Game.DealDamageToPlayer(pl, amount, ctx.SourceID)
			return nil
		}
	}
	perm := ctx.Game.FindPermanent(targetID)
	if perm != nil {
		ctx.Game.DealDamageToPermanent(perm, amount, ctx.SourceID)
	}
	return nil
}

// DealDamageToPlayersVarData deals damage from a context variable to selected players.
type DealDamageToPlayersVarData struct {
	AmountVar string
	Selector  PlayerSelector
}

func DealDamageToPlayersFromVar(amountVar string, sel PlayerSelector) EffectData {
	return &DealDamageToPlayersVarData{AmountVar: amountVar, Selector: sel}
}

func (e *DealDamageToPlayersVarData) EffectText() string            { return "deal damage to players" }
func (e *DealDamageToPlayersVarData) EffectProps() EffectProperties { return EffectProperties{Outcome: OutcomeDetriment} }

func execDealDamageToPlayersVar(ctx *EffectContext, e *DealDamageToPlayersVarData) error {
	amount := ctx.GetInt(e.AmountVar)
	if amount <= 0 {
		return nil
	}
	if vps, ok := e.Selector.(*VarPlayerSelector); ok {
		vps.context = ctx
	}
	playerIDs := e.Selector.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	for _, pid := range playerIDs {
		p := ctx.Game.GetPlayer(pid)
		if p != nil {
			ctx.Game.DealDamageToPlayer(p, amount, ctx.SourceID)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ForEachPermanent: iterate matching permanents, apply effect per-permanent
// ---------------------------------------------------------------------------

// ForEachPermanentData iterates permanents matching a filter and executes an
// inner effect for each. The inner effect receives the iterated permanent's
// ID as targets[0] in the context.
//
// If Who is non-nil, only permanents controlled by the resolved player are
// included (e.g., SelectController for "each creature you control").
type ForEachPermanentData struct {
	Filter PermanentFilter
	Who    PlayerSelector // optional: restrict to one player's permanents
	Inner  EffectData
	Txt    string
}

// ForEachPermanent creates an effect that iterates matching permanents.
func ForEachPermanent(filter PermanentFilter, inner EffectData, text string) EffectData {
	return &ForEachPermanentData{Filter: filter, Inner: inner, Txt: text}
}

// ForEachControlledPermanent creates an effect that iterates matching permanents
// controlled by the specified player.
func ForEachControlledPermanent(who PlayerSelector, filter PermanentFilter, inner EffectData, text string) EffectData {
	return &ForEachPermanentData{Filter: filter, Who: who, Inner: inner, Txt: text}
}

func (e *ForEachPermanentData) EffectText() string            { return e.Txt }
func (e *ForEachPermanentData) EffectProps() EffectProperties { return EffectProperties{Mass: true} }

func execForEachPermanent(ctx *EffectContext, e *ForEachPermanentData) error {
	perms := ctx.Game.FilterBattlefield(e.Filter)
	var controllerFilter uuid.UUID
	if e.Who != nil {
		selected := e.Who.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
		if len(selected) > 0 {
			controllerFilter = selected[0]
		}
	}
	ids := make([]uuid.UUID, 0, len(perms))
	for _, p := range perms {
		if controllerFilter != uuid.Nil && p.Controller != controllerFilter {
			continue
		}
		ids = append(ids, p.ID())
	}
	savedTargets := ctx.Targets
	for _, id := range ids {
		ctx.Targets = []uuid.UUID{id}
		if err := ExecuteEffect(ctx, e.Inner); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// ---------------------------------------------------------------------------
// IfElse: conditional branching
// ---------------------------------------------------------------------------

// ConditionData evaluates a boolean condition against the current game state.
type ConditionData interface {
	Check(ctx *EffectContext) bool
}

// IfElseData branches based on a condition.
type IfElseData struct {
	Cond ConditionData
	Then EffectData
	Else EffectData // nil = do nothing
	Txt  string
}

// IfElse creates a conditional effect.
func IfElse(text string, cond ConditionData, then, els EffectData) EffectData {
	return &IfElseData{Cond: cond, Then: then, Else: els, Txt: text}
}

func (e *IfElseData) EffectText() string            { return e.Txt }
func (e *IfElseData) EffectProps() EffectProperties { return EffectProperties{} }

func execIfElse(ctx *EffectContext, e *IfElseData) error {
	if e.Cond.Check(ctx) {
		return ExecuteEffect(ctx, e.Then)
	}
	if e.Else != nil {
		return ExecuteEffect(ctx, e.Else)
	}
	return nil
}

// --- Condition implementations ---

// HasMatchingPermanentCond checks if any permanents match a filter.
type HasMatchingPermanentCond struct {
	Filter PermanentFilter
}

func (c *HasMatchingPermanentCond) Check(ctx *EffectContext) bool {
	return ctx.Game.AnyBattlefield(c.Filter)
}

// VarGTCond checks if a context variable is greater than a value.
type VarGTCond struct {
	Name  string
	Value int
}

func (c *VarGTCond) Check(ctx *EffectContext) bool {
	return ctx.GetInt(c.Name) > c.Value
}

// TryPayManaCond tries to pay a mana cost; returns true if paid.
type TryPayManaCond struct {
	Cost string
}

func (c *TryPayManaCond) Check(ctx *EffectContext) bool {
	return ctx.Game.TryPayCostFromLands(ctx.Controller, c.Cost)
}

// VarMissingCond checks if a snapshot marked the permanent as missing.
type VarMissingCond struct {
	Name string
}

func (c *VarMissingCond) Check(ctx *EffectContext) bool {
	return ctx.GetBool(c.Name + ".missing")
}

// NotCond negates a condition.
type NotCond struct {
	Inner ConditionData
}

func (c *NotCond) Check(ctx *EffectContext) bool { return !c.Inner.Check(ctx) }

// ---------------------------------------------------------------------------
// Modal: branch on g.ModeValue()
// ---------------------------------------------------------------------------

// ModalEffectData branches on the chosen mode value (0, 1, ...).
type ModalEffectData struct {
	Modes []EffectData
	Txt   string
}

// ModalEffect creates an effect that executes one of several modes.
func ModalEffect(text string, modes ...EffectData) EffectData {
	return &ModalEffectData{Modes: modes, Txt: text}
}

func (e *ModalEffectData) EffectText() string            { return e.Txt }
func (e *ModalEffectData) EffectProps() EffectProperties { return EffectProperties{} }

func execModalEffect(ctx *EffectContext, e *ModalEffectData) error {
	mode := ctx.Game.ModeValue()
	if mode < 0 || mode >= len(e.Modes) {
		return fmt.Errorf("invalid mode %d (have %d modes)", mode, len(e.Modes))
	}
	return ExecuteEffect(ctx, e.Modes[mode])
}

// ---------------------------------------------------------------------------
// ChoosePermanent: player picks from matching permanents
// ---------------------------------------------------------------------------

// ChoosePermanentData asks a player to choose a permanent matching a filter
// and stores the result in a context variable.
type ChoosePermanentData struct {
	Player   PlayerSelector
	Filter   PermanentFilter
	Reason   string
	StoreAs  string
}

// ChoosePermanentStep creates a pipeline step where a player chooses a permanent.
func ChoosePermanentStep(player PlayerSelector, filter PermanentFilter, reason, storeAs string) EffectData {
	return &ChoosePermanentData{Player: player, Filter: filter, Reason: reason, StoreAs: storeAs}
}

func (e *ChoosePermanentData) EffectText() string            { return "" }
func (e *ChoosePermanentData) EffectProps() EffectProperties { return EffectProperties{} }

func execChoosePermanent(ctx *EffectContext, e *ChoosePermanentData) error {
	playerIDs := e.Player.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if len(playerIDs) == 0 {
		return nil
	}
	player := ctx.Game.GetPlayer(playerIDs[0])
	if player == nil {
		return nil
	}
	candidates := ctx.Game.FilterBattlefield(e.Filter)
	if len(candidates) == 0 {
		ctx.SetBool(e.StoreAs+".missing", true)
		return nil
	}
	chosen := player.ChoosePermanent(candidates, e.Reason, ctx.Game)
	if chosen == nil {
		ctx.SetBool(e.StoreAs+".missing", true)
		return nil
	}
	ctx.SetUUID(e.StoreAs, chosen.ID())
	ctx.SetInt(e.StoreAs+".power", chosen.CurrentPower(ctx.Game))
	ctx.SetInt(e.StoreAs+".toughness", chosen.CurrentToughness(ctx.Game))
	ctx.SetUUID(e.StoreAs+".controller", chosen.Controller)
	return nil
}

// ---------------------------------------------------------------------------
// SacrificeSource step (for pipelines — wraps the existing effect)
// ---------------------------------------------------------------------------

// SacrificeSourceData sacrifices the source permanent. Usable in pipelines.
type SacrificeSourceData struct{}

func SacrificeSourceStep() EffectData { return &SacrificeSourceData{} }

func (e *SacrificeSourceData) EffectText() string            { return "sacrifice" }
func (e *SacrificeSourceData) EffectProps() EffectProperties { return EffectProperties{} }

func execSacrificeSourceStep(ctx *EffectContext, _ *SacrificeSourceData) error {
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	ctx.Game.Sacrifice(perm)
	return nil
}

// ---------------------------------------------------------------------------
// ShuffleGraveyardIntoLibrary step (Feldon's Cane pattern)
// ---------------------------------------------------------------------------

type ShuffleGraveyardIntoLibraryData struct{}

func ShuffleGraveyardIntoLibrary() EffectData { return &ShuffleGraveyardIntoLibraryData{} }

func (e *ShuffleGraveyardIntoLibraryData) EffectText() string { return "shuffle graveyard into library" }
func (e *ShuffleGraveyardIntoLibraryData) EffectProps() EffectProperties {
	return EffectProperties{}
}

func execShuffleGraveyardIntoLibrary(ctx *EffectContext, _ *ShuffleGraveyardIntoLibraryData) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return nil
	}
	gy := p.Graveyard()
	lib := p.Library()
	lib = append(lib, gy...)
	p.SetLibrary(lib)
	p.ClearGraveyard()
	p.ShuffleLibrary()
	return nil
}

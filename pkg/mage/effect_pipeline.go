package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ---------------------------------------------------------------------------
// Pipeline: sequence of steps sharing an EffectContext
// ---------------------------------------------------------------------------

// PipelineData executes steps in sequence, sharing an EffectContext so
// intermediate values flow between steps without closures.
type PipelineData struct {
	Steps []Effect
	Txt   string
	Props EffectProperties
}

// Pipeline creates a pipeline effect from a sequence of steps.
func Pipeline(text string, props EffectProperties, steps ...Effect) Effect {
	return &PipelineData{Steps: steps, Txt: text, Props: props}
}

func (e *PipelineData) Text() string                 { return e.Txt }
func (e *PipelineData) Properties() EffectProperties { return e.Props }

func (e *PipelineData) Apply(ctx *EffectContext) error {
	for _, step := range e.Steps {
		if err := step.Apply(ctx); err != nil {
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
func SnapshotPermanent(sel PermanentSelector, storeAs string) Effect {
	return &SnapshotPermanentData{Selector: sel, StoreAs: storeAs}
}

func (e *SnapshotPermanentData) Text() string                 { return "" }
func (e *SnapshotPermanentData) Properties() EffectProperties { return EffectProperties{} }

func (e *SnapshotPermanentData) Apply(ctx *EffectContext) error {
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
// Gathered-permanent operations: act on a variable-bound permanent
// ---------------------------------------------------------------------------

// ExileGatheredData exiles the permanent stored in a context variable.
type ExileGatheredData struct{ VarName string }

func ExileGathered(varName string) Effect { return &ExileGatheredData{VarName: varName} }

func (e *ExileGatheredData) Text() string { return "exile" }
func (e *ExileGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *ExileGatheredData) Apply(ctx *EffectContext) error {
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
	VarName string
	NoRegen bool
}

func DestroyGathered(varName string) Effect {
	return &DestroyGatheredData{VarName: varName}
}
func DestroyGatheredNoRegen(varName string) Effect {
	return &DestroyGatheredData{VarName: varName, NoRegen: true}
}

func (e *DestroyGatheredData) Text() string { return "destroy" }
func (e *DestroyGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *DestroyGatheredData) Apply(ctx *EffectContext) error {
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

func SacrificeGathered(varName string) Effect { return &SacrificeGatheredData{VarName: varName} }

func (e *SacrificeGatheredData) Text() string                 { return "sacrifice" }
func (e *SacrificeGatheredData) Properties() EffectProperties { return EffectProperties{} }

func (e *SacrificeGatheredData) Apply(ctx *EffectContext) error {
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

func BounceGathered(varName string) Effect { return &BounceGatheredData{VarName: varName} }

func (e *BounceGatheredData) Text() string { return "return to hand" }
func (e *BounceGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, IsBounce: true}
}

func (e *BounceGatheredData) Apply(ctx *EffectContext) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm == nil {
		return nil
	}
	isToken := perm.IsToken
	card := perm.Card
	owner := card.Owner()
	if owner == uuid.Nil {
		owner = perm.Controller
	}
	ctx.Game.RemoveFromBattlefield(perm)
	if !isToken {
		p := ctx.Game.GetPlayer(owner)
		if p != nil {
			p.AddToHand(card)
		}
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

func GainLifeFromVar(playerVar, amountVar string) Effect {
	return &GainLifeVarData{PlayerVar: playerVar, AmountVar: amountVar}
}

// GainLifeControllerFromVar gains life for the controller from a context variable amount.
func GainLifeControllerFromVar(amountVar string) Effect {
	return &GainLifeVarData{AmountVar: amountVar}
}

func (e *GainLifeVarData) Text() string { return "gain life" }
func (e *GainLifeVarData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func (e *GainLifeVarData) Apply(ctx *EffectContext) error {
	var playerID uuid.UUID
	if e.PlayerVar == "" {
		playerID = ctx.Controller
	} else {
		playerID = ctx.TryGetUUID(e.PlayerVar)
	}
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

func DealDamageFromVar(amountVar string) Effect {
	return &DealDamageVarData{AmountVar: amountVar}
}

func (e *DealDamageVarData) Text() string { return "deal damage" }
func (e *DealDamageVarData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *DealDamageVarData) Apply(ctx *EffectContext) error {
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

func DealDamageToPlayersFromVar(amountVar string, sel PlayerSelector) Effect {
	return &DealDamageToPlayersVarData{AmountVar: amountVar, Selector: sel}
}

func (e *DealDamageToPlayersVarData) Text() string { return "deal damage to players" }
func (e *DealDamageToPlayersVarData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *DealDamageToPlayersVarData) Apply(ctx *EffectContext) error {
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
	Inner  Effect
	Txt    string
}

// ForEachPermanent creates an effect that iterates matching permanents.
func ForEachPermanent(filter PermanentFilter, inner Effect, text string) Effect {
	return &ForEachPermanentData{Filter: filter, Inner: inner, Txt: text}
}

// ForEachControlledPermanent creates an effect that iterates matching permanents
// controlled by the specified player.
func ForEachControlledPermanent(who PlayerSelector, filter PermanentFilter, inner Effect, text string) Effect {
	return &ForEachPermanentData{Filter: filter, Who: who, Inner: inner, Txt: text}
}

func (e *ForEachPermanentData) Text() string                 { return e.Txt }
func (e *ForEachPermanentData) Properties() EffectProperties { return EffectProperties{Mass: true} }

func (e *ForEachPermanentData) Apply(ctx *EffectContext) error {
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
		if err := e.Inner.Apply(ctx); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// ---------------------------------------------------------------------------
// IfElse: conditional branching
// ---------------------------------------------------------------------------

// IfElseData branches based on a TriggerConditionData predicate evaluated at
// resolution time against the current game state. The predicate receives a
// zero-value event (there is no triggering event mid-resolution), so only
// state predicates belong here — event predicates like EventSourceIsSelf
// always see empty event fields.
type IfElseData struct {
	Cond TriggerConditionData
	Then Effect
	Else Effect // nil = do nothing
	Txt  string
}

// IfElse creates a conditional effect.
func IfElse(text string, cond TriggerConditionData, then, els Effect) Effect {
	return &IfElseData{Cond: cond, Then: then, Else: els, Txt: text}
}

func (e *IfElseData) Text() string                 { return e.Txt }
func (e *IfElseData) Properties() EffectProperties { return EffectProperties{} }

func (e *IfElseData) Apply(ctx *EffectContext) error {
	if e.Cond.CheckTriggerCond(&GameEvent{}, ctx.Game, ctx.SourceID, ctx.Controller) {
		if e.Then != nil {
			return e.Then.Apply(ctx)
		}
		return nil
	}
	if e.Else != nil {
		return e.Else.Apply(ctx)
	}
	return nil
}

// ---------------------------------------------------------------------------
// IfVarGT: pipeline-local branching on a context variable
// ---------------------------------------------------------------------------

// IfVarGTData branches on an integer pipeline variable. Unlike IfElse, this
// is control flow over EffectContext.Vars (values gathered by earlier
// pipeline steps), not a predicate over game state, so it stays outside the
// shared TriggerConditionData vocabulary.
type IfVarGTData struct {
	Name  string
	Value int
	Then  Effect
	Else  Effect // nil = do nothing
	Txt   string
}

// IfVarGT creates an effect that runs then if the named pipeline variable is
// greater than value, otherwise els.
func IfVarGT(text, name string, value int, then, els Effect) Effect {
	return &IfVarGTData{Name: name, Value: value, Then: then, Else: els, Txt: text}
}

func (e *IfVarGTData) Text() string                 { return e.Txt }
func (e *IfVarGTData) Properties() EffectProperties { return EffectProperties{} }

func (e *IfVarGTData) Apply(ctx *EffectContext) error {
	if ctx.GetInt(e.Name) > e.Value {
		if e.Then != nil {
			return e.Then.Apply(ctx)
		}
		return nil
	}
	if e.Else != nil {
		return e.Else.Apply(ctx)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Modal: branch on g.ModeValue()
// ---------------------------------------------------------------------------

// ModalEffectData branches on the chosen mode value (0, 1, ...).
type ModalEffectData struct {
	Modes []Effect
	Txt   string
}

// ModalEffect creates an effect that executes one of several modes.
func ModalEffect(text string, modes ...Effect) Effect {
	return &ModalEffectData{Modes: modes, Txt: text}
}

func (e *ModalEffectData) Text() string                 { return e.Txt }
func (e *ModalEffectData) Properties() EffectProperties { return EffectProperties{} }

func (e *ModalEffectData) Apply(ctx *EffectContext) error {
	mode := ctx.Game.ModeValue()
	if mode < 0 || mode >= len(e.Modes) {
		return fmt.Errorf("invalid mode %d (have %d modes)", mode, len(e.Modes))
	}
	return e.Modes[mode].Apply(ctx)
}

// ---------------------------------------------------------------------------
// ChoosePermanent: player picks from matching permanents
// ---------------------------------------------------------------------------

// ChoosePermanentData asks a player to choose a permanent matching a filter
// and stores the result in a context variable.
type ChoosePermanentData struct {
	Player  PlayerSelector
	Filter  PermanentFilter
	Reason  string
	StoreAs string
}

// ChoosePermanentStep creates a pipeline step where a player chooses a permanent.
func ChoosePermanentStep(player PlayerSelector, filter PermanentFilter, reason, storeAs string) Effect {
	return &ChoosePermanentData{Player: player, Filter: filter, Reason: reason, StoreAs: storeAs}
}

func (e *ChoosePermanentData) Text() string                 { return "" }
func (e *ChoosePermanentData) Properties() EffectProperties { return EffectProperties{} }

func (e *ChoosePermanentData) Apply(ctx *EffectContext) error {
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

func SacrificeSourceStep() Effect { return &SacrificeSourceData{} }

func (e *SacrificeSourceData) Text() string                 { return "sacrifice" }
func (e *SacrificeSourceData) Properties() EffectProperties { return EffectProperties{} }

func (*SacrificeSourceData) Apply(ctx *EffectContext) error {
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

func ShuffleGraveyardIntoLibrary() Effect { return &ShuffleGraveyardIntoLibraryData{} }

func (e *ShuffleGraveyardIntoLibraryData) Text() string { return "shuffle graveyard into library" }
func (e *ShuffleGraveyardIntoLibraryData) Properties() EffectProperties {
	return EffectProperties{}
}

func (*ShuffleGraveyardIntoLibraryData) Apply(ctx *EffectContext) error {
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

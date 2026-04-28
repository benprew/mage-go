package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ---------------------------------------------------------------------------
// SnapshotAttached: read the permanent that source is attached to
// ---------------------------------------------------------------------------

// SnapshotAttachedData reads the permanent the source is attached to and stores
// its properties as context variables (same as SnapshotPermanent).
type SnapshotAttachedData struct {
	StoreAs string
}

func SnapshotAttached(storeAs string) EffectData {
	return &SnapshotAttachedData{StoreAs: storeAs}
}

func (e *SnapshotAttachedData) Text() string                 { return "" }
func (e *SnapshotAttachedData) Properties() EffectProperties { return EffectProperties{} }

func execSnapshotAttached(ctx *EffectContext, e *SnapshotAttachedData) error {
	src := ctx.Game.FindPermanent(ctx.SourceID)
	if src == nil || !src.IsAttached() {
		ctx.SetBool(e.StoreAs+".missing", true)
		return nil
	}
	perm := ctx.Game.FindPermanent(src.AttachedTo)
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
// Gathered-permanent operations: tap, untap, damage, regen, prevention
// ---------------------------------------------------------------------------

// TapGatheredData taps a var-bound permanent.
type TapGatheredData struct{ VarName string }

func TapGathered(v string) EffectData { return &TapGatheredData{VarName: v} }

func (e *TapGatheredData) Text() string { return "tap" }
func (e *TapGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func execTapGathered(ctx *EffectContext, e *TapGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm != nil {
		ctx.Game.TapPermanent(perm)
	}
	return nil
}

// UntapGatheredData untaps a var-bound permanent.
type UntapGatheredData struct{ VarName string }

func UntapGathered(v string) EffectData { return &UntapGatheredData{VarName: v} }

func (e *UntapGatheredData) Text() string { return "untap" }
func (e *UntapGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execUntapGathered(ctx *EffectContext, e *UntapGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm != nil && perm.Tapped {
		perm.Tapped = false
		ctx.Game.FireEvent(GameEvent{Type: EvtBecameUntapped, SourceID: perm.ID()})
	}
	return nil
}

// DealDamageToGatheredData deals damage to a var-bound permanent.
type DealDamageToGatheredData struct {
	VarName string
	Amount  ValueSource
}

func DealDamageToGathered(v string, amount ValueSource) EffectData {
	return &DealDamageToGatheredData{VarName: v, Amount: amount}
}

func (e *DealDamageToGatheredData) Text() string { return "deal damage" }
func (e *DealDamageToGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func execDealDamageToGathered(ctx *EffectContext, e *DealDamageToGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	amount := e.Amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount <= 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm != nil {
		ctx.Game.DealDamageToPermanent(perm, amount, ctx.SourceID)
	}
	return nil
}

// RegenerateGatheredData adds a regeneration shield to a var-bound permanent.
type RegenerateGatheredData struct{ VarName string }

func RegenerateGathered(v string) EffectData { return &RegenerateGatheredData{VarName: v} }

func (e *RegenerateGatheredData) Text() string { return "regenerate" }
func (e *RegenerateGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execRegenerateGathered(ctx *EffectContext, e *RegenerateGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	ctx.Game.AddRegenerationShield(id)
	return nil
}

// PreventDamageToGatheredData adds a prevention shield to a var-bound permanent.
type PreventDamageToGatheredData struct {
	VarName string
	Amount  ValueSource
}

func PreventDamageToGathered(v string, amount ValueSource) EffectData {
	return &PreventDamageToGatheredData{VarName: v, Amount: amount}
}

func (e *PreventDamageToGatheredData) Text() string { return "prevent damage" }
func (e *PreventDamageToGatheredData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execPreventDamageToGathered(ctx *EffectContext, e *PreventDamageToGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	amount := e.Amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	ctx.Game.AddPreventionShield(id, amount)
	return nil
}

// GrantAttrToGatheredData grants an attr to a var-bound permanent.
type GrantAttrToGatheredData struct {
	VarName string
	Attr    Attr
}

func GrantAttrToGathered(v string, a Attr) EffectData {
	return &GrantAttrToGatheredData{VarName: v, Attr: a}
}

func (e *GrantAttrToGatheredData) Text() string                 { return "" }
func (e *GrantAttrToGatheredData) Properties() EffectProperties { return EffectProperties{} }

func execGrantAttrToGathered(ctx *EffectContext, e *GrantAttrToGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(id)
	if perm != nil {
		perm.GrantBaseAttr(e.Attr)
	}
	return nil
}

// ---------------------------------------------------------------------------
// RegisterDelayedTrigger: register a delayed trigger from a pipeline
// ---------------------------------------------------------------------------

// RegisterDelayedTriggerData registers a delayed trigger that fires when the
// specified event occurs.
type RegisterDelayedTriggerData struct {
	EventType     EventType
	TargetVar     string   // read target permanent ID from context (or empty for source)
	Effects       []Effect // effects to execute when trigger fires
	MatchEventVar string   // optional: match evt.SourceID against this var
	Persistent    bool
}

func RegisterDelayedTriggerStep(evtType EventType, targetVar string, effects ...Effect) EffectData {
	return &RegisterDelayedTriggerData{EventType: evtType, TargetVar: targetVar, Effects: effects}
}

func RegisterPersistentDelayedTriggerStep(evtType EventType, targetVar string, effects ...Effect) EffectData {
	return &RegisterDelayedTriggerData{EventType: evtType, TargetVar: targetVar, Effects: effects, Persistent: true}
}

func (e *RegisterDelayedTriggerData) Text() string                 { return "" }
func (e *RegisterDelayedTriggerData) Properties() EffectProperties { return EffectProperties{} }

func execRegisterDelayedTrigger(ctx *EffectContext, e *RegisterDelayedTriggerData) error {
	targetID := ctx.SourceID
	if e.TargetVar == "_target" {
		if len(ctx.Targets) > 0 {
			targetID = ctx.Targets[0]
		}
	} else if e.TargetVar != "" {
		id := ctx.TryGetUUID(e.TargetVar)
		if id != uuid.Nil {
			targetID = id
		}
	}
	dt := &DelayedTrigger{
		EventType:  e.EventType,
		TargetID:   targetID,
		Effects:    e.Effects,
		SourceID:   ctx.SourceID,
		Controller: ctx.Controller,
		Persistent: e.Persistent,
	}
	if e.MatchEventVar != "" {
		dt.MatchEventID = ctx.TryGetUUID(e.MatchEventVar)
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}

// ---------------------------------------------------------------------------
// FlipCoin condition
// ---------------------------------------------------------------------------

// FlipCoinCond flips a coin; true = heads (win).
type FlipCoinCond struct{}

func (c *FlipCoinCond) Check(ctx *EffectContext) bool {
	return ctx.Game.FlipCoin(ctx.Controller)
}

// ---------------------------------------------------------------------------
// GrantKeyword / RevokeKeyword until end of turn (as pipeline steps)
// ---------------------------------------------------------------------------

// RevokeKeywordFromTargetUntilEOTData removes a keyword from targets[0] until EOT.
type RevokeKeywordFromTargetUntilEOTData struct {
	Keyword Keyword
}

func RevokeKeywordFromTargetUntilEOT(kw Keyword) EffectData {
	return &RevokeKeywordFromTargetUntilEOTData{Keyword: kw}
}

func (e *RevokeKeywordFromTargetUntilEOTData) Text() string { return "" }
func (e *RevokeKeywordFromTargetUntilEOTData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func execRevokeKeywordFromTargetUntilEOT(ctx *EffectContext, e *RevokeKeywordFromTargetUntilEOTData) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm == nil {
		return nil
	}
	eff := FuncContinuousEffect(LayerAbility, EndOfTurn, func(g *Game, _ uuid.UUID) error {
		p := g.FindPermanent(perm.ID())
		if p != nil {
			g.RevokeAttr(p.ID(), e.Keyword)
		}
		return nil
	})
	eff.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(eff)
	ctx.Game.ApplyContinuousEffects()
	return nil
}

// ---------------------------------------------------------------------------
// AddMana from a context variable
// ---------------------------------------------------------------------------

// AddManaFromVarData adds mana of the given color, amount from a context var.
type AddManaFromVarData struct {
	Color     Color
	AmountVar string
}

func AddManaFromVar(color Color, amountVar string) EffectData {
	return &AddManaFromVarData{Color: color, AmountVar: amountVar}
}

func (e *AddManaFromVarData) Text() string                 { return "add mana" }
func (e *AddManaFromVarData) Properties() EffectProperties { return EffectProperties{} }

func execAddManaFromVar(ctx *EffectContext, e *AddManaFromVarData) error {
	amount := ctx.GetInt(e.AmountVar)
	if amount <= 0 {
		return nil
	}
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return nil
	}
	p.ManaPool().Add(e.Color, amount)
	return nil
}

// ---------------------------------------------------------------------------
// Prevent all damage to target (for one turn)
// ---------------------------------------------------------------------------

// PreventAllDamageFromSourceData prevents all damage from the source for this turn.
type PreventAllDamageFromSourceData struct{}

func PreventAllDamageFromSource() EffectData { return &PreventAllDamageFromSourceData{} }

func (e *PreventAllDamageFromSourceData) Text() string { return "prevent all damage from source" }
func (e *PreventAllDamageFromSourceData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execPreventAllDamageFromSource(ctx *EffectContext, _ *PreventAllDamageFromSourceData) error {
	ctx.Game.PreventAllDamageFrom(ctx.SourceID)
	return nil
}

// ---------------------------------------------------------------------------
// ColorPrevention step (Circle of Protection pattern)
// ---------------------------------------------------------------------------

// AddColorPreventionData prevents all damage from one source of the given color.
type AddColorPreventionData struct {
	Color Color
}

func AddColorPreventionStep(color Color) EffectData {
	return &AddColorPreventionData{Color: color}
}

func (e *AddColorPreventionData) Text() string { return "prevent damage from color" }
func (e *AddColorPreventionData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execAddColorPrevention(ctx *EffectContext, e *AddColorPreventionData) error {
	ctx.Game.AddColorPrevention(ctx.Controller, e.Color)
	return nil
}

// ---------------------------------------------------------------------------
// AddPreventionShieldToControllerData adds a damage prevention shield to the controller.
type AddPreventionShieldToControllerData struct {
	Amount int
}

func AddPreventionShieldToControllerStep(amount int) EffectData {
	return &AddPreventionShieldToControllerData{Amount: amount}
}

func (e *AddPreventionShieldToControllerData) Text() string {
	return "prevent damage to you"
}
func (e *AddPreventionShieldToControllerData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execAddPreventionShieldToController(ctx *EffectContext, e *AddPreventionShieldToControllerData) error {
	ctx.Game.AddPreventionShield(ctx.Controller, e.Amount)
	return nil
}

// ReverseDamage step
// ---------------------------------------------------------------------------

type AddReverseDamageShieldData struct{}

func AddReverseDamageShieldStep() EffectData { return &AddReverseDamageShieldData{} }

func (e *AddReverseDamageShieldData) Text() string { return "reverse damage" }
func (e *AddReverseDamageShieldData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execAddReverseDamageShield(ctx *EffectContext, _ *AddReverseDamageShieldData) error {
	ctx.Game.AddReverseDamageShield(ctx.Controller)
	return nil
}

// ---------------------------------------------------------------------------
// SetVar: store a computed int value in context
// ---------------------------------------------------------------------------

// SetVarFromHandSizeData stores hand size (minus offset) in a context variable.
type SetVarFromHandSizeData struct {
	Player  PlayerSelector
	StoreAs string
	Offset  int // result = max(0, handSize - offset)
}

func SetVarFromHandSize(player PlayerSelector, storeAs string, offset int) EffectData {
	return &SetVarFromHandSizeData{Player: player, StoreAs: storeAs, Offset: offset}
}

func (e *SetVarFromHandSizeData) Text() string                 { return "" }
func (e *SetVarFromHandSizeData) Properties() EffectProperties { return EffectProperties{} }

func execSetVarFromHandSize(ctx *EffectContext, e *SetVarFromHandSizeData) error {
	playerIDs := e.Player.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if len(playerIDs) == 0 {
		ctx.SetInt(e.StoreAs, 0)
		return nil
	}
	p := ctx.Game.GetPlayer(playerIDs[0])
	if p == nil {
		ctx.SetInt(e.StoreAs, 0)
		return nil
	}
	val := max(len(p.Hand())-e.Offset, 0)
	ctx.SetInt(e.StoreAs, val)
	return nil
}

// ---------------------------------------------------------------------------
// ChooseColor step (stores on source permanent)
// ---------------------------------------------------------------------------

// ChooseColorStepData asks the controller to choose a color and stores it on
// the source permanent's ChosenColor field.
type ChooseColorStepData struct {
	Reason string
}

func ChooseColorStep(reason string) EffectData {
	return &ChooseColorStepData{Reason: reason}
}

func (e *ChooseColorStepData) Text() string                 { return "choose a color" }
func (e *ChooseColorStepData) Properties() EffectProperties { return EffectProperties{} }

func execChooseColorStep(ctx *EffectContext, e *ChooseColorStepData) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return nil
	}
	perm := ctx.Game.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	perm.ChosenColor = p.ChooseManaColor(e.Reason)
	return nil
}

// ---------------------------------------------------------------------------
// RemoveFromCombatGathered
// ---------------------------------------------------------------------------

type RemoveFromCombatGatheredData struct{ VarName string }

func RemoveFromCombatGathered(v string) EffectData {
	return &RemoveFromCombatGatheredData{VarName: v}
}

func (e *RemoveFromCombatGatheredData) Text() string                 { return "remove from combat" }
func (e *RemoveFromCombatGatheredData) Properties() EffectProperties { return EffectProperties{} }

func execRemoveFromCombatGathered(ctx *EffectContext, e *RemoveFromCombatGatheredData) error {
	id := ctx.TryGetUUID(e.VarName)
	if id == uuid.Nil {
		return nil
	}
	ctx.Game.RemoveFromCombat(id)
	return nil
}

// ---------------------------------------------------------------------------
// Attached permanent operations
// ---------------------------------------------------------------------------

// DestroyAttachedData destroys the permanent the source is attached to.
type DestroyAttachedData struct{}

func DestroyAttachedStep() EffectData { return &DestroyAttachedData{} }

func (e *DestroyAttachedData) Text() string { return "destroy enchanted permanent" }
func (e *DestroyAttachedData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func execDestroyAttached(ctx *EffectContext, _ *DestroyAttachedData) error {
	src := ctx.Game.FindPermanent(ctx.SourceID)
	if src == nil || src.AttachedTo == uuid.Nil {
		return nil
	}
	target := ctx.Game.FindPermanent(src.AttachedTo)
	if target != nil {
		ctx.Game.DestroyPermanent(target)
	}
	return nil
}

// GrantAttrToAttachedData grants an attribute/keyword to the attached permanent permanently.
type GrantAttrToAttachedData struct {
	Attr Keyword
}

func GrantAttrToAttachedStep(attr Keyword) EffectData {
	return &GrantAttrToAttachedData{Attr: attr}
}

func (e *GrantAttrToAttachedData) Text() string { return "grant keyword to enchanted permanent" }
func (e *GrantAttrToAttachedData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func execGrantAttrToAttached(ctx *EffectContext, e *GrantAttrToAttachedData) error {
	src := ctx.Game.FindPermanent(ctx.SourceID)
	if src == nil || src.AttachedTo == uuid.Nil {
		return nil
	}
	target := ctx.Game.FindPermanent(src.AttachedTo)
	if target != nil {
		target.GrantBaseAttr(e.Attr)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Deal damage to source
// ---------------------------------------------------------------------------

// DealDamageToSourceData deals damage to the source permanent itself.
type DealDamageToSourceData struct {
	Amount ValueSource
}

func DealDamageToSourceStep(amount ValueSource) EffectData {
	return &DealDamageToSourceData{Amount: amount}
}

func (e *DealDamageToSourceData) Text() string { return "deal damage to self" }
func (e *DealDamageToSourceData) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func execDealDamageToSource(ctx *EffectContext, e *DealDamageToSourceData) error {
	self := ctx.Game.FindPermanent(ctx.SourceID)
	if self == nil {
		return nil
	}
	amount := e.Amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if amount > 0 {
		ctx.Game.DealDamageToPermanent(self, amount, ctx.SourceID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Combat group iteration primitives
// ---------------------------------------------------------------------------

// ForEachBlockerOfSourceData iterates all creatures blocking the source
// attacker. Each blocker's ID is set as targets[0] for the inner effect.
// If Filter is set, only blockers matching the filter are included.
type ForEachBlockerOfSourceData struct {
	Filter PermanentFilter // optional: filter blockers
	Inner  EffectData
	Txt    string
}

func ForEachBlockerOfSource(inner EffectData, text string) EffectData {
	return &ForEachBlockerOfSourceData{Inner: inner, Txt: text}
}

func ForEachBlockerOfSourceMatching(filter PermanentFilter, inner EffectData, text string) EffectData {
	return &ForEachBlockerOfSourceData{Filter: filter, Inner: inner, Txt: text}
}

func (e *ForEachBlockerOfSourceData) Text() string                 { return e.Txt }
func (e *ForEachBlockerOfSourceData) Properties() EffectProperties { return EffectProperties{} }

func execForEachBlockerOfSource(ctx *EffectContext, e *ForEachBlockerOfSourceData) error {
	group := ctx.Game.CombatGroupFor(ctx.SourceID)
	if group == nil {
		return nil
	}
	var blockerIDs []uuid.UUID
	for _, bid := range group.BlockerIDs {
		if e.Filter.IsZero() {
			blockerIDs = append(blockerIDs, bid)
		} else {
			perm := ctx.Game.FindPermanent(bid)
			if perm != nil && e.Filter.Match(perm, ctx.Game) {
				blockerIDs = append(blockerIDs, bid)
			}
		}
	}
	savedTargets := ctx.Targets
	for _, bid := range blockerIDs {
		ctx.Targets = []uuid.UUID{bid}
		if err := ExecuteEffect(ctx, e.Inner); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// ForEachAttackerBlockedBySourceData iterates all attackers that the source
// creature is blocking. Each attacker's ID is set as targets[0].
type ForEachAttackerBlockedBySourceData struct {
	Inner EffectData
	Txt   string
}

func ForEachAttackerBlockedBySource(inner EffectData, text string) EffectData {
	return &ForEachAttackerBlockedBySourceData{Inner: inner, Txt: text}
}

func (e *ForEachAttackerBlockedBySourceData) Text() string                 { return e.Txt }
func (e *ForEachAttackerBlockedBySourceData) Properties() EffectProperties { return EffectProperties{} }

func execForEachAttackerBlockedBySource(ctx *EffectContext, e *ForEachAttackerBlockedBySourceData) error {
	var attackerIDs []uuid.UUID
	for _, group := range ctx.Game.CombatGroups() {
		if slices.Contains(group.BlockerIDs, ctx.SourceID) {
			attackerIDs = append(attackerIDs, group.AttackerID)
		}
	}
	savedTargets := ctx.Targets
	for _, aid := range attackerIDs {
		ctx.Targets = []uuid.UUID{aid}
		if err := ExecuteEffect(ctx, e.Inner); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// ForEachCombatOpponentData iterates all creatures in combat with the source
// (blockers if source is attacking, attacker if source is blocking).
// Each opponent's ID is set as targets[0].
type ForEachCombatOpponentData struct {
	Inner EffectData
	Txt   string
}

func ForEachCombatOpponent(inner EffectData, text string) EffectData {
	return &ForEachCombatOpponentData{Inner: inner, Txt: text}
}

func (e *ForEachCombatOpponentData) Text() string                 { return e.Txt }
func (e *ForEachCombatOpponentData) Properties() EffectProperties { return EffectProperties{} }

func execForEachCombatOpponent(ctx *EffectContext, e *ForEachCombatOpponentData) error {
	var opponentIDs []uuid.UUID
	for _, group := range ctx.Game.CombatGroups() {
		if group.AttackerID == ctx.SourceID {
			opponentIDs = append(opponentIDs, group.BlockerIDs...)
		}
		for _, bid := range group.BlockerIDs {
			if bid == ctx.SourceID {
				opponentIDs = append(opponentIDs, group.AttackerID)
			}
		}
	}
	savedTargets := ctx.Targets
	for _, id := range opponentIDs {
		ctx.Targets = []uuid.UUID{id}
		if err := ExecuteEffect(ctx, e.Inner); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// ForEachBlockerOfTargetData iterates all creatures blocking a target attacker
// (targets[0]). Each blocker's ID replaces targets[0] for the inner effect.
// Used by spells that target an attacker and affect its blockers (e.g. Feint).
type ForEachBlockerOfTargetData struct {
	Inner EffectData
	Txt   string
}

func ForEachBlockerOfTarget(inner EffectData, text string) EffectData {
	return &ForEachBlockerOfTargetData{Inner: inner, Txt: text}
}

func (e *ForEachBlockerOfTargetData) Text() string                 { return e.Txt }
func (e *ForEachBlockerOfTargetData) Properties() EffectProperties { return EffectProperties{} }

func execForEachBlockerOfTarget(ctx *EffectContext, e *ForEachBlockerOfTargetData) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	attackerID := ctx.Targets[0]
	group := ctx.Game.CombatGroupFor(attackerID)
	if group == nil {
		return nil
	}
	blockerIDs := make([]uuid.UUID, len(group.BlockerIDs))
	copy(blockerIDs, group.BlockerIDs)
	savedTargets := ctx.Targets
	for _, bid := range blockerIDs {
		ctx.Targets = []uuid.UUID{bid}
		if err := ExecuteEffect(ctx, e.Inner); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// ForEachAttackerBlockedByTargetData iterates all attackers that a target
// creature (targets[0]) is blocking. Each attacker's ID replaces targets[0].
// Used by spells that target a Wall and affect creatures it blocked (Glyph of Doom).
type ForEachAttackerBlockedByTargetData struct {
	Inner EffectData
	Txt   string
}

func ForEachAttackerBlockedByTarget(inner EffectData, text string) EffectData {
	return &ForEachAttackerBlockedByTargetData{Inner: inner, Txt: text}
}

func (e *ForEachAttackerBlockedByTargetData) Text() string                 { return e.Txt }
func (e *ForEachAttackerBlockedByTargetData) Properties() EffectProperties { return EffectProperties{} }

func execForEachAttackerBlockedByTarget(ctx *EffectContext, e *ForEachAttackerBlockedByTargetData) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	blockerID := ctx.Targets[0]
	var attackerIDs []uuid.UUID
	for _, group := range ctx.Game.CombatGroups() {
		if slices.Contains(group.BlockerIDs, blockerID) {
			attackerIDs = append(attackerIDs, group.AttackerID)
		}
	}
	savedTargets := ctx.Targets
	for _, aid := range attackerIDs {
		ctx.Targets = []uuid.UUID{aid}
		if err := ExecuteEffect(ctx, e.Inner); err != nil {
			return err
		}
	}
	ctx.Targets = savedTargets
	return nil
}

// BlockerCountVarData stores the number of blockers of the source attacker
// into a context variable. Used for Rampage calculations.
type BlockerCountVarData struct {
	StoreAs string
}

func BlockerCountVar(storeAs string) EffectData {
	return &BlockerCountVarData{StoreAs: storeAs}
}

func (e *BlockerCountVarData) Text() string                 { return "count blockers" }
func (e *BlockerCountVarData) Properties() EffectProperties { return EffectProperties{} }

func execBlockerCountVar(ctx *EffectContext, e *BlockerCountVarData) error {
	group := ctx.Game.CombatGroupFor(ctx.SourceID)
	if group == nil {
		ctx.SetInt(e.StoreAs, 0)
		return nil
	}
	ctx.SetInt(e.StoreAs, len(group.BlockerIDs))
	return nil
}

// RampageEffectData implements Rampage N: +N/+N for each blocker beyond the first.
type RampageEffectData struct {
	N int
}

func RampageEffect(n int) EffectData {
	return &RampageEffectData{N: n}
}

func (e *RampageEffectData) Text() string {
	return fmt.Sprintf("rampage %d", e.N)
}
func (e *RampageEffectData) Properties() EffectProperties { return EffectProperties{} }

func execRampageEffect(ctx *EffectContext, e *RampageEffectData) error {
	group := ctx.Game.CombatGroupFor(ctx.SourceID)
	if group == nil || len(group.BlockerIDs) <= 1 {
		return nil
	}
	bonus := e.N * (len(group.BlockerIDs) - 1)
	ce := TemporaryBoost(ctx.SourceID, bonus, bonus)
	ce.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(ce)
	return nil
}

// TargetHasRampageCond is true when targets[0] already has a Rampage triggered
// ability (used by Rapid Fire's "If it doesn't have rampage" clause).
type TargetHasRampageCond struct{}

func (c *TargetHasRampageCond) Check(ctx *EffectContext) bool {
	if len(ctx.Targets) == 0 {
		return false
	}
	p := ctx.Game.FindPermanent(ctx.Targets[0])
	if p == nil {
		return false
	}
	for _, a := range p.RuntimeAbilities {
		gt, ok := UnwrapAbility(a).(*GenericTriggered)
		if !ok {
			continue
		}
		for _, e := range gt.Effects() {
			if _, ok := e.(*RampageEffectData); ok {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// AddContinuousEffects: add continuous effects to the game
// ---------------------------------------------------------------------------

// AddContinuousEffectsData adds continuous effects created by a factory function.
// The factory is called each time the step executes, producing fresh effect
// instances (effects contain mutable state like sourceID).
type AddContinuousEffectsData struct {
	Factory func() []ContinuousEffect
}

// AddContinuousEffectsStep creates a pipeline step that adds continuous effects.
func AddContinuousEffectsStep(factory func() []ContinuousEffect) EffectData {
	return &AddContinuousEffectsData{Factory: factory}
}

func (e *AddContinuousEffectsData) Text() string                 { return "add continuous effects" }
func (e *AddContinuousEffectsData) Properties() EffectProperties { return EffectProperties{} }

func execAddContinuousEffects(ctx *EffectContext, e *AddContinuousEffectsData) error {
	for _, eff := range e.Factory() {
		eff.SetSourceID(ctx.SourceID)
		ctx.Game.AddContinuousEffect(eff)
	}
	return nil
}

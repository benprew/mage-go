package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ExecuteEffect dispatches an EffectData value to the appropriate execution
// logic. This is the central interpreter: effect data describes "what" should
// happen, and this function decides "how."
//
// New EffectData types are added as cases here. During migration, this starts
// small and grows as pre-built effects and pipeline primitives are converted.
func ExecuteEffect(ctx *EffectContext, data EffectData) error {
	switch e := data.(type) {

	// --- Damage effects (effect_damage.go) ---

	case *gainLifeEffect:
		return execGainLife(ctx, e)
	case *gainLifeTargetEffect:
		return execGainLifeTarget(ctx, e)
	case *loseLifeEffect:
		return execLoseLife(ctx, e)
	case *dealDamageEffect:
		return execDealDamage(ctx, e)
	case *dealDamageToAllCreaturesEffect:
		return execDealDamageToAllCreatures(ctx, e)
	case *dealDamageToPlayersEffect:
		return execDealDamageToPlayers(ctx, e)
	case *handSizeDamageEffect:
		return execHandSizeDamage(ctx, e)
	case *preventAllCombatDamageEffect:
		return execPreventAllCombatDamage(ctx, e)
	case *preventDamageToTargetEffect:
		return execPreventDamageToTarget(ctx, e)
	case *sacrificeOrDamageEffect:
		return execSacrificeOrDamage(ctx, e)

	// --- Spell effects (effect_spells.go) ---

	case *counterSpellEffect:
		return execCounterSpell(ctx, e)
	case *counterSpellIfColorEffect:
		return execCounterSpellIfColor(ctx, e)
	case *counterSpellIfXMeetsOrExceedsCMCEffect:
		return execCounterSpellIfXMeetsCMC(ctx, e)
	case *powerSinkEffect:
		return execPowerSink(ctx, e)
	case *addManaEffect:
		return execAddMana(ctx, e)
	case *addAnyManaEffect:
		return execAddAnyMana(ctx, e)
	case *createTokenEffect:
		return execCreateToken(ctx, e)
	case *cloneTargetEffect:
		return execCloneTarget(ctx, e)
	case *copySpellOnStackEffect:
		return execCopySpellOnStack(ctx, e)
	case *attachToTargetEffect:
		return execAttachToTarget(ctx, e)
	case *controlChangeTargetEffect:
		return execControlChangeTarget(ctx, e)
	case *extraTurnEffect:
		return execExtraTurn(ctx, e)
	case *changeColorEffect:
		return execChangeColor(ctx, e)
	case *counterUnlessPayEffect:
		return execCounterUnlessPay(ctx, e)
	case *forcefieldEffect:
		return execForcefield(ctx, e)

	// --- Removal effects (effect_removal.go) ---

	case *destroyTargetEffect:
		return execDestroyTarget(ctx, e)
	case *destroyTargetPermanentEffect:
		return execDestroyTargetPermanent(ctx, e)
	case *destroyAllMatchingEffect:
		return execDestroyAllMatching(ctx, e)
	case *destroyAllMatchingNoRegenEffect:
		return execDestroyAllMatchingNoRegen(ctx, e)
	case *destroyTargetNoRegenEffect:
		return execDestroyTargetNoRegen(ctx, e)
	case *exileTargetEffect:
		return execExileTarget(ctx, e)
	case *sacrificeSourceEffect:
		return execSacrificeSource(ctx, e)
	case *balanceEffect:
		return execBalance(ctx, e)
	case *chaosOrbEffect:
		return execChaosOrb(ctx, e)

	// --- Card effects (effect_cards.go) ---

	case *drawCardsTargetEffect:
		return execDrawCardsTarget(ctx, e)
	case *drawCardsActivePlayerEffect:
		return execDrawCardsActivePlayer(ctx, e)
	case *discardCardsEffect:
		return execDiscardCards(ctx, e)
	case *discardRandomEffect:
		return execDiscardRandom(ctx, e)
	case *returnFromGraveyardEffect:
		return execReturnFromGraveyardToBattlefield(ctx, e)
	case *returnSourceToHandEffect:
		return execReturnSourceToHand(ctx, e)
	case *returnToHandTargetEffect:
		return execReturnToHandTarget(ctx, e)
	case *returnFromGraveyardToHandTargetEffect:
		return execReturnFromGraveyardToHandTarget(ctx, e)
	case *searchLibraryEffect:
		return execSearchLibraryToHand(ctx, e)
	case *searchLibraryToTopEffect:
		return execSearchLibraryToTop(ctx, e)
	case *discardHandAndDrawEffect:
		return execDiscardHandAndDraw(ctx, e)
	case *shuffleHandAndGraveyardIntoLibraryAndDrawEffect:
		return execShuffleHandAndGraveyardIntoLibraryAndDraw(ctx, e)
	case *shuffleLibraryEffect:
		return execShuffleLibrary(ctx, e)
	case *putFromHandOntoBattlefieldEffect:
		return execPutFromHandOntoBattlefield(ctx, e)
	case *searchLibraryToBattlefieldEffect:
		return execSearchLibraryToBattlefield(ctx, e)
	case *chooseColorEffect:
		return execChooseColor(ctx, e)

	// --- Combat effects (effect_combat.go) ---

	case *addCountersEffect:
		return execAddCounters(ctx, e)
	case *removeCountersFromSourceEffect:
		return execRemoveCountersFromSource(ctx, e)
	case *addCountersUpToMaxEffect:
		return execAddCountersUpToMax(ctx, e)
	case *tapTargetEffect:
		return execTapTarget(ctx, e)
	case *untapTargetEffect:
		return execUntapTarget(ctx, e)
	case *untapSourceEffect:
		return execUntapSource(ctx, e)
	case *tapAttachedCreatureEffect:
		return execTapAttachedCreature(ctx, e)
	case *tapOrUntapTargetEffect:
		return execTapOrUntapTarget(ctx, e)
	case *tapAllLandsEffect:
		return execTapAllLands(ctx, e)
	case *removeFromCombatEffect:
		return execRemoveFromCombat(ctx, e)
	case *makeUnblockableUntilEndOfTurnEffect:
		return execMakeUnblockableUntilEndOfTurn(ctx, e)
	case *boostUntilEndOfTurnEffect:
		return execBoostUntilEndOfTurn(ctx, e)
	case *boostMatchingUntilEndOfTurnEffect:
		return execBoostMatchingUntilEndOfTurn(ctx, e)
	case *boostAllMatchingUntilEndOfTurnEffect:
		return execBoostAllMatchingUntilEndOfTurn(ctx, e)
	case *doubleSourcePowerEffect:
		return execDoubleTargetPower(ctx, e)
	case *grantKeywordUntilEndOfTurnEffect:
		return execGrantKeywordUntilEndOfTurn(ctx, e)
	case *replaceKeywordEffect:
		return execReplaceKeyword(ctx, e)
	case *regenerateSourceEffect:
		return execRegenerateSource(ctx, e)
	case *regenerateTargetEffect:
		return execRegenerateTarget(ctx, e)
	case *markDestroyAtEOTAfterNActivationsEffect:
		return execMarkDestroyAtEOTAfterNActivations(ctx, e)
	case *destroyTargetAtEndOfTurnEffect:
		return execDestroyTargetAtEndOfTurn(ctx, e)
	case *setBasePTUntilEndOfTurnEffect:
		return execSetPTUntilEndOfTurn(ctx, e)
	case *setBasePowerUntilEndOfTurnEffect:
		return execSetPowerUntilEndOfTurn(ctx, e)

	// --- Pipeline primitives (effect_pipeline.go) ---

	case *PipelineData:
		return execPipeline(ctx, e)
	case *SnapshotPermanentData:
		return execSnapshotPermanent(ctx, e)
	case *SnapshotSourceCounterData:
		return execSnapshotSourceCounter(ctx, e)
	case *ExileGatheredData:
		return execExileGathered(ctx, e)
	case *DestroyGatheredData:
		return execDestroyGathered(ctx, e)
	case *SacrificeGatheredData:
		return execSacrificeGathered(ctx, e)
	case *BounceGatheredData:
		return execBounceGathered(ctx, e)
	case *GainLifeVarData:
		return execGainLifeVar(ctx, e)
	case *DealDamageVarData:
		return execDealDamageVar(ctx, e)
	case *DealDamageToPlayersVarData:
		return execDealDamageToPlayersVar(ctx, e)
	case *ForEachPermanentData:
		return execForEachPermanent(ctx, e)
	case *IfElseData:
		return execIfElse(ctx, e)
	case *ModalEffectData:
		return execModalEffect(ctx, e)
	case *ChoosePermanentData:
		return execChoosePermanent(ctx, e)
	case *SacrificeSourceData:
		return execSacrificeSourceStep(ctx, e)
	case *ShuffleGraveyardIntoLibraryData:
		return execShuffleGraveyardIntoLibrary(ctx, e)

	// --- Extended pipeline primitives (effect_pipeline_ext.go) ---

	case *SnapshotAttachedData:
		return execSnapshotAttached(ctx, e)
	case *TapGatheredData:
		return execTapGathered(ctx, e)
	case *UntapGatheredData:
		return execUntapGathered(ctx, e)
	case *DealDamageToGatheredData:
		return execDealDamageToGathered(ctx, e)
	case *RegenerateGatheredData:
		return execRegenerateGathered(ctx, e)
	case *PreventDamageToGatheredData:
		return execPreventDamageToGathered(ctx, e)
	case *GrantAttrToGatheredData:
		return execGrantAttrToGathered(ctx, e)
	case *AddCountersToGatheredData:
		return execAddCountersToGathered(ctx, e)
	case *RegisterDelayedTriggerData:
		return execRegisterDelayedTrigger(ctx, e)
	case *GrantKeywordToTargetUntilEOTData:
		return execGrantKeywordToTargetUntilEOT(ctx, e)
	case *GrantKeywordToSourceUntilEOTData:
		return execGrantKeywordToSourceUntilEOT(ctx, e)
	case *RevokeKeywordFromTargetUntilEOTData:
		return execRevokeKeywordFromTargetUntilEOT(ctx, e)
	case *AddManaFromVarData:
		return execAddManaFromVar(ctx, e)
	case *BoostGatheredUntilEOTData:
		return execBoostGatheredUntilEOT(ctx, e)
	case *PreventAllDamageFromSourceData:
		return execPreventAllDamageFromSource(ctx, e)
	case *AddColorPreventionData:
		return execAddColorPrevention(ctx, e)
	case *AddReverseDamageShieldData:
		return execAddReverseDamageShield(ctx, e)
	case *SetVarFromHandSizeData:
		return execSetVarFromHandSize(ctx, e)
	case *ChooseColorStepData:
		return execChooseColorStep(ctx, e)
	case *RemoveFromCombatGatheredData:
		return execRemoveFromCombatGathered(ctx, e)

	// Combat group iteration
	case *ForEachBlockerOfSourceData:
		return execForEachBlockerOfSource(ctx, e)
	case *ForEachAttackerBlockedBySourceData:
		return execForEachAttackerBlockedBySource(ctx, e)
	case *ForEachCombatOpponentData:
		return execForEachCombatOpponent(ctx, e)
	case *ForEachBlockerOfTargetData:
		return execForEachBlockerOfTarget(ctx, e)
	case *ForEachAttackerBlockedByTargetData:
		return execForEachAttackerBlockedByTarget(ctx, e)
	case *BlockerCountVarData:
		return execBlockerCountVar(ctx, e)
	case *RampageEffectData:
		return execRampageEffect(ctx, e)

	default:
		_ = e
		return fmt.Errorf("executor: unhandled EffectData type %T", data)
	}
}

// --- Damage effect executors ---

func execGainLife(ctx *EffectContext, e *gainLifeEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	ctx.Game.PlayerGainLife(p, e.amount)
	if !ctx.Game.IsLichActive(ctx.Controller) {
		ctx.Game.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: ctx.Controller, Amount: e.amount})
	}
	return nil
}

func execGainLifeTarget(ctx *EffectContext, e *gainLifeTargetEffect) error {
	var targetPlayer Player
	if len(ctx.Targets) > 0 {
		targetPlayer = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if targetPlayer == nil {
		targetPlayer = ctx.Game.GetPlayer(ctx.Controller)
	}
	if targetPlayer == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller)
	ctx.Game.PlayerGainLife(targetPlayer, amount)
	if !ctx.Game.IsLichActive(targetPlayer.PlayerID()) {
		ctx.Game.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: targetPlayer.PlayerID(), Amount: amount})
	}
	return nil
}

func execLoseLife(ctx *EffectContext, e *loseLifeEffect) error {
	p := ctx.Game.GetPlayer(ctx.Controller)
	if p == nil {
		return ErrPlayerNotFound
	}
	p.LoseLife(e.amount)
	ctx.Game.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: ctx.Controller, Amount: e.amount})
	return nil
}

func execDealDamage(ctx *EffectContext, e *dealDamageEffect) error {
	if len(ctx.Targets) == 0 {
		return fmt.Errorf("no target for damage")
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller)
	if amount <= 0 {
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
	if perm == nil {
		return nil
	}

	sourceCard := ctx.Game.FindCardAnywhere(ctx.SourceID)
	if sourceCard != nil && perm.HasProtectionFrom(sourceCard) {
		return nil
	}

	ctx.Game.DealDamageToPermanent(perm, amount, ctx.SourceID)
	return nil
}

func execDealDamageToAllCreatures(ctx *EffectContext, e *dealDamageToAllCreaturesEffect) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller)
	if amount <= 0 {
		return nil
	}
	f := IsCreature
	if !e.filter.IsZero() {
		f = And(IsCreature, e.filter)
	}
	for _, p := range ctx.Game.FilterBattlefield(f) {
		ctx.Game.DealDamageToPermanent(p, amount, ctx.SourceID)
	}
	return nil
}

func execDealDamageToPlayers(ctx *EffectContext, e *dealDamageToPlayersEffect) error {
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller)
	if amount <= 0 {
		return nil
	}
	playerIDs := e.selector.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	for _, pid := range playerIDs {
		p := ctx.Game.GetPlayer(pid)
		if p != nil {
			ctx.Game.DealDamageToPlayer(p, amount, ctx.SourceID)
		}
	}
	return nil
}

func execHandSizeDamage(ctx *EffectContext, e *handSizeDamageEffect) error {
	var p Player
	if len(ctx.Targets) > 0 {
		p = ctx.Game.GetPlayer(ctx.Targets[0])
	}
	if p == nil {
		p = ctx.Game.ActivePlayerObj()
	}
	handSize := len(p.Hand())
	var damage int
	if e.above {
		damage = handSize - e.threshold
	} else {
		damage = e.threshold - handSize
	}
	if damage > 0 {
		ctx.Game.DealDamageToPlayer(p, damage, ctx.SourceID)
	}
	return nil
}

func execPreventAllCombatDamage(ctx *EffectContext, _ *preventAllCombatDamageEffect) error {
	ctx.Game.SetPreventCombatDamage()
	return nil
}

func execPreventDamageToTarget(ctx *EffectContext, e *preventDamageToTargetEffect) error {
	if len(ctx.Targets) == 0 {
		return nil
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller)
	perm := ctx.Game.FindPermanent(ctx.Targets[0])
	if perm != nil {
		ctx.Game.AddPreventionShield(perm.ID(), amount)
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Targets[0])
	if player != nil {
		ctx.Game.AddPreventionShield(player.PlayerID(), amount)
	}
	return nil
}

func execSacrificeOrDamage(ctx *EffectContext, e *sacrificeOrDamageEffect) error {
	candidates := ctx.Game.FilterBattlefield(And(ControlledBy(ctx.Controller), IsCreature, NotID(ctx.SourceID)))
	if len(candidates) > 0 {
		player := ctx.Game.GetPlayer(ctx.Controller)
		chosen := player.ChoosePermanent(candidates, "sacrifice", ctx.Game)
		if chosen != nil {
			ctx.Game.Sacrifice(chosen)
			return nil
		}
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player != nil {
		ctx.Game.DealDamageToPlayer(player, e.damage, ctx.SourceID)
	}
	return nil
}

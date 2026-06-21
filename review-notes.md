// The review is complete: 13 deduped candidates went through verification — 7 CONFIRMED, 5 PLAUSIBLE, 1 REFUTED (the optional=true double-prompt concern; the flag is provably never read by the engine). Dropping the refuted one and the two weakest cleanups (dead GetBool, double mana-solve) to fit the cap, here are the 10 findings, most severe first.

// The headline issue: migrating cards from the auto-paying TryPayManaCond to prompt-based EffectIfPaid/UnlessTargetPays changed who decides — and the AI players decide by keyword-matching the prompt string. Rupture Spire's prompt matches no accept keyword, so AI players now always sacrifice it; Brass Man and Island Fish Jasconius never untap under AI control.

[
  {
    "file": "cards/jumpstart/lands.go",
    "line": 181,
    "summary": "AI players now always sacrifice Rupture Spire: the prompt \"pay {1} or sacrifice Rupture Spire\" matches no accept keyword in AIPlayer.ChooseMayAbility (keep/draw/damage/destroy/gain/to attack) or SearchPlayer's (draw/damage/destroy), whereas the old TryPayManaCond auto-paid with no prompt",
    "failure_scenario": "AI plays Rupture Spire with untapped lands available → ETB trigger prompts → interactive/ai/ai.go:44 returns false (no keyword hit) → land is sacrificed every game. The established convention is \"pay {cost} to keep <name>\" (effect_pay_mana_unless.go:58), which the \"keep\" keyword accepts — use that phrasing or SacrificeUnlessPayMana(\"{1}\"). Force of Nature and Tabernacle escape only by coincidence (\"damage\"/\"destroy\" in their prompts hit accept keywords)."
  },
  {
    "file": "cards/arabian/creatures.go",
    "line": 87,
    "summary": "AI-controlled Brass Man (line 844) and Island Fish Jasconius can never untEffectIfPaid prompts with \"Untap thish matches no accept keyword in either AI ChooseMayAbility implementation, replacing the old auto-pay behavior",
    "failure_scenario": "AI attacks wi Mountain untapped; next upkeepChooseMayAbility(\"Untap this permanent if you {1}\") → AIPlayer returns false → Brass Manthe rest of the game. Related divergen(Farmstead, Tablet of Epityr, Urza'sChalice) are accepted by AIPlayer (\"gain\") but declined by SearchPlayer (no \"gain\" keysimulations diverge from real play."
  },                                                                                        {
    "file": "pkg/mage/effect_if_pays.go",
    "line": 26,
    "summary": "effectIfPaid.Text() — used verbatim as the player-facing prompt — renders missing the word \"pay\": fmt.Sprintf((), cost.Text()) whereManaCostPayment.Text() returns only symbols",
    "failure_scenario": "All six newlyrompts: \"gain 1 life if you {1}\"(Urza's Chalice), \"gain 1 life if you {W}{W}\" (Farmstead), \"target player draws 1 card((Urza's Miter), \"Untap this permanentish). Fix the format to \"%s if you pay%s\" — which also gives AI keyword matchers the conventional \"pay\" phrasing."             },
  {
    "file": "pkg/mage/effect_pipeline.go",
    "line": 449,
    "summary": "IfElse (and GrantKeywoaccept the full TriggerConditionDatavocabulary but evaluate with a synthesized zero-value &GameEvent{}, so any event predicateand silently misevaluates — and the st event (game.go:2193), so the sameinterface has three event contracts (real/zero/nil)",
    "failure_scenario": "IfElse(\"onlysController{}, then, els) compiles andthe then-branch never runs (zero PlayerID ≠ controller); NotTriggerCond{Inner: EventPlayeris always true; an event predicate real-derefs. Only a doc comment guardsthis. A state-predicate sub-interface (CheckStateCond(g, sourceID, controllerID)) that    IfElse/Unless/state-triggers require we error."
  },
  {
    "file": "pkg/mage/effect_unless_pays.go",
    "line": 73,
    "summary": "If cost.Pay fails after the player accepts, the ifNotPaid penalty runs witpartial payment — ManaCostPayment.Pay TapForCost's error before ManaPool().Pay independently re-checks and can fail, leaving lands tapped while the penalty also resolves",
    "failure_scenario": "Tabernacle upccepts, AutoTapForCost taps lands butpool.Pay returns \"insufficient mana\" (mana.go:278) → creature is destroyed AND lands staplayer pays the cost's side effects anquires a solver/pool discrepancy, butthe swallowed AutoTapForCost error makes it unprovable-safe."
  },
  {
    "file": "pkg/mage/effect_pipeline_ext.go",
    "line": 799,
    "summary": "TargetHasRampageCond m.GetResolvingTargets(), which is setonly during stack-object resolution (game.go:2586) and is not rebound by ForEach per-iteradirect ApplyEffect calls — evaluationsy/stale targets and return false",
    "failure_scenario": "GrantAbility(X).Unless(TargetHasRampageCond{}) inside a ForEachPecopied effect, or any direct ApplyEffe) is empty → Unless never suppresses → a creature that already has rampage gets a second RampageTrigger(2), double-counting bonusescurrent single use happens to run insinothing breaks today."
  },                                                                                        {
    "file": "pkg/mage/game_mutator.go",
    "line": 63,
    "summary": "FlipCoin — which mutates RNG state and pops the scripted gametest coinFlip(game.go:596) — was added to the read-d FlipCoinCond now satisfiesTriggerConditionData, making it type-legal in trigger/state-trigger slots that evaluate recomments guard the misuse",
    "failure_scenario": "A card author writes SetConditionData(FlipCoinCond{}) or uses it trigger: the state-trigger scan (game.ity pass, flipping dozens of coins perturn, desyncing QueueCoinFlips in tests with no error. A resolution-only sub-interface thaslots reject would turn the comment in
  },
  {
    "file": "pkg/mage/effect_unless_pays.go",
    "line": 51,
    "summary": "execUnlessPays runs ifNotPaid via ExecuteEffect in the shared surrounding EffectContext inside the per-payer loop, so with a multi-payer selector (SelectEachPlayer/SelectEachOpponent, which the function's own doc comment recommends), Vars written by payer 1's branch leak into payer 2's iteration and can clobber same-named outer pipeline vars",
    "failure_scenario": "The shared-context change is intentional and required (Tabernacle's DestroyGathered(\"self\") needs the outer snapshot), and all current call sites are single-payer — but a future multi-payer card whose ifNotPaid branch snapshots into \"x\" has payer 2 reading payer 1's stale \"x\", destroying or referencing the wrong permanent. Consider scoping branch vars or documenting the constraint at the API.",
  },
  {
    "file": "cards/arabian/creatures.go",
    "line": 87,
    "summary": "Brass Man and Island Fish Jasconius add a non-Oracle IfElse(\"Only if tapped\", SourceIsTapped{}, ...) gate — Oracle text has no tapped condition, and the identical idiom is now copy-pasted in four places (also Mana Vault at cards/limited/artifacts.go:81 and Leviathan at cards/fourthedition/creatures.go:151) with no XXX marker despite the project's never-simplify rule",
    "failure_scenario": "The guard exists for prompt-bookkeeping (per the test comment, an untapped source must not consume a queued may-choice), not card rules — so every future \"pay to untap\" card must remember to copy the 8-line wrapper or its tests desync. A shared helper (e.g. MayPayToUntapAtUpkeep(cost)) would contain the deliberate Oracle deviation in one documented place."
  },
  {
    "file": "cards/limited/creatures.go",
    "line": 302,
    "summary": "The migration leaves two parallel \"pay or suffer\" mechanisms: new Cost-based UnlessTargetPays/EffectIfPaid (ExecuteEffect, vars preserved) vs legacy string-based SacrificeUnlessPayMana/DamageUnlessPayMana/MayPayMana (TryPayCostFromLands + ApplyEffect, which still has the fresh-context var-loss this diff explicitly fixed) — and same-pattern cards now diverge (Hasran Ogress vs Force of Nature; Rupture Spire vs SacrificeUnlessPayMana users)",
    "failure_scenario": "Card authors must guess which mechanism to use; prompts, AI keyword behavior, and pipeline-var visibility differ between them, so a fix to one path (like this diff's ExecuteEffect fix) silently misses the other. Also Force of Nature hand-rolls NewTriggered(EvtUpkeep, false, ...).SetConditionData(EventPlayerIsController{}), which is verbatim BeginningOfUpkeepTrigger(..., false) (triggered.go:391). Migrate the legacy helpers to delegate to unlessPaysEffect/effectIfPaid."
  }
]

// Two notes beyond the cap: EffectContext.GetBool is now dead code (its only reader, VarMissingCond, was deleted, leaving the five SetBool(".missing") writes write-only), and the new CanPay-then-Pay flow runs the mana solver twice per accepted payment — both verified real but minor. Build, vet, and the full test suite all pass on this diff.

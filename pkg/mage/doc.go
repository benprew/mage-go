// Package mage implements the two-player Magic: The Gathering rules engine.
//
// # Controllers and Layer 2
//
// A Permanent has a default controller and a computed controller. NewPermanent
// initializes both from the controller supplied to PutOnBattlefield; this is
// important for effects that put an opponent-owned card onto the battlefield
// under your control (CR 110.2a). ControllerID returns the computed value.
//
// EffectManager.Apply snapshots the current controller, resets the computed
// value to the default, and applies control-changing effects in Layer 2 and
// timestamp order (CR 613.1b, 613.7). Only a difference between the snapshot
// and the final result is a real control change. A real change updates ability
// context and continuous-control timing for attack and {T}/{Q} restrictions
// (CR 302.6). Trigger controllers remain snapshots taken when the trigger is
// created (CR 603.3a); leaves-the-battlefield triggers use LKI (CR 603.6c).
//
// Card code reads Permanent.ControllerID and must not directly mutate control.
// GainControl is the reusable resolving-effect builder:
//
//	GainControl().Targeting(ToTarget()).Until(EndOfTurn)
//
// Conditional control uses While with ControlConditionData values. For
// tap-maintained effects such as Willow Satyr, use:
//
//	GainControl().While(
//		ControlSourceControlledByEffectController{},
//		ControlSourceTapped{},
//	).TapMaintained()
//
// Code that discovers targets dynamically at resolution uses
// Game.AddControlEffect with a ControlEffectSpec. ControlAttached is the
// canonical Control Magic-style static effect. ControlChangeContinuous is its
// compatibility name.
//
// # Graveyard reanimation
//
// MoveFromAnyGraveyard reports the player whose graveyard actually contained
// the card so zone-change events are attributed correctly.
// ReturnTargetFromAnyGraveyardToBattlefield puts that card onto the battlefield
// with the resolving effect's controller as its default controller. Later
// control effects therefore expire back to that player rather than the owner.
//
// # Sacrifice
//
// Sacrifice performs the named action and fires both its battlefield-to-
// graveyard zone-change event and EvtSacrifice. SacrificeSource,
// SacrificeTarget, and SacrificeGathered provide resolving effects;
// SacrificeSourceCost and SacrificeMatchingCost provide costs. Sacrifice costs
// record the sacrificed object's ID; resolving effects can read its generic
// PermanentLKI, including power and toughness, with LastSacrificed.
//
// # Player selectors
//
// SelectTargetPlayer reads the first resolving target and returns it only when
// it identifies a player. Trigger helpers for draw, discard, and sacrifice
// events bind the event's player in that position. Zone-change triggers bind
// the moved object instead; use SelectTargetPermanentController when an effect
// refers to that object's controller.
//
// # Combat damage assignment
//
// A blocked attacker without an explicit player-supplied assignment assigns
// lethal damage to each blocker in order, then assigns all remaining damage to
// the final blocker. With trample, remaining damage is dealt to the defender
// instead. Damage events report the full amount assigned and dealt, including
// damage beyond a blocker's lethal toughness.
//
// # Mana-production metadata
//
// ManaProductionsForAbility exposes the production profile of tap-for-mana
// abilities to engine consumers such as AI evaluation. It recognizes both
// ManaAbility values created by WithManaAbility or WithMultiManaAbility and
// equivalent activated abilities built from Tap with only AddMana or
// AddAnyMana effects. SolveMana and CanSolveMana use a fast greedy path for
// ordinary sources, then search ordered activations when targetless
// mana-producing abilities have both a mana cost and Tap. CanAfford, MaxXValue,
// and automatic mana payment all use that unified solver behavior. Automatic
// payment executes the exact planned ability, paying its activation cost before
// using the produced mana.
package mage

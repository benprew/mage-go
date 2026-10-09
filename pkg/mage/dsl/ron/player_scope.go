package ron

import "github.com/benprew/mage-go/pkg/mage"

type playerTargetPlan struct {
	name string
}

type playerReferencePlan struct {
	choice              *playerChoicePlan
	ability             *abilityTargetPlan
	landEntry           *landEntryEventPlan
	defender            *sourceCombatPlayerPlan
	action              *playerActionsPlan
	damageRecipient     *damageRecipientPlayerPlan
	damagePlayer        *damagePlayerPlan
	permanent           *permanentReferencePlan
	target              *playerTargetPlan
	spell               *spellReferencePlan
	event               *castEventPlan
	draw                *drawEventPlan
	step                *stepEventPlan
	landMana            *landManaEventPlan
	graveyard           *graveyardEventPlan
	deathController     *graveyardEventPlan
	hostDeathController *hostDeathPlan
	hostDeathOwner      *hostDeathPlan
}

type castEventPlan struct{ spellName, casterName string }

type castEventBindings struct {
	spell  *mage.SpellBinding
	caster *mage.PlayerBinding
}

type instanceBindings struct {
	groupedExiles        map[*groupedExilePlan][]*mage.CardsBinding
	libraryReveals       map[*libraryRevealPlan]*mage.CardBinding
	handReveals          map[*handRevealPlan]*mage.CardBinding
	chosenNames          map[*cardNameChoicePlan]*mage.CardNameBinding
	chosenPlayers        map[*playerChoicePlan]*mage.PlayerBinding
	chosenPermanents     map[*permanentChoicePlan]*mage.PermanentBinding
	createdValues        map[*createdValuePlan]*mage.NumberBinding
	chosenDamageSources  map[*chosenDamageSourcePlan]*mage.DamageSourceBinding
	destructionCounts    map[*destructionCountPlan]*mage.NumberBinding
	abilities            map[*abilityTargetPlan]*mage.AbilityTargetBinding
	hostDeaths           map[*hostDeathPlan]hostDeathBindings
	entryActors          map[*landEntryEventPlan]*mage.PlayerBinding
	exiles               map[*exilePaymentPlan]*mage.NumberBinding
	defenders            map[*sourceCombatPlayerPlan]*mage.PlayerBinding
	activationHistory    *mage.ActivationHistoryBinding
	deathCards           map[*deathCardPlan]*mage.CardBinding
	actionPlayers        map[*playerActionsPlan]*mage.PlayerBinding
	damageRecipients     map[*damageRecipientPlayerPlan]*mage.PlayerBinding
	damagePlayers        map[*damagePlayerPlan]*mage.PlayerBinding
	damageAmounts        map[*damageAmountPlan]*mage.NumberBinding
	graveyardControllers map[*graveyardEventPlan]*mage.PlayerBinding
	graveyardOwners      map[*graveyardEventPlan]*mage.PlayerBinding
	counters             map[*counterPaymentPlan]*mage.NumberBinding
	sacrifices           map[*sacrificeValuePlan]sacrificeValueBindings
	events               map[*castEventPlan]castEventBindings
	drawPlayers          map[*drawEventPlan]*mage.PlayerBinding
	steps                map[*stepEventPlan]*mage.PlayerBinding
	landMana             map[*landManaEventPlan]landManaEventBindings
	tapped               map[*permanentEventPlan]*mage.PermanentBinding
	permanentSets        map[*permanentTargetsPlan]*mage.PermanentTargetsBinding
	permanents           map[*permanentTargetPlan]*mage.PermanentTargetBinding
	players              map[*playerTargetPlan]*mage.PlayerTargetBinding
	spells               map[*spellTargetPlan]*mage.SpellTargetBinding
	cards                map[*cardTargetPlan]*mage.CardTargetBinding
	recipients           map[*damageTargetPlan]*mage.DamageTargetBinding
}

func compilePlayerRef(n Node, ctx scope) (playerReferencePlan, error) {
	if n.Name == "ControllerOfAbility" {
		if err := arity(n, 1); err != nil {
			return playerReferencePlan{}, err
		}
		ability, err := compileAbilityRef(n.Args[0], ctx)
		return playerReferencePlan{ability: ability}, err
	}
	if n.Name == "ControllerOfPermanent" {
		if err := arity(n, 1); err != nil {
			return playerReferencePlan{}, err
		}
		permanent, err := compilePermanentRef(n.Args[0], ctx)
		return playerReferencePlan{permanent: &permanent}, err
	}
	if n.Name == "ControllerOfSpell" {
		if err := arity(n, 1); err != nil {
			return playerReferencePlan{}, err
		}
		spell, err := compileSpellRef(n.Args[0], ctx)
		return playerReferencePlan{spell: &spell}, err
	}
	if n.Kind == Symbol {
		switch n.Name {
		case "TheChosenPlayer":
			if ctx.chosenPlayer != nil {
				return playerReferencePlan{choice: ctx.chosenPlayer}, nil
			}
		case "Trigger_DefendingPlayer":
			if ctx.defendingPlayer != nil {
				return playerReferencePlan{defender: ctx.defendingPlayer}, nil
			}
		case "HostController":
			if ctx.aura {
				return playerReferencePlan{permanent: &permanentReferencePlan{kind: permanentHost}}, nil
			}
		case "DealsDamage_ThatPlayer":
			if ctx.damageRecipient != nil {
				return *ctx.damageRecipient, nil
			}
		case "Trigger_ThatPlayer":
			if ctx.landEntryEvent != nil {
				return playerReferencePlan{landEntry: ctx.landEntryEvent}, nil
			}
			if ctx.drawEvent != nil {
				return playerReferencePlan{draw: ctx.drawEvent}, nil
			}
			if ctx.damagePlayer != nil {
				return playerReferencePlan{damagePlayer: ctx.damagePlayer}, nil
			}
			if ctx.graveyardEvent != nil {
				return playerReferencePlan{graveyard: ctx.graveyardEvent}, nil
			}
			if ctx.landManaEvent != nil {
				return playerReferencePlan{landMana: ctx.landManaEvent}, nil
			}
			if ctx.stepEvent != nil {
				return playerReferencePlan{step: ctx.stepEvent}, nil
			}
			if ctx.castEvent != nil && n.Name == ctx.castEvent.casterName {
				return playerReferencePlan{event: ctx.castEvent}, nil
			}
		case "ControllerOfDeadPermanent":
			if ctx.hostDeath != nil {
				return playerReferencePlan{hostDeathController: ctx.hostDeath}, nil
			}
			if ctx.graveyardEvent != nil {
				return playerReferencePlan{deathController: ctx.graveyardEvent}, nil
			}
		case "OwnerOfDeadPermanent":
			if ctx.hostDeath != nil {
				return playerReferencePlan{hostDeathOwner: ctx.hostDeath}, nil
			}
		case "ActionPlayer":
			if ctx.actionPlayer != nil {
				return playerReferencePlan{action: ctx.actionPlayer}, nil
			}
		case "You":
			return playerReferencePlan{}, nil
		case "ControllerOfTargetSpell":
			if ctx.spellTarget != nil {
				return playerReferencePlan{spell: &spellReferencePlan{target: ctx.spellTarget}}, nil
			}
		case "Ref_TargetPlayer":
			if ctx.playerTarget != nil && ctx.playerTarget.name == n.Name {
				return playerReferencePlan{target: ctx.playerTarget}, nil
			}
		}
	}
	return playerReferencePlan{}, problem(n, "unbound or incompatible player reference")
}

func (p playerReferencePlan) build(bindings instanceBindings) mage.PlayerRef {
	if p.choice != nil {
		return mage.FromPlayer(bindings.chosenPlayers[p.choice])
	}
	if p.ability != nil {
		return mage.ControllerOfAbility(bindings.abilities[p.ability])
	}
	if p.hostDeathController != nil {
		return mage.FromPlayer(bindings.hostDeaths[p.hostDeathController].controller)
	}
	if p.hostDeathOwner != nil {
		return mage.FromPlayer(bindings.hostDeaths[p.hostDeathOwner].owner)
	}
	if p.landEntry != nil {
		return mage.FromPlayer(bindings.entryActors[p.landEntry])
	}
	if p.defender != nil {
		return mage.FromPlayer(bindings.defenders[p.defender])
	}
	if p.action != nil {
		return mage.FromPlayer(bindings.actionPlayers[p.action])
	}
	if p.damageRecipient != nil {
		return mage.FromPlayer(bindings.damageRecipients[p.damageRecipient])
	}
	if p.damagePlayer != nil {
		return mage.FromPlayer(bindings.damagePlayers[p.damagePlayer])
	}
	if p.deathController != nil {
		return mage.FromPlayer(bindings.graveyardControllers[p.deathController])
	}
	if p.graveyard != nil {
		return mage.FromPlayer(bindings.graveyardOwners[p.graveyard])
	}
	if p.landMana != nil {
		return mage.FromPlayer(bindings.landMana[p.landMana].player)
	}
	if p.permanent != nil {
		return mage.ControllerOfWithLKI(p.permanent.build(bindings))
	}
	if p.draw != nil {
		return mage.FromPlayer(bindings.drawPlayers[p.draw])
	}
	if p.step != nil {
		return mage.FromPlayer(bindings.steps[p.step])
	}
	if p.event != nil {
		return mage.FromPlayer(bindings.events[p.event].caster)
	}
	if p.spell != nil {
		return mage.ControllerOfSpellWithLKI(p.spell.build(bindings))
	}
	if p.target == nil {
		return mage.You
	}
	return bindings.players[p.target]
}

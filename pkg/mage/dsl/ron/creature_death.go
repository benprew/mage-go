package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCreatureDeathTrigger(filter Node) (func(mage.Effect) *mage.GenericTriggered, error) {
	damaged := false
	creature := filter
	if filter.Name == "And" {
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		parts, err := list(filter.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) != 2 {
			return nil, problem(filter, "death filter requires one creature type and one source damage-history condition")
		}
		var foundCreature, foundDamage bool
		for _, part := range parts {
			switch part.Name {
			case "IsCardtype":
				if foundCreature {
					return nil, problem(part, "duplicate creature death filter")
				}
				creature, foundCreature = part, true
			case "WasDealtDamageByPermanentThisTurn":
				if foundDamage {
					return nil, problem(part, "duplicate death damage-history condition")
				}
				if err := arity(part, 1); err != nil {
					return nil, err
				}
				if part.Args[0].Kind != Symbol || part.Args[0].Name != "ThisPermanent" {
					return nil, unsupported(part.Args[0], "death damage-history source")
				}
				foundDamage = true
			default:
				return nil, unsupported(part, "creature death filter")
			}
		}
		if !foundCreature || !foundDamage {
			return nil, problem(filter, "death filter requires one creature type and one source damage-history condition")
		}
		damaged = true
	}
	if creature.Name != "IsCardtype" {
		return nil, unsupported(creature, "creature death filter")
	}
	if err := arity(creature, 1); err != nil {
		return nil, err
	}
	if creature.Args[0].Kind != Symbol || creature.Args[0].Name != "Creature" {
		return nil, unsupported(creature.Args[0], "death card type")
	}
	return func(effect mage.Effect) *mage.GenericTriggered {
		event := mage.CreatureDies()
		if damaged {
			event = event.DamagedByThisTurn()
		}
		return mage.WhenEvent(event, effect).(*mage.GenericTriggered)
	}, nil
}

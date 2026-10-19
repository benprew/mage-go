package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentTargetPlayerOption(n Node) (mage.PermanentTargetOption, bool, error) {
	if n.Name != "ControlledByAPlayer" && n.Name != "OwnedByAPlayer" {
		return nil, false, nil
	}
	if err := arity(n, 1); err != nil {
		return nil, true, err
	}
	player := n.Args[0]
	if player.Kind == Symbol && player.Name == "Opponent" {
		if n.Name == "OwnedByAPlayer" {
			return mage.TargetOpponentOwns(), true, nil
		}
		return mage.TargetOpponentControls(), true, nil
	}
	if player.Name != "SinglePlayer" {
		return nil, false, nil
	}
	if err := arity(player, 1); err != nil {
		return nil, true, err
	}
	if player.Args[0].Name != "You" {
		return nil, false, nil
	}
	if _, err := symbol(player.Args[0]); err != nil {
		return nil, true, err
	}
	if n.Name == "OwnedByAPlayer" {
		return mage.TargetYouOwn(), true, nil
	}
	return mage.TargetYouControl(), true, nil
}

func validatePermanentTargetPlayerDomains(n Node) (control, owner uint8, err error) {
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return 0, 0, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return 0, 0, err
		}
		for _, part := range parts {
			c, o, err := validatePermanentTargetPlayerDomains(part)
			if err != nil {
				return 0, 0, err
			}
			control |= c
			owner |= o
		}
		if control == 3 || owner == 3 {
			return 0, 0, problem(n, "conflicting permanent target player domains")
		}
		return control, owner, nil
	}
	_, handled, err := compilePermanentTargetPlayerOption(n)
	if err != nil || !handled {
		return 0, 0, err
	}
	value := uint8(1)
	if n.Args[0].Name == "Opponent" {
		value = 2
	}
	if n.Name == "OwnedByAPlayer" {
		return 0, value, nil
	}
	return value, 0, nil
}

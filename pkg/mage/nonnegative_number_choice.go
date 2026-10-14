package mage

import "fmt"

// NonnegativeNumberChooser permits direct input without a menu for each value.
// Players that do not supply it can enter digits through bounded choices.
type NonnegativeNumberChooser interface {
	ChooseNonnegativeNumber(reason string) int
}

func chooseNonnegativeNumber(player Player, reason string) int {
	if chooser, ok := player.(NonnegativeNumberChooser); ok {
		return chooser.ChooseNonnegativeNumber(reason)
	}
	amount := 0
	limit := int(^uint(0) >> 1)
	for {
		if amount > limit/10 {
			return amount
		}
		digitLimit := 9
		if amount == limit/10 {
			digitLimit = limit % 10
		}
		choice := player.ChooseMode([]string{fmt.Sprintf("Use %d", amount), "Append a digit"}, reason)
		if choice != 1 {
			return amount
		}
		digit := player.ChooseNumber(0, digitLimit, reason+": next digit")
		if digit < 0 || digit > digitLimit {
			return 0
		}
		amount = amount*10 + digit
	}
}

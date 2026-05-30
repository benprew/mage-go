package search

import "github.com/benprew/mage-go/pkg/mage"

func passOnly(*mage.Game, int, bool) mage.PriorityAction {
	return mage.PriorityAction{Type: mage.PriorityPass}
}

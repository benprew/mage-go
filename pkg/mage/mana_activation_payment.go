package mage

import (
	"fmt"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ManaActivationPayment saves the colors spent on one mana ability.
// Restricted mana cannot pay this cost.
type ManaActivationPayment [AnyColor + 1]int

func manaActivationPayments(pool *ManaPool, cost ManaCost) []ManaActivationPayment {
	if cost.IsZero() {
		return []ManaActivationPayment{{}}
	}
	var available ManaActivationPayment
	for _, mana := range pool.pool {
		if manaUsable(mana, nil) {
			available[mana.Color]++
		}
	}
	var requirements []manaPaymentRequirement
	for _, pip := range []struct {
		color Color
		count int
	}{{White, cost.White}, {Blue, cost.Blue}, {Black, cost.Black}, {Red, cost.Red}, {Green, cost.Green}} {
		for range pip.count {
			requirements = append(requirements, manaPaymentRequirement{candidates: pool.paymentCandidates(pip.color)})
		}
	}
	for _, hybrid := range cost.Hybrid {
		candidates := append([]Color(nil), pool.paymentCandidates(hybrid.A)...)
		for _, color := range pool.paymentCandidates(hybrid.B) {
			if !containsPaymentColor(candidates, color) {
				candidates = append(candidates, color)
			}
		}
		requirements = append(requirements, manaPaymentRequirement{candidates: candidates})
	}
	var payments []ManaActivationPayment
	var used ManaActivationPayment
	seen := make(map[ManaActivationPayment]bool)
	visited := make(map[manaPaymentState]bool)
	var generic func(int, int)
	generic = func(index, remaining int) {
		if index == len(manaPoolColors) {
			if remaining == 0 && !seen[used] {
				seen[used] = true
				payments = append(payments, used)
			}
			return
		}
		color := manaPoolColors[index]
		for count := min(available[color], remaining); count >= 0; count-- {
			used[color] += count
			generic(index+1, remaining-count)
			used[color] -= count
		}
	}
	var allocate func(int)
	allocate = func(index int) {
		state := manaPaymentState{index: index, available: [AnyColor + 1]int(available)}
		if visited[state] {
			return
		}
		visited[state] = true
		if index == len(requirements) {
			generic(0, cost.Generic)
			return
		}
		for _, color := range requirements[index].candidates {
			if available[color] == 0 {
				continue
			}
			available[color]--
			used[color]++
			allocate(index + 1)
			used[color]--
			available[color]++
		}
	}
	allocate(0)
	return payments
}

func validManaActivationPayment(pool *ManaPool, cost ManaCost, payment ManaActivationPayment) bool {
	if cost.IsZero() {
		return payment == (ManaActivationPayment{})
	}
	selected := NewManaPool()
	selected.ManaConversions = pool.ManaConversions
	selected.spendingPermissions = pool.spendingPermissions
	total := 0
	for _, color := range manaPoolColors {
		if payment[color] < 0 {
			return false
		}
		available := 0
		for _, mana := range pool.pool {
			if mana.Color == color && manaUsable(mana, nil) {
				available++
			}
		}
		if payment[color] > available {
			return false
		}
		selected.Add(color, payment[color])
		total += payment[color]
	}
	return payment[AnyColor] == 0 && total == cost.CMC() && selected.CanPay(cost, nil)
}

func payManaActivationPayment(pool *ManaPool, cost ManaCost, payment ManaActivationPayment) error {
	if !validManaActivationPayment(pool, cost, payment) {
		return fmt.Errorf("planned mana activation payment is no longer legal")
	}
	for _, color := range manaPoolColors {
		pool.removeUpTo(color, payment[color], nil)
	}
	return nil
}

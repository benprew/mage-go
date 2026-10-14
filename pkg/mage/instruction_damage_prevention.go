package mage

// DealDamageWithPreventionFrom prevents part of this damage instruction.
// The prevention uses the shared replacement order and leaves no shield.
func DealDamageWithPreventionFrom(source DamageSourceRef, recipient DamageRecipientRef, amount, prevention ResolutionValue) ResolvingEffect {
	return &damageFromEffect{source: source, recipient: recipient, amount: amount, prevention: prevention, withPrevention: true}
}

type instructionDamagePrevention struct {
	replacementBase
	amount int
}

func (r *instructionDamagePrevention) Matches(action Action, _ GameReader) bool {
	return r.amount > 0 && damageActionAmount(action) > 0
}
func (r *instructionDamagePrevention) Replace(action Action, _ *Game) Action {
	return damageActionWithAmount(action, max(0, damageActionAmount(action)-r.amount))
}
func (*instructionDamagePrevention) IsActive(GameReader) bool   { return true }
func (r *instructionDamagePrevention) Clone() ReplacementEffect { cp := *r; return &cp }

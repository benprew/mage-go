package core

// SpellPaymentContext describes the spell currently being paid for, so
// individual mana entries with spending restrictions can decide whether
// they may be spent. Built once at each spell-cast entry point.
// XSpendingRestricted and XSpendingColor retain a spell declaration.
// RestrictedGeneric saves the remaining X payment after cost reductions.
// A separate payment context prevents one query from changing another payment.
type SpellPaymentContext struct {
	IsArtifact          bool
	IsCreature          bool
	CardName            string
	XSpendingRestricted bool
	XSpendingColor      Color
	RestrictedGeneric   int
}

// ManaRestriction is a predicate over SpellPaymentContext that decides
// whether a unit of restricted mana may be spent on the given spell.
// Concrete restrictions live in this package as zero-size value structs.
type ManaRestriction interface {
	IsSatisfiedBy(SpellPaymentContext) bool
	Description() string
}

// ArtifactSpellsOnly — "Spend this mana only to cast artifact spells."
type ArtifactSpellsOnly struct{}

func (ArtifactSpellsOnly) IsSatisfiedBy(ctx SpellPaymentContext) bool {
	return ctx.IsArtifact
}

func (ArtifactSpellsOnly) Description() string {
	return "Spend this mana only to cast artifact spells"
}

// CreatureSpellsOnly — "Spend this mana only to cast creature spells."
type CreatureSpellsOnly struct{}

func (CreatureSpellsOnly) IsSatisfiedBy(ctx SpellPaymentContext) bool {
	return ctx.IsCreature
}

func (CreatureSpellsOnly) Description() string {
	return "Spend this mana only to cast creature spells"
}

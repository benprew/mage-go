package mage

// WithActivationMinimumX restricts the announced X before costs are paid.
func WithActivationMinimumX(minimum int) ActionOption {
	return func(a *ActionDefinition) { a.activationMinimumX = minimum; a.activationMinimumXOptions++ }
}

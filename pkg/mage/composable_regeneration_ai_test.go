package mage

import "testing"

func TestRegeneratesAttachment(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(*PermanentBinding, *PermanentBinding) *ActionDefinition
		want  bool
	}{
		{"direct", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(AttachedPermanent))
		}, true},
		{"source", func(a, b *PermanentBinding) *ActionDefinition { return Activate(Costs(), Regenerate(This)) }, false},
		{"snapshot", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Snapshot(AttachedPermanent, a), Regenerate(From(a))))
		}, true},
		{"capture chain", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(From(b)), CapturePermanentBeforePayment(AttachedPermanent, a), CapturePermanentBeforePayment(From(a), b))
		}, true},
		{"capture overwrite", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(From(a)), CapturePermanentBeforePayment(AttachedPermanent, a), CapturePermanentBeforePayment(This, a))
		}, false},
		{"snapshot overwrite", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Snapshot(This, a), Regenerate(From(a))), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
		{"snapshot replaces capture", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Snapshot(AttachedPermanent, a), Regenerate(From(a))), CapturePermanentBeforePayment(This, a))
		}, true},
		{"same name different binding", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Regenerate(From(b)), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
		{"nested sequence writes persist", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Sequence(Snapshot(This, a)), Regenerate(From(a))), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
		{"nested sequence", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Sequence(Snapshot(AttachedPermanent, a)), Regenerate(From(a))))
		}, true},
		{"unknown instruction", func(a, b *PermanentBinding) *ActionDefinition {
			return Activate(Costs(), Sequence(Destroy(AttachedPermanent), Regenerate(From(a))), CapturePermanentBeforePayment(AttachedPermanent, a))
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ability := tc.build(BindPermanent("host"), BindPermanent("host"))
			if got := RegeneratesAttachment(ability); got != tc.want {
				t.Fatalf("RegeneratesAttachment = %v, want %v", got, tc.want)
			}
			if got := RegeneratesAttachment(ability); got != tc.want {
				t.Fatalf("repeated inspection = %v, want %v", got, tc.want)
			}
		})
	}
}

package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestSavedPermanentComposition(t *testing.T) {
	saved := dsl.BindPermanent("host")
	var effect = dsl.Sequence(
		dsl.Snapshot(dsl.AttachedPermanent, saved),
		dsl.Sacrifice(dsl.This),
		dsl.Sacrifice(dsl.From(saved)),
	)
	metadata := dsl.InspectDefinition(effect)
	if metadata.Role != "sequence" || len(metadata.Children) != 3 || metadata.Children[0].Role != "snapshot" {
		t.Fatalf("metadata = %+v", metadata)
	}
}

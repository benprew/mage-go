package mage

import "testing"

func TestRegistryIsolation(t *testing.T) {
	a := NewRegistry()
	b := NewRegistry()

	a.Register("TestCard", func() Card {
		return NewCreature("TestCard", "{1}", 1, 1)
	})

	if !a.CardRegistered("TestCard") {
		t.Fatal("expected TestCard registered in registry a")
	}
	if b.CardRegistered("TestCard") {
		t.Fatal("expected TestCard NOT registered in registry b")
	}

	card, err := a.CreateCard("TestCard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.Name() != "TestCard" {
		t.Fatalf("expected card name TestCard, got %s", card.Name())
	}

	_, err = b.CreateCard("TestCard")
	if err == nil {
		t.Fatal("expected error creating TestCard from empty registry b")
	}
}

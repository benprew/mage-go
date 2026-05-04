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

func TestRegistryDiacriticFold(t *testing.T) {
	r := NewRegistry()
	r.Register("El-Hajjâj", func() Card { return NewCreature("El-Hajjâj", "{1}{B}{B}", 1, 1) })
	r.Register("Junún Efreet", func() Card { return NewCreature("Junún Efreet", "{1}{B}{B}", 3, 3) })

	for _, alias := range []string{"El-Hajjaj", "Junun Efreet", "El-Hajjâj", "Junún Efreet"} {
		c, err := r.CreateCard(alias)
		if err != nil {
			t.Fatalf("CreateCard(%q): %v", alias, err)
		}
		if c == nil {
			t.Fatalf("CreateCard(%q) returned nil", alias)
		}
	}

	if _, err := r.CreateCard("Nonexistent Card"); err == nil {
		t.Fatal("expected error for unknown card")
	}
}

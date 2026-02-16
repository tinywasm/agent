package agent

import "testing"

func TestFSM_Transitions(t *testing.T) {
	f := &fsm{current: StateIdle}

	// Test allowed transitions
	if err := f.transition(StateReasoning); err != nil {
		t.Fatalf("expected allowed transition Idle -> Reasoning, got %v", err)
	}

	if f.current != StateReasoning {
		t.Errorf("expected current state Reasoning, got %s", f.current)
	}

	// Test invalid transition
	if err := f.transition(StateIdle); err == nil {
		t.Fatalf("expected error transition Reasoning -> Idle")
	}

	// Continue valid transitions
	if err := f.transition(StateActing); err != nil {
		t.Fatalf("expected allowed transition Reasoning -> Acting, got %v", err)
	}

	if err := f.transition(StateReasoning); err != nil {
		t.Fatalf("expected allowed transition Acting -> Reasoning, got %v", err)
	}

	if err := f.transition(StateReflecting); err != nil {
		t.Fatalf("expected allowed transition Reasoning -> Reflecting, got %v", err)
	}

	if err := f.transition(StateResponding); err != nil {
		t.Fatalf("expected allowed transition Reflecting -> Responding, got %v", err)
	}

	if err := f.transition(StateIdle); err != nil {
		t.Fatalf("expected allowed transition Responding -> Idle, got %v", err)
	}
}

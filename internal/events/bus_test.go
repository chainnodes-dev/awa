package events

import (
	"context"
	"testing"
	"time"
)

func TestLocalBus_PublishAndReceive(t *testing.T) {
	bus := NewLocalBus()
	ctx := context.Background()

	ch, cancel := bus.Subscribe(ctx)
	defer cancel()

	want := New(RunCreated, map[string]string{"run_id": "r1"})
	if err := bus.Publish(ctx, want); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case got := <-ch:
		if got.Type != RunCreated {
			t.Errorf("event type: got %q, want %q", got.Type, RunCreated)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestLocalBus_MultipleSubscribers(t *testing.T) {
	bus := NewLocalBus()
	ctx := context.Background()

	ch1, cancel1 := bus.Subscribe(ctx)
	defer cancel1()
	ch2, cancel2 := bus.Subscribe(ctx)
	defer cancel2()

	if err := bus.Publish(ctx, New(StateChanged, nil)); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != StateChanged {
				t.Errorf("subscriber %d: got type %q, want %q", i+1, got.Type, StateChanged)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i+1)
		}
	}
}

func TestLocalBus_Cancel_StopsDelivery(t *testing.T) {
	bus := NewLocalBus()
	ctx := context.Background()

	ch, cancel := bus.Subscribe(ctx)
	cancel() // unsubscribe immediately

	// Channel should be closed after cancel.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected channel to close promptly after cancel")
	}
}

func TestLocalBus_Cancel_DoesNotReceiveAfterUnsubscribe(t *testing.T) {
	bus := NewLocalBus()
	ctx := context.Background()

	ch, cancel := bus.Subscribe(ctx)
	cancel() // unsubscribe before publish

	// Drain the closed channel, then publish — nothing should arrive.
	for range ch {
	}

	// Re-subscribe a second subscriber to ensure the bus still works.
	ch2, cancel2 := bus.Subscribe(ctx)
	defer cancel2()

	_ = bus.Publish(ctx, New(RunCompleted, nil))

	select {
	case got := <-ch2:
		if got.Type != RunCompleted {
			t.Errorf("got %q, want %q", got.Type, RunCompleted)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting on active subscriber")
	}
}

func TestLocalBus_EventTimestamp(t *testing.T) {
	before := time.Now()
	evt := New(RunFailed, nil)
	after := time.Now()

	if evt.Timestamp.Before(before) || evt.Timestamp.After(after) {
		t.Errorf("timestamp %v is outside [%v, %v]", evt.Timestamp, before, after)
	}
}

func TestLocalBus_PublishMultipleEvents(t *testing.T) {
	bus := NewLocalBus()
	ctx := context.Background()

	ch, cancel := bus.Subscribe(ctx)
	defer cancel()

	types := []string{RunCreated, StateChanged, RunCompleted}
	for _, typ := range types {
		_ = bus.Publish(ctx, New(typ, nil))
	}

	received := make([]string, 0, len(types))
	timeout := time.After(time.Second)
	for len(received) < len(types) {
		select {
		case evt := <-ch:
			received = append(received, evt.Type)
		case <-timeout:
			t.Fatalf("timed out: received %d/%d events", len(received), len(types))
		}
	}

	for i, want := range types {
		if received[i] != want {
			t.Errorf("event[%d]: got %q, want %q", i, received[i], want)
		}
	}
}

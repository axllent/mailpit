package chaos

import (
	"sync"
	"sync/atomic"
	"testing"
)

// saved puts the global settings back after the test.
func saved(t *testing.T) {
	t.Helper()

	enabled, orig := Enabled, Get()
	t.Cleanup(func() {
		_ = SetFromStruct(orig)
		Enabled = enabled
	})
}

func TestGetIsACopy(t *testing.T) {
	saved(t)
	if err := Set("sender", 452, 10); err != nil {
		t.Fatal(err)
	}

	c := Get()
	c.Sender = Trigger{ErrorCode: 599, Probability: 99}

	if got := Get().Sender; got.ErrorCode != 452 || got.Probability != 10 {
		t.Fatalf("changing what Get returned changed the configuration: %+v", got)
	}
}

// All of the triggers are valid, or none is used (it used to use the ones before the first which was not).
func TestSetFromStructIsAllOrNothing(t *testing.T) {
	saved(t)

	if err := SetFromStruct(Triggers{
		Sender:         Trigger{ErrorCode: 452, Probability: 10},
		Recipient:      Trigger{ErrorCode: 453, Probability: 20},
		Authentication: Trigger{ErrorCode: 454, Probability: 30},
	}); err != nil {
		t.Fatal(err)
	}
	want := Get()

	for name, bad := range map[string]Triggers{
		"recipient probability": {Sender: Trigger{ErrorCode: 500, Probability: 50}, Recipient: Trigger{ErrorCode: 451, Probability: 101}},
		"authentication code":   {Sender: Trigger{ErrorCode: 500, Probability: 50}, Authentication: Trigger{ErrorCode: 200, Probability: 5}},
		"negative probability":  {Sender: Trigger{ErrorCode: 500, Probability: -1}},
	} {
		if err := SetFromStruct(bad); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
		if got := Get(); got != want {
			t.Fatalf("%s: the configuration was changed to %+v by one which was not valid (want %+v)", name, got, want)
		}
	}

	// control: one which is valid is all used, and the codes which are not given are the defaults
	if err := SetFromStruct(Triggers{Sender: Trigger{ErrorCode: 500, Probability: 50}}); err != nil {
		t.Fatal(err)
	}
	got := Get()
	if got.Sender != (Trigger{ErrorCode: 500, Probability: 50}) ||
		got.Recipient != (Trigger{ErrorCode: 451, Probability: 0}) ||
		got.Authentication != (Trigger{ErrorCode: 535, Probability: 0}) {
		t.Fatalf("got %+v", got)
	}
}

func TestSetValidation(t *testing.T) {
	saved(t)
	if err := SetFromStruct(Triggers{}); err != nil {
		t.Fatal(err)
	}
	want := Get()

	for name, tc := range map[string]struct {
		key         string
		code, prob  int
		shouldError bool
	}{
		"code too low":       {"sender", 399, 10, true},
		"code too high":      {"sender", 600, 10, true},
		"probability high":   {"sender", 451, 101, true},
		"probability low":    {"sender", 451, -1, true},
		"unknown key":        {"nonsense", 451, 10, true},
		"control: sender":    {"Sender", 451, 10, false},
		"control: recipient": {"recipients", 452, 20, false},
		"control: auth":      {"AUTH", 535, 30, false},
	} {
		before := Get()
		err := Set(tc.key, tc.code, tc.prob)
		if (err != nil) != tc.shouldError {
			t.Fatalf("%s: err = %v", name, err)
		}
		if tc.shouldError && Get() != before {
			t.Fatalf("%s: the configuration was changed by one which was not valid", name)
		}
	}

	got := Get()
	if got.Sender.Probability != 10 || got.Recipient != (Trigger{ErrorCode: 452, Probability: 20}) || got.Authentication.Probability != 30 {
		t.Fatalf("got %+v (started as %+v)", got, want)
	}
}

func TestTrigger(t *testing.T) {
	saved(t)

	Enabled = true
	if fail, code := (Trigger{ErrorCode: 452, Probability: 100}).Trigger(); !fail || code != 452 {
		t.Fatalf("a probability of 100 gave %v, %d", fail, code)
	}
	for range 200 {
		if fail, _ := (Trigger{ErrorCode: 452, Probability: 0}).Trigger(); fail {
			t.Fatal("a probability of 0 triggered")
		}
	}

	Enabled = false
	if fail, _ := (Trigger{ErrorCode: 452, Probability: 100}).Trigger(); fail {
		t.Fatal("a trigger fired with chaos not enabled")
	}
}

// The triggers are changed together, so a session never sees some of a new configuration and some of the old. The
// three had a separate change each, and a reader could see a mixture: here every change sets all three to the same
// code, so every configuration which is read must have one code.
func TestConfigurationChangesAreAtomic(t *testing.T) {
	saved(t)
	Enabled = true

	// start from one code (the defaults have a different one for authentication), so what is read is only ever from
	// the changes below
	if err := SetFromStruct(Triggers{
		Sender:         Trigger{ErrorCode: 451, Probability: 1},
		Recipient:      Trigger{ErrorCode: 451, Probability: 1},
		Authentication: Trigger{ErrorCode: 451, Probability: 1},
	}); err != nil {
		t.Fatal(err)
	}

	var mixed atomic.Int64
	stop := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if c := Get(); c.Sender.ErrorCode != c.Recipient.ErrorCode || c.Recipient.ErrorCode != c.Authentication.ErrorCode {
					mixed.Add(1)
				}
			}
		}()
	}

	for i := range 3000 {
		code := 451 + i%2
		if err := SetFromStruct(Triggers{
			Sender:         Trigger{ErrorCode: code, Probability: 1},
			Recipient:      Trigger{ErrorCode: code, Probability: 1},
			Authentication: Trigger{ErrorCode: code, Probability: 1},
		}); err != nil {
			t.Fatal(err)
		}
	}
	close(stop)
	readers.Wait()

	if n := mixed.Load(); n != 0 {
		t.Fatalf("%d configurations which were read were part of one change and part of another", n)
	}
}

// The API changes the configuration while SMTP sessions use it (this is what the race detector is for).
func TestConcurrentUse(t *testing.T) {
	saved(t)
	Enabled = true

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := range 2000 {
			_ = Set("sender", 451, i%100)
		}
	}()
	go func() {
		defer wg.Done()
		for i := range 2000 {
			_ = SetFromStruct(Triggers{Recipient: Trigger{ErrorCode: 452, Probability: i % 100}})
		}
	}()
	go func() {
		defer wg.Done()
		for range 2000 {
			c := Get()
			c.Sender.Trigger()
			c.Recipient.Trigger()
			c.Authentication.Trigger()
		}
	}()
	wg.Wait()
}

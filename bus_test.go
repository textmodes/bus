package bus

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil {
		t.Error("New but not created")
	}
}

func TestSubscribe(t *testing.T) {
	bus := New()
	if bus.Subscribe("topic", func() {}) != nil {
		t.Fail()
	}
	if bus.Subscribe("topic", "invalid") == nil {
		t.Fail()
	}
}

func TestPublish(t *testing.T) {
	bus := New()
	bus.Subscribe("test", func(a, b int) {
		if a != b {
			t.Fail()
		}
	})
	bus.Publish("test", 42, 42)
}

func TestOnce(t *testing.T) {
	bus := New()
	bus.Once("test", func(a int, out *[]int) {
		*out = append(*out, a)
	})

	results := make([]int, 0)
	for n := 0; n < 23; n++ {
		bus.Publish("test", n, &results)
	}

	if len(results) != 1 {
		t.Fail()
	}
	if bus.Has("test") {
		t.Fail()
	}
}

func TestOnceAsync(t *testing.T) {
	bus := New()
	bus.OnceAsync("test", func(a int, out *[]int) {
		*out = append(*out, a)
	})

	results := make([]int, 0)
	for n := 0; n < 23; n++ {
		bus.Publish("test", n, &results)
	}
	bus.Wait()

	if len(results) != 1 {
		t.Fail()
	}
	if bus.Has("test") {
		t.Fail()
	}
}

func TestSubscribeAsync(t *testing.T) {
	bus := New()
	bus.SubscribeAsync("test", func(a int, out chan<- int) {
		out <- a
	}, false)

	results := make(chan int)
	bus.Publish("test", 1, results)
	bus.Publish("test", 2, results)

	n := 0
	go func() {
		for _ = range results {
			n++
		}
	}()

	bus.Wait()
	if n != 2 {
		t.Fail()
	}
}

func TestSubscribeAsyncTransaction(t *testing.T) {
	bus := New()
	bus.SubscribeAsync("test", func(a int, out *[]int, d string) {
		t, _ := time.ParseDuration(d)
		time.Sleep(t)
		*out = append(*out, a)
	}, true)

	results := make([]int, 0)
	bus.Publish("test", 1, &results, "1s")
	bus.Publish("test", 2, &results, "0s")
	bus.Wait()

	if len(results) != 2 {
		t.Fail()
	}
	if results[0] != 1 || results[1] != 2 {
		t.Fail()
	}
}

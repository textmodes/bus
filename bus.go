package bus

import (
	"fmt"
	"reflect"
	"sync"
)

type Bus struct {
	subscriber map[string][]*subscriber
	lock       sync.Mutex
	wg         sync.WaitGroup
}

type subscriber struct {
	cb          reflect.Value
	once        bool
	async       bool
	transaction bool
	sync.Mutex
}

func New() *Bus {
	return &Bus{
		make(map[string][]*subscriber),
		sync.Mutex{},
		sync.WaitGroup{},
	}
}

func (b *Bus) prepare(n string, fn interface{}, once, async, transaction bool) (h *subscriber, err error) {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return nil, fmt.Errorf("%T is not a function", fn)
	}
	if b.subscriber[n] == nil {
		b.subscriber[n] = make([]*subscriber, 0)
	}
	v := reflect.ValueOf(fn)
	return &subscriber{v, once, async, transaction, sync.Mutex{}}, nil
}

// Has check if a topic is present and has active subscribers.
func (b *Bus) Has(n string) bool {
	return b.subscriber[n] != nil && len(b.subscriber[n]) > 0
}

// Subscribe adds a callback to the subscribers.
func (b *Bus) Subscribe(n string, fn interface{}) (err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	var s *subscriber
	s, err = b.prepare(n, fn, false, false, false)
	if err == nil {
		b.subscriber[n] = append(b.subscriber[n], s)
	}
	return
}

// SubscribeAsync adds a callback to the subscribers. Transaction determines if the subsequent calls for a topic are done in concurrently (true) or serial (false).
func (b *Bus) SubscribeAsync(n string, fn interface{}, transaction bool) (err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	var s *subscriber
	s, err = b.prepare(n, fn, false, true, transaction)
	if err == nil {
		b.subscriber[n] = append(b.subscriber[n], s)
	}
	return
}

// OnceAsync removes the event after the callback has been fired synchronously.
func (b *Bus) Once(n string, fn interface{}) (err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	var s *subscriber
	s, err = b.prepare(n, fn, true, false, false)
	if err == nil {
		b.subscriber[n] = append(b.subscriber[n], s)
	}
	return
}

// OnceAsync removes the event after the callback has been fired asynchronously.
func (b *Bus) OnceAsync(n string, fn interface{}) (err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	var s *subscriber
	s, err = b.prepare(n, fn, true, true, false)
	if err == nil {
		b.subscriber[n] = append(b.subscriber[n], s)
	}
	return
}

// Unsubscribe removes a topic from the bus
func (b *Bus) Unsubscribe(n string) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.subscriber[n]; ok {
		delete(b.subscriber, n)
		return nil
	}
	return fmt.Errorf("Topic %q does not exist", n)
}

func (b *Bus) Publish(n string, a ...interface{}) {
	b.lock.Lock()
	if h, ok := b.subscriber[n]; ok {
		for _, s := range h {
			if s.async {
				b.wg.Add(1)
				go b.publishAsync(s, n, a...)
			} else {
				b.publish(s, n, a...)
			}
		}
	} else {
		b.lock.Unlock()
	}
}

func (b *Bus) publish(h *subscriber, n string, a ...interface{}) {
	args := b.setup(h.once, n, a...)
	h.cb.Call(args)
}

func (b *Bus) publishAsync(h *subscriber, n string, a ...interface{}) {
	defer b.wg.Done()
	if h.transaction {
		h.Lock()
		defer h.Unlock()
	}
	b.publish(h, n, a...)
}

func (b *Bus) setup(r bool, n string, a ...interface{}) []reflect.Value {
	defer b.lock.Unlock()
	args := make([]reflect.Value, 0)
	for _, arg := range a {
		args = append(args, reflect.ValueOf(arg))
	}
	if r {
		delete(b.subscriber, n)
	}
	return args
}

// Wait waits for all the async callbacks to finish
func (b *Bus) Wait() {
	b.wg.Wait()
}

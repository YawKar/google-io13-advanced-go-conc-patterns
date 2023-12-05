package feedreader

import (
	"fmt"
	"math/rand"
	"time"
)

type Item struct {
	Title, Channel, GUID string // a subset of RSS fields
}

type Fetcher interface {
	Fetch() (items []Item, next time.Time, err error)
}

type fakeFetcher struct {
	impl func() (items []Item, next time.Time, err error)
}

func (f *fakeFetcher) Fetch() (items []Item, next time.Time, err error) {
	return f.impl()
}

func Fetch(domain string) Fetcher {
	counter := 1
	return &fakeFetcher{
		func() (items []Item, next time.Time, err error) {
			for i := 0; i < 3; i++ {
				items = append(items, Item{
					fmt.Sprintf("Item %d", counter+i),
					domain,
					fmt.Sprintf("GUID: %d", rand.Int31())})
			}
			next = time.Now().Add(2 * time.Second)
			counter += 3
			return
		},
	}
}

type Subscription interface {
	Updates() <-chan Item // stream of Items
	Close() error         // shuts down the stream
}

type sub struct {
	fetcher Fetcher
	updates chan Item
	closing chan chan error
}

func (s *sub) loop() {
	maxPending := 10
	var pending []Item
	var next time.Time
	var err error
	seen := make(map[string]bool) // set of item.GUID
	type fetchResult struct {
		items []Item
		next  time.Time
		err   error
	}
	var fetchDone chan fetchResult // if non-nil, Fetch is running
	for {
		var startFetch <-chan time.Time
		if fetchDone == nil && len(pending) < maxPending {
			startFetch = time.After(time.Until(next))
		}

		var updates chan Item
		var first Item
		if len(pending) != 0 {
			first = pending[0]
			updates = s.updates
		}

		select {
		case errCh := <-s.closing:
			errCh <- err
			close(s.updates)
			return
		case <-startFetch:
			fetchDone = make(chan fetchResult, 1) // 1 for goroutine to be able to die rightaway
			go func() {
				fetched, next, err := s.fetcher.Fetch()
				fetchDone <- fetchResult{fetched, next, err}
			}()
		case f := <-fetchDone:
			fetchDone = nil
			if f.err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			for _, item := range f.items {
				if !seen[item.GUID] {
					pending = append(pending, item)
					seen[item.GUID] = true
				}
			}
		case updates <- first:
			pending = pending[1:]
		}
	}
}

func (s *sub) Close() error {
	errCh := make(chan error)
	s.closing <- errCh
	return <-errCh
}

func (s *sub) Updates() <-chan Item {
	return s.updates
}

// Converts Fetches to a stream
func Subscribe(fetcher Fetcher) Subscription {
	s := &sub{
		fetcher: fetcher,
		updates: make(chan Item), // for updates
		closing: make(chan chan error),
	}
	go s.loop()
	return s
}

type merge struct {
	updates chan Item
	subs    []Subscription
	quit    chan struct{}
	errs    chan error
}

func (m *merge) Updates() <-chan Item {
	return m.updates
}

func (m *merge) Close() (err error) {
	for range m.subs {
		m.quit <- struct{}{}
	}
	for range m.subs {
		if e := <-m.errs; e != nil {
			err = e
		}
	}
	close(m.quit)
	close(m.updates)
	return
}

// Merges several streams
func Merge(subs ...Subscription) Subscription {
	m := &merge{
		subs:    subs,
		updates: make(chan Item),
		quit:    make(chan struct{}),
		errs:    make(chan error),
	}
	for _, sub := range subs {
		go func(s Subscription) {
			for {
				var it Item
				select {
				case it = <-s.Updates():
				case <-m.quit:
					m.errs <- s.Close()
					return
				}
				select {
				case m.updates <- it:
				case <-m.quit:
					m.errs <- s.Close()
					return
				}
			}
		}(sub)
	}
	return m
}

func MainExample() {
	// subscribe to some feeds, and create a merged update stream
	merged := Merge(
		Subscribe(Fetch("blog.golang.org")),
		Subscribe(Fetch("googleblog.blogspot.com")),
		Subscribe(Fetch("googledevelopers.blogspot.com")),
	)

	// Close the subscriptions after some time
	time.AfterFunc(3*time.Second, func() {
		fmt.Println("closed:", merged.Close())
	})

	// Print the stream
	for it := range merged.Updates() {
		fmt.Println(it.Channel, it.Title)
	}

	panic("SHOW ME DEEZ STACKS")
}

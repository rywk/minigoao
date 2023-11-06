package conc

import (
	"log"
)

type BCaster[T any] struct {
	Input chan T
	reg   chan chan T
	unreg chan chan T
	subs  map[chan T]bool
}

func NewBCaster[T any](input chan T) *BCaster[T] {
	b := &BCaster[T]{
		Input: input,
		reg:   make(chan chan T),
		unreg: make(chan chan T),
		subs:  make(map[chan T]bool),
	}

	go b.run()

	return b
}

func (b *BCaster[T]) run() {
	for {
		select {
		case m := <-b.Input:
			b.broadcast(m)
		case ch, ok := <-b.reg:
			if ok {
				b.subs[ch] = true
			} else {
				return
			}
		case ch := <-b.unreg:
			delete(b.subs, ch)
			log.Println("2:DELETED A SUB")
		}
	}
}

func (b *BCaster[T]) broadcast(m T) {
	for ch := range b.subs {
		ch <- m
	}
}

func (b *BCaster[T]) Register(newch chan T) {
	b.reg <- newch
}

func (b *BCaster[T]) Unregister(newch chan T) {
	b.unreg <- newch
}

func (b *BCaster[T]) Close() error {
	close(b.reg)
	close(b.unreg)
	return nil
}

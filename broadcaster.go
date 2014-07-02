package main

type Broadcaster struct {
	dataCh       chan interface{}
	registryChan chan chan interface{}
	finishedChan chan chan interface{}
}

type BroadcastListener struct {
	ch          chan interface{}
	broadcaster *Broadcaster
}

func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		dataCh:       make(chan interface{}),
		registryChan: make(chan chan interface{}),
		finishedChan: make(chan chan interface{}),
	}
	go b.process()
	return b
}

func (b *Broadcaster) Write(v interface{}) {
	b.dataCh <- v
}

func (b *Broadcaster) Listen() *BroadcastListener {
	ch := make(chan interface{})
	b.registryChan <- ch

	return &BroadcastListener{ch, b}
}

func (b *Broadcaster) Close() {
	close(b.registryChan)
	close(b.dataCh)
}

func (b *Broadcaster) process() {
	listeners := []chan interface{}{}
	defer func() {
		for _, l := range(listeners) {
			close(l)
		}
	}()

	for {
		select {
		case v, ok := <-b.dataCh:
			if !ok {
				return
			}
			for _, ch := range listeners {
				ch <- v
			}

		case ch, ok := <-b.registryChan:
			if !ok {
				return
			}
			listeners = append(listeners, ch)

		case ch, ok := <-b.finishedChan:
			if !ok {
				return
			}

			for i, x := range listeners {
				if x == ch {
					listeners = append(listeners[0:i], listeners[i+1:]...)
					break
				}
			}

			close(ch)
		}
	}
}

func (l *BroadcastListener) Chan() chan interface{} {
	return l.ch
}

func (l *BroadcastListener) Read() interface{} {
	return <-l.ch
}

func (l *BroadcastListener) Close() {
	l.broadcaster.finishedChan <- l.ch
}

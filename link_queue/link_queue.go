package link_queue

import "sync"

type LinkQueue struct {
	store []string
	lock  sync.Mutex
}

func (q *LinkQueue) Push(link string) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.store = append(q.store, link)
}

func (q *LinkQueue) Pop() string {
	q.lock.Lock()
	defer q.lock.Unlock()
	link := q.store[0]
	q.store[0] = ""
	q.store = q.store[1:]
	return link
}

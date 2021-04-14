package link_queue

import (
	"fmt"
	"sync"
)

type LinkQueue struct {
	store []string
	lock  sync.Mutex
}

func (q *LinkQueue) Push(link string) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.store = append(q.store, link)
}

func (q *LinkQueue) Pop() (string, error) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if len(q.store) == 0 {
		return "", fmt.Errorf("queue is empty")
	}
	link := q.store[0]
	q.store[0] = ""
	q.store = q.store[1:]
	return link, nil
}

func (q *LinkQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return len(q.store)
}

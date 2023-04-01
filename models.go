package main

import "container/heap"

const outputTemplate = "%s\t%d"

var _ heap.Interface = (*PriorityQueue)(nil)

type Item struct {
	value      string
	batchIndex int
	priority   int
	index      int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].priority == pq[j].priority {
		return pq[i].batchIndex < pq[j].batchIndex
	}
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}

func (pq *PriorityQueue) Push(val any) {
	item := val.(*Item)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

type search struct {
	query string
	freq  *freq
}

type freq struct {
	count int
	pos   int
}

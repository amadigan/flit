package util

import (
	"iter"
	"slices"
	"sort"
)

type Ranges struct {
	subranges []Subrange
}

type Subrange struct {
	Start int64
	End   int64
}

func (r *Ranges) Add(start, end int64) {
	if start >= end {
		return
	}

	// binary search to find the correct position to insert the new range
	point := sort.Search(len(r.subranges), func(i int) bool {
		return r.subranges[i].Start >= start
	})

	// check if this overlaps with its neighbors
	if point > 0 && r.subranges[point-1].End >= start {
		r.subranges[point-1].End = max(r.subranges[point-1].End, end)
		return
	}

	if point < len(r.subranges) && r.subranges[point].Start <= end {
		r.subranges[point].Start = min(r.subranges[point].Start, start)
		r.subranges[point].End = max(r.subranges[point].End, end)
		return
	}

	// insert the new range at the correct position
	r.subranges = append(r.subranges, Subrange{})
	copy(r.subranges[point+1:], r.subranges[point:])
	r.subranges[point] = Subrange{Start: start, End: end}
}

func (r *Ranges) Remove(start, end int64) {
	if start >= end {
		return
	}

	// Find first range that could overlap [start, end).
	first := sort.Search(len(r.subranges), func(i int) bool {
		return r.subranges[i].End > start
	})

	if first == len(r.subranges) || r.subranges[first].Start >= end {
		return
	}

	// Find first range that starts at or after end; overlapping ranges are [first, last).
	last := first + sort.Search(len(r.subranges)-first, func(i int) bool {
		return r.subranges[first+i].Start >= end
	})

	newSubranges := make([]Subrange, 0, len(r.subranges)+1)
	newSubranges = append(newSubranges, r.subranges[:first]...)

	if r.subranges[first].Start < start {
		newSubranges = append(newSubranges, Subrange{Start: r.subranges[first].Start, End: start})
	}

	overlapTail := r.subranges[last-1]
	if overlapTail.End > end {
		newSubranges = append(newSubranges, Subrange{Start: end, End: overlapTail.End})
	}

	newSubranges = append(newSubranges, r.subranges[last:]...)

	r.subranges = newSubranges
}

func (r *Ranges) Get() []Subrange {
	return slices.Clone(r.subranges)
}

func (r *Ranges) All() iter.Seq[Subrange] {
	return func(yield func(Subrange) bool) {
		for _, subrange := range r.subranges {
			if !yield(subrange) {
				return
			}
		}
	}
}

func (r *Ranges) Len() int {
	return len(r.subranges)
}

func (r *Ranges) IsEmpty() bool {
	return len(r.subranges) == 0
}

func (r *Ranges) GetRange(i int) (Subrange, bool) {
	if i < 0 || i >= len(r.subranges) {
		return Subrange{}, false
	}
	return r.subranges[i], true
}

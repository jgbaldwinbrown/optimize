package optimize

import (
	"fmt"
	"github.com/jgbaldwinbrown/iter"
)

type MSESummer struct {
	Sum float64
	Count float64
}

func (m *MSESummer) Update(y1, y2 float64) {
	m.Sum += (y1 - y2) * (y1 - y2)
}

func (m *MSESummer) MeanSquaredError() float64 {
	return m.Sum / m.Count
}

func (m *MSESummer) Reset() {
	m.Sum = 0
	m.Count = 0
}

func NewMseSummer() *MSESummer {
	return &MSESummer{}
}

type Pair struct {
	Y1 float64
	Y2 float64
}

func (m *MSESummer) IterMSE(it iter.Iter[Pair]) (float64, error) {
	m.Reset()
	e := it.Iterate(func(p Pair) error {
		m.Update(p.Y1, p.Y2)
		return nil
	})
	return m.MeanSquaredError(), e
}

func Zip(s1 []float64, s2 []float64) (*iter.Iterator[Pair], error) {
	if len(s1) != len(s2) {
		return nil, fmt.Errorf("len(s1) %v != len(s2) %v", len(s1), len(s2))
	}

	return &iter.Iterator[Pair]{Iteratef: func(yield func(Pair) error) error {
		for i, f1 := range s1 {
			f2 := s2[i]
			e := yield(Pair{f1, f2})
			if e != nil {
				return e
			}
		}
		return nil
	}}, nil
}

type IOPair[T any] struct {
	In T
	Out float64
}

func FuncPair[T any](f func(T) float64, pairs iter.Iter[IOPair[T]]) *iter.Iterator[Pair] {
	return &iter.Iterator[Pair]{Iteratef: func(yield func(Pair) error) error {
		return pairs.Iterate(func(ip IOPair[T]) error {
			return yield(Pair{f(ip.In), ip.Out})
		})
	}}
}

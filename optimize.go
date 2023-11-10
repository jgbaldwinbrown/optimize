package optimize

import (
	"time"
	"fmt"
	"math/rand"
	"math"
)

type Func func(args ...float64) (float64, error)

type OptimizerArgs struct {
	Rand *rand.Rand
	Func Func
	Nargs int
	Start []float64
	Limits [][]float64
	Target float64
	Step float64
	Maxiter int
	Replicates int
}

type Optimizer struct {
	OptimizerArgs
	best []float64
	bestScore float64
	guess []float64
	guessScore float64
	bestGuess []float64
	bestGuessScore float64
	iterations int
	err error
}

func NewOptimizer(a OptimizerArgs) *Optimizer {
	o := new(Optimizer)
	o.OptimizerArgs = a
	return o
}

func DefaultOptimizerArgs(f Func, nargs int) OptimizerArgs {
	o := OptimizerArgs{}
	o.Func = f
	o.Nargs = nargs
	o.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	o.Target = 0.0
	o.Step = 1.0
	o.Maxiter = 10000
	o.Replicates = 1000000
	o.Limits = NewLimits(o.Nargs)
	o.Start = NewArgs(o.Nargs)
	return o
}

func NewLimits(nlims int) [][]float64 {
	l := make([][]float64, 0, nlims)
	for i := 0; i < nlims; i++ {
		l = append(l, []float64{math.Inf(-1), math.Inf(1)})
	}
	return l
}

func NewArgs(nargs int) []float64 {
	a := make([]float64, 0, nargs)
	for i := 0; i < nargs; i++ {
		a = append(a, 1.0)
	}
	return a
}

func makeGuess(dst []float64, src []float64, step float64, r *rand.Rand, limits [][]float64) []float64 {
	dst = dst[:0]
	for i, f := range src {
		roll := r.Float64() - 0.5
		guess := f + (roll * step)

		if guess < limits[i][0] {
			guess = limits[i][0]
		}
		if guess > limits[i][1] {
			guess = limits[i][1]
		}

		dst = append(dst, guess)
	}
	return dst
}

func (o *Optimizer) Handle(e error) ([]float64, int, error) {
	return o.best, o.iterations, fmt.Errorf("Optimize: guess: %v; iterations: %v; score: %v; error: %w", o.guess, o.iterations, o.guessScore, e)
}

func (o *Optimizer) Guess() error {
	h := func(e error) error {
		return fmt.Errorf("Optimizer.Guess: %w", e)
	}

	o.guess = makeGuess(o.guess, o.best, o.Step, o.Rand, o.Limits)
	if o.guessScore, o.err = o.Func(o.guess...); o.err != nil {
		return h(o.err)
	}

	if o.guessScore > o.bestGuessScore {
		o.bestGuessScore = o.guessScore
		copy(o.bestGuess, o.guess)
	}

	return nil
}

func (o *Optimizer) Optimize() ([]float64, int, error) {
	h := func(e error) ([]float64, int, error) {
		return o.Handle(e)
	}

	o.best = make([]float64, len(o.Start))
	copy(o.best, o.Start)
	if o.bestScore, o.err = o.Func(o.best...); o.err != nil {
		return h(o.err)
	}

	for o.iterations = 0; o.iterations < o.Maxiter; o.iterations++ {
		o.bestGuess = append(o.bestGuess[:0], o.best...)
		o.bestGuessScore = o.bestScore

		for rep := 0; rep < o.Replicates; rep++ {
			if e := o.Guess(); e != nil {
				return h(e)
			}
		}

		oldBestScore := o.bestScore
		if o.bestGuessScore > o.bestScore {
			o.bestScore = o.bestGuessScore
			copy(o.best, o.bestGuess)
		}
		if o.bestScore - oldBestScore <= o.Target {
			fmt.Println("o.bestScore:", o.bestScore, "o.BestGuessScore:", oldBestScore, "diff:", o.bestScore - oldBestScore)
			break
		}
	}

	return o.best, o.iterations, nil
}

func Neg(f Func) Func {
	return func(args ...float64) (float64, error) {
		out, err := f(args...)
		if err != nil {
			return 0, err
		}
		return -out, nil
	}
}

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
	Steps []float64
	Maxiter int
	Replicates int
	ReplicateSets []int
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

func LimitedReplicateSets(max int) []int {
	out := []int{32}
	for i := 32; i < max; i *= 2 {
		out = append(out, i)
	}
	return out
}

func DefaultReplicateSets() []int {
	return LimitedReplicateSets(10000)
}

func DefaultOptimizerArgs(f Func, nargs int) OptimizerArgs {
	o := OptimizerArgs{}
	o.Func = f
	o.Nargs = nargs
	o.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	o.Target = 0.0
	o.Step = 1.0
	o.Maxiter = 10000
	o.Replicates = 10000
	o.ReplicateSets = DefaultReplicateSets()
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

func makeGuess(dst []float64, src []float64, step float64, steps []float64, r *rand.Rand, limits [][]float64) []float64 {
	dst = dst[:0]
	for i, f := range src {
		istep := step
		if steps != nil {
			istep = steps[i]
		}

		roll := r.Float64() - 0.5
		guess := f + (roll * istep)

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
	o.guess = makeGuess(o.guess, o.best, o.Step, o.Steps, o.Rand, o.Limits)
	if o.guessScore, o.err = o.Func(o.guess...); o.err != nil {
		return fmt.Errorf("Optimizer.Guess: %w", o.err)
	}

	if o.guessScore > o.bestGuessScore {
		o.bestGuessScore = o.guessScore
		copy(o.bestGuess, o.guess)
	}

	return nil
}

func (o *Optimizer) updateSteps() {
	if o.Steps == nil {
		for i := 0; i < o.Nargs; i++ {
			o.Steps = append(o.Steps, o.Step)
		}
	}

	for i := 0; i < o.Nargs; i++ {
		o.Steps[i] = (o.Steps[i] + math.Abs(o.best[i] - o.bestGuess[i])) * 0.7
	}
}

func (o *Optimizer) UpdateBest() (continueLoop bool) {
	oldBestScore := o.bestScore
	if o.bestGuessScore > o.bestScore {
		o.updateSteps()

		o.bestScore = o.bestGuessScore
		copy(o.best, o.bestGuess)
	}
	if o.bestScore - oldBestScore <= o.Target {
		fmt.Println("o.bestScore:", o.bestScore, "o.BestGuessScore:", oldBestScore, "diff:", o.bestScore - oldBestScore)
		return false
	}

	return true
}

func (o *Optimizer) GuessRound() (continueLoop bool, err error) {
	o.bestGuess = append(o.bestGuess[:0], o.best...)
	o.bestGuessScore = o.bestScore

	if o.ReplicateSets == nil {
		for rep := 0; rep < o.Replicates; rep++ {
			if e := o.Guess(); e != nil {
				return false, fmt.Errorf("Optimizer.GuessRound: %w", e)
			}
		}
	} else {
		for _, reps := range o.ReplicateSets {
			for rep := 0; rep < reps; rep++ {
				if e := o.Guess(); e != nil {
					return false, fmt.Errorf("Optimizer.GuessRound: %w", e)
				}
			}
			if o.bestGuessScore > o.bestScore {
				break
			}
		}
	}

	continueLoop = o.UpdateBest()
	return continueLoop, nil
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
		continueLoop, e := o.GuessRound()
		if e != nil {
			return h(e)
		}
		if !continueLoop {
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

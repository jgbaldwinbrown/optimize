package optimize

import (
	"sync"
	"golang.org/x/sync/errgroup"
	"time"
	"fmt"
	"math/rand"
	"math"
	"log"
)

type Func func(args ...float64) (float64, error)

type OptimizerArgs struct {
	Rand *rand.Rand
	Func Func
	Nargs int
	Start []float64
	Limits [][]float64
	Target float64
	Steps []float64
	Maxiter int
	ReplicateSets []int
	Verbose bool
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

func Rep[T any](x T, n int) []T {
	a := make([]T, 0, n)
	for i := 0; i < n; i++ {
		a = append(a, x)
	}
	return a
}

func DefaultOptimizerArgs(f Func, nargs int) OptimizerArgs {
	o := OptimizerArgs{}
	o.Func = f
	o.Nargs = nargs
	o.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	o.Target = 0.0
	o.Steps = Rep(1.0, nargs)
	o.Maxiter = 10000
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

func makeGuessLocked(dst []float64, src []float64, steps []float64, r *rand.Rand, limits [][]float64, mu *sync.Mutex) []float64 {
	mu.Lock()
	defer mu.Unlock()
	return makeGuess(dst, src, steps, r, limits)
}

func makeGuess(dst []float64, src []float64, steps []float64, r *rand.Rand, limits [][]float64) []float64 {
	dst = dst[:0]
	for i, f := range src {
		istep := steps[i]

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

type GuessSet struct {
	Guess []float64
	GuessScore float64
}

func (o *Optimizer) GuessN(n int, mu *sync.Mutex) error {
	var g errgroup.Group

	guesses := make([]GuessSet, n)

	for i := 0; i < n; i++ {
		i := i
		g.Go(func() error {
			guesses[i].Guess = makeGuessLocked(guesses[i].Guess, o.best, o.Steps, o.Rand, o.Limits, mu)
			var err error
			if guesses[i].GuessScore, err = o.Func(guesses[i].Guess...); err != nil {
				return fmt.Errorf("Optimizer.Guess: %w", err)
			}
			return nil
		})
	}
	if e := g.Wait(); e != nil {
		return e
	}

	for _, guess := range guesses {
		if guess.GuessScore > o.bestGuessScore {
			o.bestGuessScore = guess.GuessScore
			copy(o.bestGuess, guess.Guess)
		}
	}

	return nil
}

func (o *Optimizer) updateSteps() {
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

func (o *Optimizer) GuessRound(mu *sync.Mutex) (continueLoop bool, err error) {
	o.bestGuess = append(o.bestGuess[:0], o.best...)
	o.bestGuessScore = o.bestScore

	for _, reps := range o.ReplicateSets {
		if e := o.GuessN(reps, mu); e != nil {
				return false, fmt.Errorf("Optimizer.GuessRound: %w", e)
		}
		if o.bestGuessScore > o.bestScore {
			break
		}
	}

	continueLoop = o.UpdateBest()
	return continueLoop, nil
}

func (o *Optimizer) LogArgsVerbose() {
	log.Printf("Optimizer Args:\n%#v\n", o.OptimizerArgs)
}

func (o *Optimizer) LogVerbose() {
	log.Printf("Optimizer:\n%#v\n", *o)
}

func (o *Optimizer) Optimize(mu *sync.Mutex) ([]float64, int, error) {
	h := func(e error) ([]float64, int, error) {
		return o.Handle(e)
	}

	o.best = make([]float64, len(o.Start))
	copy(o.best, o.Start)
	if o.bestScore, o.err = o.Func(o.best...); o.err != nil {
		return h(o.err)
	}

	if o.Verbose {
		o.LogArgsVerbose()
	}

	for o.iterations = 0; o.iterations < o.Maxiter; o.iterations++ {
		continueLoop, e := o.GuessRound(mu)
		if e != nil {
			return h(e)
		}
		if o.Verbose {
			o.LogVerbose()
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

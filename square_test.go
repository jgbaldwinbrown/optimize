package optimize

import (
	"testing"
	"fmt"
)

func square(fs ...float64) (float64, error) {
	return fs[0] * fs[0], nil
}

func TestSquare(t *testing.T) {
	o := NewOptimizer(DefaultOptimizerArgs(Neg(square), 1))
	args, niter, err := o.Optimize()
	fmt.Println(args, niter, err)
}

func mult(fs ...float64) (float64, error) {
	return fs[0] * (8.0 - fs[0]), nil
}

func TestMult(t *testing.T) {
	o := NewOptimizer(DefaultOptimizerArgs(mult, 1))
	o.Limits[0][0] = 1.0
	o.Limits[0][1] = 8.0
	args, niter, err := o.Optimize()
	fmt.Println(args, niter, err)
}


func TestMultSteps(t *testing.T) {
	o := NewOptimizer(DefaultOptimizerArgs(mult, 1))
	o.Limits[0][0] = 1.0
	o.Limits[0][1] = 8.0
	o.Steps = []float64{0.5}
	args, niter, err := o.Optimize()
	fmt.Println(args, niter, err)
}

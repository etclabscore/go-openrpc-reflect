package fakearithmetic

import (
	"context"
	"errors"
	"math"
	"math/big"

	"github.com/etclabscore/go-openrpc-reflect/internal/fakegeometry"
)

var CalculatorPublicMethodNames = []string{
	"HasBatteries",
	"Add",
	"Mul",
	"BigMul",
	"Div",
	"IsZero",
	"History",
	"Last",
	"GetRecord",
	"Reset",
}

type Calculator struct {
	h []HistoryItem
	r *Record
}

type HistoryItem struct {
	Method string
	Args   []interface{}
}

type Record struct {
	Count      int `json:"total_calculator_use"`
	Operations OpTally
}

type OpTally struct {
	m map[string]int
}

func newOpTally() OpTally {
	return OpTally{m: make(map[string]int)}
}

// memoryReset will not be an RPC eligible method because it is not exported.
func (c *Calculator) memoryReset() {
	c.h = []HistoryItem{}
	c.r = &Record{
		Count:      0,
		Operations: newOpTally(),
	}
}

func (c *Calculator) storeLatest(operationName string, args ...interface{}) {
	c.h = append(c.h, HistoryItem{operationName, []interface{}{args}})
	c.r.Count++
	_, ok := c.r.Operations.m[operationName]
	if !ok {
		c.r.Operations.m[operationName] = 1
		return
	}
	c.r.Operations.m[operationName]++
}

// Following methods implement various signatures and return values.

//HasBatteries checks whether the calculator has batteries.
func (c *Calculator) HasBatteries() bool {
	c.storeLatest("HasBatteries")
	return true
}

// Add adds two integers together.
func (c *Calculator) Add(argA, argB int) int {
	c.storeLatest("Add", argA, argB)
	return 0
}

type Pi struct{}

// ConstructCircle makes a circle.
// It returns an unnamed external package type pointer.
func (c *Calculator) ConstructCircle(x, y float64, radius float64) *fakegeometry.Circle {
	return &fakegeometry.Circle{
		Radius: radius,
		X:      x,
		Y:      y,
	}
}

// GuessAreaOfCircle returns a pretty good guess.
// It accepts an unnamed external package pointer as its only argument.
func (c *Calculator) GuessAreaOfCircle(*Pi, *fakegeometry.Circle) float64 {
	return 42
}

// Mul multiplies the arguments.
func (c *Calculator) Mul(argA int, argB int) (int, error) {
	c.storeLatest("Mul", argA, argB)
	if argA == 0 || argB == 0 {
		return math.MaxInt8, errors.New("this calculator doesn't handle multiplication by zero")
	}
	return argA * argB, nil
}

// BigMul returns a new *big.Int, the product of argA and argB.
func (c *Calculator) BigMul(argA, argB *big.Int) *big.Int {
	n := new(big.Int)
	n.Mul(argA, argB)
	return n
}

// Div doesn's actually do anything. You should use Mul instead.
// Warning: Deprecated.
func (c *Calculator) Div(int, int) error {
	c.storeLatest("Div")
	return errors.New("disused")
}

// IsZero tells you if a number is zero. People love this one.
func (c *Calculator) IsZero(argA int) bool {
	c.storeLatest("IsZero", argA)
	return argA == 0
}

// History returns the complete history of the calculator since it was last reset.
func (c *Calculator) History() []HistoryItem {
	return c.h
}

// Last returns the last command the calculator did.
func (c *Calculator) Last() (calculation *HistoryItem) {
	if len(c.h) == 0 {
		return nil
	}
	return &c.h[len(c.h)-1]
}

// GetRecord returns a special data type with a total use count and a tally of done operations.
func (c *Calculator) GetRecord() *Record {
	return c.r
}

// Reset clears the calculator memory.
func (c *Calculator) Reset() {
	c.memoryReset()
}

// ThreeRandomNumbers returns three psdeuo-random numbers.
func (c *Calculator) ThreePseudoRandomNumbers() (int, int, int) {
	return 1, 3, 2
}

// Latest error returns the latest error the calculator encountered.
// It is not usually an RPC-eligible method because it returns an error as
// the first value (go-ethereum), and returns 2 values (standard).
func (c *Calculator) LatestError() (error, bool) {
	return nil, true
}

// AddWithContext has context.Context as its first parameter,
// which ethereum/go-ethereum/rpc will skip.
func (c *Calculator) SumWithContext(ctx context.Context, number int) (int, error) {
	t := ctx.Value("target")
	return t.(int) + 1, nil
}

//  Implement an RPC interface for the calculator.

// CalculatorRPC wraps Calculator to provide an standard RPC service
// (a receiver that satifies method signature requirements for standard rpc package).
type CalculatorRPC struct {
	*Calculator
}

type HasBatteriesArg string
type HasBatteriesReply bool

// HasBatteries returns true if the calculator has batteries.
func (c *CalculatorRPC) HasBatteries(arg HasBatteriesArg, reply *HasBatteriesReply) error {
	*reply = HasBatteriesReply(c.Calculator.HasBatteries())
	return nil
}

type AddArg struct {
	A int `json:"a"`
	B int `json:"b"`
}
type AddReply int

// Add sums the A and B fields of the argument.
func (c *CalculatorRPC) Add(arg AddArg, reply *AddReply) error {
	*reply = AddReply(c.Calculator.Add(arg.A, arg.B))
	return nil
}

type BigMulArg struct {
	A *big.Int `json:"a"`
	B *big.Int
}
type BigMulReply big.Int

func (c *CalculatorRPC) BigMul(arg BigMulArg, reply *BigMulReply) error {
	(*big.Int)(reply).Mul(arg.A, arg.B)
	return nil
}

type DivArg struct {
	A int `json:"a"`
	B int `json:"b"`
}
type DivReply int

// Div is deprecated. Use Mul instead.
func (c *CalculatorRPC) Div(arg DivArg, reply *DivReply) error {
	return errors.New("disused")
}

// Mul multiplies the arguments,
// but WILL NOT be eligible in standard RPC; wrong signature.
func (c *CalculatorRPC) Mul(argA int, argB int) (int, error) {
	c.storeLatest("Mul", argA, argB)
	if argA == 0 || argB == 0 {
		return math.MaxInt8, errors.New("this calculator doesn't handle multiplication by zero")
	}
	return argA * argB, nil
}

type IsZeroArg bool

// IsZero has throwaway parameters.
func (c *CalculatorRPC) IsZero(big.Int, *IsZeroArg) error {
	return errors.New("throwaway args")
}

func (c *CalculatorRPC) BrokenReset() {
	// Will not be eligible.
}

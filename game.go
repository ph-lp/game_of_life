package main

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"time"

	"github.com/bsipos/thist"
)

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}

	return a
}

func getSum(g [][]bool) int {
	var sum int = 0
	var ix1 int = 0
	var ix2 int

	for ix1 < len(g) {
		ix2 = 0
		for ix2 < len(g) {
			if g[ix1][ix2] {
				sum++
			}
			ix2++
		}
		ix1++
	}
	return sum
}

func getBox(g [][]bool, ix1 int, ix2 int) [][]bool {
	box := initGrid(3, 3, 0)
	for c1, i1 := range makeRange(ix1-1, ix1+1) {
		if i1 == -1 {
			i1 += len(g)
		} else if i1 == len(g) {
			i1 -= len(g)
		}
		for c2, i2 := range makeRange(ix2-1, ix2+1) {
			if i2 == -1 {
				i2 += len(g)
			} else if i2 == len(g) {
				i2 -= len(g)
			}

			box[c1][c2] = g[i1][i2]
		}
	}

	return box
}

func initGrid(nGrid int, nStart int, startProb float64) [][]bool {

	if nGrid < 3 {
		nGrid = 3
	}
	if nGrid%2 != 1 {
		fmt.Printf("nGrid: %d not uneven - change to %d\n", nGrid, nGrid-1)
		nGrid--
	}

	if nStart < 3 {
		nStart = 3
	}
	if nStart%2 != 1 {
		fmt.Printf("nStart: %d not uneven - change to %d\n", nStart, nStart-1)
		nStart--
	}

	board := make([][]bool, nGrid)
	for iii := 0; iii < nGrid; iii++ {
		board[iii] = make([]bool, nGrid)
	}

	if startProb > 0 {
		var center int = (nGrid - 1) / 2
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		for _, ix1 := range makeRange(center-(nStart-1)/2, center+(nStart-1)/2) {
			for _, ix2 := range makeRange(center-(nStart-1)/2, center+(nStart-1)/2) {

				if r1.Float64() <= startProb {
					board[ix1][ix2] = true
				}
			}
		}
	}

	return board
}

type iterTrig struct {
	Ix1 int
	Ix2 int
	box [][]bool
}

type iterRes struct {
	Ix1     int
	Ix2     int
	updated bool
}

func updateCellWorker(inChan <-chan iterTrig, resChan chan<- iterRes) {

	for it := range inChan {
		var aliveNeighbours int
		if it.box[1][1] {
			aliveNeighbours = getSum(it.box) - 1
			if aliveNeighbours < 2 || aliveNeighbours > 3 {
				resChan <- iterRes{Ix1: it.Ix1, Ix2: it.Ix2, updated: false}
				continue
			}
		} else {
			aliveNeighbours = getSum(it.box)
			if aliveNeighbours == 3 {
				resChan <- iterRes{Ix1: it.Ix1, Ix2: it.Ix2, updated: true}
				continue
			}
		}

		resChan <- iterRes{Ix1: it.Ix1, Ix2: it.Ix2, updated: it.box[1][1]}
	}
}

func iterate(g [][]bool) ([][]bool, bool) {

	numJobs := len(g) * len(g)
	// numJobs := (len(g) - 2) * (len(g) - 2)
	iJobs := make(chan iterTrig, numJobs)
	iResults := make(chan iterRes, numJobs)

	for w := 1; w <= 50; w++ {
		// gameWorker plays game of life with a parameter set and returns number of iterations until steady state
		go updateCellWorker(iJobs, iResults)
	}

	newGrid := initGrid(len(g), 0, 0)
	var changed bool = true
	var box [][]bool
	for _, ix1 := range makeRange(0, len(g)-1) {
		for _, ix2 := range makeRange(0, len(g)-1) {
			// for _, ix1 := range makeRange(1, len(g)-2) {
			// 	for _, ix2 := range makeRange(1, len(g)-2) {
			if getSum(g) > 0 {
				box = getBox(g, ix1, ix2)
				iJobs <- iterTrig{Ix1: ix1, Ix2: ix2, box: box}
			} else {
				iResults <- iterRes{Ix1: ix1, Ix2: ix2, updated: false}
			}
		}
	}
	close(iJobs)

	// receive results
	for a := 1; a <= numJobs; a++ {
		res := <-iResults
		newGrid[res.Ix1][res.Ix2] = res.updated
	}
	// close(iResults)

	if reflect.DeepEqual(g, newGrid) {
		changed = false
	}

	return newGrid, changed
}

func gameWorker(triggerChan <-chan int, out chan<- float64, nGrid int, nStart int, p float64) {

	// make sure a worker only sends 1 for every trigger
	for range triggerChan {

		grid := initGrid(nGrid, nStart, p)
		var changed bool

		for t := 0; t < 200; t++ {
			grid, changed = iterate(grid)
			if !changed {
				// fmt.Println("stopped!", t)
				// stops = append(stops, t)
				out <- float64(t)
				break
			}
		}
		// to close the worker loop we check if it still has changed after all loops
		if changed {
			out <- float64(math.NaN())
		}

	}
}

func myArgMax(l []float64) int {
	var m float64 = 0
	var am int = 0
	for ix, v := range l {
		if v > m {
			m = v
			am = ix
		}
	}

	return am
}

func main() {

	// maybe use cyclic boundaries? DONE!
	// is a map more efficient?

	var nGrid int = 9
	var deltaP float64 = 0.1
	fmt.Println("game of life starts")

	for nStart := 3; nStart <= nGrid; nStart += 2 {
		fmt.Printf("nGrid: %d nStart: %d\n", nGrid, nStart)
		for p := deltaP; p < 1; p += deltaP {

			var numJobs int = 1000
			// var nGrid int = 31
			// var nGrid, nStart int = 31, 21
			// var p float64 = 0.4

			jobs := make(chan int, numJobs)
			results := make(chan float64, numJobs)

			for w := 1; w <= 100; w++ {
				// gameWorker plays game of life with a parameter set and returns number of iterations until steady state
				go gameWorker(jobs, results, nGrid, nStart, p)
			}

			for j := 1; j <= numJobs; j++ {
				jobs <- j
			}
			close(jobs)

			h := thist.NewHist(nil, "game of life", "fixed", 40, false)
			// hack to fix bins
			h.Update(0.0)
			h.Update(200.0)
			// start := time.Now()
			for a := 1; a <= numJobs; a++ {
				// fmt.Println(a)
				stop, more := <-results
				if more {
					if !math.IsNaN(stop) {
						h.Update(stop)
					}
					// if a%1000 == 0 {
					// 	fmt.Println(h.Draw())
					// }
				} else {
					close(results)
					break
				}
			}
			// elapsed := time.Since(start)
			// fmt.Println(elapsed)
			// fmt.Println(h.Draw())
			// fmt.Println(stops)
			// fmt.Println(h.Counts)
			fmt.Print(myArgMax(h.Counts), ",")
		}
		fmt.Println()
	}
}

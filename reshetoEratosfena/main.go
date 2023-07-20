package main

import (
	"fmt"
	"time"
)

func generateSlice(x int) []int {
	var sl []int
	for i := 2; i < x; i++ {
		sl = append(sl, i)
	}
	return sl
}

func main() {
	n := 50001
	items := generateSlice(n)

	t0 := time.Now()

	for i, p := range items {
		if p*p > n {
			break
		}
		for j := i + 1; j < len(items); j++ {
			if items[j]%p == 0 {
				items = append(items[:j], items[j+1:]...) // delete element
			}
		}
	}
	fmt.Printf("Elapsed time: %v\n", time.Since(t0))

	fmt.Printf("Для n = %d\nРезультат = %v", n-1, items)
}

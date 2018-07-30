package main

import (
	"fmt"
)

const A = 10
type data struct{
	ages []int
}

func main() {
	var a float64 = 0xFFFF * 0.25
	const X = 0xFFFF * 0.25
	b := int(a)
	fmt.Println(float64(b)<X)
	fmt.Println(a)
	fmt.Println(b)
}




package main

import "fmt"

const A = 10
type data struct{
	ages int
}

func main() {
	var m = make(map[string]*data)
	m["1"] = &data{
		1,
	}



	fmt.Println(m["2"])
	//d := data{2}
	//m["haha"] = &d
	//fmt.Println(m["a"])
	//fmt.Println(m["haha"].ages)
}





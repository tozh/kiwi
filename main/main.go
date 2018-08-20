package main

import "fmt"

type data struct{
	ages int
}

type IF interface {
	printAge()
}

func (d *data) printAge() {
	fmt.Print(d.ages)
}


func main() {
	var m = make(map[string]IF)
	m["1"] = &data{
		1,
	}
	m["2"] = &data{
		2,
	}

	a := m["2"]

	a.printAge()
	//d := data{2}
	//m["haha"] = &d
	//fmt.Println(m["a"])
	//fmt.Println(m["haha"].ages)
}





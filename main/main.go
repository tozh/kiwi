package main

import (
	"fmt"
)

//type data struct{
//	ages int
//}
//
//type IF interface {
//	printAge()
//}
//
//func (d *data) printAge() {
//	fmt.Print(d.ages)
//}
//type Object struct {
//	RType int64
//	Encoding int64
//
//}
//
//type StrObject struct {
//	Object
//	Value *string
//}
//
//type IntObject struct {
//	Object
//	Value *int64
//}
//type IObject interface {
//	getRType() int64
//}
//
//func (obj *Object) getRType() int64 {
//	return obj.RType
//}

func main() {
	s := "123456"
	t := s
	fmt.Println(&s)
	fmt.Println(&t)
	//so := StrObject {
	//	Object {
	//		1,
	//		1,
	//	},
	//	&s,
	//}
	//p := TryObjectEncoding(IObject(&so)).(*IntObject)
	//fmt.Println(*p.Value)
}

//func TryObjectEncoding(o IObject) IObject {
//	//length := len(*o.(*StrObject).Value)
//	value, err := strconv.ParseInt(*o.(*StrObject).Value, 10, 64)
//
//	fmt.Println(value)
//	value += 1
//	if err == nil {
//		x := StrObjectToIntObject(o.(*StrObject), value)
//		fmt.Println(x)
//		fmt.Println(*x.Value)
//		return x
//	}
//	return o
//}
//
//func StrObjectToIntObject(o *StrObject, value int64) *IntObject{
//	fmt.Println(&value)
//
//	newObj := IntObject{
//		o.Object,
//		&value,
//	}
//	fmt.Println(&newObj)
//	return &newObj
//}






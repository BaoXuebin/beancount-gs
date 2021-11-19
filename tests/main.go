package tests

import "fmt"

type Student struct {
	name string
}

func say(student Student) *Student {
	return &student
}

func say2(student Student) Student {
	return student
}

func Test() {
	str := "Hello~"
	fmt.Println(str)
	fmt.Println(&str)

	student := Student{name: "Bao"}
	fmt.Println(student)
	fmt.Println(&student)
}

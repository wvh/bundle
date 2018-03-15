package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	fmt.Println(`Hello, World!`)
	fmt.Printf("%s\n", []byte("Hello, World!"))
	fmt.Printf("%s\n", []byte{'H', 'e', 'l', 'l', 'o', ',', ' ', 'W', 'o', 'r', 'l', 'd', '!'})
}

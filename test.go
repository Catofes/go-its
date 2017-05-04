package main

import (
	"fmt"
	"time"
)

type d struct {
	name int
}

type Handler func(int) int

type h struct {
	handlers map[int]Handler
}

func (s *d) test(input int) int {
	fmt.Println("inner", s.name)
	return s.name
}

func (s *h) AddHandler(input_handler Handler) {
	if s.handlers == nil {
		s.handlers = make(map[int]Handler)
	}
	s.handlers[0] = input_handler
}

func (s *h) Call() {
	fmt.Println("outer", s.handlers[0](0))
}

func main() {
	data := d{}
	hh := h{}
	hh.AddHandler(data.test)
	hh.Call()
	data.name = 2
	hh.Call()
	fmt.Println(time.Now().Unix())
}

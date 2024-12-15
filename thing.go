package main

import (
	"strconv"
	"strings"
)

type Thing struct {
	raw string
}

func (t Thing) String() string {
	return t.raw
}

func (t Thing) List() []string {
	return strings.Split(t.raw, " ")
}

func (t Thing) Float() float64 {
	v, err := strconv.ParseFloat(t.raw, 64)
	if err != nil {
		return 0.0
	}
	return v
}

func (t Thing) Int() int64 {
	v, err := strconv.ParseInt(t.raw, 10, 64)
	if err != nil {
		return 0.0
	}
	return v
}

func (t Thing) Bool() bool {
	if t.raw != "" && t.raw != "0" && t.raw != "false" && t.raw != "no" {
		return true
	}
	return false
}

func NewThing(raw string) Thing {
	raw = strings.Trim(raw, " \t\n")
	return Thing{
		raw,
	}
}

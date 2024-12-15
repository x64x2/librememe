package main

import "io"

type Log struct {
	output io.Writer
}

func (l Log) Println(args ...string) {
	if len(args) == 0 {
		return
	}
	l.output.Write([]byte(args[0]))
	for _, arg := range args[1:] {
		l.output.Write([]byte{' '})
		l.output.Write([]byte(arg))
	}
	l.output.Write([]byte{'\n'})
}

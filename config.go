package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Config map[string]Thing

func NewConfig(configpath string) Config {
	config := map[string]Thing{
		"host":          NewThing("[::]"),
		"port":          NewThing("8293"),
		"stream":        NewThing("8296"),
		"oggdirs":       NewThing("ogg"),
		"lists":         NewThing("list.txt"),
		"log":           NewThing("true"),
		"sort_by":       NewThing("modtime"),
		"buffer":        NewThing("6000"),
		"target_in_pk3": NewThing("song.ogg"),
		"pidpath":       NewThing("xonobo-go.pid"),
		"fdpath":        NewThing("xonobo-go.ctlfd"),
	}
	file, err := os.Open(configpath)
	if err != nil {
		return config
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return config
		}
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		key, value, found := strings.Cut(line, " ")
		key = strings.Trim(key, " \t")
		if !found {
			config[key] = NewThing("")
			continue
		}
		config[key] = NewThing(value)
	}
}

func (c Config) FetchMsg(msgchan chan Thing) {
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Println("cannot initialize a communication pipe:", err)
		return
	}
	defer r.Close()
	defer w.Close()
	reader := bufio.NewReader(r)
	fmt.Printf(
		"send commands though file descriptor %d of pid %d\n"+
			"example: echo reload >>/proc/%d/fd/%d\n",
		w.Fd(), os.Getpid(), os.Getpid(), w.Fd(),
	)
	os.WriteFile(c["pidpath"].raw, []byte(fmt.Sprintf("%d", os.Getpid())), 0666)
	os.WriteFile(c["fdpath"].raw, []byte(fmt.Sprintf("%d", w.Fd())), 0666)
	go func() {
		stdin := bufio.NewReader(os.Stdin)
		for {
			line, err := stdin.ReadString('\n')
			switch err {
			case nil:
				w.WriteString(line)
			case io.EOF:
				// fmt.Println("exit")
				// w.WriteString("exit\n")
				return
			default:
				return
			}
		}
	}()
	for {
		line, err := reader.ReadString('\n')
		switch err {
		case nil:
			// break
		case io.EOF:
			fmt.Println("exit")
			msgchan <- NewThing("exit")
		default:
			fmt.Println("broken communication pipe:", err)
			return
		}
		line = strings.Trim(line, " \t\n")
		msgchan <- NewThing(line)
	}
}

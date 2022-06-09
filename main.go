package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// MARK: - commandsQueue

type commandsQueue struct {
	queue []Command
	mu    sync.Mutex
}

func (q *commandsQueue) pull() Command {
	q.mu.Lock()
	defer q.mu.Unlock()

	cmd := q.queue[0]
	q.queue = q.queue[1:]
	return cmd
}

func (q *commandsQueue) peek() Command {
	if len(q.queue) == 0 {
		return nil
	}
	return q.queue[len(q.queue)-1]
}

func (q *commandsQueue) push(cmd Command) {
	q.mu.Lock()
	defer q.mu.Unlock()

	last := q.peek()
	if stop, ok := last.(*stopCommand); ok && stop != nil {
		q.queue[len(q.queue)-1] = cmd
		q.queue = append(q.queue, last)
		return
	}
	q.queue = append(q.queue, cmd)
}

// MARK: - EventLoop

type EventLoop struct {
	queue      *commandsQueue
	stopSignal chan struct{}
	isStopped  bool
}

func NewEventLoop() EventLoop {
	return EventLoop{
		queue:      &commandsQueue{queue: make([]Command, 0)},
		stopSignal: make(chan struct{}),
		isStopped:  false,
	}
}

func (l *EventLoop) Start() {
	go func() {
		for !l.isStopped {
			if len(l.queue.queue) == 0 {
				continue
			}
			cmd := l.queue.pull()
			cmd.Execute(l)
		}
		l.stopSignal <- struct{}{}
	}()
}

func (l *EventLoop) Stop() {
	l.isStopped = true
}

func (l *EventLoop) Post(cmd Command) {
	l.queue.push(cmd)
}

func (l *EventLoop) AwaitFinish() {
	l.Post(&stopCommand{})
	<-l.stopSignal
}

type Command interface {
	Execute(handler Handler)
}

type Handler interface {
	Post(cmd Command)
	Stop()
}

// MARK: - Commands

type stopCommand struct{}

func (s *stopCommand) Execute(handler Handler) {
	handler.Stop()
}

type printCommand struct {
	arg string
}

func (p *printCommand) Execute(handler Handler) {
	fmt.Println(p.arg)
}

type addCommand struct {
	arg1, arg2 int
}

func (add *addCommand) Execute(handler Handler) {
	res := add.arg1 + add.arg2
	handler.Post(&printCommand{arg: strconv.Itoa(res)})
}

func parse(line string) Command {
	parts := strings.Fields(line)
	command, args := parts[0], parts[1:]
	switch command {
	case "print":
		return &printCommand{arg: args[0]}
	case "add":
		arg1, _ := strconv.Atoi(args[0])
		arg2, _ := strconv.Atoi(args[1])
		return &addCommand{arg1, arg2}
	}
	return nil
}

var inputPath = flag.String("f", "", "Path to file with instructions")

func main() {
	flag.Parse()
	input, err := os.Open(*inputPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	eventLoop := NewEventLoop()
	eventLoop.Start()

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		commandLine := scanner.Text()
		if cmd := parse(commandLine); cmd != nil {
			eventLoop.Post(cmd)
		}
	}
	input.Close()
	eventLoop.AwaitFinish()
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode"
)

// --- AST Definitions ---

type Command struct {
	Type string // "exec" or "print"
	Arg  string
}

type Task struct {
	Name         string
	Dependencies []string
	Commands     []Command
}

// --- Lexer ---

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenString
	TokenTask
	TokenDepends
	TokenExec
	TokenPrint
	TokenLBrace
	TokenRBrace
	TokenSemi
	TokenComma
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
}

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	if l.ch == '\n' {
		l.line++
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	var tok Token
	tok.Line = l.line

	switch l.ch {
	case '{':
		tok = Token{Type: TokenLBrace, Literal: "{"}
	case '}':
		tok = Token{Type: TokenRBrace, Literal: "}"}
	case ';':
		tok = Token{Type: TokenSemi, Literal: ";"}
	case ',':
		tok = Token{Type: TokenComma, Literal: ","}
	case '"':
		tok.Type = TokenString
		tok.Literal = l.readString()
	case 0:
		tok.Type = TokenEOF
		tok.Literal = ""
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			return tok
		}
		// Basic error handling for unknown chars could go here
	}
	l.readChar()
	return tok
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func lookupIdent(ident string) TokenType {
	switch ident {
	case "task":
		return TokenTask
	case "depends":
		return TokenDepends
	case "exec":
		return TokenExec
	case "print":
		return TokenPrint
	}
	return TokenIdent
}

// --- Parser ---

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
	errors    []string
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() []Task {
	var tasks []Task
	for p.curToken.Type != TokenEOF {
		task := p.parseTask()
		if task != nil {
			tasks = append(tasks, *task)
		} else {
			p.nextToken()
		}
	}
	return tasks
}

func (p *Parser) parseTask() *Task {
	if p.curToken.Type != TokenTask {
		p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected 'task', got '%s'", p.curToken.Line, p.curToken.Literal))
		return nil
	}
	p.nextToken()

	if p.curToken.Type != TokenIdent {
		p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected task name, got '%s'", p.curToken.Line, p.curToken.Literal))
		return nil
	}
	task := &Task{Name: p.curToken.Literal}
	p.nextToken()

	if p.curToken.Type == TokenDepends {
		p.nextToken()
		for {
			if p.curToken.Type != TokenIdent {
				p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected dependency name, got '%s'", p.curToken.Line, p.curToken.Literal))
				return nil
			}
			task.Dependencies = append(task.Dependencies, p.curToken.Literal)
			p.nextToken()
			if p.curToken.Type == TokenComma {
				p.nextToken()
			} else {
				break
			}
		}
	}

	if p.curToken.Type != TokenLBrace {
		p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected '{', got '%s'", p.curToken.Line, p.curToken.Literal))
		return nil
	}
	p.nextToken()

	for p.curToken.Type != TokenRBrace && p.curToken.Type != TokenEOF {
		cmd := p.parseCommand()
		if cmd != nil {
			task.Commands = append(task.Commands, *cmd)
		}
		p.nextToken()
	}
	return task
}

func (p *Parser) parseCommand() *Command {
	var cmd Command
	if p.curToken.Type == TokenExec || p.curToken.Type == TokenPrint {
		cmd.Type = p.curToken.Literal
		p.nextToken()
		if p.curToken.Type != TokenString {
			p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected string after '%s'", p.curToken.Line, cmd.Type))
			return nil
		}
		cmd.Arg = p.curToken.Literal
		p.nextToken()
		if p.curToken.Type != TokenSemi {
			p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected ';'", p.curToken.Line))
		}
		return &cmd
	}
	p.errors = append(p.errors, fmt.Sprintf("Line %d: Expected 'exec' or 'print', got '%s'", p.curToken.Line, p.curToken.Literal))
	return nil
}

// --- Executor ---

type Executor struct {
	tasks    map[string]Task
	status   map[string]string // "pending", "running", "done", "error"
	statusMu sync.Mutex
	wg       sync.WaitGroup
	errChan  chan error
}

func NewExecutor(tasks []Task) *Executor {
	ex := &Executor{
		tasks:  make(map[string]Task),
		status: make(map[string]string),
		errChan: make(chan error, len(tasks)),
	}
	for _, t := range tasks {
		ex.tasks[t.Name] = t
		ex.status[t.Name] = "pending"
	}
	return ex
}

func (ex *Executor) Run() error {
	// Simple cycle detection could go here
	for name := range ex.tasks {
		ex.wg.Add(1)
		go ex.runTask(name)
	}
	ex.wg.Wait()
	close(ex.errChan)

	var errs []string
	for err := range ex.errChan {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func (ex *Executor) runTask(name string) {
	defer ex.wg.Done()
	task, ok := ex.tasks[name]
	if !ok {
		ex.errChan <- fmt.Errorf("Task not found: %s", name)
		return
	}

	// Wait for dependencies
	for {
		ready := true
		ex.statusMu.Lock()
		for _, dep := range task.Dependencies {
			depStatus := ex.status[dep]
			if depStatus == "error" {
				ex.status[name] = "error"
				ex.statusMu.Unlock()
				ex.errChan <- fmt.Errorf("Task '%s' failed because dependency '%s' failed", name, dep)
				return
			}
			if depStatus != "done" {
				ready = false
				break
			}
		}
		ex.statusMu.Unlock()
		if ready {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Execute task
	ex.statusMu.Lock()
	ex.status[name] = "running"
	ex.statusMu.Unlock()

	fmt.Printf("[RUNNING] Task '%s' started\n", name)
	for _, cmd := range task.Commands {
		if cmd.Type == "print" {
			fmt.Printf("[%s] %s\n", name, cmd.Arg)
		} else if cmd.Type == "exec" {
			// Basic splitting for simplicity, a real implementation would use shlex
			parts := strings.Fields(cmd.Arg)
			if len(parts) > 0 {
				c := exec.Command(parts[0], parts[1:]...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				err := c.Run()
				if err != nil {
					ex.statusMu.Lock()
					ex.status[name] = "error"
					ex.statusMu.Unlock()
					ex.errChan <- fmt.Errorf("Task '%s' command failed: %v", name, err)
					return
				}
			}
		}
	}

	ex.statusMu.Lock()
	ex.status[name] = "done"
	ex.statusMu.Unlock()
	fmt.Printf("[DONE] Task '%s' finished\n", name)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: taskrunner <script.tr>")
		return
	}
	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	l := NewLexer(string(content))
	p := NewParser(l)
	tasks := p.ParseProgram()

	if len(p.errors) > 0 {
		fmt.Println("Parse errors:")
		for _, msg := range p.errors {
			fmt.Println(msg)
		}
		return
	}

	fmt.Println("Starting execution...")
	ex := NewExecutor(tasks)
	err = ex.Run()
	if err != nil {
		fmt.Println("\nExecution completed with errors:")
		fmt.Println(err)
	} else {
		fmt.Println("\nAll tasks completed successfully!")
	}
}

package ast

import "fmt"

type Pos struct {
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

type Node interface {
	Pos() Pos
	node()
}

// --- Script-level ---

type Script struct {
	Position    Pos
	Shebang     *ShebangDecl
	Compact     *CompactConfig
	Statements  []Node
}

func (n *Script) Pos() Pos { return n.Position }
func (n *Script) node()    {}

type ShebangDecl struct {
	Position Pos
	Engine   string
	Version  string
	Model    string
}

func (n *ShebangDecl) Pos() Pos { return n.Position }
func (n *ShebangDecl) node()    {}

type CompactConfig struct {
	Position Pos
	Mode     string
	Window   string
	Strategy string
	Raw      map[string]string
}

func (n *CompactConfig) Pos() Pos { return n.Position }
func (n *CompactConfig) node()    {}

// --- Statements ---

type VarAssign struct {
	Position Pos
	Name     string
	Value    Node
}

func (n *VarAssign) Pos() Pos { return n.Position }
func (n *VarAssign) node()    {}

type AgentCall struct {
	Position  Pos
	Prompt    Node
	Subagent  string
	Model     Node
	Dir       Node
	Compact   Node
}

func (n *AgentCall) Pos() Pos { return n.Position }
func (n *AgentCall) node()    {}

type FnDecl struct {
	Position Pos
	Name     string
	Params   []string
	Body     *Block
}

func (n *FnDecl) Pos() Pos { return n.Position }
func (n *FnDecl) node()    {}

type FnCall struct {
	Position Pos
	Name     string
	Args     []Node
}

func (n *FnCall) Pos() Pos { return n.Position }
func (n *FnCall) node()    {}

type Include struct {
	Position Pos
	Path     Node
}

func (n *Include) Pos() Pos { return n.Position }
func (n *Include) node()    {}

type Exec struct {
	Position Pos
	Cmd      Node
}

func (n *Exec) Pos() Pos { return n.Position }
func (n *Exec) node()    {}

type Print struct {
	Position Pos
	Message  Node
}

func (n *Print) Pos() Pos { return n.Position }
func (n *Print) node()    {}

type Exit struct {
	Position Pos
	Code     Node
}

func (n *Exit) Pos() Pos { return n.Position }
func (n *Exit) node()    {}

type Env struct {
	Position Pos
	Key      string
}

func (n *Env) Pos() Pos { return n.Position }
func (n *Env) node()    {}

type IfStmt struct {
	Position Pos
	Cond     Node
	Body     *Block
	ElseIfs  []ElseIf
	Else     *Block
}

func (n *IfStmt) Pos() Pos { return n.Position }
func (n *IfStmt) node()    {}

type ElseIf struct {
	Position Pos
	Cond     Node
	Body     *Block
}

func (n *ElseIf) Pos() Pos { return n.Position }
func (n *ElseIf) node()    {}

type ForStmt struct {
	Position Pos
	Var      string
	List     Node
	Body     *Block
}

func (n *ForStmt) Pos() Pos { return n.Position }
func (n *ForStmt) node()    {}

type WhileStmt struct {
	Position Pos
	Cond     Node
	Body     *Block
}

func (n *WhileStmt) Pos() Pos { return n.Position }
func (n *WhileStmt) node()    {}

type WaitBlock struct {
	Position Pos
	Body     *Block
}

func (n *WaitBlock) Pos() Pos { return n.Position }
func (n *WaitBlock) node()    {}

type Background struct {
	Position Pos
	Stmt     Node
}

func (n *Background) Pos() Pos { return n.Position }
func (n *Background) node()    {}

type DirBlock struct {
	Position Pos
	Dir      Node
	Body     *Block
}

func (n *DirBlock) Pos() Pos { return n.Position }
func (n *DirBlock) node()    {}

type CompactStmt struct {
	Position Pos
	Arg      Node
}

func (n *CompactStmt) Pos() Pos { return n.Position }
func (n *CompactStmt) node()    {}

type Return struct {
	Position Pos
	Value    Node
}

func (n *Return) Pos() Pos { return n.Position }
func (n *Return) node()    {}

type Break struct {
	Position Pos
}

func (n *Break) Pos() Pos { return n.Position }
func (n *Break) node()    {}

type Continue struct {
	Position Pos
}

func (n *Continue) Pos() Pos { return n.Position }
func (n *Continue) node()    {}

type Block struct {
	Position   Pos
	Statements []Node
}

func (n *Block) Pos() Pos { return n.Position }
func (n *Block) node()    {}

// --- Binary try ---

type BinaryTry struct {
	Position Pos
	Body     *Block
	Fail     *Block
	Max      int
}

func (n *BinaryTry) Pos() Pos { return n.Position }
func (n *BinaryTry) node()    {}

// --- Evaluated try ---

type EvalTry struct {
	Position Pos
	Body     *Block
	Eval     *Block
	Accept   *Block
	Partial  *Block
	Fail     *Block
	Max      int
}

func (n *EvalTry) Pos() Pos { return n.Position }
func (n *EvalTry) node()    {}

// --- Expressions ---

type BinaryExpr struct {
	Position  Pos
	Left      Node
	Right     Node
	Op        string
}

func (n *BinaryExpr) Pos() Pos { return n.Position }
func (n *BinaryExpr) node()    {}

type UnaryExpr struct {
	Position Pos
	Op       string
	Right    Node
}

func (n *UnaryExpr) Pos() Pos { return n.Position }
func (n *UnaryExpr) node()    {}

type VarRef struct {
	Position Pos
	Name     string
}

func (n *VarRef) Pos() Pos { return n.Position }
func (n *VarRef) node()    {}

type StringLiteral struct {
	Position   Pos
	Value      string
	Interps    []InterpSpan
}

func (n *StringLiteral) Pos() Pos { return n.Position }
func (n *StringLiteral) node()    {}

type InterpSpan struct {
	Text string
	Var  string
	Cmd  string
}

type TextBlock struct {
	Position Pos
	Value    string
	Interps  []InterpSpan
}

func (n *TextBlock) Pos() Pos { return n.Position }
func (n *TextBlock) node()    {}

type FilePath struct {
	Position Pos
	Path     Node
}

func (n *FilePath) Pos() Pos { return n.Position }
func (n *FilePath) node()    {}

type IntLiteral struct {
	Position Pos
	Value    int64
	Text     string
}

func (n *IntLiteral) Pos() Pos { return n.Position }
func (n *IntLiteral) node()    {}

type FloatLiteral struct {
	Position Pos
	Value    float64
	Text     string
}

func (n *FloatLiteral) Pos() Pos { return n.Position }
func (n *FloatLiteral) node()    {}

type BoolLiteral struct {
	Position Pos
	Value    bool
}

func (n *BoolLiteral) Pos() Pos { return n.Position }
func (n *BoolLiteral) node()    {}

type CommandSubst struct {
	Position Pos
	Cmd      string
}

func (n *CommandSubst) Pos() Pos { return n.Position }
func (n *CommandSubst) node()    {}

type ArrayLiteral struct {
	Position Pos
	Elements []Node
}

func (n *ArrayLiteral) Pos() Pos { return n.Position }
func (n *ArrayLiteral) node()    {}

type IndexExpr struct {
	Position Pos
	Object   Node
	Index    Node
}

func (n *IndexExpr) Pos() Pos { return n.Position }
func (n *IndexExpr) node()    {}

type GroupExpr struct {
	Position Pos
	Inner    Node
}

func (n *GroupExpr) Pos() Pos { return n.Position }
func (n *GroupExpr) node()    {}

package have

import (
	"bytes"
	"fmt"
)

type Expr interface {
	Pos() int
}

type expr struct {
	pos int
}

type Stmt interface {
	Expr
	Label() *Object
}

type stmt struct {
	expr
	label *Object
}

// Simple statements are those that can be used in the 3rd
// clause of the `for` loop.
type SimpleStmt interface {
	Stmt
}

type ObjectType int

const (
	OBJECT_VAR = ObjectType(iota + 1)
	OBJECT_TYPE
	OBJECT_PACKAGE
	OBJECT_LABEL
)

// This serves a similar purpose to Go's types.Object
type Object interface {
	Name() string
	ObjectType() ObjectType
}

// Implements Object
type Variable struct {
	name string
	Type Type
	Init Expr
}

func (o *Variable) Name() string           { return o.name }
func (o *Variable) ObjectType() ObjectType { return OBJECT_VAR }

// implements Object and Stmt
type LabelStmt struct {
	stmt
	name       string
	Branchable Stmt // Either nil or branchable statement (for, switch, etc)
}

func (l *LabelStmt) Name() string           { return l.name }
func (l *LabelStmt) ObjectType() ObjectType { return OBJECT_LABEL }

// Implements Object and Stmt
type TypeDecl struct {
	stmt

	name        string
	AliasedType Type
}

func (o *TypeDecl) Name() string           { return o.name }
func (o *TypeDecl) ObjectType() ObjectType { return OBJECT_TYPE }
func (o *TypeDecl) Type() Type {
	// TODO: cache this thing, don't produce new instances every time
	if o.AliasedType == nil {
		return &SimpleType{simpleTypeStrToID[o.name]}
	}
	return &CustomType{Name: o.name, Decl: o}
}

var builtinTypeNames []string = []string{"bool", "byte", "complex128", "complex64", "error", "float32",
	"float64", "int", "int16", "int32", "int64", "int8", "rune",
	"string", "uint", "uint16", "uint32", "uint64", "uint8", "uintptr"}

var builtinTypes map[string]*TypeDecl = map[string]*TypeDecl{}

func initVarDecls() {
	for _, name := range builtinTypeNames {
		builtinTypes[name] = &TypeDecl{
			name: name,
			//Type: &SimpleType{ID: simpleTypeStrToID[name]},
			AliasedType: nil, // Simple types aren't aliases
		}
	}
}

var builtinFuncs = []string{
	// These should be changed once we have varags etc.
	"func print(s string) bool:\n pass",
	"func read() string:\n pass",
}

func GetBuiltinType(name string) (*TypeDecl, bool) {
	t, ok := builtinTypes[name]
	return t, ok
}

type CodeBlock struct {
	Statements []Stmt
	Labels     map[string]*LabelStmt
}

func (cb *CodeBlock) AddLabel(l *LabelStmt) error {
	if _, ok := cb.Labels[l.Name()]; ok {
		return fmt.Errorf("Label `%s` declared more than once", l.Name())
	}
	cb.Labels[l.Name()] = l
	return nil
}

// implements SimpleStmt
type AssignStmt struct {
	stmt
	Lhs, Rhs []Expr
	Token    *Token
}

// implements Stmt
type StructStmt struct {
	stmt
	Struct *StructType
}

// implements Stmt
type VarStmt struct {
	stmt
	Vars       []*Variable
	IsFuncStmt bool
}

// implements Stmt
type PassStmt struct {
	stmt
}

// implements Stmt
type IfBranch struct {
	stmt
	ScopedVarDecl *VarStmt
	Condition     Expr
	Code          *CodeBlock
}

// implements Stmt
type IfStmt struct {
	stmt
	Branches []*IfBranch
}

// implements Stmt
type ForStmt struct {
	stmt

	// TODO: foreach-like loops can't be handled by this
	ScopedVarDecl *VarStmt
	Condition     Expr
	RepeatStmt    SimpleStmt
	Code          *CodeBlock
}

// implements Stmt
// Statement wrapper for expressions.
type ExprStmt struct {
	stmt
	Expression Expr
}

// Break, continue, goto
type BranchStmt struct {
	stmt
	Token *Token
	Right *Ident

	// Is nil for breaks/continues in unnamed loops/switches.
	GotoLabel *LabelStmt
	// Is nil for goto statements. Otherwise it is the statement
	// that the continue/break refers to.
	Branchable Stmt
}

type Kind int

const (
	KIND_SIMPLE = Kind(iota + 1)
	KIND_ARRAY
	KIND_SLICE
	KIND_MAP
	KIND_POINTER
	KIND_CUSTOM
	KIND_STRUCT
	KIND_TUPLE
	KIND_FUNC
	KIND_UNKNOWN
)

type Type interface {
	// True means no underscores beneath, no type inference needed.
	Known() bool
	String() string
	Kind() Kind
	Negotiate(other Type) (Type, error)
}

type SimpleTypeID int

const (
	SIMPLE_TYPE_INT = SimpleTypeID(iota + 1)
	SIMPLE_TYPE_STRING
	SIMPLE_TYPE_BOOL
	// TODO: add others
)

var simpleTypeAsStr = map[SimpleTypeID]string{
	SIMPLE_TYPE_INT:    "int",
	SIMPLE_TYPE_STRING: "string",
	SIMPLE_TYPE_BOOL:   "bool",
}

var simpleTypeStrToID = map[string]SimpleTypeID{}

func initSimpleTypeIDs() {
	for k, v := range simpleTypeAsStr {
		simpleTypeStrToID[v] = k
	}
}

type SimpleType struct {
	ID SimpleTypeID
}

func (t *SimpleType) Known() bool    { return true }
func (t *SimpleType) String() string { return simpleTypeAsStr[t.ID] }
func (t *SimpleType) Kind() Kind     { return KIND_SIMPLE }

type ArrayType struct {
	Size int
	Of   Type
}

func (t *ArrayType) Known() bool    { return t.Of.Known() }
func (t *ArrayType) String() string { return fmt.Sprintf("[%d]%s", t.Size, t.Of.String()) }
func (t *ArrayType) Kind() Kind     { return KIND_ARRAY }

type SliceType struct {
	Of Type
}

func (t *SliceType) Known() bool    { return t.Of.Known() }
func (t *SliceType) String() string { return "[]" + t.Of.String() }
func (t *SliceType) Kind() Kind     { return KIND_SLICE }

type MapType struct {
	By, Of Type
}

func (t *MapType) Known() bool    { return t.By.Known() && t.Of.Known() }
func (t *MapType) String() string { return "map[" + t.By.String() + "]" + t.Of.String() }
func (t *MapType) Kind() Kind     { return KIND_MAP }

type FuncType struct {
	Args, Results []Type
}

func (t *FuncType) Known() bool {
	for _, a := range t.Args {
		if !a.Known() {
			return false
		}
	}
	for _, r := range t.Results {
		if !r.Known() {
			return false
		}
	}
	return true
}

func (t *FuncType) String() string {
	out := &bytes.Buffer{}
	out.WriteString("func(")
	for c, a := range t.Args {
		out.WriteString(a.String())
		if (c + 1) < len(t.Args) {
			out.WriteString(", ")
		}
	}
	out.WriteString(")")
	switch len(t.Results) {
	case 0:
	case 1:
		out.WriteString(" " + t.Results[0].String())
	default:
		out.WriteString(" (")
		for c, r := range t.Results {
			out.WriteString(r.String())
			if (c + 1) < len(t.Results) {
				out.WriteString(", ")
			}
		}
		out.WriteString(")")
	}
	return out.String()
}

func (t *FuncType) Kind() Kind { return KIND_FUNC }

type PointerType struct {
	To Type
}

func (t *PointerType) Known() bool    { return t.To.Known() }
func (t *PointerType) String() string { return "*" + t.To.String() }
func (t *PointerType) Kind() Kind     { return KIND_POINTER }

type TupleType struct {
	Members []Type
}

func (t *TupleType) Known() bool {
	for _, t := range t.Members {
		if !t.Known() {
			return false
		}
	}
	return true
}

func (t *TupleType) String() string {
	out := &bytes.Buffer{}
	out.WriteByte('(')
	for c, v := range t.Members {
		fmt.Fprintf(out, v.String())
		if c < len(t.Members) {
			out.Write([]byte(", "))
		}
	}
	out.WriteByte(')')
	return out.String()
}

func (t *TupleType) Kind() Kind { return KIND_TUPLE }

type StructType struct {
	Members map[string]Type
	// Keys in the order of declaration
	Keys    []string
	Methods []*FuncDecl
	Name    string
}

func (st *StructType) GetTypeN(n int) Type {
	return st.Members[st.Keys[n]]
}

func (t *StructType) Known() bool {
	for _, t := range t.Members {
		if !t.Known() {
			return false
		}
	}
	return true
}

func (t *StructType) String() string {
	out := &bytes.Buffer{}
	out.WriteByte('{')
	for i, k := range t.Keys {
		fmt.Fprintf(out, "%s: %s", k, t.Members[k].String())
		if (i + 1) < len(t.Members) {
			out.Write([]byte(", "))
		}
	}
	out.WriteByte('}')
	return out.String()
}

func (t *StructType) Kind() Kind { return KIND_STRUCT }

type CustomType struct {
	Name    string
	Package string // "" means local
	Decl    *TypeDecl
}

func (t *CustomType) Known() bool    { return true }
func (t *CustomType) String() string { return t.Name }
func (t *CustomType) Kind() Kind     { return KIND_CUSTOM }
func (t *CustomType) RootType() Type {
	current := t.Decl.AliasedType
	for current.Kind() == KIND_CUSTOM {
		current = current.(*CustomType).Decl.AliasedType
	}
	return current
}

type UnknownType struct {
}

func (t *UnknownType) Known() bool    { return false }
func (t *UnknownType) String() string { return "_" }
func (t *UnknownType) Kind() Kind     { return KIND_UNKNOWN }

type TypeExpr struct {
	expr
	typ Type
}

func (e *expr) Pos() int {
	return e.pos
}

func (s *stmt) Label() *Object {
	return s.label
}

// Blank expression, represents no expression.
// Sometimes useful.
// implements Expr
type BlankExpr struct {
	expr
}

func NewBlankExpr() *BlankExpr { return &BlankExpr{expr{0}} }

// implements Expr
type BasicLit struct {
	expr
	typ Type

	token *Token
}

type CompoundLitKind int

const (
	COMPOUND_UNKNOWN CompoundLitKind = iota
	COMPOUND_EMPTY
	COMPOUND_LISTLIKE
	COMPOUND_MAPLIKE
)

// implements Expr
type CompoundLit struct {
	expr
	typ        Type
	kind       CompoundLitKind
	elems      []Expr
	contentPos int
}

func (cl *CompoundLit) updatePosWithType(typ Expr) {
	cl.contentPos = cl.pos
	cl.pos = typ.Pos()
}

// implements Expr
type BinaryOp struct {
	expr

	Left, Right Expr
	op          *Token
}

// implements Expr
type UnaryOp struct {
	expr

	Right Expr
	op    *Token
}

type PrimaryExpr interface {
	Expr
}

// implements PrimaryExpr
type ArrayExpr struct {
	expr

	Left, Index Expr
}

type DotSelector struct {
	expr

	Left  Expr
	Right *Ident
}

// implements PrimaryExpr
type FuncCallExpr struct {
	expr

	Left Expr
	//args []*Expr
	Args []Expr
}

// implements PrimaryExpr
type FuncDecl struct {
	expr

	name          string
	typ           *FuncType
	Args, Results []*Variable
	Code          *CodeBlock
	// Is nil for non-method functions
	Receiver *Variable
}

// implements PrimaryExpr
type Ident struct {
	expr

	name   string
	object Object
	//token *Token
}

type Node interface {
	//Pos() int // TODO
	//End() int // TODO
}

type Package struct {
	name string
}

func (p *Package) Get(name string) Object {
	panic("todo")
}

func (o *Package) Name() string           { return o.name }
func (o *Package) ObjectType() ObjectType { return OBJECT_PACKAGE }

func init() {
	initSimpleTypeIDs()
	initVarDecls()
}

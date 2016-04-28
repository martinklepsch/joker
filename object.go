package main

import (
	"fmt"
	"math/big"
	"strings"
)

type (
	Position struct {
		line   int
		column int
	}
	Equality interface {
		Equals(interface{}) bool
	}
	Object interface {
		Equality
		ToString(escape bool) string
		GetInfo() *ObjectInfo
		WithInfo(*ObjectInfo) Object
	}
	Error interface {
		error
		Object
		Type() Symbol
	}
	Meta interface {
		GetMeta() *ArrayMap
		WithMeta(*ArrayMap) Object
	}
	MetaHolder struct {
		meta *ArrayMap
	}
	ObjectInfo struct {
		Position
	}
	InfoHolder struct {
		info *ObjectInfo
	}
	Char struct {
		InfoHolder
		ch rune
	}
	Double struct {
		InfoHolder
		d float64
	}
	Int struct {
		InfoHolder
		i int
	}
	BigInt struct {
		InfoHolder
		b big.Int
	}
	BigFloat struct {
		InfoHolder
		b big.Float
	}
	Ratio struct {
		InfoHolder
		r big.Rat
	}
	Bool struct {
		InfoHolder
		b bool
	}
	Nil struct {
		InfoHolder
		n struct{}
	}
	Keyword struct {
		InfoHolder
		k string
	}
	Symbol struct {
		InfoHolder
		MetaHolder
		ns   *string
		name *string
	}
	String struct {
		InfoHolder
		s string
	}
	Regex struct {
		InfoHolder
		r string
	}
	Var struct {
		InfoHolder
		MetaHolder
		ns    *Namespace
		name  Symbol
		value Object
	}
	Proc func([]Object) Object
	Fn   struct {
		InfoHolder
		MetaHolder
		fnExpr *FnExpr
		env    *LocalEnv
	}
	ExInfo struct {
		InfoHolder
		msg  String
		data *ArrayMap
	}
	RecurBindings []Object
)

func panicArity(n int) {
	name := RT.currentExpr.(*CallExpr).name
	panic(RT.newError(fmt.Sprintf("Wrong number of args (%d) passed to %s", n, name)))
}

func checkArity(args []Object, min int, max int) {
	n := len(args)
	if n < min || n > max {
		panicArity(n)
	}
}

func (rb RecurBindings) ToString(escape bool) string {
	return "RecurBindings"
}

func (rb RecurBindings) Equals(other interface{}) bool {
	return false
}

func (rb RecurBindings) GetInfo() *ObjectInfo {
	return nil
}

func (rb RecurBindings) WithInfo(info *ObjectInfo) Object {
	return rb
}

func (exInfo *ExInfo) ToString(escape bool) string {
	return exInfo.msg.ToString(escape)
}

func (exInfo *ExInfo) Type() Symbol {
	return MakeSymbol("ExInfo")
}

func (exInfo *ExInfo) Equals(other interface{}) bool {
	switch other := other.(type) {
	case *ExInfo:
		return exInfo.msg == other.msg && exInfo.data.Equals(other.data)
	default:
		return false
	}
}

func (exInfo *ExInfo) WithInfo(info *ObjectInfo) Object {
	exInfo.info = info
	return exInfo
}

func (exInfo *ExInfo) Error() string {
	return exInfo.msg.s
}

func (fn *Fn) ToString(escape bool) string {
	return "function"
}

func (fn *Fn) Equals(other interface{}) bool {
	switch other := other.(type) {
	case *Fn:
		return fn == other
	default:
		return false
	}
}

func (fn *Fn) WithMeta(meta *ArrayMap) Object {
	res := *fn
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (fn *Fn) WithInfo(info *ObjectInfo) Object {
	fn.info = info
	return fn
}

func (fn *Fn) Call(args []Object) Object {
	for _, arity := range fn.fnExpr.arities {
		if len(arity.args) == len(args) {
			RT.pushFrame()
			defer RT.popFrame()
			return evalLoop(arity.body, fn.env.addFrame(args))
		}
	}
	v := fn.fnExpr.variadic
	if v == nil || len(args) < len(v.args)-1 {
		panicArity(len(args))
	}
	restArgs := &ArraySeq{arr: args, index: len(v.args) - 1}
	vargs := make([]Object, len(v.args))
	for i := 0; i < len(vargs)-1; i++ {
		vargs[i] = args[i]
	}
	vargs[len(vargs)-1] = restArgs
	RT.pushFrame()
	defer RT.popFrame()
	return evalLoop(v.body, fn.env.addFrame(vargs))
}

func (p Proc) Call(args []Object) Object {
	return p(args)
}

func (p Proc) ToString(escape bool) string {
	return "primitive function"
}

func (p Proc) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Proc:
		return &p == &other
	default:
		return false
	}
}

func (p Proc) GetInfo() *ObjectInfo {
	return nil
}

func (p Proc) WithInfo(*ObjectInfo) Object {
	return p
}

func (i InfoHolder) GetInfo() *ObjectInfo {
	return i.info
}

func (m MetaHolder) GetMeta() *ArrayMap {
	return m.meta
}

func (sym Symbol) WithMeta(meta *ArrayMap) Object {
	res := sym
	res.meta = SafeMerge(res.meta, meta)
	return res
}

func (v *Var) WithMeta(meta *ArrayMap) Object {
	res := *v
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (v *Var) WithInfo(info *ObjectInfo) Object {
	v.info = info
	return v
}

func MakeQualifiedSymbol(ns, name string) Symbol {
	return Symbol{
		ns:   STRINGS.Intern(ns),
		name: STRINGS.Intern(name),
	}
}

func MakeSymbol(nsname string) Symbol {
	index := strings.IndexRune(nsname, '/')
	if index == -1 || nsname == "/" {
		return Symbol{
			ns:   nil,
			name: STRINGS.Intern(nsname),
		}
	}
	return Symbol{
		ns:   STRINGS.Intern(nsname[0:index]),
		name: STRINGS.Intern(nsname[index+1 : len(nsname)]),
	}
}

func (v *Var) ToString(escape bool) string {
	return "#'" + v.ns.name.ToString(false) + "/" + v.name.ToString(false)
}

func (v *Var) Equals(other interface{}) bool {
	// TODO: revisit this
	return v == other
}

func (n Nil) ToString(escape bool) string {
	return "nil"
}

func (n Nil) Equals(other interface{}) bool {
	switch other.(type) {
	case Nil:
		return true
	default:
		return false
	}
}

func (n Nil) WithInfo(info *ObjectInfo) Object {
	n.info = info
	return n
}

func (rat *Ratio) ToString(escape bool) string {
	return rat.r.String()
}

func (rat *Ratio) Equals(other interface{}) bool {
	if rat == other {
		return true
	}
	switch r := other.(type) {
	case *Ratio:
		return rat.r.Cmp(&r.r) == 0
	case *BigInt:
		var otherRat big.Rat
		otherRat.SetInt(&r.b)
		return rat.r.Cmp(&otherRat) == 0
	case Int:
		var otherRat big.Rat
		otherRat.SetInt64(int64(r.i))
		return rat.r.Cmp(&otherRat) == 0
	}
	return false
}

func (rat *Ratio) WithInfo(info *ObjectInfo) Object {
	rat.info = info
	return rat
}

func (bi *BigInt) ToString(escape bool) string {
	return bi.b.String() + "N"
}

func (bi *BigInt) Equals(other interface{}) bool {
	if bi == other {
		return true
	}
	switch b := other.(type) {
	case *BigInt:
		return bi.b.Cmp(&b.b) == 0
	case Int:
		bi2 := big.NewInt(int64(b.i))
		return bi.b.Cmp(bi2) == 0
	}
	return false
}

func (bi *BigInt) WithInfo(info *ObjectInfo) Object {
	bi.info = info
	return bi
}

func (bf *BigFloat) ToString(escape bool) string {
	return bf.b.Text('g', 256) + "M"
}

func (bf *BigFloat) Equals(other interface{}) bool {
	if bf == other {
		return true
	}
	switch b := other.(type) {
	case *BigFloat:
		return bf.b.Cmp(&b.b) == 0
	case Double:
		bf2 := big.NewFloat(b.d)
		return bf.b.Cmp(bf2) == 0
	}
	return false
}

func (bf *BigFloat) WithInfo(info *ObjectInfo) Object {
	bf.info = info
	return bf
}

func (c Char) ToString(escape bool) string {
	if escape {
		return escapeRune(c.ch)
	}
	return string(c.ch)
}

func (c Char) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Char:
		return c.ch == other.ch
	default:
		return false
	}
}

func (c Char) WithInfo(info *ObjectInfo) Object {
	c.info = info
	return c
}

func (d Double) ToString(escape bool) string {
	return fmt.Sprintf("%f", d.d)
}

func (d Double) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Double:
		return d.d == other.d
	default:
		return false
	}
}

func (d Double) WithInfo(info *ObjectInfo) Object {
	d.info = info
	return d
}

func (i Int) ToString(escape bool) string {
	return fmt.Sprintf("%d", i.i)
}

func (i Int) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Int:
		return i.i == other.i
	default:
		return false
	}
}

func (i Int) WithInfo(info *ObjectInfo) Object {
	i.info = info
	return i
}

func (b Bool) ToString(escape bool) string {
	return fmt.Sprintf("%t", b.b)
}

func (b Bool) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Bool:
		return b.b == other.b
	default:
		return false
	}
}

func (b Bool) WithInfo(info *ObjectInfo) Object {
	b.info = info
	return b
}

func (k Keyword) ToString(escape bool) string {
	return k.k
}

func (k Keyword) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Keyword:
		return k.k == other.k
	default:
		return false
	}
}

func (k Keyword) WithInfo(info *ObjectInfo) Object {
	k.info = info
	return k
}

func (rx Regex) ToString(escape bool) string {
	if escape {
		return "#" + escapeString(rx.r)
	}
	return "#" + rx.r
}

func (rx Regex) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Regex:
		return rx.r == other.r
	default:
		return false
	}
}

func (rx Regex) WithInfo(info *ObjectInfo) Object {
	rx.info = info
	return rx
}

func (s Symbol) ToString(escape bool) string {
	if s.ns != nil {
		return *s.ns + "/" + *s.name
	}
	return *s.name
}

func (s Symbol) Equals(other interface{}) bool {
	switch other := other.(type) {
	case Symbol:
		return s.ns == other.ns && s.name == other.name
	default:
		return false
	}
}

func (s Symbol) WithInfo(info *ObjectInfo) Object {
	s.info = info
	return s
}

func (s String) ToString(escape bool) string {
	if escape {
		return escapeString(s.s)
	}
	return s.s
}

func (s String) Equals(other interface{}) bool {
	switch other := other.(type) {
	case String:
		return s.s == other.s
	default:
		return false
	}
}

func (s String) WithInfo(info *ObjectInfo) Object {
	s.info = info
	return s
}

func IsSymbol(obj Object) bool {
	switch obj.(type) {
	case Symbol:
		return true
	default:
		return false
	}
}

func IsVector(obj Object) bool {
	switch obj.(type) {
	case *Vector:
		return true
	default:
		return false
	}
}

func IsSeq(obj Object) bool {
	switch obj.(type) {
	case Seq:
		return true
	default:
		return false
	}
}

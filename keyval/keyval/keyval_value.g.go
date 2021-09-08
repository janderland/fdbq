// Code generated by visitor -visitor Value -acceptor value -types Tuple,Nil,Int,Uint,Bool,Float,String,UUID,Bytes,Variable,Clear. DO NOT EDIT.
package keyval

type (
	ValueVisitor interface {
		VisitTuple(Tuple)
		VisitNil(Nil)
		VisitInt(Int)
		VisitUint(Uint)
		VisitBool(Bool)
		VisitFloat(Float)
		VisitString(String)
		VisitUUID(UUID)
		VisitBytes(Bytes)
		VisitVariable(Variable)
		VisitClear(Clear)
	}

	value interface {
		Value(ValueVisitor)
	}
)

func _() {
	var (
		Tuple    Tuple
		Nil      Nil
		Int      Int
		Uint     Uint
		Bool     Bool
		Float    Float
		String   String
		UUID     UUID
		Bytes    Bytes
		Variable Variable
		Clear    Clear

		_ value = &Tuple
		_ value = &Nil
		_ value = &Int
		_ value = &Uint
		_ value = &Bool
		_ value = &Float
		_ value = &String
		_ value = &UUID
		_ value = &Bytes
		_ value = &Variable
		_ value = &Clear
	)
}

func (x Tuple) Value(v ValueVisitor) {
	v.VisitTuple(x)
}

func (x Nil) Value(v ValueVisitor) {
	v.VisitNil(x)
}

func (x Int) Value(v ValueVisitor) {
	v.VisitInt(x)
}

func (x Uint) Value(v ValueVisitor) {
	v.VisitUint(x)
}

func (x Bool) Value(v ValueVisitor) {
	v.VisitBool(x)
}

func (x Float) Value(v ValueVisitor) {
	v.VisitFloat(x)
}

func (x String) Value(v ValueVisitor) {
	v.VisitString(x)
}

func (x UUID) Value(v ValueVisitor) {
	v.VisitUUID(x)
}

func (x Bytes) Value(v ValueVisitor) {
	v.VisitBytes(x)
}

func (x Variable) Value(v ValueVisitor) {
	v.VisitVariable(x)
}

func (x Clear) Value(v ValueVisitor) {
	v.VisitClear(x)
}


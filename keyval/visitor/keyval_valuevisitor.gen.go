// Code generated with args "-visitor ValueVisitor -acceptor value -types Int,Uint,Bool,Float,String,Variable,Clear". DO NOT EDIT.
package keyval

type (
	ValueVisitor interface {
		VisitInt(Int)
		VisitUint(Uint)
		VisitBool(Bool)
		VisitFloat(Float)
		VisitString(String)
		VisitVariable(Variable)
		VisitClear(Clear)
	}

	value interface {
		Value(ValueVisitor)
	}
)

func _() {
	var (
		Int_      Int
		Uint_     Uint
		Bool_     Bool
		Float_    Float
		String_   String
		Variable_ Variable
		Clear_    Clear

		_ value = &Int_
		_ value = &Uint_
		_ value = &Bool_
		_ value = &Float_
		_ value = &String_
		_ value = &Variable_
		_ value = &Clear_
	)
}

func (x *Int) Value(v ValueVisitor) {
	v.VisitInt(*x)
}

func (x *Uint) Value(v ValueVisitor) {
	v.VisitUint(*x)
}

func (x *Bool) Value(v ValueVisitor) {
	v.VisitBool(*x)
}

func (x *Float) Value(v ValueVisitor) {
	v.VisitFloat(*x)
}

func (x *String) Value(v ValueVisitor) {
	v.VisitString(*x)
}

func (x *Variable) Value(v ValueVisitor) {
	v.VisitVariable(*x)
}

func (x *Clear) Value(v ValueVisitor) {
	v.VisitClear(*x)
}

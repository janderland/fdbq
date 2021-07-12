// Code generated with args "-visitor TupleVisitor -acceptor TupElement -types Int,Uint,Bool,Float,BigInt,String,UUID,Bytes,Variable,MaybeMore". DO NOT EDIT.
package keyval

type (
	TupleVisitor interface {
		VisitInt(Int)
		VisitUint(Uint)
		VisitBool(Bool)
		VisitFloat(Float)
		VisitBigInt(BigInt)
		VisitString(String)
		VisitUUID(UUID)
		VisitBytes(Bytes)
		VisitVariable(Variable)
		VisitMaybeMore(MaybeMore)
	}

	TupElement interface {
		TupElement(TupleVisitor)
	}
)

func _() {
	var (
		Int_       Int
		Uint_      Uint
		Bool_      Bool
		Float_     Float
		BigInt_    BigInt
		String_    String
		UUID_      UUID
		Bytes_     Bytes
		Variable_  Variable
		MaybeMore_ MaybeMore

		_ TupElement = &Int_
		_ TupElement = &Uint_
		_ TupElement = &Bool_
		_ TupElement = &Float_
		_ TupElement = &BigInt_
		_ TupElement = &String_
		_ TupElement = &UUID_
		_ TupElement = &Bytes_
		_ TupElement = &Variable_
		_ TupElement = &MaybeMore_
	)
}

func (x *Int) TupElement(v TupleVisitor) {
	v.VisitInt(*x)
}

func (x *Uint) TupElement(v TupleVisitor) {
	v.VisitUint(*x)
}

func (x *Bool) TupElement(v TupleVisitor) {
	v.VisitBool(*x)
}

func (x *Float) TupElement(v TupleVisitor) {
	v.VisitFloat(*x)
}

func (x *BigInt) TupElement(v TupleVisitor) {
	v.VisitBigInt(*x)
}

func (x *String) TupElement(v TupleVisitor) {
	v.VisitString(*x)
}

func (x *UUID) TupElement(v TupleVisitor) {
	v.VisitUUID(*x)
}

func (x *Bytes) TupElement(v TupleVisitor) {
	v.VisitBytes(*x)
}

func (x *Variable) TupElement(v TupleVisitor) {
	v.VisitVariable(*x)
}

func (x *MaybeMore) TupElement(v TupleVisitor) {
	v.VisitMaybeMore(*x)
}

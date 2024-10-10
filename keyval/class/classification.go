package class

import q "github.com/janderland/fql/keyval"

var (
	_ q.DirectoryOperation = &dirClassification{}
	_ q.TupleOperation     = &tupClassification{}
	_ q.ValueOperation     = &valClassification{}
)

type dirClassification struct{ out subClass }

func (x *dirClassification) ForString(q.String) {}

func (x *dirClassification) ForVariable(q.Variable) {
	x.out = variableSubClass
}

type tupClassification struct{ out subClass }

func (x *tupClassification) ForTuple(e q.Tuple) {
	x.out = classifyTuple(e)
}

func (x *tupClassification) ForVariable(q.Variable) {
	x.out = variableSubClass
}

func (x *tupClassification) ForMaybeMore(q.MaybeMore) {
	x.out = variableSubClass
}

func (x *tupClassification) ForNil(q.Nil) {}

func (x *tupClassification) ForInt(q.Int) {}

func (x *tupClassification) ForUint(q.Uint) {}

func (x *tupClassification) ForBool(q.Bool) {}

func (x *tupClassification) ForFloat(q.Float) {}

// TODO: Add support for BigInt.
/*
func (x *tupClassification) ForBigInt(q.BigInt) {}
*/

func (x *tupClassification) ForString(q.String) {}

func (x *tupClassification) ForUUID(q.UUID) {}

func (x *tupClassification) ForBytes(q.Bytes) {}

type valClassification struct{ out subClass }

func (x *valClassification) ForTuple(e q.Tuple) {
	x.out = classifyTuple(e)
}

func (x *valClassification) ForVariable(q.Variable) {
	x.out = variableSubClass
}

func (x *valClassification) ForClear(q.Clear) {
	x.out = clearSubClass
}

func (x *valClassification) ForNil(q.Nil) {}

func (x *valClassification) ForInt(q.Int) {}

func (x *valClassification) ForUint(q.Uint) {}

func (x *valClassification) ForBool(q.Bool) {}

func (x *valClassification) ForFloat(q.Float) {}

func (x *valClassification) ForString(q.String) {}

func (x *valClassification) ForUUID(q.UUID) {}

func (x *valClassification) ForBytes(q.Bytes) {}

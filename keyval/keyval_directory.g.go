// Code generated by visitor -visitor Directory -acceptor DirElement -types String,Variable. DO NOT EDIT.
package keyval

type (
	DirectoryVisitor interface {
		VisitString(String)
		VisitVariable(Variable)
	}

	DirElement interface {
		DirElement(DirectoryVisitor)
		Eq(interface{}) bool
	}
)

func _() {
	var (
		String   String
		Variable Variable

		_ DirElement = &String
		_ DirElement = &Variable
	)
}

func (x String) DirElement(v DirectoryVisitor) {
	v.VisitString(x)
}

func (x Variable) DirElement(v DirectoryVisitor) {
	v.VisitVariable(x)
}


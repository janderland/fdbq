// Code generated with args "-visitor DirectoryVisitor -acceptor DirElement -types String,Variable". DO NOT EDIT.
package keyval

type (
	DirectoryVisitor interface {
		VisitString(String)
		VisitVariable(Variable)
	}

	DirElement interface {
		DirElement(DirectoryVisitor)
	}
)

func _() {
	var (
		String_   String
		Variable_ Variable

		_ DirElement = &String_
		_ DirElement = &Variable_
	)
}

func (x *String) DirElement(v DirectoryVisitor) {
	v.VisitString(*x)
}

func (x *Variable) DirElement(v DirectoryVisitor) {
	v.VisitVariable(*x)
}

// Code generated by: operation -op-name Directory -param-name DirElement -types String,Variable. DO NOT EDIT.

package keyval

type (
	DirectoryOperation interface {
		// ForString performs the DirectoryOperation if the given DirElement is of type String.
		ForString(String)
		// ForVariable performs the DirectoryOperation if the given DirElement is of type Variable.
		ForVariable(Variable)
	}

	DirElement interface {
		// DirElement executes the given DirectoryOperation on this DirElement.
		DirElement(DirectoryOperation)

		// Eq returns true if the given value is equal to this DirElement.
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

func (x String) DirElement(op DirectoryOperation) {
	op.ForString(x)
}

func (x Variable) DirElement(op DirectoryOperation) {
	op.ForVariable(x)
}


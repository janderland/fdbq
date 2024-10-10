// Package compare validates tuples against a schema.
package compare

import (
	q "github.com/janderland/fql/keyval"
)

// Tuples checks if the candidate Tuple conforms to the given schema.
// The schema Tuple may contain Variable or MaybeMore while the candidate
// must not contain either. The elements of each Tuple are compared for
// equality. If an element of the schema Tuple is a Variable, then the
// candidate's corresponding element must conform for the constraints
// of the Variable. If all the elements match then nil is returned. If
// an element doesn't match, then an array is returned specifying the
// index path to the first mismatching element. For instance, given the
// following candidate tuple...
//
//	Tuple{55, Tuple{"hello", "world", Tuple{67}}}
//
// ...if the element with value `67` didn't match then the returned array
// would be `[]int{1,2,0}`. If the tuples aren't the same length then the
// length of the shorter Tuple is used as the mismatching index.
func Tuples(schema q.Tuple, candidate q.Tuple) []int {
	// If the schema is empty, the candidate must
	// be empty as well.
	if len(schema) == 0 {
		if len(candidate) == 0 {
			return nil
		}
		return []int{0}
	}

	// If this schema ends with a MaybeMore, we
	// don't need to check if the candidate is
	// longer. Get rid of the MaybeMore as it's
	// not used in the actual comparisons.
	switch schema[len(schema)-1].(type) {
	case q.MaybeMore:
		schema = schema[:len(schema)-1]
	default:
		if len(schema) < len(candidate) {
			return []int{len(schema)}
		}
	}

	// The candidate must be at least as long
	// as the schema.
	if len(schema) > len(candidate) {
		return []int{len(candidate)}
	}

	// Loop over both tuples, comparing their elements.
	for i, element := range schema {
		c := comparison{i: i, candidate: candidate[i]}
		element.TupElement(&c)
		if c.out != nil {
			return c.out
		}
	}
	return nil
}

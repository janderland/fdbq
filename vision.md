# Foundation QL Vision

Foundation QL is a query language for Foundation DB. FQL
aims to make FDB's semantics feel natural and intuitive.
Common patterns like index indirection and chunked range
reads are first class citizens.

> __NOTE:__ This document expects the reader to have a basic
> knowledge of FDB, and it's tuple & directory layers.

FQL may also be used as a [Go API](#language-integration)
which is structurally equivalent to the query language. 

## Query Basics

An FQL query looks like a key-value. Queries include
a directory, tuple, and value. FQL can only access keys
encoded by the directory & tuples layers.

```fql
/my/directory("my","tuple")=4000
```

FQL queries may define a single key-value to be written (as
shown above) or may define a set of key-values to be read
from the database.

```fql
/my/directory("my","tuple")=<>
```

The query above has a variable (`<>`) in its value section
and will read a single key from the database. 

FQL queries can also perform range reads by including
a variable in the tuple section.

```fql
/my/directory(<>,"tuple")=<>
```

All key-values with a certain prefix can be range read by
ending the tuple with `...`.

```fql
/my/directory("my","tuple",...)=<>
```

Including a variable in the directory section tells FQL to
perform the read on all directory paths matching the
pattern.

```fql
/<>/directory("my","tuple")=<>
```

The value section may be omitted to imply a variable in
that section, meaning the following query is semantically
identical to the one above.

```fql
/<>/directory("my","tuple")
```

## Data Elements

In an FQL query, the directory, tuple, and value 
contain instances of data elements. FQL utilizes the 
same types of elements as the tuple layer. Example 
instances of these types can be seen below.

| Type     | Example                                |
|:---------|:---------------------------------------|
| `nil`    | `nil`                                  |
| `int`    | `-14`                                  |
| `uint`   | `7`                                    |
| `bool`   | `true`                                 |
| `float`  | `33.4`                                 |
| `string` | `"string"`                             |
| `bytes`  | `0xa2bff2438312aac032`                 |
| `uuid`   | `5a5ebefd-2193-47e2-8def-f464fc698e31` |
| `tuple`  | `("hello",27.4,nil)`                   |

> __NOTE:__ FDB tuples can also include variable length
> integers (bigint). While these are not currently
> supported by FQL, they will be in the future.

The directory may only contain strings. Directory strings 
don't need to be quoted if they only contain 
alphanumerics, `.`, or `_`. The tuple & value may 
contain any of the data elements.

## Data Encoding

TODO: Finish section.

## Variables

Queries without any variables result in a single key-value
being written. You can think of these queries as explicitly
defining a single key-value.

> __NOTE:__ Queries lacking a value section imply a variable
> in said section and therefore do not result in a write
> operation.

Queries with variables or `...` result in zero or more
key-values being read. You can think of these queries as
defining a set of possible key-values stored in the DB.

You can further limit the set of key-values read by
including a type constraint in the variable.

```fql
/my/directory("tuple",<int|string>)=<tuple>
```

In the query above, the 2nd element of the key's tuple must
be either an integer or string. Likewise, the value must be
a tuple.

> __NOTE:__ FQL does not currently provide a way to
> constrain a variable to range of values, though this could
> be added in the future.

## Index Indirection

TODO: Finish section.

```fql
/user/index/surname("Johnson",<userID:int>)
/user/entry(:userID,...)
```

## Transaction Boundaries

TODO: Finish section.

## Language Integration

When integrating SQL into other languages, there are usually
two choices each with their own drawbacks:

1. Write literal SQL strings into your code. This is simple
   but type safety isn't usually checked till runtime.

2. Use an ORM and/or code generation. This is more complex
   and sometimes doesn't perfectly model SQL semantics, but
   does provide type safety.

FQL leans towards option #2 by providing a Go API which is
structurally equivalent to the query language. This allows
FQL semantics to be modeled in the host language's type
system.

This Go API may also be viewed as an FDB layer which unifies
the directory & tuple layers with the FDB base API.

```go
package example

import (
  "github.com/apple/foundationdb/bindings/go/src/fdb"
  "github.com/apple/foundationdb/bindings/go/src/fdb/directory"

  "github.com/janderland/fdbq/engine"
  "github.com/janderland/fdbq/engine/facade"
  kv "github.com/janderland/fdbq/keyval"
)

func _() {
  fdb.MustAPIVersion(620)
  eg := engine.New(facade.NewTransactor(
    fdb.MustOpenDefault(),
    // Choose a global directory root for all queries.
    directory.Root(),
  ))

  // /user/entry(22573,"Goodwin","Samuels")=nil
  query := kv.KeyValue{
    Key: kv.Key{
      Directory: kv.Directory{
        kv.String("user"),
        kv.String("entry"),
      },
      Tuple: kv.Tuple{
        kv.Int(22573),
        kv.String("Goodwin"),
        kv.String("Samuels"),
      },
    },
    Value: kv.Nil{},
  }

  err := eg.Set(query);
  if err != nil {
    panic(err)
  }
}
```

Code generation tooling could be provided to lessen the
verbosity of the host language's syntax.

```go
package example

// Generate a function which binds args to the query.
//go:generate fql --gen getFullName '/user/entry(?,<string>,<string>)'

func _() {
  fdb.MustAPIVersion(620)
  eg := engine.New(facade.NewTransactor(
    fdb.MustOpenDefault(),
    // Choose a global directory root for all queries.
    directory.Root(),
  ))

  const userID = 22573
  query := getFullName(userID)
  res, err := eg.SingleRead(query)
  if err != nil {
    panic(err)
  }
}
```

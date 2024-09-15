---
title: FQL

# We include this intro via the 'include-before'
# metadata field so it's placed before the TOC.
include-before: |
  ```lang-fql {.query}
  /user/index/surname("Johnson",<userID:int>)
  /user(:userID,...)
  ```
  ```lang-fql {.result}
  /user(9323,"Timothy","Johnson",37)=nil
  /user(24335,"Andrew","Johnson",42)=nil
  /user(33423,"Ryan","Johnson",0x0ffa83,42.2)=nil
  ```
  FQL is an [open source](https://github.com/janderland/fdbq)
  query language for
  [Foundation DB](https://www.foundationdb.org/). It's query
  semantics mirror Foundation DB's [core data
  model](https://apple.github.io/foundationdb/data-modeling.html).
  Fundamental patterns like range-reads and indirection are first
  class citizens.
...

# Overview

FQL is specified as a [context-free
grammar](https://github.com/janderland/fdbq/blob/main/syntax.ebnf).
The queries look like key-values encoded using the directory
& tuple layers.

```lang-fql {.query}
/my/directory("my","tuple")=4000
```

FQL queries may define a single key-value to be written, as
shown above, or may define a set of key-values to be read,
as shown below.

```lang-fql {.query}
/my/directory("my","tuple")=<int>
```

```lang-fql {.result}
/my/directory("my","tuple")=4000
```

The query above has a variable `<int>` as its value.
Variables act as placeholders for any of the supported [data
elements](#data-elements). In this case, the variable also
tells FQL how to decode the value's bytes. This query will
return a single key-value from the database, if such a key
exists.

FQL queries can also perform range reads & filtering by
including a variable in the key's tuple. The query below
will return all key-values which conform to the schema
defined by the query.

```lang-fql {.query}
/my/directory(<>,"tuple")=nil
```

```lang-fql {.result}
/my/directory("your","tuple")=nil
/my/directory(42,"tuple")=nil
```

All key-values with a certain key prefix can be range read
by ending the key's tuple with `...`.

```lang-fql {.query}
/my/directory("my","tuple",...)=<>
```

```lang-fql {.result}
/my/directory("my","tuple")=0x0fa0
/my/directory("my","tuple",47.3)=0x8f3a
/my/directory("my","tuple",false,0xff9a853c12)=nil
```

A query's value may be omitted to imply a variable, meaning
the following query is semantically identical to the one
above.

```lang-fql {.query}
/my/directory("my","tuple",...)
```

```lang-fql {.result}
/my/directory("my","tuple")=0x0fa0
/my/directory("my","tuple",47.3)=0x8f3a
/my/directory("my","tuple",false,0xff9a853c12)=nil
```

Including a variable in the directory tells FQL to perform
the read on all directory paths matching the schema.

```lang-fql {.query}
/<>/directory("my","tuple")
```

```lang-fql {.result}
/my/directory("my","tuple")=0x0fa0
/your/directory("my","tuple")=nil
```

Key-values can be cleared by using the special `clear` token
as the value.

```lang-fql {.query}
/my/directory("my","tuple")=clear
```

The directory layer can be queried by only including
a directory path.

```lang-fql {.query}
/my/<>
```

```lang-fql {.result}
/my/directory
```

# Data Elements

An FQL query contains instances of data elements. These are
the same types of elements found in the [tuple
layer](https://github.com/apple/foundationdb/blob/main/design/tuple.md).
Descriptions of these elements can be seen below.

<div>

| Type    | Description                            |
|:--------|:---------------------------------------|
| `nil`   | `nil`                                  |
| `bool`  | `true`                                 |
| `int`   | `-14`                                  |
| `uint`  | `7`                                    |
| `bint`  | `#35299340192843523485929848293291842` |
| `num`   | `33.4`                                 |
| `str`   | `"string"`                             |
| `uuid`  | `5a5ebefd-2193-47e2-8def-f464fc698e31` |
| `bytes` | `0xa2bff2438312aac032`                 |
| `tup`   | `("hello",27.4,nil)`                   |

</div>

> `bint` support is not yet implemented.

Tuples & values may contain any of the data elements.

```lang-fql {.query}
/region/north_america(22.3,-8)=("rain","fog")
/region/east_asia("japan",nil)=0xff
```

Strings are the only data element allowed in directories. If
a directory string only contains alphanumericals,
underscores, dashes, and periods then the quotes don't need
to be included.

```lang-fql {.query}
/quoteless-string_in.dir(true)=false
/"other ch@r@cters must be quoted!"(20)=32.3
```

Quoted strings may contain quotes via backslash escapes.

```lang-fql {.query}
/my/dir("I said \"hello\"")=nil
```

# Value Encoding

The directory and tuple layers are responsible for encoding
the data elements in the key. As for the value, FDB doesn't
provide a standard encoding.

The table below outlines how FQL encodes data elements as
values. Endianness is configurable.

<div>

| Type    | Encoding                        |
|:--------|:--------------------------------|
| `nil`   | empty value                     |
| `bool`  | single byte, `0x00` means false |
| `int`   | 64-bit, 1's compliment          |
| `uint`  | 64-bit                          |
| `bint`  | not implemented yet             |
| `num`   | IEEE 754                        |
| `str`   | ASCII                           |
| `uuid`  | RFC 4122                        |
| `bytes` | as provided                     |
| `tup`   | tuple layer                     |

</div>

# Variables & Schemas

Variables allow FQL to describe key-value schemas. Any [data
element](#data-elements) may be represented with a variable.
Variables are specified as a list of element types,
separated by `|`, wrapped in angled braces.

```lang-fql
<uint|str|uuid|bytes>
```

The variable's type list describes which data elements are
allowed at the variable's position. A variable may be empty,
including no element types, meaning it represents all
element types.

```lang-fql {.query}
/user(<int>,<str>,<>)=<>
```

```lang-fql {.result}
/user(0,"jon",0xffab0c)=nil
/user(20,"roger",22.3)=0xff
/user(21,"",nil)="nothing"
```

Before the type list, a variable can be given a name. This
name is used to reference the variable in subsequent
queries, allowing for [index
indirection](#index-indirection).

```lang-fql {.query}
/index("cars",<varName:int>)
/data(:varName,...)
```

```lang-fql {.result}
/user(33,"mazda")=nil
/user(320,"ford")=nil
/user(411,"chevy")=nil
```

# Space & Comments

Whitespace and newlines are allowed within a tuple, between
its elements.

```lang-fql {.query}
/account/private(
  <uint>,
  <uint>,
  <str>,
)=<int>
```

Comments start with a `%` and continue until the end of the
line. They can be used to describe a tuple's elements.

```lang-fql
% private account balances
/account/private(
  <uint>, % user ID
  <uint>, % group ID
  <str>,  % account name
)=<int>   % balance in USD
```

# Kinds of Queries

FQL queries can write/clear a single key-value, read one or
more key-values, or list directories. Throughout this
section, snippets of Go code are included to show how the
queries interact with the FDB API.

## Mutations

Queries lacking both [variables](#variables) and the `...`
token perform mutations on the database by either writing
a key-value or clearing an existing one.

> Queries lacking a value altogether imply an empty
> [variable](#variables) as the value and should not be
> confused with mutation queries.

Mutation queries with a [data element](#data-elements) as
their value perform a write operation.

```lang-fql {.query}
/my/dir("hello","world")=42
```

```lang-go {.equiv-go}
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.CreateOrOpen(tr, []string{"my", "dir"}, nil)
  if err != nil {
    return nil, err
  }

  val := make([]byte, 8)
  // Endianness is configurable...
  binary.LittleEndian.PutUint64(val, 42)

  tr.Set(dir.Pack(tuple.Tuple{"hello", "world"}), val)
  return nil, nil
})
```

Mutation queries with the `clear` token as their value
perform a clear operation.

```lang-fql {.query}
/my/dir("hello","world")=clear
```

```lang-go {.equiv-go}
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"my", "dir"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  tr.Clear(dir.Pack(tuple.Tuple{"hello", "world"}))
  return nil, nil
})
```

## Single Reads

If the query has [variables](#variables) or the `...` token
in its value (but not in its key) then it reads a single
key-value, if the key-value exists.

```lang-fql {.query}
/my/dir(99.8, 7dfb10d1-2493-4fb5-928e-889fdc6a7136)=<int|str>
```

```lang-go {.equiv-go}
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"my", "dir"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  // Read the value's raw bytes...
  val := tr.MustGet(dir.Pack(tuple.Tuple{99.8,
    tuple.UUID{
      0x7d, 0xfb, 0x10, 0xd1,
      0x24, 0x93, 0x4f, 0xb5,
      0x92, 0x8e, 0x88, 0x9f,
      0xdc, 0x6a, 0x71, 0x36}))

  // Try to decode the value as a uint.
  if len(val) == 8 {
      return binary.LittleEndian.Uint64(val), nil
  }

  // If the value isn't a uint, assume it's a string.
  return string(val), nil
})
```

FQL attempts to decode the value as each of the types listed
in the variable, stopping at first success. If the value
cannot be decoded, the key-value does not match the schema.

If the value is specified as an empty variable, then the raw
bytes are returned.

```lang-fql {.query}
/some/data(10139)=<>
```

```lang-go {.equiv-go}
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"some", "data"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  // No value decoding...
  return tr.MustGet(dir.Pack(tuple.Tuple{10139})), nil
})
```

## Range Reads

TODO: Show a simpler query here. Use this query as the
filtering example.

Queries with [variables](#variables) or the `...` token in
their key (and optionally in their value) result in a range
of key-values being read.

```lang-fql {.query}
/people(3392,<str|int>,<>)=(<uint>,...)
```

```lang-go {.equiv-go}
db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"people"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  rng, err := fdb.PrefixRange(dir.Pack(tuple.Tuple{3392}))
  if err != nil {
    return nil, err
  }

  var results []fdb.KeyValue
  iter := tr.GetRange(rng, fdb.RangeOptions{}).Iterator()
  for iter.Advance() {
    kv := iter.MustGet()

    tup, err := dir.Unpack(kv.Key)
    if err != nil {
      return nil, err
    }

    // Our query specifies a key-tuple
    // with 3 elements...
    if len(tup) != 3 {
      continue
    }

    // The 2nd element must be either a
    // string or an int64...
    switch tup[1].(type) {
    default:
      continue
    case string | int64:
    }

    // The query tells us to assume the value
    // is a packed tuple...
    val, err := tuple.Unpack(kv.Value)
    if err != nil {
      continue
    }

    // The value-tuple must have one or more
    // elements in it...
    if len(val) == 0 {
      continue
    }

    // The first element of the value-tuple must
    // be a uint64...
    if _, isInt := val[0].(uint64); !isInt {
      continue
    }

    results = append(results, kv)
  }
  return results, nil
})
```

The actual implementation pipelines the reading, filtering,
and value decoding across multiple threads.

# Filtering

Read queries define a schema to which key-values may or
may-not conform. In the Go snippets above, non-conformant
key-values were being filtered out of the results.

> Filtering is performed on the client-side and may result
> in lots of data being transferred to the client machine.

Alternatively, FQL can throw an error when encountering
non-conformant key-values. This helps enforce the assumption
that all key-values within a directory conform to a certain
schema.

# Indirection

In Foundation DB, indexes are implemented by having one
key-value (the index) point at another key-value. This is
also called "indirection".

> Indirection is not yet implemented.

Suppose we have a large list of people, one key-value for
each person.

```lang-fql {.query}
/people(<id:uint>,<firstName:str>,<lastName:str>,<age:int>)=nil
```

If we wanted to read all records with the last name of
"Johnson", we'd have to perform a linear search across the
entire "people" directory. To make this kind of search more
efficient, we can store an index of last names in a separate
directory.

```lang-fql {.query}
/index/last_name(<lastName:str>,<id:uint>)=nil
```

FQL can forward the observed values of named variables from
one query to the next, allowing us to efficiently query for
all people with the last name of "Johnson".

```lang-fql {.query}
/index/last_name("Johnson",<id:uint>)
/people(:id,...)
```
```lang-fql {.result}
/people(23,"Lenny","Johnson",22,"Mechanic")=nil
/people(348,"Roger","Johnson",54,"Engineer")=nil
/people(2003,"Larry","Johnson",8,"N/A")=nil
```

The first query returned 3 key-values containing the IDs of
23, 348, & 2003 which were then fed into the second query
resulting in 3 individual [single reads](#single-reads).

```lang-fql {.query}
/index/last_name("Johnson",<id:uint>)
```
```lang-fql {.result}
/index/last_name("Johnson",23)=nil
/index/last_name("Johnson",348)=nil
/index/last_name("Johnson",2003)=nil
```

# Aggregation

> The design of aggregation queries is not complete. This
> section describes the general idea. Exact syntax may
> change. This feature is not currently included in the
> grammar nor has it been implemented.

Foundation DB performs best when key-values are kept small.
When [storing large
blobs](https://apple.github.io/foundationdb/blob.html), the
data is usually split into 10 kB chunks stored in the value.
The respective key contain the byte offset of the chunk.

```lang-fql {.query}
/blob(
  "my file",    % The identifier of the blob.
  <offset:int>, % The byte offset within the blob.
)=<chunk:bytes> % A chunk of the blob.
```

```lang-fql {.result}
/blob("my file",0)=10e3_bytes
/blob("my file",10000)=10e3_bytes
/blob("my file",20000)=2.7e3_bytes
```

> Instead of printing the actual byte strings in these
> results, only the byte lengths are printed. This is an
> option provided by the CLI to lower result verbosity.

This gets the job done, but it would be nice if the client
could obtain the entire blob instead of having to append the
chunks themselves. This can be done using aggregation
queries.

FQL provides a pseudo data type named `agg` which performs
the aggregation.

```lang-fql {.query}
/blob("my file",...)=<blob:agg>
```

```lang-fql {.result}
/blob("my file",...)=22.7e3_bytes
```

Aggregation queries always result in a single key-value.
With non-aggregation queries, variables & the `...` token
are resolved as actual data elements in the query results.
For aggregation queries, only aggregation variables are
resolved.

A similar pseudo data type for summing integers could be
provided as well.

```lang-fql {.query}
/deltas("group A",<int>)
```

```lang-fql {.result}
/deltas("group A",20)=nil
/deltas("group A",-18)=nil
/deltas("group A",3)=nil
```

```lang-fql {.query}
/deltas("group A",<sum>)
```

```lang-fql {.result}
/deltas("group A",5)=<>
```

# Using FQL

## CLI

TODO: Finish section.

## API

TODO: Review this section.

When integrating SQL into other languages, there are usually
two choices each with their own drawbacks:

1. Write literal _SQL strings_ into your code. This is
   simple but type safety isn't usually checked till
   runtime.

2. Use an _ORM_. This is more complex and sometimes doesn't
   perfectly model SQL semantics, but does provide type
   safety.

FQL leans towards option #2 by providing a Go API which is
structurally equivalent to the query language, allowing FQL
semantics to be modeled in the host language's type system.

This Go API may also be viewed as an FDB layer which unifies
the directory & tuple layers with the FDB base API.

```lang-go
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
    fdb.MustOpenDefault(), directory.Root()))

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

  // Perform the write.
  err := eg.Set(query);
  if err != nil {
    panic(err)
  }
}
```


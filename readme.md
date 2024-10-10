# FQL

![demo gif](vhs/demo.gif)

FQL provides a query language and an alternative client API for Foundation DB.
Some things this project aims to achieve are:

- [x] Provide a query language for FDB.
- [x] Provide a textual description of key-value schemas.
- [x] Provide a Go API which is structurally equivalent to the query language.
- [ ] Simplify the ergonomics of the FoundationDB API.
    - [ ] Gracefully handle multi-transaction range-reads.
    - [ ] Gracefully handle transient errors.
- [ ] Provide an environment for exploring FDB data.
- [ ] Import/Export subsets of FDB data.

## Building & Running

### Without Docker

With the Foundation DB client library (>= v6.2.0) and Go (>= v1.20) installed,
you can simply run `go build` in the root of this repo. This will create an
`fql` binary in the root of the repo.

### Docker Environment

Building, linting, and testing can all be performed in a Docker environment.
This allows any host to perform these operations with only Docker as a
dependency. The [build.sh](build.sh) script can be used to perform these
operations. This is the same script used by the CI/CD workflow of this repo.

To build, lint, & test the current state of the codebase, run `./build.sh 
--verify`. To learn more about the build script, run `./build.sh --help`.

### Docker Image

FQL is available as a Docker image for executing queries. The first argument
passed to the container is the contents of the cluster file. The remaining
arguments are passed to the FQL binary.

```bash
# 'my_cluster:baoeA32@172.20.3.33:4500' is used as the contents
# for the cluster file. '-log' and '/my/dir(<>)=42' are passed
# as args to the FQL binary.
docker run docker.io/janderland/fql 'my_cluster:baoeA32@172.20.3.33:4500' -log '/my/dir(<>)=42'
```

Within the cluster file contents (first argument), any instances of a 
hostname wrapped in curly braces (e.g. '{my_hostname}') are replaced by the 
equivalent IP address. FDB doesn't support connecting to a cluster via 
hostnames, so this functional provides a workaround. This can simplify 
connecting to a Docker instance of FDB.

```bash
docker network create my_net
docker run --network my_net --name fdb -d foundationdb/foundationdb

# The substring '{fdb}' in the first argument will be replaced with
# the IP address of the FDB container started above before the cluster
# file is written to disk.
docker run --network my_net docker.io/janderland/fql 'docker:docker@{fdb}:4500' -log '/my/dir(<>)=42'
```

## Query Language

Here is the [syntax definition](syntax.ebnf) for the query language. Currently,
FQL is focused on reading & writing key-values created using the directory and
tuple layers. Reading or writing keys of arbitrary byte strings is not 
supported.

FQL queries are a textual representation of a specific key-value or a schema
describing the structure of many key-values. These queries have the ability to
write a key-value, read one or more key-values, and list directories.

### Components & Structure

This section will explain the components and structure of an FQL query. The
semantic meaning of these queries will be explained below in the [Kinds of 
Queries](#kinds-of-queries) section.

#### Primitives

FQL utilizes textual representations of the element types supported by the
tuple layer. These types are known as primitives. Besides as tuple elements,
primitives can also be used as the value portion of a key-value.

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

When primitives are used as tuple elements, they are encoded using the tuple 
layer. When they are used as the value portion of a key-value, they are 
encoded by FQL as outlined below.

| Type     | Encoding                          |
|:---------|:----------------------------------|
| `nil`    | `nil`                             |
| `int`    | 64-bit, endianness configurable   |
| `uint`   | 64-bit, endianness configurable   |
| `bool`   | single bit, `0` means false       |
| `float`  | IEEE 754, endianness configurable |
| `string` | ASCII byte string                 |
| `bytes`  | As provided                       |
| `uuid`   | 16-byte string                    |

Ideally, the encoding of these primitives would align with common community 
practices to maximize usefulness. Let me know if you believe it doesn't.

Even though a big int encoding is supported by the tuple layer, FQL does 
not currently support using big ints.

#### Directories

A directory is specified as a sequence of strings, each prefixed by a forward
slash:

```fql
/my/dir/path_way
```

The strings of the directory do not need quotes if they only contain
alphanumericals, underscores, dashes, or periods. To use other symbols, the
strings must be quoted:

```
/my/"dir@--\o/"/path_way
```

The quote character may be backslash escaped:

```
/my/"\"dir\""/path_way
```

#### Tuples

A tuple is specified as a sequence of elements, separated by commas, wrapped in
a pair of curly braces. The elements may be a tuple or any of the primitive
types.

```fql
("one", 2, 0x03, ( "subtuple" ), 5825d3f8-de5b-40c6-ac32-47ea8b98f7b4)
```

The last element of a tuple may be the `...` token.

```fql
(0xFF, "thing", ...)
```

Any combination of spaces, tabs, and newlines is allowed after the opening  
brace and commas.

```fql
(
  1,
  2,
  3,
)
```

#### Key-Values

A key-value is specified as a directory, tuple, equal symbol, and value appended
together:

```fql
/my/dir("this", 0)=0xabcf03
```

The value following the equal symbol may be any of the primitives or a tuple:

```fql
/my/dir(22.3, -8)=("another", "tuple")
```

The value can also be the `clear` token.

```fql
/some/where("home", "town", 88.3)=clear
```

#### Variables

A variable may be used in place of a directory element, tuple element, or value.

```fql
/my/dir/<>("first", <>, "third")=<>
```

If the variable is a tuple element or value, it may contain a list of primitive
types separated by pipes, except for the `nil` type. The variable may also
contain the `any` type which is equivalent to specifying every type. Specifying
no types is also equivalent to specifying the `any` type.

```fql
/my/dir("that", <int|float|bytes>)=<any>
```

### Kinds of Queries

This section showcases the various kinds of FQL queries, their semantic
meaning, and the equivalent FDB API calls implemented in Go.

#### Set

Set queries write a single key-value. The query must not contain the `clear`
or `...` tokens, nor a variable.

```fql
/my/dir("hello", "world")=42
```

```go
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.CreateOrOpen(tr, []string{"my", "dir"}, nil)
  if err != nil {
    return nil, err
  }

  val := make([]byte, 8)
  binary.LittleEndian.PutUint64(val, 42)
  tr.Set(dir.Pack(tuple.Tuple{"hello", "world"}), val)
  return nil, nil
})
```

#### Clear

Clear queries delete a single key-value. The query must contain the `clear`
token as it's value and must not contain the `...` token or variables.

```fql
/my/dir("hello", "world")=clear
```

```go
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

#### Read Single Key

Read-single queries read a single key-value. These queries must not have the
`...` token or a variable in their key. The value must be a variable.  
Deserialization of the value is attempted for each type in the order specified
by the variable. The first successful deserialization is used as the output. If
the value cannot be deserialized as any of the types specified then the
key-value is not returned or an error is returned, depending on configuration.

```fql
/my/dir(99.8, 7dfb10d1-2493-4fb5-928e-889fdc6a7136)=<int|string>
```

```go
db.Transact(func(tr fdb.Transaction) (interface{}, error) {
  dir, err := directory.Open(tr, []string{"my", "dir"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  val := tr.MustGet(dir.Pack(tuple.Tuple{99.8,
    tuple.UUID{0x7d, 0xfb, 0x10, 0xd1, 0x24, 0x93, 0x4f, 0xb5, 0x92, 0x8e, 0x88, 0x9f, 0xdc, 0x6a, 0x71, 0x36}))
  
     
  if len(val) == 8 {
      return binary.LittleEndian.Uint64(val), nil
  }
  return string(val), nil
})
```

As a shorthand, these query may be specified without the `=` token or value. 
This implies an empty variable as the value. In the code block below, the 
three queries are equivalent.

```fql
/my/dir(99.8, 7dfb10d1-2493-4fb5-928e-889fdc6a7136)
/my/dir(99.8, 7dfb10d1-2493-4fb5-928e-889fdc6a7136)=<>
/my/dir(99.8, 7dfb10d1-2493-4fb5-928e-889fdc6a7136)=<any>
```

#### Read Range of Keys

Read-many queries read a range of values based on a key prefix. These 
queries have a `...` token or a variable in their key. If a key-value is 
encountered which does not match the schema defined by the query then the
key-value is not returned or an error is returned, depending on configuration.
These queries are implemented using FDB's range-read mechanism with 
additional filtering performed on the client. Care must be taken with these 
queries as they may result in large amounts of data being sent to the 
client and most of the data being filtered out.

```fql
/people(3392, <string|int>, <>)=(<uint>, ...)
```

```go
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

    if len(tup) != 3 {
      return nil, fmt.Errorf("invalid kv: %v", kv)
    }

    switch tup[0].(type) {
    default:
      return nil, fmt.Errorf("invalid kv: %v", kv)
    case string | int64:
    }

    val, err := tuple.Unpack(kv.Value)
    if err != nil {
      return nil, fmt.Errorf("invalid kv: %v", kv)
    }
    if len(val) == 0 {
      return nil, fmt.Errorf("invalid kv: %v", kv)
    }
    if _, isInt := val[0].(uint64); !isInt {
      return nil, fmt.Errorf("invalid kv: %v", kv)
    }

    results = append(results, kv)
  }
  return results, nil
})
```

#### List Directory Paths

If only a directory is provided as a query, then the directory layer is queried.
Empty variables may be included as placeholders for any directory name.

```fql
/root/<>/items/<>
```

```go
db.ReadTransact(func(tr fdb.ReadTransaction) (interface{}, error) {
  root, err := directory.Open(tr, []string{"root"}, nil)
  if err != nil {
    if errors.Is(err, directory.ErrDirNotExists) {
      return nil, nil
    }
    return nil, err
  }

  oneDeep, err := root.List(tr, nil)
  if err != nil {
    return nil, err
  }

  var results [][]string
  for _, dir1 := range oneDeep {
    twoDeep, err := root.List(tr, []string{dir1, "items"})
    if err != nil {
      return nil, err
    }

    for _, dir2 := range twoDeep {
      results = append(results, []string{"root", dir1, dir2})
    }
  }
  return results, nil
})
```

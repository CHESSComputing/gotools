### CHESS/FOXDEN did composition
This codebase contains prototype codebase for did composition.
Given any JSON and set of keys it will construct did (dataset identifier).

```
# to build the code use
go build

# to compose did from example.json and foo, attributes
dset -encode example.json --attrs=foo,bla
/bla:value/foo:1

# decode back given did path
dset -decode "/bla:value/foo:1"
{
  "bla": "value",
  "foo": 1
}
```

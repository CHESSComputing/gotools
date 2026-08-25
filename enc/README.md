Basic tool to encrypt and decrypt given text on command line. The encrypted
result shown as hex string.

```
# build tool
go build

# encrypt some entry
./enc -cipher aes -entry test -secret bla -action encrypt
dd15043547b9d422d5859e853a33f71921b9257b2ca181183c6aa99411390a38

# encrypt some entry and store it into binary file
./enc -cipher aes -entry test -secret bla -action encrypt -fout /tmp/file.bin


# decrypt entry
./enc -cipher aes -entry dd15043547b9d422d5859e853a33f71921b9257b2ca181183c6aa99411390a38 -secret bla -action decrypt
test

# decrypt data from binary file
./enc -cipher aes -secret bla -action decrypt -fin /tmp/file.bin
```

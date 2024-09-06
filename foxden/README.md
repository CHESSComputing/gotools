# foxden CLI
`foxden` Command Line Interface (CLI) provides an uniform interface to all
[FOXDEN services](https://foxden.classe.cornell.edu:8344/docs). Its usage,
commands and examples can be fetched as following:

```
./foxden -h
foxden command line tool
Complete documentation at https://foxden.classe.cornell.edu:8344/docs

Usage:
  foxden [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      foxden config commamd
  describe    foxden describe command
  doi         foxden doi command
  help        Help about any command
  mc          foxden MaterialCommons commands
  meta        foxden MetaData commands
  ml          foxden ml commands
  prov        foxden provenance commands
  s3          foxden s3 commands
  search      foxden search commands
  spec        foxden SpecScans commands
  sync        foxden sync command
  token       foxden token commands
  version     foxden version commamd
  view        foxden view commands

Flags:
      --config string   config file (default is $HOME/.foxden.yaml)
  -h, --help            help for foxden
      --verbose int     verbosity level)

Use "foxden [command] --help" for more information about a command.
```

Each command has its own documentation section along with necessary examples, e.g.
```
./foxden meta --help
foxden MetaData commands to access FOXDEN MetaData service
Complete documentation at https://foxden.classe.cornell.edu:8344/docs

foxden meta <ls|rm|view> [options]
foxden meta add <file.json> {options}
options: --schema=<schema> --did-attrs=<attrs> --did-sep=<separator> --did-div=<divider> --json

Examples:

# list all meta data records:
foxden meta ls

# list specific meta-data record:
foxden meta view <DID>

# remove meta-data record:
foxden meta rm 123xyz

# add meta-data record with given schema, file and did attributes which create a did value:
foxden meta add <file.json> --schema=<schema> --did-attrs=beamline,btr,cycle,sample_name --did-sep=/ --did-div==

# the same as above since it is default values
foxden meta add <file.json> --schema=<schema>

# the same as above but provide json output
foxden meta add <file.json> --schema=<schema> --json
```

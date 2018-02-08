# Service Instance Reaper

This repository provides a utility for reaping, that is deleting, old Cloud Foundry service instances.

## Building

To build the utility, install go and govendor (see the [Go Development](docs/go.adoc) guide for instructions) and issue:
```bash
$ cd $GOPATH/src/github.com/pivotal-cf/service-instance-reaper
$ govendor install +local
```

## Simply installing

To install the utility without first cloning this repository or using govendor, issue:
```bash
$ go get -u github.com/pivotal-cf/service-instance-reaper

```

## Running

Ensure that `$GOPATH/bin` is on the path and then issue:
```bash
service-instance-reaper help
```
to display the command line syntax of the utility.

## Go Development

See the [Go Development](docs/go.adoc) guide.
(If you just want to build and install the plugin, simply install go and govendor.)

## Testing

Run the tests as follows:
```bash
$ cd $GOPATH/src/github.com/pivotal-cf/service-instance-reaper
$ govendor test +local
```

## License

The Service Instance Reaper is Open Source software released under the
[Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0.html).

## Contributing

Contributions are welcomed. Please refer to the [Contributor's Guide](CONTRIBUTING.md).


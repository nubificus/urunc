title: Setup a Build environment
------

Most of the steps are covered in the [installation](/installation) document. Please refer to it for:

- installing a recent version of Go (> 1.20)

- installing `containerd` and `runc`

- setting up the devmapper snapshotter

- installing `nerdctl` and the `CNI` plugins

- installing the relevant hypervisors

The next step is to clone the repo:

```bash
git clone https://github.com/nubificus/urunc.git
```

and start experimenting with the code. A first step could be running the tests:

```bash
make test
```

## Contribution Guidelines

Follow the
[contributing](/developer-guide/contribute) guidelines before submitting any pull requests.


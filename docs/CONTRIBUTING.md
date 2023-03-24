# TiDB Operator Development Guide

## Prerequisites

Please install [Go 1.19.x](https://go.dev/doc/install). If you want to run TiDB Operator locally, please also install the latest version of [Docker](https://www.docker.com/get-started/), [kind](https://kind.sigs.k8s.io/docs/user/quick-start/), [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) and [Helm](https://helm.sh/docs/intro/quickstart/).

## Workflow

### Step 1: Fork TiDB Operator on GitHub

Visit https://github.com/pingcap/tidb-operator

Click `Fork` button (top right) to establish a cloud-based fork.

### Step 2: Clone fork to local machine

Define a local working directory:

```sh
$ working_dir=$GOPATH/src/github.com/pingcap
```

Set `user` to match your github profile name:

```sh
$ user={your github profile name}
```

Create your clone:

```sh
$ mkdir -p $working_dir
$ cd $working_dir
$ git clone git@github.com:$user/tidb-operator.git
```

Set your clone to track upstream repository.

```sh
$ cd $working_dir/tidb-operator
$ git remote add upstream https://github.com/pingcap/tidb-operator
```

Since you don't have write access to the upstream repository, you need to disable pushing to upstream master:

```sh
$ git remote set-url --push upstream no_push
$ git remote -v
```

The output should look like:

```sh
origin    git@github.com:$(user)/tidb-operator.git (fetch)
origin    git@github.com:$(user)/tidb-operator.git (push)
upstream  https://github.com/pingcap/tidb-operator (fetch)
upstream  no_push (push)
```

### Step 3: Branch

Get your local master up to date:

```sh
$ cd $working_dir/tidb-operator
$ git fetch upstream
$ git checkout master
$ git rebase upstream/master
```

Branch from master:

```sh
$ git checkout -b myfeature
```

### Step 4: Develop

#### Edit the code

You can now edit the code on the `myfeature` branch.

#### Check

At first, you must have [jq](https://stedolan.github.io/jq/) installed.

Run following commands to check your code change.

```sh
$ make check
```

This will show errors if your code change does not pass checks (e.g. fmt, lint). Please fix them before submitting the PR.

If you change code related to CRD, such as type definitions in `pkg/apis/pingcap/v1alpha1/types.go`, please also run following commands to generate necessary code and artifacts.

```sh
$ hack/update-all.sh
```

#### Start TiDB Operator locally and do manual tests

At first, you must have [Docker](https://www.docker.com/get-started/) installed and running.

We uses [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to
start a Kubernetes cluster locally and
[kubectl](https://kubernetes.io/docs/reference/kubectl/overview/) must be
installed to access Kubernetes cluster.

You can refer to their official references to install them on your machine, or
run the following command to install them into our local binary directory:
`output/bin`.

```sh
$ ./hack/local-up-operator.sh -i
$ export PATH=$(pwd)/output/bin:$PATH
```

Make sure they are installed correctly:

```sh
$ kind --version
...
$ kubectl version --client
...
```

Create a Kubernetes cluster with `kind`:

```sh
$ kind create cluster
```

Build and run tidb-operator:

```sh
$ ./hack/local-up-operator.sh
```

Start a basic TiDB cluster:

```sh
$ kubectl apply -f examples/basic/tidb-cluster.yaml
```

#### Run unit tests

Before running your code in a real Kubernetes cluster, make sure it passes all (1300+) unit tests.

```sh
$ make test
```

#### Run e2e tests

Now you can run the following command to run all e2e test.

```sh
$ ./hack/e2e.sh
```

> **Note:**
>
> - You can run `make docker` if you only want to build images.
> - Running all e2e tests typically takes hours and consumes a lot of system resources, so it's better to limit specs to run, for example: `./hack/e2e.sh -- --ginkgo.focus='Basic'`.
> - It's possible to reuse the kind cluster, e.g pass `SKIP_DOWN=y` for the first time and pass `SKIP_UP=y SKIP_DOWN=y` later.
> - If you have configured multi docker registry repos, please ensure docker hub is used when building images.
> - `hack/run-in-container.sh` can start a dev container the same as our CI environment. This is the recommended way to run e2e tests, e.g: `./hack/run-in-container.sh sleep 1d`. You can start more than one terminals and run `./hack/run-in-container.sh` to enter into the same container for debugging. Run `./hack/run-in-container.sh -h` to see help.
> - We don't support bash version < 4 for now. For those who are using a not supported version of bash, especially macOS (which default bash version is 3.2) users, please run `hack/run-in-container.sh` to start a containerized environment or install bash 4+ manually.


Run the following command to see help:

```sh
$ ./hack/e2e.sh -h
```

Arguments for some e2e test pipelines run in our CI (including 180+ cases):

- pull-e2e-kind: `--ginkgo.focus='DMCluster|TiDBCluster' --ginkgo.skip="\[TiDBCluster:\sBasic\]"`
- pull-e2e-kind-across-kubernetes: `--ginkgo.focus='\[Across\sKubernetes\]' --install-dm-mysql=false`
- pull-e2e-kind-serial: `--ginkgo.focus='\[Serial\]' --install-operator=false`
- pull-e2e-kind-tikv-scale-simultaneously: `--ginkgo.focus='Scale\sin\ssimultaneously'`
- pull-e2e-kind-tngm: `--ginkgo.focus='TiDBNGMonitoring'`
- pull-e2e-kind-br: `--ginkgo.focus='Backup\sand\sRestore'`
- pull-e2e-kind-basic: `--ginkgo.focus='\[TiDBCluster:\sBasic\]' --install-dm-mysql=false`

In PR comments, you can run `/test ${case-name}` (e.g `/test pull-e2e-kind`) to trigger the case manually.

### Step 5: Keep your branch in sync

While on your `myfeature` branch, run the following commands:

```sh
$ git fetch upstream
$ git rebase upstream/master
```

### Step 6: Commit

Before you commit, make sure that all the checks and unit tests are passed:

```sh
$ make check
$ make test
```

Then commit your changes.

```sh
$ git commit
```

Likely you'll go back and edit/build/test some more than `commit --amend`
in a few cycles.

### Step 7: Push

When your commit is ready for review (or just to establish an offsite backup of your work),
push your branch to your fork on `github.com`:

```sh
$ git push -f origin myfeature
```

### Step 8: Create a pull request

1. Visit your fork at https://github.com/$user/tidb-operator (replace `$user` obviously).
2. Click the `Compare & pull request` button next to your `myfeature` branch.
3. Edit the description of the pull request to match your change, and if your pull request introduce a user-facing change, a release note is required.

> You can refer to [Release Notes Language Style Guide](./release-note-guide.md) for how to write proper release notes.

### Step 9: Get a code review

Once your pull request has been opened, it will be assigned to at least two
reviewers. Those reviewers will do a thorough code review, looking for
correctness, bugs, opportunities for improvement, documentation and comments,
and style.

Commit changes made in response to review comments to the same branch on your
fork.

Very small PRs are easy to review. Very large PRs are very difficult to
review.

## Developer Docs

There are api reference docs, design proposals, and other developer related docs in `docs` directory. Feel free to check things there. Happy Hacking!

# Building TiDB Operator from Source Code

## Go
TiDB Operator is written in [Go](https://golang.org). If you don't have a Go development environment, [set one up](https://golang.org/doc/code.html).

The version of Go should be 1.9 or later.

After Go is installed, you need to define `GOPATH` and modify `PATH` modified to access your Go binaries.

You can configure them as follows, or you can Google a setup as you like.

```sh
$ export GOPATH=$HOME/go
$ export PATH=$PATH:$GOPATH/bin
```

## Dependency management

TiDB Operator uses [retool](https://github.com/twitchtv/retool) to manage Go related tools.

```sh
$ go get -u github.com/twitchtv/retool
```

## Workflow

### Step 1: Fork TiDB Operator on GitHub

1. visit https://github.com/pingcap/tidb-operator
2. Click `Fork` button (top right) to establish a cloud-based fork.

### Step 2: Clone fork to local machine

Per Go's [workspace instructions](https://golang.org/doc/code.html#Workspaces), place TiDB Operator code on your
`GOPATH` using the following cloning procedure.

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
$ git remote add upstream git@github.com:pingcap/tidb-operator.git
```

Since you don't have write access to the upstream repository, you need to disable pushing to upstream master:

```sh
$ git remote set-url --push upstream no_push
$ git remote -v
```

The output should look like:

```
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

#### Run unit tests

Before running your code in a real Kubernetes cluster, make sure it passes all unit tests.

```sh
$ make test
```

#### Run e2e tests

For e2e tests, we recommend DinD Kubernetes environment. Follow [this guide](./local-dind-tutorial.md) to spin up a local DinD Kubernetes cluster.

Then you can build and push Docker images to the DinD Docker registry. The DinD Docker registry is available as `localhost:5000` both on the host machine and inside the DinD.

```sh
$ make docker-push
$ make e2e-docker-push
```

After Docker images are pushed to the DinD Docker registry, run e2e tests:

```sh
$ kubectl apply -f manifests/tidb-operator-e2e.yaml
```

### Step 5: Keep your branch in sync

While on your `myfeature` branch, run the following commands:

```sh
$ git fetch upstream
$ git rebase upstream/master
```

### Step 6: Commit

Commit your changes.

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

### Step 9: Get a code review

Once your pull request has been opened, it will be assigned to at least two
reviewers. Those reviewers will do a thorough code review, looking for
correctness, bugs, opportunities for improvement, documentation and comments,
and style.

Commit changes made in response to review comments to the same branch on your
fork.

Very small PRs are easy to review. Very large PRs are very difficult to
review.

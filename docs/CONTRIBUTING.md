# TiDB Operator(v2) Development Guide

<!-- toc -->
- [Prerequisites](#prerequisites)
- [Workflow](#workflow)
  - [Step 1: Fork TiDB Operator on GitHub](#step-1-fork-tidb-operator-on-github)
  - [Step 2: Clone fork to local machine](#step-2-clone-fork-to-local-machine)
  - [Step 3: Branch](#step-3-branch)
  - [Step 4: Develop](#step-4-develop)
    - [Edit the code](#edit-the-code)
    - [Genearate and check](#genearate-and-check)
    - [Start TiDB Operator locally and do manual tests](#start-tidb-operator-locally-and-do-manual-tests)
  - [Step 5: Keep your branch in sync](#step-5-keep-your-branch-in-sync)
  - [Step 6: Commit](#step-6-commit)
  - [Step 7: Push](#step-7-push)
  - [Step 8: Create a pull request](#step-8-create-a-pull-request)
  - [Step 9: Get a code review](#step-9-get-a-code-review)
- [Developer Docs](#developer-docs)
<!-- /toc -->

## Prerequisites

Please install [Go 1.23.x](https://go.dev/doc/install). If you want to run TiDB Operator locally, please also install the latest version of [Docker](https://www.docker.com/get-started/).

## Workflow

### Step 1: Fork TiDB Operator on GitHub

Visit [TiDB Operator](https://github.com/pingcap/tidb-operator)

Click `Fork` button (top right) to establish a cloud-based fork.

### Step 2: Clone fork to local machine

Define a local working directory:

```sh
working_dir=$GOPATH/src/github.com/pingcap
```

Set `user` to match your github profile name:

```sh
user={your github profile name}
```

Create your clone:

```sh
mkdir -p $working_dir
cd $working_dir
git clone git@github.com:$user/tidb-operator.git
```

Set your clone to track upstream repository.

```sh
cd $working_dir/tidb-operator
git remote add upstream https://github.com/pingcap/tidb-operator
```

Since you don't have write access to the upstream repository, you need to disable pushing to upstream master:

```sh
git remote set-url --push upstream no_push
git remote -v
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
cd $working_dir/tidb-operator
git fetch upstream
git checkout v2
git rebase upstream/v2
```

Branch from v2:

```sh
git checkout -b myfeature
```

### Step 4: Develop

#### Edit the code

You can now edit the code on the `myfeature` branch.

#### Genearate and check

Sometimes you may have to re-generate code by the following commands. If you don't know whether you need to run it, just run it.

```sh
make generate
```

Run following commands to check your code change.

```sh
make check
```

This will show errors if your code change does not pass checks (e.g. unit, lint). Please fix them before submitting the PR.


#### Start TiDB Operator locally and do manual tests

At first, you must have [Docker](https://www.docker.com/get-started/) installed and running.

We use [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to
start a Kubernetes cluster locally.

Run following commands to run e2e

```sh
make e2e
```

We use [Ginkgo](https://github.com/onsi/ginkgo) to write our e2e cases, So you can run a specified case by following commands

```sh
GINKGO_OPTS='--focus "regexp of case"' make e2e
```

You can also skip preparing e2e environment and run e2e directly by following commands

```sh
GINKGO_OPTS='--focus "regexp of case"' make e2e/run
```

You can see logs of operator by following commands

```sh
make logs/operator
```

And if you have some changes but just want to update operator, you can

```sh
make push && make reload/operator
```

You can also deploy and re-deploy manifests by

```sh
make deploy
```


### Step 5: Keep your branch in sync

While on your `myfeature` branch, run the following commands:

```sh
git fetch upstream
git rebase upstream/v2
```

### Step 6: Commit

Before you commit, make sure that all checks are passed:

```sh
make check
```

Then commit your changes.

```sh
git commit
```

Likely you'll go back and edit/build/test some more than `commit --amend`
in a few cycles.

### Step 7: Push

When your commit is ready for review (or just to establish an offsite backup of your work),
push your branch to your fork on `github.com`:

```sh
git push origin myfeature
```

### Step 8: Create a pull request

1. Visit your fork at `https://github.com/$user/tidb-operator` (replace `$user` obviously).
2. Click the `Compare & pull request` button next to your `myfeature` branch.
3. Edit the description of the pull request to match your change, and if your pull request introduce a user-facing change, a release note is required.

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

If you hope to submit a new feature, please see [RFCs Template](./rfcs/0000-template.md)


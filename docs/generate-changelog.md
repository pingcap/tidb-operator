# How to gather release notes and generate the changelog

1. install `release-notes` bin:

    ```shell
    $ go get k8s.io/release/cmd/release-notes@v0.2.0
    ```

2. generate the changelog:

    ```shell
    $ export GITHUB_TOKEN=<your-github-token>
    $ release-notes --start-rev <start> --end-rev <end> --branch <branch> --github-org pingcap --github-repo tidb-operator --output <output> --requiredAuthor ""
    ```

    - `<branch>` the branch to scrape, e.g. `release-1.1`
    - `<start>` the start revision, e.g. `v1.1.0-beta.1`
    - `<end>` the end revision, e.g. `release-1.1` (HEAD of the branch)
    - `<output>` the output file, e.g. `note.md`

    Full example:

    ```shell
    $ release-notes --start-rev v1.1.0-beta.1 --end-rev release-1.1 --branch release-1.1 --github-org pingcap --github-repo tidb-operator --output note.md --debug --format markdown --requiredAuthor ""
    $ cat note.md
    ## Other Notable Changes

    - Remove some not very useful update events ([#1486](https://github.com/pingcap/tidb-operator/pull/1486), [@weekface](https://github.com/weekface))
    ```

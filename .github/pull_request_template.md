<!--
Thank you for contributing to TiDB Operator!
Please complete the following template before creating a PR :D
Ref: TiDB Operator [CONTRIBUTING](https://github.com/pingcap/tidb-operator/blob/master/CONTRIBUTING.md) document
-->

### What problem does this PR solve?
<!--
Please describe the problem AS DETAILED AS POSSIBLE.
Add and ISSUE LINK WITH SUMMARY if exists.

For example:

    Fix the bug that syncing `Backup` CR will crash tidb-controller-manager pod when client TLS feature is enabled.

    Closes #xxx (issue number)
-->

### What is changed and how does it work?
<!--
Please describe the design that your implementation follows AS DETAILED AS POSSIBLE.

For example:

    The root cause is a nil pointer dereferencing. Add a `ptr != nil` check before access members of `ptr` to prevent the crash.
-->

### Code changes

- [ ] Has Go code change
- [ ] Has CI related scripts change

### Tests
<!-- AT LEAST ONE test must be included. -->

- [ ] Unit test <!-- If you added any unit test cases, check this box -->
- [ ] E2E test <!-- If you added any e2e test cases, check this box -->
- [ ] Manual test <!-- If this PR needs manual test, check this box, and add detailed manual test scripts or steps below, so that ANYONE CAN REPRODUCE IT. Ref: https://github.com/pingcap/tidb-operator/pull/3517 -->
- [ ] No code <!-- If this PR contains no code changes, check this box -->

### Side effects

- [ ] Breaking backward compatibility <!-- If this PR breaks things deployed with previous TiDB Operator versions, check this box -->
- [ ] Other side effects: <!-- Any other side effects, such as requiring additional storage / consumes substantial memory / potential reconcilation latency -->

### Related changes

- [ ] Need to cherry-pick to the release branch <!-- If this PR should also appear in the current release branch, check this box -->
- [ ] Need to update the documentation <!-- If this PR introduces new features / change previous usages, check this box -->

### Release Notes
<!--
If no need to add a release note, just type `NONE` in the following `release-note` block.
If the PR requires additional action from users to deploy the new release, start the release note with "ACTION REQUIRED: ".
-->
Please refer to [Release Notes Language Style Guide](https://github.com/pingcap/tidb-operator/blob/master/docs/release-note-guide.md) before writing the release note.

```release-note

```

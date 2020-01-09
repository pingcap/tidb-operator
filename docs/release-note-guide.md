# Release Notes Language Style Guide

When you write a release note for your pull request, make sure that your language style meets the following rules:

1. Include `ACTION REQUIRED:` at the beginning if the change requires user action, e.g. deprecating or abandoning features:

    - ACTION REQUIRED: Add the `timezone` support for [all charts]

2. Every note starts with the "do" form of a verb. For example:

    - Support backup to S3 with [Backup & Restore (BR)](https://github.com/pingcap/br)
    - Fix Docker ulimit configuring for the latest EKS AMI

3. Ensure no period at the end of note.

4. Use a single backquote (``) to frame the following elements in your release notes:

    - Custom Resource name
    - Variable name
    - Variable value
    - Error message
    - Single-line SQL statement
    - Configuration item name
    - Field name
    - The element that needs emphasis (for example, table name, column name, schema name, etc.)
  
5. Pay attention to the capitalization of the following terms that are often misspelled:

    - PD, TiKV, TiDB (not pd, tikv, tidb)
    - TiDB Operator or `tidb-operator` (not tidb operator)
    - TiDB Binlog (not tidb binlog)

6. The following templates are commonly used in release notes:

    - Fix the issue that ... when doing (an operation)/ when (... occurs)
    - Fix the issue that ... because ... (the cause of the problem)
    - Add the feature of (something/doing something) to do (the purpose)
    - Support (something/doing something)

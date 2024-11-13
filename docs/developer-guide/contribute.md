---
layout: default
title: "Contributing"
description: "Contributing guidelines"
---

Urunc is an open-source project licenced under the [Apache License 2.0](https://github.com/nubificus/urunc/blob/main/LICENSE).
We welcome anyone who would be interested in contributing to `urunc`.
As a first step, please take a look at the following document.
The current document provides a high level overview of `urunc`'s code structure, along with a few guidelines regarding contributions to the project.

## Table of contents:

1. [Code organization](#code-organization)
2. [How to contribute](#how-to-contribute)
3. [Opening an issue](#opening-an-issue)
4. [Requesting new features](#requesting-new-features)
5. [Submitting a PR](#submitting-a-pr)
6. [Style guide](#style-guide)
7. [Contact](#contact)

## Code organization

Urunc is written in Go and we structure the code and other files as follows:

- `/`: The root directory contains the Makefile to build `urunc`, along with other non-code files, such as the licence, readme and more.
- `/docs`: This directory contains all the documentation related to `urunc`, such as the installation guide, timestamping and more.
- `/cmd`: This directory contains handlers for the various command line options of `urunc` and the implementation of containerd-shim.
- `/internal/metrics`: This directory contains the implementation of the metrics logger, which is used for the internal measuring of `urunc`'s setup steps.
- `/pkg`: This directory contains the majority of the code for `urunc`. In particular, the subdirectory `/pkg/network/` contains network related code as expected, while the `/pkg/unikontainers/` subdirectory contains the main logic of `urunc`, along with the VMM/unikernel related logic.

Therefore, we expect any new documentation related files to be placed under `/docs` and any changes or new files in code to be either in the `/cmd/` or `/pkg/` directory.

## How to contribute

There are plenty of ways to contribute to an open source project, even without changing or touching the code.
Therefore, anyone who is interested in this project is very welcome to contribute in one of the following ways:

1.  Using `urunc`. Try it out yourself and let us know your experience. Did everything work well? Were the instructions clear?
2.  Improve or suggest changes to the documentation of the project. Documentation is very important for every project, hence any ideas on how to improve the documentation to make it more clear are more than welcome.
3.  Request new features. Any proposals for improving or adding new features are very welcome.
4.  Find a bug and report it. Bugs are everywhere and some are hidden very well. As a result, we would really appreciate it if someone found a bug and reported it to the maintainers.
5.  Make changes to the code. Improve the code, add new functionalities and make `urunc` even more useful.

## Opening an issue

We use Github issues to track bugs and requests for new features.
Anyone is welcome to open a new issue, which is either related to a bug or a request for a new feature.

### Reporting bugs

In order to report a bug or misbehavior in `urunc`, a user can open a new issue explaining the problem.
For the time being, we do not use any strict template for reporting any issues.
However, in order to easily identify and fix the problem, it would be very helpful to provide enough information.
In that context, when opening a new issue regarding a bug, we kindly ask you to:

- Mark the issue with the bug label
- Provide the following information:

    1. A short description of the bug.
    2. The respective logs both from the output and containerd.
    3. Urunc's version (either the commit's hash or the version).
    4. The CPU architecture, VMM and the Unikernel framework used.
    5. Any particular steps to reproduce the issue.
- Keep an eye on the issue for possible questions from the maintainers.

A template for an issue could be the following one:
```
## Description
An explanation of the issue 

## System info

- Urunc version:
- Arch:
- VMM:
- Unikernel:

## Steps to reproduce
A list of steps that can reproduce the issue.
```

### Requesting new features

We will be very happy to listen from users about new features that they would like to see in `urunc`.
One way to communicate such a request is using Github issues.
For the time being, we do not use any strict template for requesting new features.
However, we kindly ask you to mark the issue with the enhancement label and provide a description of the new feature.

## Submitting a PR

Anyone should feel free to submit a change or an addition to the codebase of `urunc`.
Currently, we use Github's Pull Requests (PRs) to submit changes to `urunc`'s codebase.
Before creating a new PR, please follow the guidelines below:

- Make sure that the changes do not break the building process of `urunc`.
- Make sure that all the tests run successfully.
- Make sure that no commit in a PR breaks the building process of `urunc`
- Make sure to sign-off your commits.
- Provide meaningful commit messages, describing shortly the changes.
- Provide a meaningful PR message

As soon as a new PR is created the following workflow will take place:

  1. The creator of the PR should invoke the tests by adding the `ok-to-test` label.
  2. If the tests pass, request from one or more `urunc`'s maintainers to review the PR.
  3. The reviewers submit their review.
  4. The author of the PR should address all the comments from the reviewers.
  5. As soon as a reviewer approves the PR, an action will add the appropriate git trailers in the PR's commits.
  6. The reviewer who accepted the changes will merge the new changes.

## Labels for the CI

We use github workflows to invoke some tests when a new PR opens for `urunc`.
In particular, we perform the following workflows tests:

- Linting of the commit message. Please check the [git commit message style](CONTRIBUTING.md#Git-commit-messages) below for more info.
- Spell check, since `urunc` repository contains its documentation too.
- License check
- Code linting for Go.
- Building artifacts for amd64 and aarch64.
- Unit tests
- End-to-end tests

For a better control over the tests and workflows that run in a PR, we define
three labels which can be used:

- `ok-to-test`: Runs a full CI workflow, meaning all lint tests (commit
  message, spellcheck, license), Go's linting, building for x86 and aarch64,
  unit tests and at last end-to-end tests.
- `skip-build`: Skips the building workflows along with unit and end-to end tests
  running all the linting tests. This is useful when
  the PR is related to docs and it can help for catching spelling errors etc. In
  addition, if the changes are not related to the codebase, running the
  end-to-end tests is not required and saves some time.
- `skip-lint`: Skips the linting phase. This is particularly useful on draft
  PRs, when we want to just test the functionality of the code (either a bug
  fix, or a new feature) and defer the cleanup/polishing of commits, code, and
  docs to when the PR will be ready for review.

**Note**: Both `skip-build` and `skip-lint` assume that the `ok-to-test` label
is added.

## Style guide

### Git commit messages

Please follow the below guidelines for your commit messages:

- Limit the first line to 72 characters or less.
- Limit all the other lines to 80 characters
- Follow the [Conventional Commits](https://www.conventionalcommits.org/)
  specification and, specifically, format the header as `<type>[optional scope]:
  <description>`, where `description` must not end with a fullstop and `type`
  can be one of:

  - *feat*: A new feature
  - *fix*: A bug fix
  - *docs*: Documentation only changes
  - *style*: Changes that do not affect the meaning of the code (white-space,
    formatting, missing semi-colons, etc)
  - *refactor*: A code change that neither fixes a bug nor adds a feature
  - *perf*: A code change that improves performance
  - *test*: Adding missing tests
  - *build*: Changes that affect the build system or external dependencies
    (example scopes: gulp, broccoli, npm)
  - *ci*: Changes to our CI configuration files and scripts (example scopes:
    Travis, Circle, BrowserStack, SauceLabs)
  - *chore*: Other changes that don't modify src or test files
  - *revert*: Reverts a previous commit
- In case the PR is associated with an issue, please refer to it, using the git trailer `Fixes: #Nr_issue`
- Always sign-off your commit message

### Golang code styde

We follow gofmt's rules on formatting GO code. Therefore, we ask all contributors to do the same.
Go provides the `gofmt` tool, which can be used for formatting your code.

## Contact

Feel free to contact any of the authors directly using their emails in the commit messages or using one of the below email addresses:

- urunc@nubificus.co.uk
- urunc@nubis-pc.eu

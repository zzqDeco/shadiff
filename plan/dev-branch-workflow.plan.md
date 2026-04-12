# Dev Branch Workflow Plan

## Goal

Establish `dev` as the integration branch for day-to-day changes while keeping `master` as the stable promotion baseline.

## Scope

In scope:

- create and publish `dev` from the current `master`
- document the new branch flow in contributor-facing docs
- clarify that the repository currently uses `master`, not `main`

Out of scope:

- renaming `master` to `main`
- branch protection rules or CI enforcement
- release automation for `dev -> master` promotion

## Approach

- Seed `dev` from the current `master` tip so both branches start aligned.
- Update repository guidance to use short-lived working branches created from `dev`.
- Route normal feature, fix, docs, refactor, and test pull requests into `dev` first.
- Keep `master` as the stable promotion branch and require a separate `dev -> master` pull request when ready.

## Tasks

- Create and push the `dev` branch from the current `master`.
- Update `plan/README.md` to track this workflow rollout.
- Update `AGENTS.md` and `CLAUDE.md` with the new branch policy.
- Update `README.md` and `README_CN.md` with a concise development branch flow note.
- Verify the remote branch exists and the docs use the same branch names consistently.

## Verification

- `git branch -a`
- `gh api repos/zzqDeco/shadiff/branches/dev`
- Manual review of updated docs for branch-name consistency

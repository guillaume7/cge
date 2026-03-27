# Cross-Repo Embedding Blueprint — VP3

## Intent

VP3 should make CGE adoptable in other repositories with a small, repeatable
integration kit.

However, VP3 should reach that goal by proving the pattern in this repo first,
then extracting the reusable pieces second.

That kit should be composable enough to fit repos with different:

- prompt stacks
- agent squads
- story/backlog conventions
- release workflows

## Minimum Reusable Assets

The reusable workflow pack should cover at least the delegated-subtask golden
path:

1. install or refresh instructions for CGE workflow assets
2. repo-local graph initialization and seeding
3. delegated-subtask kickoff workflow guidance
4. delegated-subtask finish workflow guidance
5. prompt snippets for kickoff, delegation, and handoff
6. skill or instruction snippets describing when to read and write graph memory

The initial implementation should optimize for repo-local usefulness and later
extract these pieces into a portable pack once the workflow feels natural here.

The current repo's biggest adoption gap is not only graph initialization; it is
getting prompts, skills, and instructions to rely on the graph naturally during
real work.

## Standard Repo Inputs

The first reusable version should assume common repo artifacts such as:

- `README.md`
- `docs/architecture/`
- `docs/plan/backlog.yaml`
- `docs/themes/`
- `.github/` prompts, agents, and skills

If some inputs are missing, the workflow should degrade gracefully instead of
failing the entire bootstrap flow.

## Composition Model

The reusable embedding should separate:

- **workflow commands** — stable CLI entry points such as init/start/finish
- **workflow assets** — prompts, skills, or instruction snippets
- **seed sources** — repo artifacts transformed into baseline graph memory
- **repo overrides** — local adjustments without forking the whole integration

This lets other repos adopt CGE without losing their own conventions.

The preferred shape is **composable snippets** rather than a rigid metadata
takeover.

## Adoption Success Signal

Another repo should be able to:

1. install CGE
2. run the workflow bootstrap
3. see a seeded graph and reusable workflow assets
4. start and finish non-trivial delegated subtasks with structured graph-backed briefs
5. customize the integration without breaking future upgrades

## Lightweight Automation Guidance

Reuse should favor lightweight automation such as:

- commands that bundle repeated best-practice steps
- snippets that remind agents when graph usage is appropriate
- helpers that prepare kickoff or handoff payloads
- small wrappers or hooks that make graph-backed kickoff and handoff the default
  for most non-trivial delegated tasks

Reuse should avoid opaque automation such as:

- hidden background graph mutation
- mandatory repo-wide wrappers that obscure normal CLI behavior
- integrations that make it hard for maintainers to understand what will happen

## Benchmarking Guidance

The reusable pattern should eventually support benchmark runs that compare:

- with graph-backed kickoff/handoff
- without graph-backed kickoff/handoff

on the same delegated-subtask task families and acceptance criteria.

At minimum, the benchmark should record:

- token or prompt volume
- recovery or orientation effort
- task quality/completeness
- handoff quality for the next agent

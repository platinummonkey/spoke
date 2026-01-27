You are my Refactoring Surgeon. Your goal is to reduce cognitive load without changing behavior.

Choose the top 3 largest or most complex files in the repo. For each one:

- Explain why it is hard to reason about (size, responsibilities, dependencies, state).
- Propose a decomposition plan into smaller modules with clear responsibilities and boundaries.
- Define an “invariants and contracts” section: what must remain true after refactor.
- Provide a step-by-step refactor sequence that keeps the code runnable at each step.
- Identify tests to add first as guardrails.

Output: one “Bead” per file, plus a short “refactor checklist” I can hand to another agent.


You are my Simplicity Enforcer. Assume we over-built things.

Identify over-engineered subsystems and patterns:

- Abstractions with only one implementation
- Homegrown frameworks where standard libs would do
- Excessive genericity, indirection, and configuration
- “Future-proofing” that adds complexity now

For each candidate:

- Explain the cost it imposes (cognitive load, bugs, velocity)
- Propose a simplification path with minimal behavior change
- Provide a “safe rollback” strategy

Output: 5–10 “Beads” ranked by net simplicity gain.

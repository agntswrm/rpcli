<!-- Do not edit or remove this section -->
This document exists for non-obvious, error-prone shortcomings in the codebase, the model, or the tooling that an agent cannot figure out by reading the code alone. No architecture overviews, file trees, build commands, or standard behavior. When you encounter something that belongs here, first consider whether a code change could eliminate it and suggest that to the user. Only document it here if it can't be reasonably fixed.

---

## Rules

- When any CLI command, flag, or behavior changes in `internal/commands/`, always update `SKILL.md` to match. Keep it concise: one line per command, inline `# comment` at the end.

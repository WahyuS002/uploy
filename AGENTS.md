# Codebase Knowledge Graph

This project uses `codebase-memory-mcp` to maintain a knowledge graph of the codebase.
Always prefer MCP graph tools over grep, glob, or file search for code discovery.

1. `search_graph` — find functions, classes, routes, and variables.
2. `trace_path` — trace callers and callees.
3. `get_code_snippet` — read a specific function or class.
4. `query_graph` — run complex graph queries.
5. `get_architecture` — inspect high-level structure.

Fall back to grep or glob only for string literals, errors, config values, non-code files, or insufficient graph results.

# UI Verification

Do not use BrowserAct for visual or UI verification unless the user explicitly requests it.

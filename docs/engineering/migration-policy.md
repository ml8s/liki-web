# Migration policy

Migrations are forward-only and have no down migrations.

Rules:

1. every migration ID is globally unique;
2. an ID that has entered `main` is frozen and must never be modified;
3. schema corrections require a new migration;
4. runtime code never reads legacy tables or columns;
5. a fresh database after migration must match the canonical schema snapshot;
6. deleting a legacy object requires a destructive migration;
7. destructive migrations require a fresh production backup and assembly test.

Do not add nullable JSON state columns. Model zero-or-one state as a child row
with required columns, or as an explicit enum/state table.

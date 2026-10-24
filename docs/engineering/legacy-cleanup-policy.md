# Legacy cleanup policy

No retired URL, product, scenario alias, or DB schema fallback is kept for
compatibility.

Before deletion, record the object in the legacy inventory with:

1. runtime references;
2. production row count;
3. business impact;
4. required backup;
5. deletion method.

Delete only when no runtime code, test, or current document depends on it.
Uncertain objects stay in the inventory and are reviewed later; they must not
silently remain as runtime compatibility branches.

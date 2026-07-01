# Memory Commands

Mova Context stores the active project memory in `memory.md` and can automatically archive older entries based on the configured retention policy.

## Read Memory

Display the current project memory.

```bash
mova memory-read [project]
```

---

## Clear Memory

Delete memory using different criteria.

### Delete all memory

Deletes both the active memory and all archived history.

```bash
mova memory-clear [project]
```

### Delete only archived memory

Keeps `memory.md` and removes all archived months.

```bash
mova memory-clear [project] --archived
```

### Delete a specific day

Removes all entries from a given date.

```bash
mova memory-clear [project] --date 2024-06-15
```

### Delete a date range

Removes all entries between two dates.

```bash
mova memory-clear [project] --from 2024-06-01 --to 2024-06-30
```

### Keep active memory

Deletes archived memory while preserving `memory.md`.

```bash
mova memory-clear [project] --keep-active
```

---

# Configure Automatic Archive

Enable or disable automatic archiving and configure the retention period.

### Enable automatic archive

```bash
mova memory-config [project] enable
```

### Disable automatic archive

```bash
mova memory-config [project] disable
```

### Configure retention

Set the number of days that entries remain in the active memory before being archived.

```bash
mova memory-config [project] days N
```

Example:

```bash
mova memory-config my-project days 30
```

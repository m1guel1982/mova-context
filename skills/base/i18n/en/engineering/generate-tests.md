# Objective

Generate consistent and minimal tests.
KISS+DRY: see `kiss-dry-core.md`.

# Structure

```txt
describe('Module') {
  describe('method()') {
    it('should [result] when [condition]')
    it('should throw [error] when [condition]')
  }
}
```

# Minimum Coverage per Function

Successful case · invalid/null input · edge case (empty, 0) · system error (DB down, 404)

# Rules

One `expect` per test when possible · explicit hardcoded test data, no magic factories · mocks only for HTTP/DB/filesystem · test name = documentation

## jikan

an open source terminal based time tracker .

### run

```sh
go run .
```

Time sessions are stored in a local SQLite database named `jikan.db` by
default. Use a different database file with:

```sh
go run . --db path/to/jikan.db
```

Export completed timer instances to CSV from the terminal UI with `e`, or from
the command line with:

```sh
go run . --export jikan_sessions.csv
```

### keys

- `j` / `k`: move through projects
- `space` / `enter`: start or stop the selected project timer
- `n`: add a new project
- `r`: reset the selected project
- `e`: export completed sessions to `jikan_sessions.csv`
- `q`: quit

The CSV contains one row per completed timer instance with project name, start
time, end time, and duration columns. The storage code uses Go's `database/sql`
package so the local SQLite implementation can be replaced with PostgreSQL
later without changing the Bubble Tea model.

# go-mine

Interactive terminal data explorer for CSV, TSV, Parquet, and JSON files. Built with [bubbletea](https://github.com/charmbracelet/bubbletea) and [golars](https://github.com/msjurset/golars).

## Features

- Load and explore CSV, TSV, Parquet, and JSON files
- Five interactive views: Table, Stats, Filter, SQL, and Columns
- Paginated table with vim-style navigation and column sorting
- Per-column statistics with distribution histograms for numeric data
- Filter rows with expressions (`age > 30 AND city == "Seattle"`)
- Regex matching (`name.matches("^A.*")`), startswith, endswith
- Export filtered data to CSV, Parquet, or JSON
- SQL query interface with full SELECT/WHERE/GROUP BY/ORDER BY support and interactive results table
- Column detail view with type info, null counts, unique values, and samples
- Row detail overlay for inspecting individual records
- Built-in sample data generator for quick demos

## Install

```
make deploy
```

This builds the binary, installs it to `~/.local/bin/`, installs the man page, and sets up zsh completions.

## Usage

```
go-mine [flags] <file.csv|file.tsv|file.parquet|file.json>
go-mine -generate [-rows N]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-generate` | `false` | Generate sample data instead of loading a file |
| `-rows` | `10000` | Number of rows to generate (with `-generate`) |
| `-info` | `false` | Print data summary to stdout and exit |
| `-version` | — | Print version and exit |
| `-completion` | — | Print shell completion script (`zsh`, `bash`) |

### Interactive Keys

| Key | Action |
|-----|--------|
| `1`-`5` / `tab` | Switch views: Table, Stats, Filter, SQL, Columns |
| `j`/`k` or `↑`/`↓` | Navigate rows / select column |
| `h`/`l` or `←`/`→` | Scroll columns |
| `s` | Sort by current column (asc → desc → none) |
| `S` | Clear sort |
| `enter` | Open row detail / apply filter (blank clears) / execute SQL |
| `esc` | Toggle between SQL input and result table navigation |
| `pgup`/`pgdn` | Page through data |
| `g`/`G` | Jump to top/bottom |
| `ctrl+e` | Export current data to file |
| `?` | Toggle help |
| `q` | Quit |

### Filter Syntax

```
column > 42
column == "text"
column >= 10 AND column <= 100
column.is_null
column.is_not_null
column.contains("substr")
column.matches("^regex.*pattern$")
column.startswith("prefix")
column.endswith("suffix")
```

### Examples

```bash
# Explore a CSV file
go-mine data.csv

# Open a Parquet file
go-mine sales.parquet

# Explore tab-separated data
go-mine records.tsv

# Generate sample data and explore
go-mine -generate -rows 50000

# Print schema and stats without launching TUI
go-mine -info data.csv

# Install zsh completions
go-mine -completion zsh > ~/.oh-my-zsh/custom/completions/_go-mine
```

## Build

```
make build
```

## License

MIT

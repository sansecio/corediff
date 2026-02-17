# Magento Corediff

![](https://buq.eu/screenshots/6595XfnX5wwUPzbFQGkU0GgN.png)

A forensic tool to quickly find unauthorized modifications in an open source code base, such as Magento. Corediff compares each line of code with a database of 4.3M legitimate code hashes and shows you the lines that have not been seen before. A bit like [@NYT_first_said](https://maxbittker.github.io/clear-pipes/).

> _"Corediff saved us countless hours"_

> _"Very useful to gauge foreign codebases"_

Corediff was created by Sansec, specialist in Magento security and digital forensics since 2010. Corediff analysis helped to uncover numerous cases of server side payment skimming that would otherwise have gone undetected.

# Usage

## Scanning

```
corediff [OPTIONS] <path>...
```

Scan a codebase against the hash database. In default mode, only official platform paths are checked. Use `--ignore-paths` to scan all files.

| Flag | Description |
|------|-------------|
| `-d, --database` | Hash database path (default: download Sansec database) |
| `-i, --ignore-paths` | Scan everything, not just core paths |
| `-s, --suspect` | Show suspect code lines only |
| `-t, --text` | Scan all valid UTF-8 text files |
| `--no-platform` | Don't check for app root |
| `-f, --path-filter` | Filter paths before diffing (e.g. `vendor/magento`) |
| `-v, --verbose` | Verbose output (`-vv` for per-file details) |

In the following example, Corediff reports a malicious backdoor in `cron.php`:

![](https://buq.eu/screenshots/y76R3uN9CrCFN6GEji4uSPtM.png)

## Database management

### Index local paths

```bash
corediff db index -d custom.db <path>...
```

### Index Packagist packages

```bash
corediff db index -d m2.db -p brick/math composer/ca-bundle guzzlehttp/guzzle
```

The `-p` / `--packagist` flag treats positional arguments as Packagist package names. Supports version pinning with `vendor/pkg:1.2.3` or `vendor/pkg@1.2.3`.

### Index from composer.json

```bash
corediff db index -d m2.db --composer /path/to/composer.json
```

### Update previously indexed packages

```bash
corediff db index -d m2.db --update
```

### Merge databases

```bash
corediff db merge -d all.db magento1.db magento2.db *.db
```

### Self-update

```bash
corediff update
```

# Installation

Use our binary package (available for Linux & Mac, arm64 & amd64)

```sh
osarch=$(uname -sm | tr 'LD ' 'ld-')
curl https://sansec.io/downloads/$osarch/corediff -O
chmod 755 corediff
./corediff <store-path> | less -SR
```

Or compile from source (requires recent Go version):

```sh
go install github.com/sansecio/corediff/cmd/corediff@latest
corediff <store-path>
```

At the first run, `corediff` will automatically download the hash database.

# Community contributed datasets

[@fros_it](https://twitter.com/fros_it) has kindly contributed hashes for his collection of Magento Connect extensions, including all available historical copies. Download the [extension hash database](https://sansec.io/downloads/corediff-db/m1ext.db) here (62MB) and use it like this:

![](https://buq.eu/screenshots/RXdQ1Mmg5KliivMtK6DlHTcP.png)

# Contributing

Adding or maintaining hashes?

```bash
# Index a local path
corediff db index -d custom.db <path>

# Index Packagist packages
corediff db index -d custom.db -p vendor/package

# Merge databases for release
corediff db merge -d all.db magento1.db magento2.db *.db
```

In some cases, it is better to not add file paths to the hash database. All paths found in the database will be examined and reported in default mode. Should many varieties exist for a particular file (such as in DI/compiled code), we would rather exclude its scanning from default output, until we can be reasonably certain to have coverage for 99%+ versions out there. So for volatile paths, use the `--ignore-paths` flag when indexing.

Contributions welcome! Naturally, we only accept hashes from trusted sources.

# About

Created by Willem de Groot. Malware contributions welcome.

(C) 2023-2026 Sansec BV

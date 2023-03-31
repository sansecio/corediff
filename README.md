# Magento Corediff

![](https://buq.eu/screenshots/6595XfnX5wwUPzbFQGkU0GgN.png)

A forensic tool to quickly find unauthorized modifications in an open source code base, such as Magento. Corediff compares each line of code with a database of 1.7M legitimate code hashes and shows you the lines that have not been seen before. A bit like [@NYT_first_said](https://maxbittker.github.io/clear-pipes/).

> _"Corediff saved us countless hours"_

> _"Very useful to gauge foreign codebases"_

Corediff was created by [Sansec](https://sansec.io/?corediff), specialists in Magento security and digital forensics since 2010. Corediff analysis helped us to uncover numerous cases of server side payment skimming that would otherwise have gone undetected.

# Usage

```
Usage:
  corediff [OPTIONS] <path>...

Application Options:
  -d, --database=     Hash database path (default: download Sansec database)
  -a, --add           Add new hashes to DB, do not check
  -m, --merge         Merge databases
  -i, --ignore-paths  Scan everything, not just core paths.
      --no-cms        Don't check for CMS root when adding hashes. Do add file paths.
  -v, --verbose       Show what is going on
```

In the following example, Corediff reports a malicious backdoor in `cron.php`:

![](https://buq.eu/screenshots/y76R3uN9CrCFN6GEji4uSPtM.png)

In default mode, Corediff will only check official Magento paths. In order to find these, you should point Corediff to the root of a Magento installation.

Alternatively you can scan all files with the `--ignore-paths` option. NB this will produce more output and requires more interpretation by a developer or forensic analyst.

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
git clone https://github.com/sansecio/corediff.git
cd corediff
go run . <store-path>
```

At the first run, `corediff` will automatically download the Sansec hash database.

# Community contributed datasets

[@fros_it](https://twitter.com/fros_it) has kindly contributed hashes for his collection of Magento Connect extensions, including all available historical copies. Download the [extension hash database](https://sansec.io/downloads/corediff-db/m1ext.db) here (62MB) and use it like this:

![](https://buq.eu/screenshots/RXdQ1Mmg5KliivMtK6DlHTcP.png)

# Todo

- [ ] Compression of hash db? Eg https://github.com/Smerity/govarint, https://github.com/bits-and-blooms/bloom

# Contributing

Adding or maintaining hashes?

```bash
# Create or update custom.db with all hashes from `<path>`.
corediff --database=custom.db --add <path>

# Merge databases for release
corediff --database=all.db --merge magento1.db magento2.db *.db
```

In some cases, it is better to not add file paths to the hash database. All paths found in the database will be examined and reported in default mode. Should many varieties exist for a particular file (such as in DI/compiled code), we would rather exclude its scanning from default output, until we can be reasonably certain to have coverage for 99%+ versions out there. So for volatile paths, use the `--add --ignore-paths` options.

Contributions welcome! Naturally, we only accept hashes from trusted sources. [Contact us to discuss your contribution](mailto:info@sansec.io).

# About Sansec

Sansec's flagship software [eComscan](https://sansec.io/?corediff) is used by ecommerce agencies, law enforcement and PCI forensic investigators. We are proud to open source many of our internal tools and hope that it will benefit our partners and customers. Malware contributions welcome.

(C) 2023 [Sansec BV](https://sansec.io/?corediff) // info@sansec.io

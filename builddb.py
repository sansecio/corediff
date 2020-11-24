#!/usr/bin/env python3

"""
Magento Core Differ
(C) 2020 Sansec BV https://sansec.io

Quickly find core modifications in any Magento 1 or 2 code base.

See https://github.com/sansecio/magento-corediff 
"""

from __future__ import print_function
import os
import sys
import hashlib
import time
from io import open  # py26

DBPATH = 'corediff.bin'
EXTS = ('php', 'phtml', 'js', 'htaccess', 'sh')
SKIP_LINE_PATTERN = ('* ', '/*', '//', '#')


def find_files(startpath):
    if os.path.isfile(startpath):
        yield startpath
        return

    for root, _, files in os.walk(startpath):
        for f in files:
            if f.rpartition('.')[2] in EXTS:
                yield os.path.join(root, f)


def md5(src):
    return hashlib.md5(src.strip().encode('utf-8')).digest()


def store(path, base, db):
    # first, add path without base to db
    relpath = "path:" + path[len(base) + 1:]
    relpath_md5 = md5(relpath)
    if relpath_md5 not in db:
        print("Adding relpath to db:", relpath)
        db.add(relpath_md5)

    with open(path, encoding='utf-8', errors='ignore') as fh:
        for line in fh:
            db.add(md5(line))


def check(path, db):
    with open(path, encoding='utf-8', errors='ignore') as fh:
        found = []
        for line in fh:
            if any(line.lstrip().startswith(p) for p in SKIP_LINE_PATTERN):
                continue
            if md5(line) in db:
                continue
            found.append(line)

    if found:
        print("\n\n>>>", path)
        for line in found:
            print(line.rstrip())


def usage():
    print("Usage: %s <add|check> <path>" % sys.argv[0])
    sys.exit(1)


def loaddb(path):
    db = set()
    if os.path.isfile(path):
        with open(path, 'rb') as fh:
            h = fh.read(16)
            while h:
                db.add(h)
                h = fh.read(16)
    print("Loaded %d hashes" % len(db))
    return db


def savedb(path, db):
    print("Saving %d hashes" % len(db))
    with open(path, 'wb') as fh:
        for k in db:
            fh.write(k)


if __name__ == '__main__':

    try:
        verb = sys.argv[1]
        target = sys.argv[2]
        assert verb in ['add', 'check']
    except:
        usage()

    start = time.time()
    db = loaddb(DBPATH)
    print("That took",  time.time() - start)
    base = os.path.abspath(target)
    targetisdir = os.path.isdir(base)

    if verb == 'add':
        for path in find_files(base):
            store(path, base, db)
        savedb(DBPATH, db)
    elif verb == 'check':
        for path in find_files(base):
            # skip non-core files
            relpath = path[len(base) + 1:]
            if targetisdir and md5("path:"+relpath) not in db:
                continue

            check(path, db)

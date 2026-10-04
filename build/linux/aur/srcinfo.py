#!/usr/bin/env python3
"""Prints .SRCINFO for the simple single-package PKGBUILD beside it: scalar and array fields only."""
import re
import sys

text = open(sys.argv[1]).read()
fields = {}
for m in re.finditer(r"^(\w+)=(\(.*?\)|'[^']*'|\S+)$", text, re.M | re.S):
    key, raw = m.groups()
    if raw.startswith("("):
        fields[key] = re.findall(r"""'([^']*)'|"([^"]*)\"""", raw[1:-1])
        fields[key] = [a or b for a, b in fields[key]]
    else:
        fields[key] = [raw.strip("'")]

ver = fields["pkgver"][0]
subst = lambda v: v.replace("${pkgver}", ver).replace("$pkgver", ver)
arches = fields["arch"]
order = ["pkgdesc", "pkgver", "pkgrel", "url", "arch", "license", "depends", "provides", "conflicts"]
order += ["source", "sha256sums"]
for a in arches:
    order += [f"source_{a}", f"sha256sums_{a}"]

print(f"pkgbase = {fields['pkgname'][0]}")
for key in order:
    for v in fields.get(key, []):
        print(f"\t{key} = {subst(v)}")
print()
print(f"pkgname = {fields['pkgname'][0]}")

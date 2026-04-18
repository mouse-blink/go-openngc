#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Adam Stevko
# SPDX-License-Identifier: MIT
"""Generate golden test files and object fixture data from PyOngc.

Prerequisites:
    pip install pyongc

Usage:
    python scripts/generate_testdata.py

Outputs:
    testdata/golden/*.txt         -- byte-exact CLI output for golden file tests
    testdata/golden/commands.json -- manifest of CLI commands for TestGolden
    testdata/objects.json         -- per-object fixture data (flat array) for TestFixtureDriven
    testdata/catalog_fields.json  -- full catalog dump for TestFullCatalogParity
    testdata/errors.json          -- PyOngc error message fixtures for TestErrorMessageParity
    testdata/fixtures_meta.json   -- separations and stats metadata
"""

import json
import math
import os
import subprocess
import sys

# Test objects covering all edge cases
TEST_OBJECTS = [
    # Galaxies
    "NGC0001",    # Galaxy in Peg, has identifiers, radvel, redshift
    "NGC0224",    # M31, Andromeda Galaxy, common name
    "NGC5248",    # Galaxy in Boo, uptovmag=10 test case
    # HII regions
    "NGC7000",    # North America Nebula, common name, OpenNGC notes
    # Supernova remnant
    "NGC1952",    # M1, Crab Nebula
    # Planetary nebula (central star data)
    "NGC7027",    # PN in Cyg, should have central star mags
    "NGC6720",    # M57, Ring Nebula
    # Clusters
    "NGC0869",    # h Persei, open cluster (separation test pair)
    "NGC0884",    # chi Persei, open cluster (separation test pair)
    "NGC5139",    # Omega Centauri, globular cluster
    "NGC7789",    # Open cluster in Cas
    # IC objects
    "IC0673",     # Galaxy in Leo (nearby test)
    "IC1396",     # Cl+N in Cep
    # Messier lookups
    "M1",         # resolves to NGC1952
    "M31",        # resolves to NGC0224
    "M42",        # Orion Nebula
    "M45",        # Pleiades (addendum object, no NGC number)
    "M57",        # Ring Nebula
    # Edge cases
    "NGC0001A",   # Component letter
]

# CLI commands to capture as golden files — positional args after flags
CLI_COMMANDS = [
    # view without details
    (["ongc", "view", "NGC0001"], "view_ngc0001.txt"),
    (["ongc", "view", "NGC7000"], "view_ngc7000.txt"),
    (["ongc", "view", "M1"], "view_m1.txt"),
    # view with details (flags before positionals for Go stdlib flag)
    (["ongc", "view", "--details", "NGC0001"], "view_ngc0001_details.txt"),
    (["ongc", "view", "--details", "NGC7000"], "view_ngc7000_details.txt"),
    (["ongc", "view", "--details", "M1"], "view_m1_details.txt"),
    (["ongc", "view", "--details", "NGC7027"], "view_ngc7027_details.txt"),
    (["ongc", "view", "--details", "NGC0224"], "view_ngc0224_details.txt"),
    (["ongc", "view", "--details", "NGC6720"], "view_ngc6720_details.txt"),
    (["ongc", "view", "--details", "NGC5139"], "view_ngc5139_details.txt"),
    (["ongc", "view", "--details", "IC1396"], "view_ic1396_details.txt"),
    # nearby
    (["ongc", "nearby", "11:08:44", "-00:09:01.3"], "nearby_ic0673.txt"),
    (["ongc", "nearby", "--radius", "30", "20:59:17.14", "+44:31:43.6"], "nearby_ngc7000.txt"),
    # neighbors (flags before positionals)
    (["ongc", "neighbors", "--radius", "15", "ngc521"], "neighbors_ngc521.txt"),
    (["ongc", "neighbors", "--radius", "30", "NGC7000"], "neighbors_ngc7000.txt"),
    (["ongc", "neighbors", "--radius", "60", "M31"], "neighbors_m31.txt"),
    # separation
    (["ongc", "separation", "ngc869", "ngc884"], "separation_double_cluster.txt"),
    (["ongc", "separation", "NGC0224", "NGC0221"], "separation_m31_m32.txt"),
    # search
    (["ongc", "search", "--catalog", "NGC", "--constellation", "Aql,Boo", "--type", "G", "--uptovmag", "10"], "search_galaxy_boo_aql.txt"),
    (["ongc", "search", "--type", "SNR"], "search_snr.txt"),
    (["ongc", "search", "--catalog", "NGC", "--type", "PN", "--constellation", "Cyg"], "search_pn_cyg.txt"),
    (["ongc", "search", "--withname", "--type", "GCl", "--catalog", "NGC"], "search_named_gcl.txt"),
    # stats
    (["ongc", "stats"], "stats.txt"),
]

# Manifest for TestGolden — Go CLI requires flags before positional args
COMMANDS_JSON = [
    {"filename": "view_ngc0001.txt",          "args": ["view", "NGC0001"]},
    {"filename": "view_ngc7000.txt",          "args": ["view", "NGC7000"]},
    {"filename": "view_m1.txt",               "args": ["view", "M1"]},
    {"filename": "view_ngc0001_details.txt",  "args": ["view", "--details", "NGC0001"]},
    {"filename": "view_ngc7000_details.txt",  "args": ["view", "--details", "NGC7000"]},
    {"filename": "view_m1_details.txt",       "args": ["view", "--details", "M1"]},
    {"filename": "view_ngc7027_details.txt",  "args": ["view", "--details", "NGC7027"]},
    {"filename": "view_ngc0224_details.txt",  "args": ["view", "--details", "NGC0224"]},
    {"filename": "view_ngc6720_details.txt",  "args": ["view", "--details", "NGC6720"]},
    {"filename": "view_ngc5139_details.txt",  "args": ["view", "--details", "NGC5139"]},
    {"filename": "view_ic1396_details.txt",   "args": ["view", "--details", "IC1396"]},
    {"filename": "nearby_ic0673.txt",         "args": ["nearby", "11:08:44", "-00:09:01.3"]},
    {"filename": "nearby_ngc7000.txt",        "args": ["nearby", "--radius", "30", "20:59:17.14", "+44:31:43.6"]},
    {"filename": "neighbors_ngc521.txt",      "args": ["neighbors", "--radius", "15", "ngc521"]},
    {"filename": "neighbors_ngc7000.txt",     "args": ["neighbors", "--radius", "30", "NGC7000"]},
    {"filename": "neighbors_m31.txt",         "args": ["neighbors", "--radius", "60", "M31"]},
    {"filename": "separation_double_cluster.txt", "args": ["separation", "ngc869", "ngc884"]},
    {"filename": "separation_m31_m32.txt",    "args": ["separation", "NGC0224", "NGC0221"]},
    {"filename": "search_galaxy_boo_aql.txt", "args": ["search", "--catalog", "NGC", "--constellation", "Aql,Boo", "--type", "G", "--uptovmag", "10"]},
    {"filename": "search_snr.txt",            "args": ["search", "--type", "SNR"]},
    {"filename": "search_pn_cyg.txt",         "args": ["search", "--catalog", "NGC", "--type", "PN", "--constellation", "Cyg"]},
    {"filename": "search_named_gcl.txt",      "args": ["search", "--withname", "--type", "GCl", "--catalog", "NGC"]},
    {"filename": "stats.txt",                 "args": ["stats"]},
]


def obj_ra_rad(obj):
    """Return RA in radians from PyOngc Dso, or 0.0 if not available."""
    coords = obj.rad_coords
    if coords is None:
        return 0.0
    return float(coords[0])


def obj_dec_rad(obj):
    """Return Dec in radians from PyOngc Dso, or 0.0 if not available."""
    coords = obj.rad_coords
    if coords is None:
        return 0.0
    return float(coords[1])


def messier_str(obj):
    """Return plain integer string for messier number, or empty string."""
    messier = obj._messier
    if messier:
        try:
            return str(int(messier))
        except (ValueError, TypeError):
            return str(messier) if messier else ""
    return ""


def generate_golden_files():
    """Capture PyOngc CLI output as golden files."""
    golden_dir = os.path.join("testdata", "golden")
    os.makedirs(golden_dir, exist_ok=True)

    for cmd, filename in CLI_COMMANDS:
        filepath = os.path.join(golden_dir, filename)
        print(f"  {' '.join(cmd)} -> {filepath}")
        try:
            result = subprocess.run(
                cmd, capture_output=True, text=True, timeout=30
            )
            with open(filepath, "w", newline="\n") as f:
                f.write(result.stdout)
            if result.returncode != 0:
                print(f"    WARNING: exit code {result.returncode}")
                if result.stderr:
                    print(f"    stderr: {result.stderr.strip()}")
        except FileNotFoundError:
            print(f"    ERROR: 'ongc' CLI not found. Install with: pip install pyongc")
            sys.exit(1)
        except subprocess.TimeoutExpired:
            print(f"    ERROR: command timed out")
            sys.exit(1)

    # Write the commands manifest consumed by TestGolden
    manifest_path = os.path.join(golden_dir, "commands.json")
    with open(manifest_path, "w", newline="\n") as f:
        json.dump(COMMANDS_JSON, f, indent=2)
    print(f"  -> {manifest_path} ({len(COMMANDS_JSON)} entries)")


def generate_object_fixtures():
    """Export per-object fixture data as a flat JSON array for TestFixtureDriven."""
    try:
        from pyongc import ongc
    except ImportError:
        print("ERROR: pyongc not installed. Run: pip install pyongc")
        sys.exit(1)

    os.makedirs("testdata", exist_ok=True)

    entries = []
    for name in TEST_OBJECTS:
        try:
            obj = ongc.get(name)
        except Exception as e:
            print(f"  WARNING: get({name}) raised {e}, skipping")
            continue
        if obj is None:
            print(f"  WARNING: {name} not found, skipping")
            continue

        entry = {
            "name": obj.name,
            "type": obj.type,
            "ra_rad": obj_ra_rad(obj),
            "dec_rad": obj_dec_rad(obj),
            "const": obj.constellation or "",
            "messier": messier_str(obj),
        }
        entries.append(entry)
        print(f"  {name} -> {obj.name} ({obj.type})")

    filepath = os.path.join("testdata", "objects.json")
    with open(filepath, "w", newline="\n") as f:
        json.dump(entries, f, indent=2, ensure_ascii=False)
    print(f"  -> {filepath} ({len(entries)} objects)")


def generate_catalog_fields():
    """Full catalog dump for TestFullCatalogParity — one compact entry per object."""
    try:
        from pyongc import ongc
    except ImportError:
        print("ERROR: pyongc not installed")
        sys.exit(1)

    os.makedirs("testdata", exist_ok=True)

    all_objects = ongc.listObjects()
    entries = []
    for obj in all_objects:
        entries.append({
            "name": obj.name,
            "type": obj.type,
            "ra_rad": obj_ra_rad(obj),
            "dec_rad": obj_dec_rad(obj),
            "const": obj.constellation or "",
            "messier": messier_str(obj),
        })

    path = os.path.join("testdata", "catalog_fields.json")
    with open(path, "w", newline="\n") as f:
        json.dump(entries, f, indent=None, separators=(',', ':'))
    print(f"  -> {path} ({len(entries)} objects)")


def generate_errors():
    """PyOngc error message fixtures for TestErrorMessageParity."""
    try:
        from pyongc import ongc
    except ImportError:
        print("ERROR: pyongc not installed")
        sys.exit(1)

    os.makedirs("testdata", exist_ok=True)

    errors_list = []
    for bad_name in ["NOTREAL", "NGC9999", "IC0000"]:
        try:
            ongc.get(bad_name)
        except Exception as e:
            errors_list.append({"op": "get", "input": bad_name, "error": str(e)})

    if not errors_list:
        print("  WARNING: no errors captured — check PyOngc behavior for bad names")

    path = os.path.join("testdata", "errors.json")
    with open(path, "w", newline="\n") as f:
        json.dump(errors_list, f, indent=2)
    print(f"  -> {path} ({len(errors_list)} entries)")


def generate_fixtures_meta():
    """Separation pairs and stats metadata, separate from objects.json."""
    try:
        from pyongc import ongc
    except ImportError:
        print("ERROR: pyongc not installed")
        sys.exit(1)

    os.makedirs("testdata", exist_ok=True)

    separations = []
    pairs = [("NGC0869", "NGC0884"), ("NGC0224", "NGC0221")]
    for name1, name2 in pairs:
        sep = ongc.getSeparation(name1, name2)
        separations.append({
            "obj1": name1,
            "obj2": name2,
            "separation_deg": float(sep[0]),
            "delta_ra_deg": float(sep[1]),
            "delta_dec_deg": float(sep[2]),
        })
        print(f"  separation({name1}, {name2}) = {sep[0]:.6f}°")

    stats = ongc.stats()
    meta = {
        "pyongc_version": stats[0] if isinstance(stats, tuple) else str(stats),
        "database_version": str(stats[1]) if isinstance(stats, tuple) and len(stats) > 1 else "",
        "total_objects": stats[2] if isinstance(stats, tuple) and len(stats) > 2 else 0,
        "separations": separations,
    }

    path = os.path.join("testdata", "fixtures_meta.json")
    with open(path, "w", newline="\n") as f:
        json.dump(meta, f, indent=2, ensure_ascii=False)
    print(f"  -> {path}")


def main():
    print("Generating golden files...")
    generate_golden_files()
    print()

    print("Generating object fixtures (objects.json)...")
    generate_object_fixtures()
    print()

    print("Generating full catalog fields (catalog_fields.json)...")
    generate_catalog_fields()
    print()

    print("Generating error fixtures (errors.json)...")
    generate_errors()
    print()

    print("Generating fixtures metadata (fixtures_meta.json)...")
    generate_fixtures_meta()
    print()

    print("Done! Test fixtures ready in testdata/")


if __name__ == "__main__":
    main()

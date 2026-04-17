#!/usr/bin/env python3
"""Generate golden test files and object fixture data from PyOngc.

Prerequisites:
    pip install pyongc

Usage:
    python scripts/generate_testdata.py

Outputs:
    testdata/golden/*.txt    — byte-exact CLI output for golden file tests
    testdata/objects.json    — per-object field values for property tests
"""

import json
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
    # Duplicates and nonexistent
    # (add more as needed)
]

# CLI commands to capture as golden files
CLI_COMMANDS = [
    # view without details
    (["ongc", "view", "NGC0001"], "view_ngc0001.txt"),
    (["ongc", "view", "NGC7000"], "view_ngc7000.txt"),
    (["ongc", "view", "M1"], "view_m1.txt"),
    # view with details
    (["ongc", "view", "NGC0001", "--details"], "view_ngc0001_details.txt"),
    (["ongc", "view", "NGC7000", "--details"], "view_ngc7000_details.txt"),
    (["ongc", "view", "M1", "--details"], "view_m1_details.txt"),
    (["ongc", "view", "NGC7027", "--details"], "view_ngc7027_details.txt"),
    (["ongc", "view", "NGC0224", "--details"], "view_ngc0224_details.txt"),
    (["ongc", "view", "NGC6720", "--details"], "view_ngc6720_details.txt"),
    (["ongc", "view", "NGC5139", "--details"], "view_ngc5139_details.txt"),
    (["ongc", "view", "IC1396", "--details"], "view_ic1396_details.txt"),
    # nearby
    (["ongc", "nearby", "11:08:44", "-00:09:01.3"], "nearby_ic0673.txt"),
    (["ongc", "nearby", "20:59:17.14", "+44:31:43.6", "--radius", "30"], "nearby_ngc7000.txt"),
    # neighbors
    (["ongc", "neighbors", "ngc521", "--radius", "15"], "neighbors_ngc521.txt"),
    (["ongc", "neighbors", "NGC7000", "--radius", "30"], "neighbors_ngc7000.txt"),
    (["ongc", "neighbors", "M31", "--radius", "60"], "neighbors_m31.txt"),
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


def generate_golden_files():
    """Capture CLI output as golden files."""
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


def generate_object_fixtures():
    """Export per-object field values as JSON."""
    try:
        import pyongc as ongc
    except ImportError:
        print("ERROR: pyongc not installed. Run: pip install pyongc")
        sys.exit(1)

    objects = []
    for name in TEST_OBJECTS:
        obj = ongc.get(name)
        if obj is None:
            print(f"  WARNING: {name} not found, skipping")
            continue

        entry = {
            "input_name": name,
            "name": obj.name,
            "type": obj.type,
            "ra": list(obj.ra) if obj.ra is not None else None,
            "dec": list(obj.dec) if obj.dec is not None else None,
            "const": obj.const,
            "majax": obj.majax,
            "minax": obj.minax,
            "pa": obj.pa,
            "bmag": obj.bmag,
            "vmag": obj.vmag,
            "jmag": obj.jmag,
            "hmag": obj.hmag,
            "kmag": obj.kmag,
            "sbrightn": obj.sbrightn,
            "hubble": obj.hubble,
            "parallax": obj.parallax,
            "pmra": obj.pmra,
            "pmdec": obj.pmdec,
            "radvel": obj.radvel,
            "redshift": obj.redshift,
            "cstarumag": obj.cstarumag,
            "cstarbmag": obj.cstarbmag,
            "cstarvmag": obj.cstarvmag,
            "messier": obj.messier,
            "ngc": obj.ngc if hasattr(obj, "ngc") else "",
            "ic": obj.ic if hasattr(obj, "ic") else "",
            "str": str(obj),
        }

        # Handle list/tuple fields carefully
        for field in ["cstarnames", "identifiers", "commonnames"]:
            val = getattr(obj, field, None)
            if val is None:
                entry[field] = None
            elif isinstance(val, (list, tuple)):
                entry[field] = list(val)
            elif isinstance(val, str):
                entry[field] = [v.strip() for v in val.split(",") if v.strip()] if val else []
            else:
                entry[field] = None

        # Notes
        entry["nednotes"] = getattr(obj, "nednotes", "") or ""
        entry["ongnotes"] = getattr(obj, "ongnotes", "") or ""

        # Replace NaN with None for JSON
        for key, val in entry.items():
            if isinstance(val, float) and (val != val):  # NaN check
                entry[key] = None

        objects.append(entry)
        print(f"  {name} -> {obj.name} ({obj.type})")

    # Also capture separation reference values
    separations = []
    pairs = [("NGC0869", "NGC0884"), ("NGC0224", "NGC0221")]
    for name1, name2 in pairs:
        sep = ongc.getSeparation(name1, name2)
        separations.append({
            "obj1": name1,
            "obj2": name2,
            "separation_deg": sep[0],
            "delta_ra_deg": sep[1],
            "delta_dec_deg": sep[2],
        })
        print(f"  separation({name1}, {name2}) = {sep[0]:.6f}°")

    # Capture stats
    stats = ongc.stats()

    fixture = {
        "pyongc_version": stats[0] if isinstance(stats, tuple) else str(stats),
        "database_version": stats[1] if isinstance(stats, tuple) else "",
        "total_objects": stats[2] if isinstance(stats, tuple) else 0,
        "objects": objects,
        "separations": separations,
    }

    filepath = os.path.join("testdata", "objects.json")
    with open(filepath, "w", newline="\n") as f:
        json.dump(fixture, f, indent=2, ensure_ascii=False)
    print(f"  -> {filepath} ({len(objects)} objects)")


def main():
    print("Generating golden files...")
    generate_golden_files()
    print()
    print("Generating object fixtures...")
    generate_object_fixtures()
    print()
    print("Done! Test fixtures ready in testdata/")


if __name__ == "__main__":
    main()

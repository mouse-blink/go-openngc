#!/bin/bash
# SPDX-FileCopyrightText: 2026 Adam Stevko
# SPDX-License-Identifier: MIT
# scripts/download_data.sh — downloads OpenNGC data at the latest upstream tag.
set -euo pipefail

TAG=$(curl -fsSL https://api.github.com/repos/mattiaverga/OpenNGC/tags \
        | python3 -c 'import json,sys; print(json.load(sys.stdin)[0]["name"])')

echo "Downloading OpenNGC ${TAG}"
curl -fsSL "https://raw.githubusercontent.com/mattiaverga/OpenNGC/${TAG}/database_files/NGC.csv" \
    -o ongc/data/NGC.csv
curl -fsSL "https://raw.githubusercontent.com/mattiaverga/OpenNGC/${TAG}/database_files/addendum.csv" \
    -o ongc/data/addendum.csv
echo "${TAG}" > ongc/data/VERSION
echo "Done. VERSION=${TAG}"

#!/usr/bin/env python3
import os
import re
from pathlib import Path

# Base directory
docs_site_path = Path("/home/beagle/work/brummer/docs-site")

# Find all markdown files (excluding node_modules)
md_files = []
for file in docs_site_path.rglob("*.md"):
    if "node_modules" not in str(file):
        md_files.append(file)

# Regex to find image references
img_pattern = re.compile(r'!\[([^\]]*)\]\(([^)]+)\)')

# Store all image references
image_refs = []

for md_file in md_files:
    with open(md_file, 'r', encoding='utf-8') as f:
        content = f.read()
        matches = img_pattern.findall(content)
        for alt_text, img_path in matches:
            if any(img_path.endswith(ext) for ext in ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp']):
                image_refs.append({
                    'file': str(md_file.relative_to(docs_site_path)),
                    'alt_text': alt_text,
                    'path': img_path
                })

# Print only documentation images
print("=== Documentation Image References ===")
for ref in sorted(image_refs, key=lambda x: x['path']):
    print(f"File: {ref['file']}")
    print(f"  Alt: {ref['alt_text']}")
    print(f"  Path: {ref['path']}")
    print()

# Extract unique image paths and normalize them
unique_paths = set()
for ref in image_refs:
    path = ref['path']
    # Normalize relative paths
    if path.startswith('../'):
        path = path[3:]  # Remove ../
    elif path.startswith('./'):
        path = path[2:]  # Remove ./
    unique_paths.add(path)

print("\n=== Unique Image Paths (normalized) ===")
for path in sorted(unique_paths):
    print(path)

# Check which files exist
print("\n=== Missing Images ===")
for path in sorted(unique_paths):
    full_path = docs_site_path / "static" / path
    if not full_path.exists():
        print(f"MISSING: {path}")
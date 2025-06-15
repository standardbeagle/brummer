#!/usr/bin/env python3
import os
import re
from pathlib import Path

# Base directory
docs_site_path = Path("/home/beagle/work/brummer/docs-site")

# Find all markdown files
md_files = list(docs_site_path.rglob("*.md"))

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

# Print results
print("=== All Image References Found ===")
for ref in sorted(image_refs, key=lambda x: x['path']):
    print(f"File: {ref['file']}")
    print(f"  Alt: {ref['alt_text']}")
    print(f"  Path: {ref['path']}")
    print()

# Extract unique image paths
unique_paths = sorted(set(ref['path'] for ref in image_refs))
print("\n=== Unique Image Paths ===")
for path in unique_paths:
    print(path)
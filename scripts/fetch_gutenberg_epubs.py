#!/usr/bin/env python3
"""
Fetch random EPUB books from Project Gutenberg OPDS feed.

This script downloads approximately 100 EPUB files from Project Gutenberg
and organizes them in a directory structure: Author/Book_name/file.epub

Usage:
    python fetch_gutenberg_epubs.py [--output-dir PATH] [--count N]
"""

import argparse
import random
import re
import sys
import time
import urllib.request
import xml.etree.ElementTree as ET
from pathlib import Path
from urllib.parse import urljoin

# Project Gutenberg OPDS feed base URL
GUTENBERG_BASE = "https://www.gutenberg.org"
GUTENBERG_CATALOG_URL = "https://www.gutenberg.org/ebooks/search.opds/?sort_order=random"

# Namespaces used in OPDS/Atom feeds
ATOM_NS = "http://www.w3.org/2005/Atom"


def sanitize_filename(name: str) -> str:
    """Remove or replace characters that are invalid in filenames."""
    name = re.sub(r'[<>:"/\\|?*]', '_', name)
    name = name.strip().strip('.')
    if len(name) > 100:
        name = name[:100]
    return name or "Unknown"


def fetch_xml(url: str, max_retries: int = 3) -> ET.Element:
    """Fetch and parse XML from the given URL."""
    for attempt in range(max_retries):
        try:
            req = urllib.request.Request(
                url,
                headers={'User-Agent': 'BiblioOPDS-EPUBFetcher/1.0'}
            )
            with urllib.request.urlopen(req, timeout=30) as response:
                return ET.fromstring(response.read())
        except Exception as e:
            if attempt < max_retries - 1:
                print(f"  Retry {attempt + 1}/{max_retries}: {e}")
                time.sleep(2 ** attempt)
            else:
                raise


def download_file(url: str, dest_path: Path, max_retries: int = 3) -> bool:
    """Download a file from URL to destination path."""
    for attempt in range(max_retries):
        try:
            req = urllib.request.Request(
                url,
                headers={'User-Agent': 'BiblioOPDS-EPUBFetcher/1.0'}
            )
            with urllib.request.urlopen(req, timeout=60) as response:
                dest_path.parent.mkdir(parents=True, exist_ok=True)
                with open(dest_path, 'wb') as f:
                    f.write(response.read())
                return True
        except Exception as e:
            if attempt < max_retries - 1:
                print(f"    Retry {attempt + 1}/{max_retries}: {e}")
                time.sleep(2 ** attempt)
            else:
                print(f"    Failed: {e}")
                return False
    return False


def parse_catalog_entry(entry: ET.Element) -> dict | None:
    """
    Parse a catalog entry to extract book ID, title, and author.
    Returns dict with id, title, author, epub_url or None.
    """
    # Get book ID from the subsection link (e.g., /ebooks/12345.opds -> 12345)
    for link in entry.findall(f'{{{ATOM_NS}}}link'):
        if link.get('rel') == 'subsection':
            href = link.get('href', '')
            # Extract ID from /ebooks/12345.opds
            match = re.search(r'/ebooks/(\d+)\.opds', href)
            if match:
                book_id = match.group(1)
                
                # Get title
                title_elem = entry.find(f'{{{ATOM_NS}}}title')
                title = title_elem.text if title_elem is not None and title_elem.text else "Unknown Title"
                
                # Get author from content element (format: "Author Name")
                author = "Unknown Author"
                content_elem = entry.find(f'{{{ATOM_NS}}}content')
                if content_elem is not None and content_elem.text:
                    author = content_elem.text.strip()
                
                # Construct direct EPUB URL (with images)
                epub_url = f"{GUTENBERG_BASE}/ebooks/{book_id}.epub.images"
                
                return {
                    'id': book_id,
                    'title': title,
                    'author': author,
                    'epub_url': epub_url,
                }
    return None


def fetch_catalog_page(url: str) -> tuple[list[dict], str | None]:
    """
    Fetch books from an OPDS catalog page.
    Returns (list of book dicts, next page URL or None).
    """
    books = []
    next_url = None
    
    try:
        root = fetch_xml(url)
    except Exception as e:
        print(f"Failed to fetch catalog: {e}")
        return books, None
    
    # Parse all entries
    for entry in root.findall(f'{{{ATOM_NS}}}entry'):
        book = parse_catalog_entry(entry)
        if book:
            books.append(book)
    
    # Find next page link
    for link in root.findall(f'{{{ATOM_NS}}}link'):
        if link.get('rel') == 'next':
            next_url = link.get('href')
            if next_url and not next_url.startswith('http'):
                next_url = urljoin(GUTENBERG_BASE, next_url)
            break
    
    return books, next_url


def collect_books(target_count: int) -> list[dict]:
    """Collect book info from Project Gutenberg OPDS catalog."""
    books = []
    seen_ids = set()
    
    print(f"Collecting catalog entries for {target_count} books...")
    
    # Random feed doesn't paginate - call it multiple times to get more books
    fetch_count = 0
    max_fetches = (target_count // 15) + 5  # Each page has ~20 books, add buffer
    
    while len(books) < target_count and fetch_count < max_fetches:
        fetch_count += 1
        print(f"  Fetching random batch {fetch_count}...")
        page_books, _ = fetch_catalog_page(GUTENBERG_CATALOG_URL)
        
        new_count = 0
        for book in page_books:
            if book['id'] not in seen_ids:
                seen_ids.add(book['id'])
                books.append(book)
                new_count += 1
        
        print(f"    Found {new_count} new entries, total: {len(books)}")
        
        if len(books) >= target_count:
            break
        
        time.sleep(0.5)  # Be nice to the server
    
    random.shuffle(books)
    return books[:target_count]


def main():
    parser = argparse.ArgumentParser(
        description='Fetch random EPUB books from Project Gutenberg OPDS feed'
    )
    parser.add_argument(
        '--output-dir', '-o',
        default='./testdata/epub_library',
        help='Output directory for downloaded EPUBs (default: ./testdata/epub_library)'
    )
    parser.add_argument(
        '--count', '-n',
        type=int,
        default=100,
        help='Number of EPUB files to download (default: 100)'
    )
    parser.add_argument(
        '--flat',
        action='store_true',
        help='Store all files in root directory without subdirectories'
    )
    
    args = parser.parse_args()
    
    output_dir = Path(args.output_dir)
    target_count = args.count
    
    print(f"Target: {target_count} EPUB files")
    print(f"Output directory: {output_dir.absolute()}")
    print()
    
    # Collect book info from catalog (single step - no detail page fetch needed)
    books = collect_books(target_count)
    
    if not books:
        print("No books found in OPDS catalog!")
        sys.exit(1)
    
    print(f"\nDownloading {len(books)} books...\n")
    
    downloaded = 0
    failed = 0
    
    for i, book in enumerate(books):
        title = book['title']
        author = book['author']
        epub_url = book['epub_url']
        
        # Sanitize names for filesystem
        safe_author = sanitize_filename(author)
        safe_title = sanitize_filename(title)
        
        # Determine file path
        if args.flat:
            filename = f"{safe_author} - {safe_title}.epub"
            file_path = output_dir / sanitize_filename(filename)
        else:
            filename = f"{safe_title}.epub"
            file_path = output_dir / safe_author / safe_title / filename
        
        # Skip if already exists
        if file_path.exists():
            print(f"[{i + 1}/{len(books)}] Exists: {title[:50]}")
            downloaded += 1
            continue
        
        print(f"[{i + 1}/{len(books)}] {title[:50]}")
        print(f"    Author: {author}")
        
        if download_file(epub_url, file_path):
            downloaded += 1
            print(f"    Saved: {file_path.name}")
        else:
            failed += 1
        
        time.sleep(0.3)
    
    print()
    print("=" * 60)
    print(f"Download complete!")
    print(f"  Successfully downloaded: {downloaded}")
    print(f"  Failed: {failed}")
    print(f"  Output directory: {output_dir.absolute()}")


if __name__ == '__main__':
    main()

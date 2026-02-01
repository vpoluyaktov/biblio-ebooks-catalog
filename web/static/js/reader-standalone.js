// Standalone Ebook Reader with Pagination

class StandaloneReader {
    constructor() {
        this.currentBook = null;
        this.currentChapterIndex = 0;
        this.currentPage = 0;
        this.totalPages = 1;
        this.pages = [];
        this.settings = this.loadSettings();
        this.bookId = this.getBookIdFromURL();
        this.resizeTimeout = null;
        this.layout = this.settings.layout || 'single'; // 'single' or 'double'
        this.init();
    }

    getBookIdFromURL() {
        const params = new URLSearchParams(window.location.search);
        return params.get('id');
    }

    init() {
        this.applySettings();
        this.attachEventListeners();
        
        if (this.bookId) {
            this.loadBook(this.bookId);
        } else {
            this.showError('No book ID provided');
        }
    }

    attachEventListeners() {
        // Close button - navigate back to catalog
        document.getElementById('reader-close').addEventListener('click', () => {
            // Try to go back in history, or redirect to catalog
            if (window.history.length > 1) {
                window.history.back();
            } else {
                // Fallback: redirect to catalog root
                window.location.href = window.APP_BASE_PATH || '/catalog';
            }
        });

        // Chapter navigation buttons
        document.getElementById('reader-prev-chapter').addEventListener('click', () => this.previousChapter());
        document.getElementById('reader-next-chapter').addEventListener('click', () => this.nextChapter());

        // Page navigation buttons
        document.getElementById('reader-page-prev').addEventListener('click', () => this.previousPage());
        document.getElementById('reader-page-next').addEventListener('click', () => this.nextPage());

        // Click zones for page navigation
        document.getElementById('reader-nav-prev').addEventListener('click', () => this.previousPage());
        document.getElementById('reader-nav-next').addEventListener('click', () => this.nextPage());

        // Font controls
        document.getElementById('reader-font-family').addEventListener('change', (e) => {
            this.settings.fontFamily = e.target.value;
            this.saveSettings();
            this.applySettings();
            this.repaginate();
        });

        document.getElementById('reader-font-decrease').addEventListener('click', () => this.changeFontSize(-1));
        document.getElementById('reader-font-increase').addEventListener('click', () => this.changeFontSize(1));

        // Theme control
        document.getElementById('reader-theme').addEventListener('change', (e) => {
            this.settings.theme = e.target.value;
            this.saveSettings();
            this.applySettings();
        });

        // Line height control
        document.getElementById('reader-line-height').addEventListener('change', (e) => {
            this.settings.lineHeight = e.target.value;
            this.saveSettings();
            this.applySettings();
            this.repaginate();
        });

        // Layout toggle (1 page / 2 pages)
        document.getElementById('reader-layout-toggle').addEventListener('click', () => {
            this.toggleLayout();
        });

        // Chapters dropdown
        document.getElementById('reader-chapters-btn').addEventListener('click', (e) => {
            e.stopPropagation();
            const menu = document.getElementById('reader-chapters-menu');
            menu.classList.toggle('active');
        });

        // Close chapters menu when clicking outside
        document.addEventListener('click', (e) => {
            const menu = document.getElementById('reader-chapters-menu');
            const btn = document.getElementById('reader-chapters-btn');
            if (!menu.contains(e.target) && !btn.contains(e.target)) {
                menu.classList.remove('active');
            }
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ignore if typing in an input
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'SELECT') return;
            
            switch(e.key) {
                case 'Escape':
                    if (window.opener) window.close();
                    else history.back();
                    break;
                case 'ArrowLeft':
                    e.preventDefault();
                    this.previousPage();
                    break;
                case 'ArrowRight':
                    e.preventDefault();
                    this.nextPage();
                    break;
                case 'ArrowUp':
                    e.preventDefault();
                    this.previousChapter();
                    break;
                case 'ArrowDown':
                    e.preventDefault();
                    this.nextChapter();
                    break;
                case 'Home':
                    e.preventDefault();
                    this.goToPage(0);
                    break;
                case 'End':
                    e.preventDefault();
                    this.goToPage(this.totalPages - 1);
                    break;
                case ' ':
                    e.preventDefault();
                    if (e.shiftKey) this.previousPage();
                    else this.nextPage();
                    break;
            }
        });

        // Touch gestures for mobile
        let touchStartX = 0;
        let touchStartY = 0;

        const mainArea = document.getElementById('reader-main');
        mainArea.addEventListener('touchstart', (e) => {
            touchStartX = e.changedTouches[0].screenX;
            touchStartY = e.changedTouches[0].screenY;
        }, { passive: true });

        mainArea.addEventListener('touchend', (e) => {
            const touchEndX = e.changedTouches[0].screenX;
            const touchEndY = e.changedTouches[0].screenY;
            const diffX = touchStartX - touchEndX;
            const diffY = touchStartY - touchEndY;
            const swipeThreshold = 50;

            // Only handle horizontal swipes (ignore vertical scrolling attempts)
            if (Math.abs(diffX) > swipeThreshold && Math.abs(diffX) > Math.abs(diffY)) {
                if (diffX > 0) {
                    this.nextPage();
                } else {
                    this.previousPage();
                }
            }
        }, { passive: true });

        // Handle window resize - repaginate
        window.addEventListener('resize', () => {
            clearTimeout(this.resizeTimeout);
            this.resizeTimeout = setTimeout(() => {
                this.repaginate();
            }, 250);
        });
    }

    async loadBook(bookId) {
        try {
            this.showLoading();

            const basePath = window.APP_BASE_PATH || '';
            const response = await fetch(`${basePath}/api/books/${bookId}/content`);
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || 'Failed to load book content');
            }

            this.currentBook = await response.json();
            this.currentChapterIndex = 0;
            this.currentPage = 0;

            // Update page title and UI
            document.title = this.currentBook.title || 'Ebook Reader';
            document.getElementById('reader-book-title').textContent = this.currentBook.title || 'Unknown Title';
            document.getElementById('reader-book-author').textContent = this.currentBook.author || '';

            // Build chapters menu
            this.buildChaptersMenu();

            // Display first chapter
            this.displayChapter();

        } catch (error) {
            console.error('Error loading book:', error);
            this.showError(error.message);
        }
    }

    buildChaptersMenu() {
        const menu = document.getElementById('reader-chapters-menu');
        menu.innerHTML = '';

        this.currentBook.chapters.forEach((chapter, index) => {
            const item = document.createElement('div');
            item.className = 'reader-chapter-item';
            if (index === this.currentChapterIndex) {
                item.classList.add('active');
            }

            item.innerHTML = `
                <span class="reader-chapter-number">${index + 1}.</span>
                ${this.escapeHtml(chapter.title || 'Untitled')}
            `;

            item.addEventListener('click', () => {
                this.currentChapterIndex = index;
                this.currentPage = 0;
                this.displayChapter();
                menu.classList.remove('active');
            });

            menu.appendChild(item);
        });
    }

    displayChapter() {
        if (!this.currentBook || !this.currentBook.chapters[this.currentChapterIndex]) {
            return;
        }

        const chapter = this.currentBook.chapters[this.currentChapterIndex];
        const pageContentLeft = document.getElementById('reader-page-content-left');
        const pageContentRight = document.getElementById('reader-page-content-right');

        // Apply settings to both pages
        [pageContentLeft, pageContentRight].forEach(pageContent => {
            if (pageContent) {
                pageContent.setAttribute('data-font-size', this.settings.fontSize);
                pageContent.setAttribute('data-font-family', this.settings.fontFamily);
                pageContent.style.lineHeight = this.settings.lineHeight;
            }
        });

        // Set chapter content to left page (will be paginated)
        pageContentLeft.innerHTML = chapter.content;

        // Paginate the content
        this.paginateContent();

        // Update navigation
        this.updateChapterNavigation();
        this.updateChaptersMenuActive();
    }

    paginateContent() {
        const pageContentLeft = document.getElementById('reader-page-content-left');
        const containerLeft = document.getElementById('reader-page-left');
        
        // Use FIXED page dimensions for consistency
        // Single page: max-width 720px, padding 56px each side = 608px content
        // Double page: max-width 600px each, padding 56px each side = 488px content
        const SINGLE_PAGE_WIDTH = 608;
        const DOUBLE_PAGE_WIDTH = 488;
        const PADDING_VERTICAL = 96; // 48px top + 48px bottom
        
        const isDoubleLayout = this.layout === 'double' && window.innerWidth > 1000;
        const fixedColumnWidth = isDoubleLayout ? DOUBLE_PAGE_WIDTH : SINGLE_PAGE_WIDTH;
        
        // Get height from container (this is more stable)
        const containerHeight = containerLeft.clientHeight;
        const availableHeight = Math.max(400, containerHeight - PADDING_VERTICAL);

        // Store original content
        const originalContent = pageContentLeft.innerHTML;
        
        // Create a temporary container with CSS columns to measure total width needed
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = originalContent;
        tempDiv.style.cssText = `
            position: absolute;
            visibility: hidden;
            height: ${availableHeight}px;
            width: ${fixedColumnWidth}px;
            column-width: ${fixedColumnWidth}px;
            column-gap: 0;
            column-fill: auto;
            font-size: ${getComputedStyle(pageContentLeft).fontSize};
            font-family: ${getComputedStyle(pageContentLeft).fontFamily};
            line-height: ${getComputedStyle(pageContentLeft).lineHeight};
        `;
        document.body.appendChild(tempDiv);

        // Calculate number of pages based on scroll width
        const totalWidth = tempDiv.scrollWidth;
        this.totalPages = Math.max(1, Math.ceil(totalWidth / fixedColumnWidth));

        // Store the content and column settings for display
        this.chapterContent = originalContent;
        this.columnWidth = fixedColumnWidth;
        this.columnHeight = availableHeight;

        // Cleanup
        document.body.removeChild(tempDiv);
        
        // Ensure current page is valid
        if (this.currentPage >= this.totalPages) {
            this.currentPage = this.totalPages - 1;
        }
        if (this.currentPage < 0) {
            this.currentPage = 0;
        }

        // Display current page
        this.showCurrentPage();
    }

    showCurrentPage() {
        const pageContentLeft = document.getElementById('reader-page-content-left');
        const pageContentRight = document.getElementById('reader-page-content-right');
        
        if (!this.chapterContent) return;

        const isDoubleLayout = this.layout === 'double' && window.innerWidth > 1000;
        
        // In double layout, we show 2 pages at a time (left and right)
        // currentPage represents the left page index
        const leftPageIndex = isDoubleLayout ? this.currentPage * 2 : this.currentPage;
        const rightPageIndex = leftPageIndex + 1;

        // Use CSS columns with transform to show the correct "page"
        const leftOffset = leftPageIndex * this.columnWidth;
        const rightOffset = rightPageIndex * this.columnWidth;
        
        // Render left page - wrapper width must match columnWidth to show only 1 column
        pageContentLeft.innerHTML = `
            <div class="reader-columns-wrapper" style="
                width: ${this.columnWidth}px;
                height: ${this.columnHeight}px;
                overflow: hidden;
            ">
                <div class="reader-columns-content" style="
                    column-width: ${this.columnWidth}px;
                    column-gap: 0;
                    column-fill: auto;
                    height: ${this.columnHeight}px;
                    transform: translateX(-${leftOffset}px);
                ">
                    ${this.chapterContent}
                </div>
            </div>
        `;

        // Render right page (in double layout)
        if (isDoubleLayout) {
            if (rightPageIndex < this.totalPages) {
                // Right page has content
                pageContentRight.innerHTML = `
                    <div class="reader-columns-wrapper" style="
                        width: ${this.columnWidth}px;
                        height: ${this.columnHeight}px;
                        overflow: hidden;
                    ">
                        <div class="reader-columns-content" style="
                            column-width: ${this.columnWidth}px;
                            column-gap: 0;
                            column-fill: auto;
                            height: ${this.columnHeight}px;
                            transform: translateX(-${rightOffset}px);
                        ">
                            ${this.chapterContent}
                        </div>
                    </div>
                `;
            } else {
                // Right page is empty but should maintain dimensions
                pageContentRight.innerHTML = `
                    <div class="reader-columns-wrapper" style="
                        width: ${this.columnWidth}px;
                        height: ${this.columnHeight}px;
                    "></div>
                `;
            }
        } else {
            pageContentRight.innerHTML = '';
        }

        this.updatePageNavigation();
    }

    repaginate() {
        if (!this.currentBook) return;
        this.paginateContent();
    }

    goToPage(pageIndex) {
        if (pageIndex >= 0 && pageIndex < this.totalPages) {
            this.currentPage = pageIndex;
            this.showCurrentPage();
        }
    }

    previousPage() {
        const isDoubleLayout = this.layout === 'double' && window.innerWidth > 1000;
        
        if (this.currentPage > 0) {
            this.currentPage--;
            this.showCurrentPage();
        } else if (this.currentChapterIndex > 0) {
            // Go to previous chapter, last page/spread
            this.currentChapterIndex--;
            this.currentPage = Infinity; // Will be clamped in paginateContent
            this.displayChapter();
            // Set to last spread in double layout
            if (isDoubleLayout) {
                this.currentPage = Math.ceil(this.totalPages / 2) - 1;
            } else {
                this.currentPage = this.totalPages - 1;
            }
            this.showCurrentPage();
        }
    }

    nextPage() {
        const isDoubleLayout = this.layout === 'double' && window.innerWidth > 1000;
        const maxPage = isDoubleLayout ? Math.ceil(this.totalPages / 2) - 1 : this.totalPages - 1;
        
        if (this.currentPage < maxPage) {
            this.currentPage++;
            this.showCurrentPage();
        } else if (this.currentChapterIndex < this.currentBook.chapters.length - 1) {
            // Go to next chapter, first page
            this.currentChapterIndex++;
            this.currentPage = 0;
            this.displayChapter();
        }
    }

    previousChapter() {
        if (this.currentChapterIndex > 0) {
            this.currentChapterIndex--;
            this.currentPage = 0;
            this.displayChapter();
        }
    }

    nextChapter() {
        if (this.currentChapterIndex < this.currentBook.chapters.length - 1) {
            this.currentChapterIndex++;
            this.currentPage = 0;
            this.displayChapter();
        }
    }

    updatePageNavigation() {
        const prevBtn = document.getElementById('reader-page-prev');
        const nextBtn = document.getElementById('reader-page-next');
        const indicator = document.getElementById('reader-page-indicator');
        const chapterInfo = document.getElementById('reader-chapter-info');
        const prevZone = document.getElementById('reader-nav-prev');
        const nextZone = document.getElementById('reader-nav-next');

        const isDoubleLayout = this.layout === 'double' && window.innerWidth > 1000;
        
        // Calculate effective page numbers for display
        let displayPage, displayTotal;
        if (isDoubleLayout) {
            // In double layout, currentPage is the spread index (0, 1, 2...)
            // Each spread shows 2 pages
            displayPage = this.currentPage * 2 + 1; // Show left page number
            displayTotal = this.totalPages;
            const maxSpread = Math.ceil(this.totalPages / 2) - 1;
            var isLastSpread = this.currentPage >= maxSpread;
        } else {
            displayPage = this.currentPage + 1;
            displayTotal = this.totalPages;
            var isLastSpread = this.currentPage === this.totalPages - 1;
        }

        const isFirstPage = this.currentPage === 0 && this.currentChapterIndex === 0;
        const isLastPage = isLastSpread && this.currentChapterIndex === this.currentBook.chapters.length - 1;

        prevBtn.disabled = isFirstPage;
        nextBtn.disabled = isLastPage;

        prevZone.classList.toggle('disabled', isFirstPage);
        nextZone.classList.toggle('disabled', isLastPage);

        if (isDoubleLayout) {
            const rightPage = Math.min(displayPage + 1, displayTotal);
            indicator.textContent = `Pages ${displayPage}-${rightPage} of ${displayTotal}`;
        } else {
            indicator.textContent = `Page ${displayPage} of ${displayTotal}`;
        }

        // Update chapter info (just the title, no "Chapter X:" prefix)
        const chapter = this.currentBook.chapters[this.currentChapterIndex];
        chapterInfo.textContent = chapter.title || 'Untitled';
    }

    updateChapterNavigation() {
        const prevBtn = document.getElementById('reader-prev-chapter');
        const nextBtn = document.getElementById('reader-next-chapter');

        prevBtn.disabled = this.currentChapterIndex === 0;
        nextBtn.disabled = this.currentChapterIndex === this.currentBook.chapters.length - 1;
    }

    updateChaptersMenuActive() {
        const items = document.querySelectorAll('.reader-chapter-item');
        items.forEach((item, index) => {
            item.classList.toggle('active', index === this.currentChapterIndex);
        });
    }

    changeFontSize(delta) {
        const sizes = ['small', 'medium', 'large', 'xlarge'];
        const currentIndex = sizes.indexOf(this.settings.fontSize);
        const newIndex = Math.max(0, Math.min(sizes.length - 1, currentIndex + delta));
        
        this.settings.fontSize = sizes[newIndex];
        this.saveSettings();
        this.applySettings();
        this.repaginate();
    }

    applySettings() {
        // Apply theme to container
        const container = document.getElementById('reader-container');
        container.setAttribute('data-reader-theme', this.settings.theme);
        document.getElementById('reader-theme').value = this.settings.theme;

        // Apply font family
        document.getElementById('reader-font-family').value = this.settings.fontFamily;

        // Apply line height
        document.getElementById('reader-line-height').value = this.settings.lineHeight;

        // Apply layout
        const content = document.getElementById('reader-content');
        content.setAttribute('data-layout', this.layout);
        this.updateLayoutButton();

        // Apply to both page content elements
        const pageContentLeft = document.getElementById('reader-page-content-left');
        const pageContentRight = document.getElementById('reader-page-content-right');
        
        [pageContentLeft, pageContentRight].forEach(pageContent => {
            if (pageContent) {
                pageContent.setAttribute('data-font-size', this.settings.fontSize);
                pageContent.setAttribute('data-font-family', this.settings.fontFamily);
                pageContent.style.lineHeight = this.settings.lineHeight;
            }
        });
    }

    toggleLayout() {
        this.layout = this.layout === 'single' ? 'double' : 'single';
        this.settings.layout = this.layout;
        this.saveSettings();
        
        // Apply layout change
        const content = document.getElementById('reader-content');
        content.setAttribute('data-layout', this.layout);
        this.updateLayoutButton();
        
        // Reset to page 0 and repaginate after DOM updates with new layout
        this.currentPage = 0;
        // Use requestAnimationFrame to ensure layout has been applied
        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                this.repaginate();
            });
        });
    }

    updateLayoutButton() {
        const icon = document.getElementById('reader-layout-icon');
        const label = document.getElementById('reader-layout-label');
        
        if (this.layout === 'double') {
            icon.textContent = '☐☐';
            label.textContent = '2 Pages';
        } else {
            icon.textContent = '☐';
            label.textContent = '1 Page';
        }
    }

    showLoading() {
        const pageContent = document.getElementById('reader-page-content-left');
        pageContent.innerHTML = `
            <div class="reader-loading">
                <div class="reader-loading-spinner"></div>
                <div>Loading book...</div>
            </div>
        `;
        // Clear right page
        const pageContentRight = document.getElementById('reader-page-content-right');
        if (pageContentRight) pageContentRight.innerHTML = '';
    }

    showError(message) {
        const pageContent = document.getElementById('reader-page-content-left');
        pageContent.innerHTML = `
            <div class="reader-loading">
                <div style="color: #dc3545; font-size: 24px; margin-bottom: 16px;">⚠</div>
                <div style="font-weight: 600; margin-bottom: 8px;">Failed to load book</div>
                <div style="font-size: 14px; opacity: 0.8; max-width: 400px; text-align: center;">${this.escapeHtml(message)}</div>
                <button class="reader-btn" onclick="history.back()" style="margin-top: 24px;">
                    ← Go Back
                </button>
            </div>
        `;
        // Clear right page
        const pageContentRight = document.getElementById('reader-page-content-right');
        if (pageContentRight) pageContentRight.innerHTML = '';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    loadSettings() {
        const defaults = {
            fontSize: 'medium',
            fontFamily: 'serif',
            theme: 'light',
            lineHeight: '1.6',
            layout: 'single'
        };

        try {
            const saved = localStorage.getItem('ebookReaderSettings');
            return saved ? { ...defaults, ...JSON.parse(saved) } : defaults;
        } catch (e) {
            return defaults;
        }
    }

    saveSettings() {
        try {
            localStorage.setItem('ebookReaderSettings', JSON.stringify(this.settings));
        } catch (e) {
            console.error('Failed to save reader settings:', e);
        }
    }
}

// Initialize reader when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        new StandaloneReader();
    });
} else {
    new StandaloneReader();
}

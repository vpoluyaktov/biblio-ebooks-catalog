// Standalone Ebook Reader with Pagination

// Reading History Manager - tracks last 10 books with reading position
class ReadingHistory {
    constructor() {
        this.maxEntries = 10;
        this.storageKey = 'readingHistory';
    }

    getAll() {
        try {
            const data = localStorage.getItem(this.storageKey);
            return data ? JSON.parse(data) : [];
        } catch (e) {
            console.error('Failed to load reading history:', e);
            return [];
        }
    }

    save(history) {
        try {
            localStorage.setItem(this.storageKey, JSON.stringify(history));
            console.log('[ReadingHistory] Saved to localStorage:', history.length, 'entries');
        } catch (e) {
            console.error('Failed to save reading history:', e);
        }
    }

    addOrUpdate(entry) {
        const history = this.getAll();
        const bookId = parseInt(entry.bookId, 10);
        const libraryId = entry.libraryId ? parseInt(entry.libraryId, 10) : null;
        
        console.log('[ReadingHistory] addOrUpdate called for bookId:', bookId);
        
        const existingIndex = history.findIndex(h => parseInt(h.bookId, 10) === bookId);
        if (existingIndex !== -1) {
            history.splice(existingIndex, 1);
        }
        
        const newEntry = {
            bookId: bookId,
            libraryId: libraryId,
            title: entry.title,
            author: entry.author,
            chapterIndex: entry.chapterIndex || 0,
            pageIndex: entry.pageIndex || 0,
            totalChapters: entry.totalChapters || 1,
            lastRead: new Date().toISOString()
        };
        history.unshift(newEntry);
        console.log('[ReadingHistory] Added entry:', newEntry);
        
        while (history.length > this.maxEntries) {
            history.pop();
        }
        
        this.save(history);
    }

    updatePosition(bookId, chapterIndex, pageIndex) {
        const history = this.getAll();
        const numBookId = parseInt(bookId, 10);
        const entry = history.find(h => parseInt(h.bookId, 10) === numBookId);
        if (entry) {
            entry.chapterIndex = chapterIndex;
            entry.pageIndex = pageIndex;
            entry.lastRead = new Date().toISOString();
            this.save(history);
            console.log('[ReadingHistory] Updated position for bookId:', numBookId, 'chapter:', chapterIndex, 'page:', pageIndex);
        }
    }

    getPosition(bookId) {
        const history = this.getAll();
        const numBookId = parseInt(bookId, 10);
        const entry = history.find(h => parseInt(h.bookId, 10) === numBookId);
        if (entry) {
            console.log('[ReadingHistory] Found saved position for bookId:', numBookId);
        }
        return entry ? {
            chapterIndex: entry.chapterIndex || 0,
            pageIndex: entry.pageIndex || 0
        } : null;
    }
}

// Global reading history instance
const readingHistory = new ReadingHistory();

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
        
        // Listen for system theme changes
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
            // Only auto-switch if user hasn't set a manual preference
            if (!localStorage.getItem('theme')) {
                this.applySettings();
                this.displayChapter();
            }
        });
        
        if (this.bookId) {
            this.loadBook(this.bookId);
        } else {
            this.showError('No book ID provided');
        }
    }

    attachEventListeners() {
        // Close button - close the tab
        document.getElementById('reader-close').addEventListener('click', () => {
            window.close();
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
            this.displayChapter();
        });

        document.getElementById('reader-font-decrease').addEventListener('click', () => this.changeFontSize(-1));
        document.getElementById('reader-font-increase').addEventListener('click', () => this.changeFontSize(1));

        // Theme control - uses shared localStorage 'theme' key
        document.getElementById('reader-theme').addEventListener('change', (e) => {
            localStorage.setItem('theme', e.target.value);
            this.applySettings();
            this.displayChapter();
        });

        // Line height control
        document.getElementById('reader-line-height').addEventListener('change', (e) => {
            this.settings.lineHeight = e.target.value;
            this.saveSettings();
            this.applySettings();
            this.displayChapter();
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

        // Mobile settings panel
        const mobileSettingsBtn = document.getElementById('reader-mobile-settings-btn');
        if (mobileSettingsBtn) {
            mobileSettingsBtn.addEventListener('click', () => this.openMobileSettings());
        }

        const mobileSettingsClose = document.getElementById('reader-mobile-settings-close');
        if (mobileSettingsClose) {
            mobileSettingsClose.addEventListener('click', () => this.closeMobileSettings());
        }

        // Mobile chapter selector
        const mobileChapter = document.getElementById('reader-mobile-chapter');
        if (mobileChapter) {
            mobileChapter.addEventListener('change', (e) => {
                this.currentChapterIndex = parseInt(e.target.value);
                this.currentPage = 0;
                this.displayChapter();
                this.closeMobileSettings();
            });
        }

        // Mobile font family
        const mobileFontFamily = document.getElementById('reader-mobile-font-family');
        if (mobileFontFamily) {
            mobileFontFamily.addEventListener('change', (e) => {
                this.settings.fontFamily = e.target.value;
                this.saveSettings();
                this.applySettings();
                this.displayChapter();
            });
        }

        // Mobile font size
        const mobileFontDecrease = document.getElementById('reader-mobile-font-decrease');
        if (mobileFontDecrease) {
            mobileFontDecrease.addEventListener('click', () => this.changeFontSize(-1));
        }

        const mobileFontIncrease = document.getElementById('reader-mobile-font-increase');
        if (mobileFontIncrease) {
            mobileFontIncrease.addEventListener('click', () => this.changeFontSize(1));
        }

        // Mobile line height
        const mobileLineHeight = document.getElementById('reader-mobile-line-height');
        if (mobileLineHeight) {
            mobileLineHeight.addEventListener('change', (e) => {
                this.settings.lineHeight = e.target.value;
                this.saveSettings();
                this.applySettings();
                this.displayChapter();
            });
        }

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ignore if typing in an input
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'SELECT') return;
            
            switch(e.key) {
                case 'Escape':
                    window.close();
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
            
            // Check for saved position
            const savedPosition = readingHistory.getPosition(bookId);
            if (savedPosition) {
                this.currentChapterIndex = savedPosition.chapterIndex;
                this.currentPage = savedPosition.pageIndex;
                console.log('[StandaloneReader] Restoring position - chapter:', this.currentChapterIndex, 'page:', this.currentPage);
            } else {
                this.currentChapterIndex = 0;
                this.currentPage = 0;
            }

            // Update page title and UI
            document.title = this.currentBook.title || 'Ebook Reader';
            document.getElementById('reader-book-title').textContent = this.currentBook.title || 'Unknown Title';
            document.getElementById('reader-book-author').textContent = this.currentBook.author || '';

            // Build chapters menu
            this.buildChaptersMenu();

            // Display chapter (will restore page position after pagination)
            this.displayChapter();

            // Add to reading history
            readingHistory.addOrUpdate({
                bookId: parseInt(bookId, 10),
                libraryId: null,
                title: this.currentBook.title || 'Unknown Title',
                author: this.currentBook.author || 'Unknown Author',
                chapterIndex: this.currentChapterIndex,
                pageIndex: this.currentPage,
                totalChapters: this.currentBook.chapters.length
            });

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
        
        // Get actual dimensions from the rendered page element
        // The CSS now controls the page width (90% of screen)
        const containerHeight = containerLeft.clientHeight;
        const containerWidth = containerLeft.clientWidth;
        
        // Get padding from computed styles
        const styles = getComputedStyle(pageContentLeft);
        const paddingTop = parseInt(styles.paddingTop) || 48;
        const paddingBottom = parseInt(styles.paddingBottom) || 48;
        const paddingLeft = parseInt(styles.paddingLeft) || 56;
        const paddingRight = parseInt(styles.paddingRight) || 56;
        
        const availableHeight = Math.max(400, containerHeight - paddingTop - paddingBottom);
        const availableWidth = Math.max(200, containerWidth - paddingLeft - paddingRight);

        // Store original content
        const originalContent = pageContentLeft.innerHTML;
        
        // Create a temporary container with CSS columns to measure total width needed
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = originalContent;
        tempDiv.style.cssText = `
            position: absolute;
            visibility: hidden;
            height: ${availableHeight}px;
            width: ${availableWidth}px;
            column-width: ${availableWidth}px;
            column-gap: 0;
            column-fill: auto;
            font-size: ${styles.fontSize};
            font-family: ${styles.fontFamily};
            line-height: ${styles.lineHeight};
        `;
        document.body.appendChild(tempDiv);

        // Calculate number of pages based on scroll width
        const totalWidth = tempDiv.scrollWidth;
        this.totalPages = Math.max(1, Math.ceil(totalWidth / availableWidth));

        // Store the content and column settings for display
        this.chapterContent = originalContent;
        this.columnWidth = availableWidth;
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
        
        // Save position to reading history
        if (this.bookId) {
            readingHistory.updatePosition(this.bookId, this.currentChapterIndex, this.currentPage);
        }
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
        this.displayChapter();
    }

    getSystemTheme() {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    applySettings() {
        // Apply theme from shared localStorage 'theme' key, or use system theme if not set
        let theme = localStorage.getItem('theme');
        if (!theme) {
            theme = this.getSystemTheme();
        }
        const container = document.getElementById('reader-container');
        container.setAttribute('data-reader-theme', theme);
        document.getElementById('reader-theme').value = theme;

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
        
        // Reset to page 0 and force full chapter re-display after DOM updates
        this.currentPage = 0;
        // Use requestAnimationFrame to ensure layout has been applied
        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                // Force full re-display of chapter content
                this.displayChapter();
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

    openMobileSettings() {
        const panel = document.getElementById('reader-mobile-settings');
        if (panel) {
            panel.classList.add('active');
            this.updateMobileChapterSelector();
            // Sync mobile controls with current settings
            const mobileFontFamily = document.getElementById('reader-mobile-font-family');
            const mobileLineHeight = document.getElementById('reader-mobile-line-height');
            if (mobileFontFamily) mobileFontFamily.value = this.settings.fontFamily;
            if (mobileLineHeight) mobileLineHeight.value = this.settings.lineHeight;
        }
    }

    closeMobileSettings() {
        const panel = document.getElementById('reader-mobile-settings');
        if (panel) {
            panel.classList.remove('active');
        }
    }

    updateMobileChapterSelector() {
        const select = document.getElementById('reader-mobile-chapter');
        if (!select || !this.currentBook || !this.currentBook.chapters) return;
        
        select.innerHTML = this.currentBook.chapters.map((chapter, index) => 
            `<option value="${index}" ${index === this.currentChapterIndex ? 'selected' : ''}>${index + 1}. ${this.escapeHtml(chapter.title || 'Untitled')}</option>`
        ).join('');
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
            // Save reader-specific settings (theme is stored separately in root 'theme' key)
            const settingsToSave = { ...this.settings };
            delete settingsToSave.theme; // Don't save theme here, it's in root localStorage
            localStorage.setItem('ebookReaderSettings', JSON.stringify(settingsToSave));
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

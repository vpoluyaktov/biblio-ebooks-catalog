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
        // Close button
        document.getElementById('reader-close').addEventListener('click', () => {
            if (window.opener) {
                window.close();
            } else {
                history.back();
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
        const pageContent = document.getElementById('reader-page-content');

        // Apply settings
        pageContent.setAttribute('data-font-size', this.settings.fontSize);
        pageContent.setAttribute('data-font-family', this.settings.fontFamily);
        pageContent.style.lineHeight = this.settings.lineHeight;

        // Set chapter content
        pageContent.innerHTML = chapter.content;

        // Paginate the content
        this.paginateContent();

        // Update navigation
        this.updateChapterNavigation();
        this.updateChaptersMenuActive();
    }

    paginateContent() {
        const pageContent = document.getElementById('reader-page-content');
        const container = document.getElementById('reader-page');
        
        // Get available dimensions
        const containerHeight = container.clientHeight;
        const paddingTop = parseInt(getComputedStyle(pageContent).paddingTop) || 48;
        const paddingBottom = parseInt(getComputedStyle(pageContent).paddingBottom) || 48;
        const paddingLeft = parseInt(getComputedStyle(pageContent).paddingLeft) || 56;
        const paddingRight = parseInt(getComputedStyle(pageContent).paddingRight) || 56;
        const availableHeight = containerHeight - paddingTop - paddingBottom;
        const availableWidth = pageContent.clientWidth - paddingLeft - paddingRight;

        // Store original content
        const originalContent = pageContent.innerHTML;
        
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
            font-size: ${getComputedStyle(pageContent).fontSize};
            font-family: ${getComputedStyle(pageContent).fontFamily};
            line-height: ${getComputedStyle(pageContent).lineHeight};
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
        const pageContent = document.getElementById('reader-page-content');
        
        if (!this.chapterContent) return;

        // Use CSS columns with transform to show the correct "page"
        const offset = this.currentPage * this.columnWidth;
        
        pageContent.innerHTML = `
            <div class="reader-columns-wrapper" style="
                height: ${this.columnHeight}px;
                overflow: hidden;
            ">
                <div class="reader-columns-content" style="
                    column-width: ${this.columnWidth}px;
                    column-gap: 0;
                    column-fill: auto;
                    height: ${this.columnHeight}px;
                    transform: translateX(-${offset}px);
                ">
                    ${this.chapterContent}
                </div>
            </div>
        `;

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
        if (this.currentPage > 0) {
            this.currentPage--;
            this.showCurrentPage();
        } else if (this.currentChapterIndex > 0) {
            // Go to previous chapter, last page
            this.currentChapterIndex--;
            this.currentPage = Infinity; // Will be clamped in paginateContent
            this.displayChapter();
            this.currentPage = this.totalPages - 1;
            this.showCurrentPage();
        }
    }

    nextPage() {
        if (this.currentPage < this.totalPages - 1) {
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

        const isFirstPage = this.currentPage === 0 && this.currentChapterIndex === 0;
        const isLastPage = this.currentPage === this.totalPages - 1 && 
                          this.currentChapterIndex === this.currentBook.chapters.length - 1;

        prevBtn.disabled = isFirstPage;
        nextBtn.disabled = isLastPage;

        prevZone.classList.toggle('disabled', isFirstPage);
        nextZone.classList.toggle('disabled', isLastPage);

        indicator.textContent = `Page ${this.currentPage + 1} of ${this.totalPages}`;

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

        // Apply to page content if it exists
        const pageContent = document.getElementById('reader-page-content');
        if (pageContent) {
            pageContent.setAttribute('data-font-size', this.settings.fontSize);
            pageContent.setAttribute('data-font-family', this.settings.fontFamily);
            pageContent.style.lineHeight = this.settings.lineHeight;
        }
    }

    showLoading() {
        const pageContent = document.getElementById('reader-page-content');
        pageContent.innerHTML = `
            <div class="reader-loading">
                <div class="reader-loading-spinner"></div>
                <div>Loading book...</div>
            </div>
        `;
    }

    showError(message) {
        const pageContent = document.getElementById('reader-page-content');
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
            lineHeight: '1.6'
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

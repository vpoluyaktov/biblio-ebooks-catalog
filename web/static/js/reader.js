// Ebook Reader Module

class EbookReader {
    constructor() {
        this.currentBook = null;
        this.currentChapterIndex = 0;
        this.settings = this.loadSettings();
        this.overlay = null;
        this.init();
    }

    init() {
        this.createReaderOverlay();
        this.applySettings();
        this.attachEventListeners();
    }

    createReaderOverlay() {
        const overlay = document.createElement('div');
        overlay.className = 'reader-overlay';
        const appTheme = localStorage.getItem('theme') || 'dark';
        overlay.setAttribute('data-reader-theme', appTheme);
        
        overlay.innerHTML = `
            <div class="reader-toolbar">
                <div class="reader-toolbar-left">
                    <button class="reader-btn" id="reader-close">
                        <span class="reader-btn-icon">✕</span>
                        Close
                    </button>
                </div>
                <div class="reader-toolbar-center">
                    <div class="reader-title" id="reader-book-title"></div>
                    <div class="reader-author" id="reader-book-author"></div>
                </div>
                <div class="reader-toolbar-right">
                    <div class="reader-settings">
                        <div class="reader-setting-group">
                            <span class="reader-setting-label">Font:</span>
                            <select class="reader-select" id="reader-font-family">
                                <option value="serif">Serif</option>
                                <option value="sans-serif">Sans-serif</option>
                                <option value="monospace">Monospace</option>
                            </select>
                        </div>
                        <div class="reader-setting-group">
                            <button class="reader-icon-btn" id="reader-font-decrease" title="Decrease font size">A-</button>
                            <button class="reader-icon-btn" id="reader-font-increase" title="Increase font size">A+</button>
                        </div>
                        <div class="reader-setting-group">
                            <span class="reader-setting-label">Line Height:</span>
                            <select class="reader-select" id="reader-line-height">
                                <option value="1.4">Compact</option>
                                <option value="1.6">Normal</option>
                                <option value="1.8">Relaxed</option>
                                <option value="2.0">Loose</option>
                            </select>
                        </div>
                    </div>
                    <div class="reader-chapters-dropdown">
                        <button class="reader-btn" id="reader-chapters-btn">
                            <span class="reader-btn-icon">☰</span>
                            Chapters
                        </button>
                        <div class="reader-chapters-menu" id="reader-chapters-menu"></div>
                    </div>
                    <button class="reader-mobile-settings-btn" id="reader-mobile-settings-btn" title="Settings">⚙</button>
                </div>
            </div>
            <div class="reader-content" id="reader-content">
                <div class="reader-loading">
                    <div class="reader-loading-spinner"></div>
                    <div>Loading book...</div>
                </div>
            </div>
            <div class="reader-footer">
                <div class="reader-progress" id="reader-progress"></div>
                <div class="reader-navigation">
                    <button class="reader-btn" id="reader-prev-chapter" disabled>
                        <span class="reader-btn-icon">←</span>
                        Previous
                    </button>
                    <button class="reader-btn" id="reader-next-chapter" disabled>
                        Next
                        <span class="reader-btn-icon">→</span>
                    </button>
                </div>
            </div>
            <div class="reader-mobile-settings" id="reader-mobile-settings">
                <div class="reader-mobile-settings-header">
                    <span class="reader-mobile-settings-title">Settings</span>
                    <button class="reader-mobile-settings-close" id="reader-mobile-settings-close">✕</button>
                </div>
                <div class="reader-mobile-setting-row">
                    <span class="reader-mobile-setting-label">Chapter</span>
                    <select class="reader-mobile-select" id="reader-mobile-chapter"></select>
                </div>
                <div class="reader-mobile-setting-row">
                    <span class="reader-mobile-setting-label">Font</span>
                    <select class="reader-mobile-select" id="reader-mobile-font-family">
                        <option value="serif">Serif</option>
                        <option value="sans-serif">Sans-serif</option>
                        <option value="monospace">Monospace</option>
                    </select>
                </div>
                <div class="reader-mobile-setting-row">
                    <span class="reader-mobile-setting-label">Font Size</span>
                    <div class="reader-mobile-setting-control">
                        <button class="reader-icon-btn" id="reader-mobile-font-decrease">A-</button>
                        <button class="reader-icon-btn" id="reader-mobile-font-increase">A+</button>
                    </div>
                </div>
                <div class="reader-mobile-setting-row">
                    <span class="reader-mobile-setting-label">Line Height</span>
                    <select class="reader-mobile-select" id="reader-mobile-line-height">
                        <option value="1.4">Compact</option>
                        <option value="1.6">Normal</option>
                        <option value="1.8">Relaxed</option>
                        <option value="2.0">Loose</option>
                    </select>
                </div>
            </div>
        `;

        document.body.appendChild(overlay);
        this.overlay = overlay;
    }

    attachEventListeners() {
        // Close button
        document.getElementById('reader-close').addEventListener('click', () => this.close());

        // Navigation buttons
        document.getElementById('reader-prev-chapter').addEventListener('click', () => this.previousChapter());
        document.getElementById('reader-next-chapter').addEventListener('click', () => this.nextChapter());

        // Font controls
        document.getElementById('reader-font-family').addEventListener('change', (e) => {
            this.settings.fontFamily = e.target.value;
            this.saveSettings();
            this.applySettings();
        });

        document.getElementById('reader-font-decrease').addEventListener('click', () => this.changeFontSize(-1));
        document.getElementById('reader-font-increase').addEventListener('click', () => this.changeFontSize(1));


        // Line height control
        document.getElementById('reader-line-height').addEventListener('change', (e) => {
            this.settings.lineHeight = e.target.value;
            this.saveSettings();
            this.applySettings();
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
        document.getElementById('reader-mobile-settings-btn').addEventListener('click', () => {
            this.openMobileSettings();
        });

        document.getElementById('reader-mobile-settings-close').addEventListener('click', () => {
            this.closeMobileSettings();
        });

        // Mobile chapter selector
        document.getElementById('reader-mobile-chapter').addEventListener('change', (e) => {
            this.currentChapterIndex = parseInt(e.target.value);
            this.displayChapter();
            this.closeMobileSettings();
        });

        // Mobile font family
        document.getElementById('reader-mobile-font-family').addEventListener('change', (e) => {
            this.settings.fontFamily = e.target.value;
            this.saveSettings();
            this.applySettings();
        });

        // Mobile font size
        document.getElementById('reader-mobile-font-decrease').addEventListener('click', () => this.changeFontSize(-1));
        document.getElementById('reader-mobile-font-increase').addEventListener('click', () => this.changeFontSize(1));

        // Mobile line height
        document.getElementById('reader-mobile-line-height').addEventListener('change', (e) => {
            this.settings.lineHeight = e.target.value;
            this.saveSettings();
            this.applySettings();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (!this.overlay.classList.contains('active')) return;

            switch(e.key) {
                case 'Escape':
                    this.close();
                    break;
                case 'ArrowLeft':
                    this.previousChapter();
                    break;
                case 'ArrowRight':
                    this.nextChapter();
                    break;
            }
        });

        // Touch gestures for mobile
        let touchStartX = 0;
        let touchEndX = 0;

        this.overlay.addEventListener('touchstart', (e) => {
            touchStartX = e.changedTouches[0].screenX;
        });

        this.overlay.addEventListener('touchend', (e) => {
            touchEndX = e.changedTouches[0].screenX;
            this.handleSwipe();
        });

        const handleSwipe = () => {
            const swipeThreshold = 50;
            const diff = touchStartX - touchEndX;

            if (Math.abs(diff) > swipeThreshold) {
                if (diff > 0) {
                    // Swipe left - next chapter
                    this.nextChapter();
                } else {
                    // Swipe right - previous chapter
                    this.previousChapter();
                }
            }
        };

        this.handleSwipe = handleSwipe;
    }

    async openBook(bookId) {
        try {
            this.overlay.classList.add('active');
            this.showLoading();

            // Fetch book content from API
            const response = await fetch(`${APP_BASE_PATH}/api/books/${bookId}/content`);
            if (!response.ok) {
                throw new Error('Failed to load book content');
            }

            this.currentBook = await response.json();
            this.currentChapterIndex = 0;

            // Update UI
            document.getElementById('reader-book-title').textContent = this.currentBook.title || 'Unknown Title';
            document.getElementById('reader-book-author').textContent = this.currentBook.author || 'Unknown Author';

            // Build chapters menu
            this.buildChaptersMenu();

            // Display first chapter
            this.displayChapter();

        } catch (error) {
            console.error('Error opening book:', error);
            alert('Failed to open book: ' + error.message);
            this.close();
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
                ${chapter.title}
            `;

            item.addEventListener('click', () => {
                this.currentChapterIndex = index;
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
        const content = document.getElementById('reader-content');

        // Create chapter container
        const chapterDiv = document.createElement('div');
        chapterDiv.className = 'reader-chapter';
        chapterDiv.setAttribute('data-font-size', this.settings.fontSize);
        chapterDiv.setAttribute('data-font-family', this.settings.fontFamily);
        chapterDiv.style.lineHeight = this.settings.lineHeight;

        // Set chapter content
        chapterDiv.innerHTML = chapter.content;

        // Replace content
        content.innerHTML = '';
        content.appendChild(chapterDiv);

        // Scroll to top
        content.scrollTop = 0;

        // Update navigation buttons
        this.updateNavigation();

        // Update progress
        this.updateProgress();

        // Update chapters menu active state
        this.updateChaptersMenuActive();
    }

    updateNavigation() {
        const prevBtn = document.getElementById('reader-prev-chapter');
        const nextBtn = document.getElementById('reader-next-chapter');

        prevBtn.disabled = this.currentChapterIndex === 0;
        nextBtn.disabled = this.currentChapterIndex === this.currentBook.chapters.length - 1;
    }

    updateProgress() {
        const progress = document.getElementById('reader-progress');
        const current = this.currentChapterIndex + 1;
        const total = this.currentBook.chapters.length;
        progress.textContent = `Chapter ${current} of ${total}`;
    }

    updateChaptersMenuActive() {
        const items = document.querySelectorAll('.reader-chapter-item');
        items.forEach((item, index) => {
            if (index === this.currentChapterIndex) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });
    }

    previousChapter() {
        if (this.currentChapterIndex > 0) {
            this.currentChapterIndex--;
            this.displayChapter();
        }
    }

    nextChapter() {
        if (this.currentChapterIndex < this.currentBook.chapters.length - 1) {
            this.currentChapterIndex++;
            this.displayChapter();
        }
    }

    changeFontSize(delta) {
        const sizes = ['small', 'medium', 'large', 'xlarge'];
        const currentIndex = sizes.indexOf(this.settings.fontSize);
        const newIndex = Math.max(0, Math.min(sizes.length - 1, currentIndex + delta));
        
        this.settings.fontSize = sizes[newIndex];
        this.saveSettings();
        this.applySettings();

        // Update current chapter display
        const chapterDiv = document.querySelector('.reader-chapter');
        if (chapterDiv) {
            chapterDiv.setAttribute('data-font-size', this.settings.fontSize);
        }
    }

    applySettings() {
        // Apply theme from app's global theme setting
        const appTheme = localStorage.getItem('theme') || 'dark';
        this.overlay.setAttribute('data-reader-theme', appTheme);

        // Apply font family (desktop)
        document.getElementById('reader-font-family').value = this.settings.fontFamily;

        // Apply line height (desktop)
        document.getElementById('reader-line-height').value = this.settings.lineHeight;

        // Apply to mobile controls as well
        document.getElementById('reader-mobile-font-family').value = this.settings.fontFamily;
        document.getElementById('reader-mobile-line-height').value = this.settings.lineHeight;

        // Font size is applied per chapter
    }

    openMobileSettings() {
        const panel = document.getElementById('reader-mobile-settings');
        panel.classList.add('active');
        
        // Update chapter selector
        this.updateMobileChapterSelector();
    }

    closeMobileSettings() {
        const panel = document.getElementById('reader-mobile-settings');
        panel.classList.remove('active');
    }

    updateMobileChapterSelector() {
        const select = document.getElementById('reader-mobile-chapter');
        if (!this.currentBook || !this.currentBook.chapters) return;
        
        select.innerHTML = this.currentBook.chapters.map((chapter, index) => 
            `<option value="${index}" ${index === this.currentChapterIndex ? 'selected' : ''}>${index + 1}. ${chapter.title}</option>`
        ).join('');
    }

    showLoading() {
        const content = document.getElementById('reader-content');
        content.innerHTML = `
            <div class="reader-loading">
                <div class="reader-loading-spinner"></div>
                <div>Loading book...</div>
            </div>
        `;
    }

    close() {
        this.overlay.classList.remove('active');
        this.currentBook = null;
        this.currentChapterIndex = 0;
    }

    loadSettings() {
        const defaults = {
            fontSize: 'medium',
            fontFamily: 'serif',
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
let ebookReader;
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        ebookReader = new EbookReader();
    });
} else {
    ebookReader = new EbookReader();
}

// Export for use in other modules
window.openEbookReader = function(bookId) {
    if (ebookReader) {
        ebookReader.openBook(bookId);
    }
};

// Standalone Ebook Reader for separate tab/window

class StandaloneReader {
    constructor() {
        this.currentBook = null;
        this.currentChapterIndex = 0;
        this.settings = this.loadSettings();
        this.bookId = this.getBookIdFromURL();
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
        document.getElementById('reader-close').addEventListener('click', () => window.close());

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
            switch(e.key) {
                case 'Escape':
                    window.close();
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

        document.addEventListener('touchstart', (e) => {
            touchStartX = e.changedTouches[0].screenX;
        });

        document.addEventListener('touchend', (e) => {
            touchEndX = e.changedTouches[0].screenX;
            const swipeThreshold = 50;
            const diff = touchStartX - touchEndX;

            if (Math.abs(diff) > swipeThreshold) {
                if (diff > 0) {
                    this.nextChapter();
                } else {
                    this.previousChapter();
                }
            }
        });
    }

    async loadBook(bookId) {
        try {
            this.showLoading();

            const basePath = window.APP_BASE_PATH || '';
            const response = await fetch(`${basePath}/api/books/${bookId}/content`);
            if (!response.ok) {
                throw new Error('Failed to load book content');
            }

            this.currentBook = await response.json();
            this.currentChapterIndex = 0;

            // Update page title and UI
            document.title = this.currentBook.title || 'Ebook Reader';
            document.getElementById('reader-book-title').textContent = this.currentBook.title || 'Unknown Title';
            document.getElementById('reader-book-author').textContent = this.currentBook.author || 'Unknown Author';

            // Build chapters menu
            this.buildChaptersMenu();

            // Display first chapter
            this.displayChapter();

        } catch (error) {
            console.error('Error loading book:', error);
            this.showError('Failed to load book: ' + error.message);
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
        // Apply theme to container
        const container = document.getElementById('reader-container');
        container.setAttribute('data-reader-theme', this.settings.theme);
        document.getElementById('reader-theme').value = this.settings.theme;

        // Apply font family
        document.getElementById('reader-font-family').value = this.settings.fontFamily;

        // Apply line height
        document.getElementById('reader-line-height').value = this.settings.lineHeight;
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

    showError(message) {
        const content = document.getElementById('reader-content');
        content.innerHTML = `
            <div class="reader-loading">
                <div style="color: #dc3545; margin-bottom: 16px;">⚠️ Error</div>
                <div>${message}</div>
                <button class="reader-btn" onclick="window.close()" style="margin-top: 20px;">Close Window</button>
            </div>
        `;
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

// Mobile-specific UI for Biblio OPDS Server

const MobileUI = {
  screen: 'home',
  history: [],
  selectedAuthor: null,
  selectedSeries: null,
  selectedGenre: null,
  selectedBook: null,

  init() {
    // Set initial mobile state
    App.isMobile = window.innerWidth <= 768;
    
    // Listen for resize events
    window.addEventListener('resize', () => this.checkMobile());
  },

  checkMobile() {
    const wasMobile = App.isMobile;
    App.isMobile = window.innerWidth <= 768;
    
    if (wasMobile !== App.isMobile && App.currentView === 'browser') {
      App.renderBrowser();
    }
  },

  navigateTo(screen, data = {}) {
    this.history.push({ screen: this.screen, data: this.getCurrentData() });
    this.screen = screen;
    this.params = data;
    Object.assign(this, data);
    this.render();
  },

  goBack() {
    if (this.history.length > 0) {
      const prev = this.history.pop();
      this.screen = prev.screen;
      Object.assign(this, prev.data);
      this.render();
    }
  },

  getCurrentData() {
    return {
      selectedAuthor: this.selectedAuthor,
      selectedSeries: this.selectedSeries,
      selectedGenre: this.selectedGenre,
      selectedBook: this.selectedBook
    };
  },

  render() {
    const container = document.getElementById('mobile-container');
    if (!container) return;

    switch (this.screen) {
      case 'home':
        container.innerHTML = this.renderHome();
        break;
      case 'authors':
        container.innerHTML = this.renderAuthors();
        this.loadAuthors();
        break;
      case 'series':
        container.innerHTML = this.renderSeriesList();
        this.loadSeries();
        break;
      case 'genres':
        container.innerHTML = this.renderGenres();
        const parentGenreId = this.params?.parentGenreId || 0;
        this.loadGenres(parentGenreId);
        break;
      case 'search':
        container.innerHTML = this.renderSearch();
        break;
      case 'books':
        container.innerHTML = this.renderBooks();
        this.loadBooks();
        break;
      case 'book-detail':
        container.innerHTML = this.renderBookDetail();
        break;
      case 'config':
        container.innerHTML = this.renderConfig();
        break;
    }

    this.bindMobileEvents();
  },

  renderHome() {
    const currentLib = App.libraries.find(l => l.id === App.currentLibrary);
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <div class="mobile-header-content">
            <div class="mobile-logo">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
              </svg>
              <span>Biblio OPDS Server</span>
            </div>
            <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
              <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="5"></circle>
                <line x1="12" y1="1" x2="12" y2="3"></line>
                <line x1="12" y1="21" x2="12" y2="23"></line>
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
                <line x1="1" y1="12" x2="3" y2="12"></line>
                <line x1="21" y1="12" x2="23" y2="12"></line>
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
                <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
              </svg>
              <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
              </svg>
            </button>
          </div>
        </div>

        <div class="mobile-content">
          ${App.libraries.length > 0 ? `
            <div class="mobile-library-selector" style="padding: 1rem; background: var(--bg-secondary); border-bottom: 1px solid var(--border);">
              <label style="display: block; font-size: 0.875rem; color: var(--text-secondary); margin-bottom: 0.5rem;">Library:</label>
              <select id="mobile-home-library-select" class="mobile-select">
                ${App.libraries.map(lib => `
                  <option value="${lib.id}" ${lib.id === App.currentLibrary ? 'selected' : ''}>${lib.name}</option>
                `).join('')}
              </select>
            </div>
          ` : ''}
          <div class="mobile-search-box">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <path d="m21 21-4.35-4.35"/>
            </svg>
            <input type="text" placeholder="Global search..." id="mobile-global-search">
          </div>

          <div class="mobile-menu">
            <button class="mobile-menu-item" data-mobile-nav="authors">
              <div class="mobile-menu-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
                  <circle cx="12" cy="7" r="4"></circle>
                </svg>
              </div>
              <div class="mobile-menu-text">
                <div class="mobile-menu-title">Search by Authors</div>
                <div class="mobile-menu-subtitle">Browse books by author</div>
              </div>
              <svg class="mobile-menu-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </button>

            <button class="mobile-menu-item" data-mobile-nav="series">
              <div class="mobile-menu-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
                  <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
                </svg>
              </div>
              <div class="mobile-menu-text">
                <div class="mobile-menu-title">Search by Series</div>
                <div class="mobile-menu-subtitle">Browse book series</div>
              </div>
              <svg class="mobile-menu-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </button>

            <button class="mobile-menu-item" data-mobile-nav="genres">
              <div class="mobile-menu-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 12h-4l-3 9L9 3l-3 9H2"></path>
                </svg>
              </div>
              <div class="mobile-menu-text">
                <div class="mobile-menu-title">Search by Genres</div>
                <div class="mobile-menu-subtitle">Browse by category</div>
              </div>
              <svg class="mobile-menu-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </button>

            <button class="mobile-menu-item" data-mobile-nav="search">
              <div class="mobile-menu-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/>
                  <path d="m21 21-4.35-4.35"/>
                </svg>
              </div>
              <div class="mobile-menu-text">
                <div class="mobile-menu-title">Advanced Search</div>
                <div class="mobile-menu-subtitle">Search with filters</div>
              </div>
              <svg class="mobile-menu-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </button>

            <button class="mobile-menu-item" data-mobile-nav="config">
              <div class="mobile-menu-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"></circle>
                  <path d="M12 1v6m0 6v6"></path>
                  <path d="m1 12 6 0m6 0 6 0"></path>
                </svg>
              </div>
              <div class="mobile-menu-text">
                <div class="mobile-menu-title">Configuration</div>
                <div class="mobile-menu-subtitle">Settings and libraries</div>
              </div>
              <svg class="mobile-menu-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </button>
          </div>
        </div>
      </div>
    `;
  },

  renderAuthors() {
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">Authors</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-filter-box">
          <input type="text" placeholder="Filter authors..." id="mobile-filter-input">
        </div>

        <div class="mobile-content">
          <div class="mobile-list" id="mobile-authors-list">
            <div class="mobile-loading">Loading authors...</div>
          </div>
        </div>
      </div>
    `;
  },

  renderSeriesList() {
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">Series</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-filter-box">
          <input type="text" placeholder="Filter series..." id="mobile-filter-input">
        </div>

        <div class="mobile-content">
          <div class="mobile-list" id="mobile-series-list">
            <div class="mobile-loading">Loading series...</div>
          </div>
        </div>
      </div>
    `;
  },

  renderGenres() {
    const title = this.params?.parentGenreName || 'Genres';
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">${title}</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-content">
          <div class="mobile-list" id="mobile-genres-list">
            <div class="mobile-loading">Loading...</div>
          </div>
        </div>
      </div>
    `;
  },

  renderSearch() {
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">Advanced Search</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-content">
          <div class="mobile-search-form">
            <div class="mobile-form-group">
              <label>Search query</label>
              <input type="text" placeholder="Enter search terms..." id="mobile-search-query">
            </div>
            <button class="mobile-btn-primary" id="mobile-search-btn">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
              Search
            </button>
          </div>
          <div class="mobile-list" id="mobile-search-results"></div>
        </div>
      </div>
    `;
  },

  renderBooks() {
    const title = this.selectedAuthor ? `Books by ${this.selectedAuthor.name}` :
                  this.selectedSeries ? `Series: ${this.selectedSeries.name}` :
                  this.selectedGenre ? `Genre: ${this.selectedGenre}` : 'Books';
    
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">${title}</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-content">
          <div class="mobile-books-list" id="mobile-books-list">
            <div class="mobile-loading">Loading books...</div>
          </div>
        </div>
      </div>
    `;
  },

  renderBookDetail() {
    if (!this.selectedBook) return '';
    
    const book = this.selectedBook;
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">Book Details</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-content">
          <div class="mobile-book-detail">
            <div class="mobile-book-cover">
              ${book.cover_url ? 
                `<img src="${book.cover_url}" alt="${book.title}">` :
                `<div class="mobile-book-cover-placeholder">📚</div>`
              }
            </div>
            <div class="mobile-book-info">
              <h2 class="mobile-book-title">${book.title}</h2>
              <div class="mobile-book-author">${book.author}</div>
              ${book.series ? `<div class="mobile-book-series">${book.series}</div>` : ''}
              <div class="mobile-book-meta">
                <span>${book.lang?.toUpperCase() || 'N/A'}</span>
                <span>${book.size || 'N/A'}</span>
                ${book.date ? `<span>${book.date}</span>` : ''}
              </div>
              ${book.genre ? `<div class="mobile-book-genre">${book.genre}</div>` : ''}
              ${book.annotation ? `<div class="mobile-book-annotation">${book.annotation}</div>` : ''}
            </div>
            <div class="mobile-book-actions">
              <a href="${book.download_url || '#'}" class="mobile-btn-primary mobile-btn-download" ${!book.download_url ? 'style="opacity:0.5;pointer-events:none"' : ''}>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                  <polyline points="7 10 12 15 17 10"></polyline>
                  <line x1="12" y1="15" x2="12" y2="3"></line>
                </svg>
                Download
              </a>
            </div>
          </div>
        </div>
      </div>
    `;
  },

  renderConfig() {
    const currentLib = App.libraries.find(l => l.id === App.currentLibrary);
    return `
      <div class="mobile-screen">
        <div class="mobile-header">
          <button class="mobile-back-btn" data-mobile-back>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="mobile-header-title">Configuration</div>
          <button type="button" class="mobile-icon-btn" data-action="toggleTheme">
            <svg class="icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
            <svg class="icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </button>
        </div>

        <div class="mobile-content">
          <div class="mobile-config">
            <div class="mobile-config-section">
              <h3>Current Library</h3>
              <select id="mobile-library-select" class="mobile-select">
                ${App.libraries.map(lib => `
                  <option value="${lib.id}" ${lib.id === App.currentLibrary ? 'selected' : ''}>${lib.name}</option>
                `).join('')}
              </select>
            </div>

            ${App.user?.role === 'admin' ? `
              <div class="mobile-config-section">
                <h3>Administration</h3>
                <a href="#libraries" class="mobile-btn-secondary">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
                    <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
                  </svg>
                  Manage Libraries
                </a>
              </div>
            ` : ''}

            <div class="mobile-config-section">
              <h3>Account</h3>
              <button class="mobile-btn-secondary" data-action="logout">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
                  <polyline points="16,17 21,12 16,7"></polyline>
                  <line x1="21" y1="12" x2="9" y2="12"></line>
                </svg>
                Logout
              </button>
            </div>
          </div>
        </div>
      </div>
    `;
  },

  bindMobileEvents() {
    // Back button
    document.querySelectorAll('[data-mobile-back]').forEach(btn => {
      btn.addEventListener('click', () => this.goBack());
    });

    // Navigation buttons
    document.querySelectorAll('[data-mobile-nav]').forEach(btn => {
      btn.addEventListener('click', () => {
        const screen = btn.dataset.mobileNav;
        this.navigateTo(screen);
      });
    });

    // Filter input
    const filterInput = document.getElementById('mobile-filter-input');
    if (filterInput) {
      let timeout;
      filterInput.addEventListener('input', (e) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => {
          const query = e.target.value.trim();
          if (this.screen === 'authors') {
            this.loadAuthors(query);
          } else if (this.screen === 'series') {
            this.loadSeries(query);
          }
        }, 300);
      });
    }

    // Library selector in config screen
    const libSelect = document.getElementById('mobile-library-select');
    if (libSelect) {
      libSelect.addEventListener('change', (e) => {
        App.currentLibrary = parseInt(e.target.value);
        this.navigateTo('home');
      });
    }

    // Library selector on home screen
    const homeLibSelect = document.getElementById('mobile-home-library-select');
    if (homeLibSelect) {
      homeLibSelect.addEventListener('change', (e) => {
        App.currentLibrary = parseInt(e.target.value);
        this.render(); // Re-render home screen with new library
      });
    }

    // Search button
    const searchBtn = document.getElementById('mobile-search-btn');
    if (searchBtn) {
      searchBtn.addEventListener('click', () => this.performSearch());
    }

    // Global search
    const globalSearch = document.getElementById('mobile-global-search');
    if (globalSearch) {
      globalSearch.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
          this.performGlobalSearch(e.target.value);
        }
      });
    }
  },

  async loadAuthors(filter = '') {
    try {
      // Ensure library is set
      if (!App.currentLibrary && App.libraries.length > 0) {
        App.currentLibrary = App.libraries[0].id;
      }
      
      if (!App.currentLibrary) {
        const list = document.getElementById('mobile-authors-list');
        if (list) list.innerHTML = '<div class="mobile-empty">No library selected</div>';
        return;
      }

      // Use same API call as desktop UI
      const url = `/api/libraries/${App.currentLibrary}/authors?limit=50&offset=0&filter=${encodeURIComponent(filter)}`;
      const data = await App.fetchAPI(url);
      const list = document.getElementById('mobile-authors-list');
      if (!list) return;

      const authors = data?.authors || [];
      console.log('Authors loaded:', authors.length, 'filter:', filter);
      
      if (authors.length === 0) {
        list.innerHTML = '<div class="mobile-empty">No authors found</div>';
        return;
      }

      list.innerHTML = authors.map(author => {
        const fullName = [author.last_name, author.first_name, author.middle_name]
          .filter(n => n && n.trim())
          .join(' ') || 'Unknown Author';
        return `
          <div class="mobile-list-item" data-author-id="${author.id}">
            <div class="mobile-list-item-text">
              <div class="mobile-list-item-title">${fullName}</div>
              <div class="mobile-list-item-subtitle">${author.BookCount} book${author.BookCount !== 1 ? 's' : ''}</div>
            </div>
            <svg class="mobile-list-item-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </div>
        `;
      }).join('');

      // Bind click events
      list.querySelectorAll('[data-author-id]').forEach(item => {
        item.addEventListener('click', () => {
          const authorId = parseInt(item.dataset.authorId);
          const author = authors.find(a => a.id === authorId);
          if (author) {
            author.name = [author.last_name, author.first_name, author.middle_name]
              .filter(n => n && n.trim())
              .join(' ') || 'Unknown Author';
            this.navigateTo('books', { selectedAuthor: author });
          }
        });
      });
    } catch (e) {
      console.error('Failed to load authors:', e);
    }
  },

  async loadSeries(filter = '') {
    try {
      // Ensure library is set
      if (!App.currentLibrary && App.libraries.length > 0) {
        App.currentLibrary = App.libraries[0].id;
      }
      
      if (!App.currentLibrary) {
        const list = document.getElementById('mobile-series-list');
        if (list) list.innerHTML = '<div class="mobile-empty">No library selected</div>';
        return;
      }

      // Use same API call as desktop UI
      const url = `/api/libraries/${App.currentLibrary}/series?limit=50&offset=0&filter=${encodeURIComponent(filter)}`;
      const data = await App.fetchAPI(url);
      const list = document.getElementById('mobile-series-list');
      if (!list) return;

      const series = data?.series || [];
      if (series.length === 0) {
        list.innerHTML = '<div class="mobile-empty">No series found</div>';
        return;
      }

      list.innerHTML = series.map(s => `
        <div class="mobile-list-item" data-series-id="${s.id}">
          <div class="mobile-list-item-text">
            <div class="mobile-list-item-title">${s.name}</div>
            <div class="mobile-list-item-subtitle">${s.BookCount} book${s.BookCount !== 1 ? 's' : ''}</div>
          </div>
          <svg class="mobile-list-item-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </div>
      `).join('');

      list.querySelectorAll('[data-series-id]').forEach(item => {
        item.addEventListener('click', () => {
          const seriesId = parseInt(item.dataset.seriesId);
          const selectedSeries = series.find(s => s.id === seriesId);
          this.navigateTo('books', { selectedSeries });
        });
      });
    } catch (e) {
      console.error('Failed to load series:', e);
    }
  },

  async loadGenres(parentId = 0) {
    try {
      // Ensure library is set
      if (!App.currentLibrary && App.libraries.length > 0) {
        App.currentLibrary = App.libraries[0].id;
      }
      
      if (!App.currentLibrary) {
        const list = document.getElementById('mobile-genres-list');
        if (list) list.innerHTML = '<div class="mobile-empty">No library selected</div>';
        return;
      }

      const genres = await App.fetchAPI('/api/genres');
      const list = document.getElementById('mobile-genres-list');
      if (!list) return;

      // Filter genres by parent_id
      const filteredGenres = genres.filter(g => g.parent_id === parentId);
      
      if (filteredGenres.length === 0) {
        list.innerHTML = '<div class="mobile-empty">No genres found</div>';
        return;
      }

      list.innerHTML = filteredGenres.map(genre => {
        // Check if this genre has children
        const hasChildren = genres.some(g => g.parent_id === genre.id);
        
        return `
          <div class="mobile-list-item" data-genre-id="${genre.id}" data-genre-name="${genre.name}" data-has-children="${hasChildren}">
            <div class="mobile-list-item-text">
              <div class="mobile-list-item-title">${genre.name}</div>
            </div>
            <svg class="mobile-list-item-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </div>
        `;
      }).join('');

      list.querySelectorAll('[data-genre-id]').forEach(item => {
        item.addEventListener('click', () => {
          const genreId = parseInt(item.dataset.genreId);
          const genreName = item.dataset.genreName;
          const hasChildren = item.dataset.hasChildren === 'true';
          
          if (hasChildren) {
            // Navigate to child genres
            this.navigateTo('genres', { parentGenreId: genreId, parentGenreName: genreName });
          } else {
            // Navigate to books for this genre
            this.navigateTo('books', { selectedGenre: genreId });
          }
        });
      });
    } catch (e) {
      console.error('Failed to load genres:', e);
    }
  },

  async loadBooks() {
    try {
      // Ensure library is set
      if (!App.currentLibrary && App.libraries.length > 0) {
        App.currentLibrary = App.libraries[0].id;
      }
      
      if (!App.currentLibrary) {
        const list = document.getElementById('mobile-books-list');
        if (list) list.innerHTML = '<div class="mobile-empty">No library selected</div>';
        return;
      }

      let opdsUrl = '';
      
      if (this.selectedAuthor) {
        opdsUrl = `/opds/${App.currentLibrary}/author/${this.selectedAuthor.id}`;
      } else if (this.selectedSeries) {
        opdsUrl = `/opds/${App.currentLibrary}/series/${this.selectedSeries.id}`;
      } else if (this.selectedGenre) {
        opdsUrl = `/opds/${App.currentLibrary}/genres/${encodeURIComponent(this.selectedGenre)}`;
      }

      if (!opdsUrl) {
        const list = document.getElementById('mobile-books-list');
        if (list) list.innerHTML = '<div class="mobile-empty">No selection</div>';
        return;
      }

      // Fetch OPDS feed
      const response = await fetch(opdsUrl);
      const text = await response.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      
      // Parse OPDS entries
      const entries = xml.querySelectorAll('entry');
      const books = Array.from(entries).map(entry => {
        const id = entry.querySelector('id')?.textContent || '';
        const bookId = id.split(':').pop();
        const title = entry.querySelector('title')?.textContent || 'Unknown';
        const author = entry.querySelector('author name')?.textContent || 'Unknown';
        const content = entry.querySelector('content')?.textContent || '';
        
        // Extract series from content if present
        let series = '';
        const seriesMatch = content.match(/Series:\s*([^<\n]+)/);
        if (seriesMatch) series = seriesMatch[1].trim();
        
        // Extract language from dcterms:language
        const lang = entry.querySelector('language, [*|language]')?.textContent || '';
        
        // Get download link and extract file size
        const downloadLink = Array.from(entry.querySelectorAll('link')).find(
          link => link.getAttribute('type')?.includes('application/')
        );
        const downloadUrl = downloadLink?.getAttribute('href') || '';
        const lengthAttr = downloadLink?.getAttribute('length') || '0';
        const sizeBytes = parseInt(lengthAttr);
        const size = sizeBytes > 0 ? this.formatFileSize(sizeBytes) : '';
        
        // Construct cover URL like desktop UI does
        const coverUrl = bookId ? `/opds/${App.currentLibrary}/covers/${bookId}/cover.jpg` : '';
        
        return {
          id: bookId,
          title,
          author,
          series,
          lang,
          size,
          download_url: downloadUrl,
          cover_url: coverUrl
        };
      });

      const list = document.getElementById('mobile-books-list');
      if (!list) return;

      if (!books || books.length === 0) {
        list.innerHTML = '<div class="mobile-empty">No books found</div>';
        return;
      }

      list.innerHTML = books.map(book => `
        <div class="mobile-book-item" data-book-id="${book.id}">
          <div class="mobile-book-item-cover">
            ${book.cover_url ? 
              `<img src="${book.cover_url}" alt="${book.title}">` :
              `<div class="mobile-book-item-cover-placeholder">📚</div>`
            }
          </div>
          <div class="mobile-book-item-info">
            <div class="mobile-book-item-title">${book.title}</div>
            <div class="mobile-book-item-author">${book.author}</div>
            ${book.series ? `<div class="mobile-book-item-series">${book.series}</div>` : ''}
            <div class="mobile-book-item-meta">
              ${book.lang ? `<span>${book.lang.toUpperCase()}</span>` : ''}
              ${book.size ? `<span>${book.size}</span>` : ''}
            </div>
          </div>
          <svg class="mobile-list-item-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </div>
      `).join('');

      list.querySelectorAll('[data-book-id]').forEach(item => {
        item.addEventListener('click', () => {
          const bookId = item.dataset.bookId;
          const book = books.find(b => String(b.id) === String(bookId));
          if (book) {
            console.log('Navigating to book:', book);
            this.navigateTo('book-detail', { selectedBook: book });
          } else {
            console.error('Book not found:', bookId);
          }
        });
      });
    } catch (e) {
      console.error('Failed to load books:', e);
    }
  },

  filterList(query) {
    // Filter visible list items based on query
    const items = document.querySelectorAll('.mobile-list-item');
    const lowerQuery = query.toLowerCase();
    
    items.forEach(item => {
      const title = item.querySelector('.mobile-list-item-title')?.textContent.toLowerCase() || '';
      if (title.includes(lowerQuery)) {
        item.style.display = '';
      } else {
        item.style.display = 'none';
      }
    });
  },

  formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
  },

  async performSearch() {
    const query = document.getElementById('mobile-search-query')?.value;
    if (!query) return;

    try {
      // Ensure library is set
      if (!App.currentLibrary && App.libraries.length > 0) {
        App.currentLibrary = App.libraries[0].id;
      }
      
      if (!App.currentLibrary) {
        const results = document.getElementById('mobile-search-results');
        if (results) results.innerHTML = '<div class="mobile-empty">No library selected</div>';
        return;
      }

      // Use OPDS search endpoint like desktop UI
      const opdsUrl = `/opds/${App.currentLibrary}/search?q=${encodeURIComponent(query)}`;
      const response = await fetch(opdsUrl);
      const text = await response.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      
      // Parse OPDS entries
      const entries = xml.querySelectorAll('entry');
      const books = Array.from(entries).map(entry => {
        const id = entry.querySelector('id')?.textContent || '';
        const bookId = id.split(':').pop();
        const title = entry.querySelector('title')?.textContent || 'Unknown';
        const author = entry.querySelector('author name')?.textContent || 'Unknown';
        const content = entry.querySelector('content')?.textContent || '';
        
        let series = '';
        const seriesMatch = content.match(/Series:\s*([^<\n]+)/);
        if (seriesMatch) series = seriesMatch[1].trim();
        
        const lang = entry.querySelector('language, [*|language]')?.textContent || '';
        
        const downloadLink = Array.from(entry.querySelectorAll('link')).find(
          link => link.getAttribute('type')?.includes('application/')
        );
        const downloadUrl = downloadLink?.getAttribute('href') || '';
        const lengthAttr = downloadLink?.getAttribute('length') || '0';
        const sizeBytes = parseInt(lengthAttr);
        const size = sizeBytes > 0 ? this.formatFileSize(sizeBytes) : '';
        
        // Construct cover URL like desktop UI does
        const coverUrl = bookId ? `/opds/${App.currentLibrary}/covers/${bookId}/cover.jpg` : '';
        
        return {
          id: bookId,
          title,
          author,
          series,
          lang,
          size,
          download_url: downloadUrl,
          cover_url: coverUrl
        };
      });

      const results = document.getElementById('mobile-search-results');
      if (!results) return;

      if (books.length === 0) {
        results.innerHTML = '<div class="mobile-empty">No results found</div>';
        return;
      }

      results.innerHTML = books.map(book => `
        <div class="mobile-book-item" data-book-id="${book.id}">
          <div class="mobile-book-item-cover">
            ${book.cover_url ? 
              `<img src="${book.cover_url}" alt="${book.title}">` :
              `<div class="mobile-book-item-cover-placeholder">📚</div>`
            }
          </div>
          <div class="mobile-book-item-info">
            <div class="mobile-book-item-title">${book.title}</div>
            <div class="mobile-book-item-author">${book.author}</div>
            ${book.series ? `<div class="mobile-book-item-series">${book.series}</div>` : ''}
            <div class="mobile-book-item-meta">
              ${book.lang ? `<span>${book.lang.toUpperCase()}</span>` : ''}
              ${book.size ? `<span>${book.size}</span>` : ''}
            </div>
          </div>
          <svg class="mobile-list-item-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </div>
      `).join('');

      results.querySelectorAll('[data-book-id]').forEach(item => {
        item.addEventListener('click', () => {
          const bookId = item.dataset.bookId;
          const book = books.find(b => String(b.id) === String(bookId));
          if (book) {
            this.navigateTo('book-detail', { selectedBook: book });
          }
        });
      });
    } catch (e) {
      console.error('Search failed:', e);
      const results = document.getElementById('mobile-search-results');
      if (results) results.innerHTML = '<div class="mobile-empty">Search failed. Please try again.</div>';
    }
  },

  async performGlobalSearch(query) {
    if (!query) return;
    this.navigateTo('search');
    const searchInput = document.getElementById('mobile-search-query');
    if (searchInput) {
      searchInput.value = query;
      await this.performSearch();
    }
  }
};

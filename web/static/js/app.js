// Biblio OPDS Server Web UI

const App = {
  user: null,
  currentLibrary: null,
  currentView: 'home',
  theme: 'dark',
  libraries: [],
  currentTab: 'authors',
  currentAuthor: null,
  currentSeries: null,
  currentBook: null,
  books: [],
  authors: [],
  series: [],
  sortColumn: 'title',
  sortDirection: 'asc',
  
  // Mobile navigation state
  mobileScreen: 'home', // home, authors, series, genres, search, books, book-detail, config
  mobileHistory: [],
  isMobile: false,

  async init() {
    this.loadTheme();
    await this.checkAuth();
    await this.loadLibraries();
    this.bindEvents();
    
    // Initialize mobile UI if available
    if (typeof MobileUI !== 'undefined') {
      MobileUI.init();
    }
    
    this.router();
    window.addEventListener('hashchange', () => this.router());
  },

  async loadLibraries() {
    try {
      this.libraries = await this.fetchAPI('/api/libraries') || [];
    } catch (e) {
      this.libraries = [];
    }
  },

  loadTheme() {
    const saved = localStorage.getItem('theme') || 'dark';
    this.theme = saved;
    document.documentElement.setAttribute('data-theme', saved);
  },

  toggleTheme() {
    this.theme = this.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', this.theme);
    localStorage.setItem('theme', this.theme);
  },

  renderHeader(title, extraContent = '') {
    return `
      <header class="header">
        <h1 class="header-title">${title}</h1>
        <div class="header-actions">
          ${extraContent}
          <button type="button" class="theme-toggle" data-action="toggleTheme" title="Toggle theme">
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
      </header>
    `;
  },

  async checkAuth() {
    try {
      const res = await fetch('/api/auth/me');
      const data = await res.json();
      if (data.authenticated) {
        this.user = data.user;
      }
    } catch (e) {
      console.error('Auth check failed:', e);
    }
  },

  async checkSetup() {
    try {
      const res = await fetch('/api/setup/check');
      const data = await res.json();
      return data.setup_required;
    } catch (e) {
      return false;
    }
  },

  bindEvents() {
    document.addEventListener('click', (e) => {
      const actionEl = e.target.closest('[data-action]');
      if (actionEl) {
        const action = actionEl.dataset.action;
        console.log('Action clicked:', action, 'Method exists:', !!this[action]);
        if (this[action]) {
          e.preventDefault();
          this[action](actionEl);
        }
      }
    });

    document.addEventListener('submit', (e) => {
      if (e.target.matches('[data-form]')) {
        e.preventDefault();
        const form = e.target.dataset.form;
        if (this[form]) {
          this[form](e.target);
        }
      }
    });
  },

  async router() {
    const hash = window.location.hash.slice(1) || 'home';
    const [view, ...params] = hash.split('/');

    // Check if setup is required
    const setupRequired = await this.checkSetup();
    if (setupRequired && view !== 'setup') {
      window.location.hash = '#setup';
      return;
    }

    // Check auth for protected routes
    if (!this.user && !['login', 'setup'].includes(view)) {
      window.location.hash = '#login';
      return;
    }

    this.currentView = view;
    
    switch (view) {
      case 'setup':
        this.renderSetup();
        break;
      case 'login':
        this.renderLogin();
        break;
      case 'home':
        this.renderHome();
        break;
      case 'browser':
        if (params[0]) this.currentLibrary = parseInt(params[0]);
        this.renderBrowser();
        break;
      case 'library':
        this.currentLibrary = parseInt(params[0]) || 1;
        this.renderLibrary(params[1] || 'authors');
        break;
      case 'authors':
        this.currentLibrary = parseInt(params[0]) || 1;
        this.renderAuthors(params[1]);
        break;
      case 'author':
        this.renderAuthor(params[0]);
        break;
      case 'series':
        this.currentLibrary = parseInt(params[0]) || 1;
        this.renderSeries();
        break;
      case 'series-books':
        this.renderSeriesBooks(params[0]);
        break;
      case 'genres':
        this.currentLibrary = parseInt(params[0]) || 1;
        this.renderGenres();
        break;
      case 'genre':
        this.renderGenreBooks(params[0]);
        break;
      case 'search':
        this.currentLibrary = parseInt(params[0]) || 1;
        this.renderSearch();
        break;
      case 'settings':
        this.renderSettings();
        break;
      case 'libraries':
        this.renderLibraries();
        break;
      default:
        this.renderHome();
    }
  },

  // Render Methods
  renderSetup() {
    document.getElementById('app').innerHTML = `
      <div class="login-page">
        <div class="login-card">
          <div class="login-logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
            </svg>
          </div>
          <h1 class="login-title">Welcome to Biblio OPDS Server</h1>
          <p class="login-subtitle">Create your admin account to get started</p>
          <form data-form="submitSetup">
            <div class="form-group">
              <label class="form-label">Username</label>
              <input type="text" name="username" class="form-input" placeholder="admin" required>
            </div>
            <div class="form-group">
              <label class="form-label">Password</label>
              <input type="password" name="password" class="form-input" placeholder="••••••••" required>
            </div>
            <div class="form-group">
              <label class="form-label">Confirm Password</label>
              <input type="password" name="confirm" class="form-input" placeholder="••••••••" required>
            </div>
            <button type="submit" class="btn btn-primary" style="width:100%">Create Account</button>
            <p id="setup-error" class="text-center text-muted mt-2" style="color:var(--danger)"></p>
          </form>
        </div>
      </div>
    `;
  },

  renderLogin() {
    document.getElementById('app').innerHTML = `
      <div class="login-page">
        <button type="button" class="theme-toggle" data-action="toggleTheme" title="Toggle theme" style="position:absolute;top:1rem;right:1rem">
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
        <div class="login-card">
          <div class="login-logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
            </svg>
          </div>
          <h1 class="login-title">Biblio OPDS Server</h1>
          <p class="login-subtitle">Sign in to your account</p>
          <form data-form="submitLogin">
            <div class="form-group">
              <label class="form-label">Username</label>
              <input type="text" name="username" class="form-input" placeholder="Username" required>
            </div>
            <div class="form-group">
              <label class="form-label">Password</label>
              <input type="password" name="password" class="form-input" placeholder="••••••••" required>
            </div>
            <button type="submit" class="btn btn-primary" style="width:100%">Sign In</button>
            <p id="login-error" class="text-center text-muted mt-2" style="color:var(--danger)"></p>
          </form>
        </div>
      </div>
    `;
  },

  async renderHome() {
    // If user has libraries, go directly to browser view
    if (this.libraries.length > 0) {
      this.currentLibrary = this.libraries[0].id;
      window.location.hash = `#browser`;
      return;
    }

    // Show welcome screen if no libraries
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('home')}
        <div class="main">
          ${this.renderHeader('Dashboard')}
          <div class="content">
            <div class="empty-state" style="padding:4rem">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:64px;height:64px;margin-bottom:1rem">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
              </svg>
              <h2>Welcome to Biblio OPDS Server</h2>
              <p class="text-muted">No libraries imported yet. Go to Libraries to import your first library.</p>
              <a href="#libraries" class="btn btn-primary mt-2">Import Library</a>
            </div>
          </div>
        </div>
      </div>
    `;
  },

  // ========== FreeLib-style Browser View ==========
  async renderBrowser() {
    if (!this.currentLibrary && this.libraries.length > 0) {
      this.currentLibrary = this.libraries[0].id;
    }

    if (!this.currentLibrary) {
      window.location.hash = '#home';
      return;
    }

    // Check if mobile and use mobile UI
    if (this.isMobile && typeof MobileUI !== 'undefined') {
      MobileUI.render();
      return;
    }

    const currentLib = this.libraries.find(l => l.id === this.currentLibrary);

    document.getElementById('app').innerHTML = `
      <div class="app-browser">
        <!-- Toolbar with library selector -->
        <div class="toolbar">
          <div class="toolbar-group">
            <div class="app-logo">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:20px;height:20px;color:var(--primary)">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
              </svg>
              <span style="font-weight:600;font-size:1rem">Biblio OPDS Server</span>
            </div>
            <div class="toolbar-divider"></div>
            <div class="library-selector">
              <label>Library:</label>
              <select id="library-select">
                ${this.libraries.map(lib => `
                  <option value="${lib.id}" ${lib.id === this.currentLibrary ? 'selected' : ''}>${lib.name}</option>
                `).join('')}
              </select>
              <span id="library-stats" class="library-stats"></span>
            </div>
          </div>
          <div class="toolbar-group" style="margin-left:auto">
            <div class="search-box" style="width:250px">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:16px;height:16px;position:absolute;left:10px;top:50%;transform:translateY(-50%);color:var(--text-muted)">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
              <input type="text" class="form-input" placeholder="Search..." id="search-input" style="padding-left:32px;height:32px;font-size:0.875rem">
            </div>
            <button type="button" class="theme-toggle" data-action="toggleTheme" title="Toggle theme" style="width:32px;height:32px">
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
            ${this.user?.role === 'admin' ? `
              <a href="#libraries" class="btn btn-sm btn-outline" title="Manage Libraries">⚙️</a>
            ` : ''}
            <button class="btn btn-sm btn-outline" data-action="logout" title="Logout">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16,17 21,12 16,7"/>
                <line x1="21" y1="12" x2="9" y2="12"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- Tabs -->
        <div class="tabs">
          <button class="tab ${this.currentTab === 'authors' ? 'active' : ''}" data-tab="authors">Authors</button>
          <button class="tab ${this.currentTab === 'series' ? 'active' : ''}" data-tab="series">Series</button>
          <button class="tab ${this.currentTab === 'genres' ? 'active' : ''}" data-tab="genres">Genres</button>
          <button class="tab ${this.currentTab === 'search' ? 'active' : ''}" data-tab="search">Search</button>
        </div>

        <!-- Three-panel layout -->
        <div class="browser-layout">
          <!-- Mobile overlay -->
          <div class="mobile-overlay" id="mobile-overlay"></div>

          <!-- Left panel: Authors/Series list -->
          <div class="panel-left" id="panel-left">
            <div class="panel-header">
              <input type="text" class="panel-filter" placeholder="Filter..." id="panel-filter">
            </div>
            <div class="panel-list" id="panel-list">
              <div class="text-muted text-center" style="padding:2rem">Loading...</div>
            </div>
          </div>
          <div class="panel-resizer" id="resizer-left"></div>

          <!-- Center panel: Books table -->
          <div class="panel-center">
            <div class="books-table-wrapper">
              <table class="books-table" id="books-table">
                <thead>
                  <tr>
                    <th data-sort="author" class="sortable ${this.sortColumn === 'author' ? 'sorted-' + this.sortDirection : ''}">Author</th>
                    <th data-sort="title" class="sortable ${this.sortColumn === 'title' ? 'sorted-' + this.sortDirection : ''}">Title</th>
                    <th data-sort="size" class="col-size sortable ${this.sortColumn === 'size' ? 'sorted-' + this.sortDirection : ''}">Size</th>
                    <th data-sort="date" class="col-date sortable ${this.sortColumn === 'date' ? 'sorted-' + this.sortDirection : ''}">Date</th>
                    <th data-sort="genre" class="col-genre sortable ${this.sortColumn === 'genre' ? 'sorted-' + this.sortDirection : ''}">Genre</th>
                    <th data-sort="lang" class="sortable ${this.sortColumn === 'lang' ? 'sorted-' + this.sortDirection : ''}">Lang</th>
                  </tr>
                </thead>
                <tbody id="books-tbody">
                  <tr><td colspan="6" class="text-muted text-center" style="padding:2rem">Select an author or series</td></tr>
                </tbody>
              </table>
            </div>
          </div>

          <div class="panel-resizer" id="resizer-right"></div>
          <!-- Right panel: Book details -->
          <div class="panel-right" id="panel-right">
            <div class="book-details">
              <div class="book-cover-placeholder">📚</div>
              <div class="text-muted text-center">Select a book to view details</div>
            </div>
          </div>

          <!-- Mobile panel toggle buttons -->
          <button class="mobile-panel-toggle left" id="mobile-toggle-left" title="Show navigation">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="3" y1="12" x2="21" y2="12"></line>
              <line x1="3" y1="6" x2="21" y2="6"></line>
              <line x1="3" y1="18" x2="21" y2="18"></line>
            </svg>
          </button>
          <button class="mobile-panel-toggle right" id="mobile-toggle-right" title="Show book details">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
              <polyline points="13 2 13 9 20 9"></polyline>
            </svg>
          </button>
        </div>

        <!-- Status bar -->
        <div class="status-bar">
          <span id="status-left">${currentLib?.name || 'No library'}</span>
          <span id="status-right"></span>
        </div>
      </div>
    `;

    this.bindBrowserEvents();
    
    // Load initial data based on tab
    this.loadTabContent();
  },

  bindBrowserEvents() {
    // Library selector
    document.getElementById('library-select')?.addEventListener('change', (e) => {
      this.currentLibrary = parseInt(e.target.value);
      this.currentAuthor = null;
      this.currentSeries = null;
      this.currentBook = null;
      this.renderBrowser();
    });

    // Load library stats
    this.loadLibraryStats();

    // Tabs
    document.querySelectorAll('.tab').forEach(tab => {
      tab.addEventListener('click', () => {
        this.currentTab = tab.dataset.tab;
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        this.loadTabContent();
      });
    });

    // Panel filter with debounce for backend filtering
    let filterTimeout;
    document.getElementById('panel-filter')?.addEventListener('input', (e) => {
      clearTimeout(filterTimeout);
      filterTimeout = setTimeout(() => {
        const query = e.target.value.trim();
        if (this.currentTab === 'authors') {
          this.filterAuthors(query);
        } else if (this.currentTab === 'series') {
          this.filterSeries(query);
        } else {
          this.filterPanelList(query);
        }
      }, 300); // 300ms debounce
    });

    // Search input in toolbar
    document.getElementById('search-input')?.addEventListener('keypress', (e) => {
      if (e.key === 'Enter' && e.target.value.trim()) {
        this.currentTab = 'search';
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelector('[data-tab="search"]')?.classList.add('active');
        this.performBrowserSearch(e.target.value, 'all');
      }
    });

    // Sortable columns
    document.querySelectorAll('.books-table th[data-sort]').forEach(th => {
      th.addEventListener('click', () => {
        const column = th.dataset.sort;
        if (this.sortColumn === column) {
          this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
          this.sortColumn = column;
          this.sortDirection = 'asc';
        }
        this.sortAndRenderBooks();
      });
    });

    // Panel resizers
    this.initPanelResizers();

    // Mobile panel toggles
    this.initMobilePanelToggles();
  },

  initPanelResizers() {
    const resizerLeft = document.getElementById('resizer-left');
    const resizerRight = document.getElementById('resizer-right');
    const panelLeft = document.getElementById('panel-left');
    const panelRight = document.getElementById('panel-right');

    if (resizerLeft && panelLeft) {
      this.makeResizable(resizerLeft, panelLeft, 'left');
    }
    if (resizerRight && panelRight) {
      this.makeResizable(resizerRight, panelRight, 'right');
    }
  },

  makeResizable(resizer, panel, side) {
    let startX, startWidth;

    const onMouseMove = (e) => {
      const dx = e.clientX - startX;
      let newWidth;
      if (side === 'left') {
        newWidth = startWidth + dx;
      } else {
        newWidth = startWidth - dx;
      }
      // Clamp width
      newWidth = Math.max(200, Math.min(500, newWidth));
      panel.style.width = newWidth + 'px';
    };

    const onMouseUp = () => {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    resizer.addEventListener('mousedown', (e) => {
      startX = e.clientX;
      startWidth = panel.offsetWidth;
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    });
  },

  initMobilePanelToggles() {
    const toggleLeft = document.getElementById('mobile-toggle-left');
    const toggleRight = document.getElementById('mobile-toggle-right');
    const panelLeft = document.getElementById('panel-left');
    const panelRight = document.getElementById('panel-right');
    const overlay = document.getElementById('mobile-overlay');

    if (!toggleLeft || !toggleRight || !panelLeft || !panelRight || !overlay) return;

    // Toggle left panel (authors/series) - starts open by default
    toggleLeft.addEventListener('click', () => {
      const isClosed = panelLeft.classList.contains('mobile-closed');
      if (isClosed) {
        // Open left panel
        panelLeft.classList.remove('mobile-closed');
        panelRight.classList.remove('mobile-open');
        overlay.classList.remove('hidden');
      } else {
        // Close left panel
        panelLeft.classList.add('mobile-closed');
        overlay.classList.add('hidden');
      }
    });

    // Toggle right panel (book details)
    toggleRight.addEventListener('click', () => {
      const isOpen = panelRight.classList.contains('mobile-open');
      if (isOpen) {
        // Close right panel
        panelRight.classList.remove('mobile-open');
        // Restore left panel
        panelLeft.classList.remove('mobile-closed');
        overlay.classList.remove('hidden');
      } else {
        // Open right panel, close left
        panelLeft.classList.add('mobile-closed');
        panelRight.classList.add('mobile-open');
        overlay.classList.remove('hidden');
      }
    });

    // Close panels when clicking overlay - return to default state (left open)
    overlay.addEventListener('click', () => {
      panelLeft.classList.remove('mobile-closed');
      panelRight.classList.remove('mobile-open');
      overlay.classList.remove('hidden');
    });

    // Close left panel when selecting an item, show books
    panelLeft.addEventListener('click', (e) => {
      if (e.target.closest('.panel-item')) {
        setTimeout(() => {
          panelLeft.classList.add('mobile-closed');
          overlay.classList.add('hidden');
        }, 300);
      }
    });
  },

  closeMobilePanels() {
    const panelLeft = document.getElementById('panel-left');
    const panelRight = document.getElementById('panel-right');
    const overlay = document.getElementById('mobile-overlay');
    
    // Return to default state: left panel open, right panel closed
    if (panelLeft) panelLeft.classList.remove('mobile-closed');
    if (panelRight) panelRight.classList.remove('mobile-open');
    if (overlay) overlay.classList.remove('hidden');
  },

  sortAndRenderBooks() {
    if (this.books.length === 0) return;

    // Sort books
    this.books.sort((a, b) => {
      let valA = a[this.sortColumn] || '';
      let valB = b[this.sortColumn] || '';

      // Handle size sorting (parse numeric value)
      if (this.sortColumn === 'size') {
        valA = this.parseSizeToBytes(valA);
        valB = this.parseSizeToBytes(valB);
      }

      if (typeof valA === 'string') {
        valA = valA.toLowerCase();
        valB = valB.toLowerCase();
      }

      let result = 0;
      if (valA < valB) result = -1;
      else if (valA > valB) result = 1;

      return this.sortDirection === 'asc' ? result : -result;
    });

    // Update header classes
    document.querySelectorAll('.books-table th[data-sort]').forEach(th => {
      th.classList.remove('sorted-asc', 'sorted-desc');
      if (th.dataset.sort === this.sortColumn) {
        th.classList.add('sorted-' + this.sortDirection);
      }
    });

    // Re-render table body
    this.renderBooksTable();
  },

  parseSizeToBytes(sizeStr) {
    if (!sizeStr) return 0;
    const match = sizeStr.match(/(\d+(?:\.\d+)?)\s*(B|KB|MB|GB)/i);
    if (!match) return 0;
    const value = parseFloat(match[1]);
    const unit = match[2].toUpperCase();
    const multipliers = { 'B': 1, 'KB': 1024, 'MB': 1024*1024, 'GB': 1024*1024*1024 };
    return value * (multipliers[unit] || 1);
  },

  renderBooksTable() {
    const tbody = document.getElementById('books-tbody');
    if (!tbody) return;

    tbody.innerHTML = this.books.map((book, idx) => `
      <tr data-book-idx="${idx}">
        <td class="col-author" title="${book.author}">${book.author}</td>
        <td class="col-title" title="${book.title}">${book.title}</td>
        <td class="col-size">${book.size}</td>
        <td class="col-date">${book.date}</td>
        <td class="col-genre" title="${book.genre}">${book.genre}</td>
        <td>${book.lang}</td>
      </tr>
    `).join('');

    // Bind row click
    tbody.querySelectorAll('tr').forEach(row => {
      row.addEventListener('click', () => {
        this.selectBookRow(row);
      });
    });

    // Make table focusable for keyboard navigation
    const table = document.getElementById('books-table');
    if (table && !table.hasAttribute('tabindex')) {
      table.setAttribute('tabindex', '0');
      table.addEventListener('keydown', (e) => this.handleBooksKeyboard(e));
    }
  },

  selectBookRow(row) {
    const tbody = document.getElementById('books-tbody');
    if (!tbody || !row) return;
    
    tbody.querySelectorAll('tr').forEach(r => r.classList.remove('selected'));
    row.classList.add('selected');
    
    // Scroll row into view
    row.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    
    const idx = parseInt(row.dataset.bookIdx);
    if (!isNaN(idx) && this.books[idx]) {
      this.showBookDetails(this.books[idx]);
    }
  },

  handleBooksKeyboard(e) {
    const tbody = document.getElementById('books-tbody');
    if (!tbody) return;

    const rows = Array.from(tbody.querySelectorAll('tr[data-book-idx]'));
    if (rows.length === 0) return;

    const selectedRow = tbody.querySelector('tr.selected');
    let currentIdx = selectedRow ? rows.indexOf(selectedRow) : -1;
    let newIdx = currentIdx;

    switch (e.key) {
      case 'ArrowUp':
        e.preventDefault();
        newIdx = Math.max(0, currentIdx - 1);
        break;
      case 'ArrowDown':
        e.preventDefault();
        newIdx = Math.min(rows.length - 1, currentIdx + 1);
        break;
      case 'PageUp':
        e.preventDefault();
        newIdx = Math.max(0, currentIdx - 10);
        break;
      case 'PageDown':
        e.preventDefault();
        newIdx = Math.min(rows.length - 1, currentIdx + 10);
        break;
      case 'Home':
        e.preventDefault();
        newIdx = 0;
        break;
      case 'End':
        e.preventDefault();
        newIdx = rows.length - 1;
        break;
      case 'Enter':
        e.preventDefault();
        if (this.currentBook && this.currentBook.downloadLink) {
          window.location.href = this.currentBook.downloadLink;
        }
        return;
      default:
        return;
    }

    if (newIdx !== currentIdx && rows[newIdx]) {
      this.selectBookRow(rows[newIdx]);
    }
  },

  loadTabContent() {
    const panelList = document.getElementById('panel-list');
    const panelFilter = document.getElementById('panel-filter');
    
    // Clear filter
    if (panelFilter) panelFilter.value = '';
    
    if (this.currentTab === 'authors') {
      this.loadAuthorsList();
    } else if (this.currentTab === 'series') {
      this.loadSeriesList();
    } else if (this.currentTab === 'genres') {
      this.loadGenresList();
    } else if (this.currentTab === 'search') {
      this.showSearchPanel();
    }
  },

  // Virtual scrolling state
  vsAuthors: { items: [], total: 0, offset: 0, loading: false, filter: '' },
  vsSeries: { items: [], total: 0, offset: 0, loading: false, filter: '' },
  VS_PAGE_SIZE: 50,
  VS_ITEM_HEIGHT: 33, // pixels per item
  panelScrollHandler: null, // Store bound scroll handler for cleanup
  
  // Books pagination state
  booksNextUrl: null,
  booksLoading: false,
  booksScrollHandler: null,

  async loadLibraryStats() {
    if (!this.currentLibrary) return;
    
    try {
      const res = await fetch(`/api/libraries/${this.currentLibrary}/stats`);
      const data = await res.json();
      
      const statsEl = document.getElementById('library-stats');
      if (statsEl && data.book_count !== undefined) {
        statsEl.textContent = `(${data.book_count.toLocaleString()} books)`;
      }
    } catch (e) {
      console.error('Failed to load library stats:', e);
    }
  },

  async loadAuthorsList() {
    // Reset virtual scroll state
    this.vsAuthors = { items: [], total: 0, offset: 0, loading: false, filter: '' };
    
    const panelList = document.getElementById('panel-list');
    panelList.innerHTML = '<div class="text-muted text-center" style="padding:1rem">Loading...</div>';

    // Setup scroll event with cleanup of previous handler
    this.setupPanelScrollHandler(panelList);

    // Load first batch
    await this.loadAuthorsPage();
  },

  async loadAuthorsPage() {
    if (this.vsAuthors.loading) return;
    this.vsAuthors.loading = true;
    const isLoadingMore = this.vsAuthors.offset > 0;

    // Show loading toast
    this.showToast(isLoadingMore ? 'Loading more authors...' : 'Loading authors...', 'loading');

    try {
      const filter = encodeURIComponent(this.vsAuthors.filter);
      const res = await fetch(`/api/libraries/${this.currentLibrary}/authors?limit=${this.VS_PAGE_SIZE}&offset=${this.vsAuthors.offset}&filter=${filter}`);
      const data = await res.json();

      if (this.vsAuthors.offset === 0) {
        this.vsAuthors.items = [];
      }

      // Map authors to display format
      const newItems = (data.authors || []).map(a => ({
        id: a.id,
        name: [a.last_name, a.first_name, a.middle_name].filter(Boolean).join(' ').trim() || 'Unknown',
        book_count: a.BookCount || a.book_count || 0
      }));

      this.vsAuthors.items.push(...newItems);
      this.vsAuthors.total = data.total;
      this.vsAuthors.offset += newItems.length;

      this.renderAuthorsVirtualList();
      document.getElementById('status-right').textContent = `${this.vsAuthors.total} authors`;

    } catch (e) {
      console.error('Failed to load authors:', e);
      this.showToast('Failed to load authors', 'error');
    } finally {
      this.vsAuthors.loading = false;
    }
  },

  renderAuthorsVirtualList() {
    const panelList = document.getElementById('panel-list');
    if (!panelList) return;

    panelList.innerHTML = this.vsAuthors.items.map(author => `
      <div class="panel-item" data-author-id="${author.id}">
        <span>${author.name}</span>
        ${author.book_count ? `<span class="panel-item-count">${author.book_count}</span>` : ''}
      </div>
    `).join('');

    // Add "load more" button if there's more data
    if (this.vsAuthors.offset < this.vsAuthors.total) {
      panelList.innerHTML += `
        <div class="vs-load-more">
          <button class="btn btn-sm btn-outline" id="load-more-authors" style="width:100%;margin:0.5rem 0">
            Load more (${this.vsAuthors.offset} of ${this.vsAuthors.total})
          </button>
        </div>
      `;
      document.getElementById('load-more-authors')?.addEventListener('click', () => this.loadAuthorsPage());
    }

    // Bind click events
    panelList.querySelectorAll('.panel-item').forEach(item => {
      item.addEventListener('click', () => {
        panelList.querySelectorAll('.panel-item').forEach(i => i.classList.remove('selected'));
        item.classList.add('selected');
        this.loadBooksByAuthor(item.dataset.authorId);
      });
    });
  },

  onAuthorsScroll() {
    const panelList = document.getElementById('panel-list');
    if (!panelList) return;

    // Load more when near bottom
    const threshold = 100;
    if (panelList.scrollTop + panelList.clientHeight >= panelList.scrollHeight - threshold) {
      if (this.vsAuthors.offset < this.vsAuthors.total && !this.vsAuthors.loading) {
        this.loadAuthorsPage();
      }
    }
  },

  async filterAuthors(query) {
    this.vsAuthors.filter = query;
    this.vsAuthors.offset = 0;
    this.vsAuthors.items = [];
    await this.loadAuthorsPage();
  },

  showSearchPanel() {
    const panelList = document.getElementById('panel-list');
    panelList.innerHTML = `
      <div style="padding:1rem">
        <div class="form-group" style="margin-bottom:0.75rem">
          <label class="form-label" style="font-size:0.75rem;margin-bottom:0.25rem">Search by:</label>
          <select id="search-type" class="form-input" style="font-size:0.875rem;padding:0.5rem">
            <option value="all">All fields</option>
            <option value="title">Title</option>
            <option value="author">Author</option>
            <option value="series">Series</option>
          </select>
        </div>
        <div class="form-group" style="margin-bottom:0.75rem">
          <input type="text" id="search-query" class="form-input" placeholder="Enter search term..." style="font-size:0.875rem;padding:0.5rem">
        </div>
        <button class="btn btn-primary btn-sm" id="search-btn" style="width:100%">Search</button>
        <div id="search-results-list" style="margin-top:1rem"></div>
      </div>
    `;

    // Bind search events
    const searchBtn = document.getElementById('search-btn');
    const searchQuery = document.getElementById('search-query');
    const searchType = document.getElementById('search-type');

    searchBtn?.addEventListener('click', () => {
      const query = searchQuery.value.trim();
      const type = searchType.value;
      if (query) {
        this.performBrowserSearch(query, type);
      }
    });

    searchQuery?.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        const query = searchQuery.value.trim();
        const type = searchType.value;
        if (query) {
          this.performBrowserSearch(query, type);
        }
      }
    });
  },

  async loadSeriesList() {
    // Reset virtual scroll state
    this.vsSeries = { items: [], total: 0, offset: 0, loading: false, filter: '' };
    
    const panelList = document.getElementById('panel-list');
    panelList.innerHTML = '<div class="text-muted text-center" style="padding:1rem">Loading...</div>';

    // Setup scroll event with cleanup of previous handler
    this.setupPanelScrollHandler(panelList);

    // Load first batch
    await this.loadSeriesPage();
  },

  async loadSeriesPage() {
    if (this.vsSeries.loading) return;
    this.vsSeries.loading = true;
    const isLoadingMore = this.vsSeries.offset > 0;

    // Show loading toast
    this.showToast(isLoadingMore ? 'Loading more series...' : 'Loading series...', 'loading');

    try {
      const filter = encodeURIComponent(this.vsSeries.filter);
      const res = await fetch(`/api/libraries/${this.currentLibrary}/series?limit=${this.VS_PAGE_SIZE}&offset=${this.vsSeries.offset}&filter=${filter}`);
      const data = await res.json();

      if (this.vsSeries.offset === 0) {
        this.vsSeries.items = [];
      }

      // Map series to display format
      const newItems = (data.series || []).map(s => ({
        id: s.id,
        name: s.name || 'Unknown',
        book_count: s.BookCount || s.book_count || 0
      }));

      this.vsSeries.items.push(...newItems);
      this.vsSeries.total = data.total;
      this.vsSeries.offset += newItems.length;

      this.renderSeriesVirtualList();
      document.getElementById('status-right').textContent = `${this.vsSeries.total} series`;

    } catch (e) {
      console.error('Failed to load series:', e);
      this.showToast('Failed to load series', 'error');
    } finally {
      this.vsSeries.loading = false;
    }
  },

  renderSeriesVirtualList() {
    const panelList = document.getElementById('panel-list');
    if (!panelList) return;

    panelList.innerHTML = this.vsSeries.items.map(s => `
      <div class="panel-item" data-series-id="${s.id}">
        <span>${s.name}</span>
        ${s.book_count ? `<span class="panel-item-count">${s.book_count}</span>` : ''}
      </div>
    `).join('');

    // Add "load more" button if there's more data
    if (this.vsSeries.offset < this.vsSeries.total) {
      panelList.innerHTML += `
        <div class="vs-load-more">
          <button class="btn btn-sm btn-outline" id="load-more-series" style="width:100%;margin:0.5rem 0">
            Load more (${this.vsSeries.offset} of ${this.vsSeries.total})
          </button>
        </div>
      `;
      document.getElementById('load-more-series')?.addEventListener('click', () => this.loadSeriesPage());
    }

    // Bind click events
    panelList.querySelectorAll('.panel-item').forEach(item => {
      item.addEventListener('click', () => {
        panelList.querySelectorAll('.panel-item').forEach(i => i.classList.remove('selected'));
        item.classList.add('selected');
        this.loadBooksBySeries(item.dataset.seriesId);
      });
    });
  },

  onSeriesScroll() {
    const panelList = document.getElementById('panel-list');
    if (!panelList) return;

    // Load more when near bottom
    const threshold = 100;
    if (panelList.scrollTop + panelList.clientHeight >= panelList.scrollHeight - threshold) {
      if (this.vsSeries.offset < this.vsSeries.total && !this.vsSeries.loading) {
        this.loadSeriesPage();
      }
    }
  },

  // Unified scroll handler that checks currentTab to prevent cross-tab data loading
  setupPanelScrollHandler(panelList) {
    // Remove previous scroll handler if exists
    if (this.panelScrollHandler) {
      panelList.removeEventListener('scroll', this.panelScrollHandler);
    }
    
    // Create new bound handler
    this.panelScrollHandler = () => {
      if (this.currentTab === 'authors') {
        this.onAuthorsScroll();
      } else if (this.currentTab === 'series') {
        this.onSeriesScroll();
      }
      // Genres tab doesn't need scroll loading - it loads all at once
    };
    
    panelList.addEventListener('scroll', this.panelScrollHandler);
  },

  async filterSeries(query) {
    this.vsSeries.filter = query;
    this.vsSeries.offset = 0;
    this.vsSeries.items = [];
    await this.loadSeriesPage();
  },

  async loadGenresList() {
    const panelList = document.getElementById('panel-list');
    panelList.innerHTML = '<div class="text-muted text-center" style="padding:2rem">Loading...</div>';

    // Remove scroll handler - genres don't use virtual scrolling
    if (this.panelScrollHandler) {
      panelList.removeEventListener('scroll', this.panelScrollHandler);
      this.panelScrollHandler = null;
    }

    try {
      const genres = await this.fetchAPI('/api/genres');
      const topLevel = genres.filter(g => g.parent_id === 0);

      panelList.innerHTML = topLevel.map(genre => {
        const children = genres.filter(g => g.parent_id === genre.id);
        return `
          <div class="panel-item" style="flex-direction:column;align-items:flex-start">
            <strong style="margin-bottom:0.25rem">${genre.name}</strong>
            ${children.length > 0 ? `
              <div style="display:flex;flex-wrap:wrap;gap:0.25rem">
                ${children.map(c => `
                  <span class="badge" style="cursor:pointer;font-size:0.7rem" data-genre-id="${c.id}">${c.name}</span>
                `).join('')}
              </div>
            ` : ''}
          </div>
        `;
      }).join('');

      panelList.querySelectorAll('[data-genre-id]').forEach(item => {
        item.addEventListener('click', (e) => {
          e.stopPropagation();
          this.loadBooksByGenre(item.dataset.genreId);
        });
      });

      // Update status bar with genre count
      document.getElementById('status-right').textContent = `${genres.length} genres`;
    } catch (e) {
      panelList.innerHTML = '<div class="text-muted text-center" style="padding:2rem">Failed to load</div>';
    }
  },

  async loadBooksByAuthor(authorId) {
    this.currentAuthor = authorId;
    await this.loadBooks(`/opds/${this.currentLibrary}/author/${authorId}`);
  },

  async loadBooksBySeries(seriesId) {
    this.currentSeries = seriesId;
    await this.loadBooks(`/opds/${this.currentLibrary}/series/${seriesId}`);
  },

  async loadBooksByGenre(genreId) {
    await this.loadBooks(`/opds/${this.currentLibrary}/genres/${genreId}`);
  },

  async performBrowserSearch(query, type = 'all') {
    if (!query.trim()) return;
    
    // Build search URL with type parameter
    let searchUrl = `/opds/${this.currentLibrary}/search?q=${encodeURIComponent(query)}`;
    if (type && type !== 'all') {
      searchUrl += `&type=${type}`;
    }
    
    await this.loadBooks(searchUrl);
  },

  async loadBooks(url, append = false) {
    const tbody = document.getElementById('books-tbody');
    
    // Show loading toast
    this.showToast(append ? 'Loading more books...' : 'Loading books...', 'loading');

    if (!append) {
      tbody.innerHTML = '';
      this.books = [];
      this.booksNextUrl = null;
    }
    
    if (this.booksLoading) return;
    this.booksLoading = true;

    try {
      const res = await fetch(url);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const entries = xml.querySelectorAll('entry');
      
      // Parse pagination links
      const links = xml.querySelectorAll('feed > link');
      this.booksNextUrl = null;
      for (const link of links) {
        if (link.getAttribute('rel') === 'next') {
          this.booksNextUrl = link.getAttribute('href');
          break;
        }
      }

      const newBooks = Array.from(entries).map(entry => {
        const title = entry.querySelector('title')?.textContent || 'Untitled';
        const author = entry.querySelector('author name')?.textContent || '';
        const content = entry.querySelector('content')?.textContent || '';
        const updated = entry.querySelector('updated')?.textContent || '';
        
        // Get language from dc:language element
        const lang = entry.querySelector('language')?.textContent || 
                     entry.querySelector('[*|language]')?.textContent || 'ru';
        
        // Get format from dc:format element
        const format = entry.querySelector('format')?.textContent || 
                       entry.querySelector('[*|format]')?.textContent || '';
        
        // Get size from dc:extent element
        const extent = entry.querySelector('extent')?.textContent || 
                       entry.querySelector('[*|extent]')?.textContent || '';
        
        // Get genres from category elements
        const categories = entry.querySelectorAll('category');
        const genres = Array.from(categories).map(c => c.getAttribute('label') || c.getAttribute('term')).filter(Boolean);
        const genre = genres.join(', ');
        
        // Find acquisition link
        let downloadLink = '';
        const links = entry.querySelectorAll('link');
        for (const l of links) {
          const rel = l.getAttribute('rel') || '';
          if (rel.includes('acquisition')) {
            downloadLink = l.getAttribute('href');
            break;
          }
        }
        
        const bookId = downloadLink.match(/book\/(\d+)/)?.[1];
        
        // Use extent for size, fallback to format
        const size = extent || format.toUpperCase() || '';

        return { 
          id: bookId, 
          title, 
          author, 
          size, 
          date: updated ? new Date(updated).toLocaleDateString() : '',
          genre,
          lang,
          downloadLink,
          content
        };
      });
      
      // Append or replace books
      if (append) {
        this.books.push(...newBooks);
      } else {
        this.books = newBooks;
      }

      if (this.books.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-muted text-center" style="padding:2rem">No books found</td></tr>';
        return;
      }

      this.renderBooksTableFull();

      // Setup scroll handler for loading more books
      this.setupBooksScrollHandler();

      // Update status with pagination info
      const statusText = this.booksNextUrl 
        ? `${this.books.length} books (scroll for more)`
        : `${this.books.length} books`;
      document.getElementById('status-right').textContent = statusText;

      // Auto-select first book and focus table (only on initial load)
      if (!append && this.books.length > 0) {
        const table = document.getElementById('books-table');
        const firstRow = tbody.querySelector('tr');
        if (firstRow) {
          this.selectBookRow(firstRow);
          table?.focus();
        }
      }

    } catch (e) {
      console.error('Failed to load books:', e);
      if (!append) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-muted text-center" style="padding:2rem">Failed to load</td></tr>';
      }
      this.showToast('Failed to load books', 'error');
    } finally {
      this.booksLoading = false;
    }
  },
  
  renderBooksTableFull() {
    const tbody = document.getElementById('books-tbody');
    if (!tbody) return;
    
    tbody.innerHTML = this.books.map((book, idx) => `
      <tr data-book-idx="${idx}">
        <td class="col-author" title="${this.escapeHtml(book.author)}">${this.escapeHtml(book.author)}</td>
        <td class="col-title" title="${this.escapeHtml(book.title)}">${this.escapeHtml(book.title)}</td>
        <td class="col-size">${book.size}</td>
        <td class="col-date">${book.date}</td>
        <td class="col-genre" title="${this.escapeHtml(book.genre)}">${this.escapeHtml(book.genre)}</td>
        <td>${book.lang}</td>
      </tr>
    `).join('');

    // Bind row click
    tbody.querySelectorAll('tr').forEach(row => {
      row.addEventListener('click', () => {
        this.selectBookRow(row);
      });
    });

    // Make table focusable for keyboard navigation
    const table = document.getElementById('books-table');
    if (table && !table.hasAttribute('tabindex')) {
      table.setAttribute('tabindex', '0');
      table.addEventListener('keydown', (e) => this.handleBooksKeyboard(e));
    }
  },
  
  escapeHtml(text) {
    if (!text) return '';
    return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  },
  
  setupBooksScrollHandler() {
    const wrapper = document.querySelector('.books-table-wrapper');
    if (!wrapper) return;
    
    // Remove previous handler
    if (this.booksScrollHandler) {
      wrapper.removeEventListener('scroll', this.booksScrollHandler);
    }
    
    this.booksScrollHandler = () => {
      if (!this.booksNextUrl || this.booksLoading) return;
      
      const threshold = 100;
      if (wrapper.scrollTop + wrapper.clientHeight >= wrapper.scrollHeight - threshold) {
        this.loadBooks(this.booksNextUrl, true);
      }
    };
    
    wrapper.addEventListener('scroll', this.booksScrollHandler);
  },

  async showBookDetails(book) {
    this.currentBook = book;
    const panel = document.getElementById('panel-right');
    const coverUrl = book.id ? `/opds/${this.currentLibrary}/covers/${book.id}/cover.jpg` : '';

    // Fetch annotation from the book file if not already present
    let description = book.content || '';
    if (!description && book.id) {
      try {
        const res = await fetch(`/opds/${this.currentLibrary}/annotation/${book.id}`);
        if (res.ok) {
          description = await res.text();
        }
      } catch (e) {
        // Annotation not available
      }
    }

    panel.innerHTML = `
      <div class="book-details">
        ${coverUrl ? `
          <img src="${coverUrl}" class="book-cover-large" onerror="this.outerHTML='<div class=\\'book-cover-placeholder\\'>📚</div>'">
        ` : `
          <div class="book-cover-placeholder">📚</div>
        `}
        <div class="book-title-large">${book.title}</div>
        <div class="book-meta">
          <div class="book-meta-row">
            <span class="book-meta-label">Author:</span>
            <span class="book-meta-value">${book.author}</span>
          </div>
          ${book.genre ? `
            <div class="book-meta-row">
              <span class="book-meta-label">Genre:</span>
              <span class="book-meta-value">${book.genre}</span>
            </div>
          ` : ''}
          ${book.size ? `
            <div class="book-meta-row">
              <span class="book-meta-label">Size:</span>
              <span class="book-meta-value">${book.size}</span>
            </div>
          ` : ''}
          ${book.lang ? `
            <div class="book-meta-row">
              <span class="book-meta-label">Language:</span>
              <span class="book-meta-value">${book.lang}</span>
            </div>
          ` : ''}
          ${book.date ? `
            <div class="book-meta-row">
              <span class="book-meta-label">Date:</span>
              <span class="book-meta-value">${book.date}</span>
            </div>
          ` : ''}
        </div>
        ${description ? `
          <div class="book-description">
            <div class="book-description-label">Description:</div>
            <div class="book-description-text">${description}</div>
          </div>
        ` : ''}
        <div class="book-actions">
          ${book.downloadLink ? `
            <a href="${book.downloadLink}" class="btn btn-primary btn-sm" download>Download</a>
          ` : ''}
        </div>
      </div>
    `;
  },

  filterPanelList(query) {
    const items = document.querySelectorAll('#panel-list .panel-item');
    const q = query.toLowerCase();
    items.forEach(item => {
      const text = item.textContent.toLowerCase();
      item.style.display = text.includes(q) ? '' : 'none';
    });
  },

  async renderLibrary(tab = 'authors') {
    const library = await this.fetchAPI(`/api/libraries/${this.currentLibrary}`);
    
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('library', this.currentLibrary)}
        <div class="main">
          <header class="header">
            <h1 class="header-title">${library.name}</h1>
            <div class="header-actions">
              <div class="search-box">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/>
                  <path d="m21 21-4.35-4.35"/>
                </svg>
                <input type="text" class="form-input" placeholder="Search books..." id="search-input">
              </div>
              <button type="button" class="theme-toggle" data-action="toggleTheme" title="Toggle theme">
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
          </header>
          <div class="content" id="library-content">
            <div class="loading"><div class="spinner"></div></div>
          </div>
        </div>
      </div>
    `;

    document.getElementById('search-input').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        window.location.hash = `#search/${this.currentLibrary}?q=${encodeURIComponent(e.target.value)}`;
      }
    });

    this.loadLibraryTab(tab);
  },

  async loadLibraryTab(tab) {
    const content = document.getElementById('library-content');
    
    switch (tab) {
      case 'authors':
        await this.loadAuthors(content);
        break;
      case 'series':
        await this.loadSeries(content);
        break;
      case 'genres':
        await this.loadGenres(content);
        break;
    }
  },

  async loadAuthors(container) {
    const letters = 'АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЭЮЯ'.split('');
    const latinLetters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('');
    
    container.innerHTML = `
      <div class="alphabet-filter">
        ${[...letters, ...latinLetters].map(l => `
          <button class="alphabet-btn" data-letter="${l}">${l}</button>
        `).join('')}
      </div>
      <div id="authors-list" class="card">
        <div class="card-body">
          <p class="text-muted text-center">Select a letter to browse authors</p>
        </div>
      </div>
    `;

    container.querySelectorAll('.alphabet-btn').forEach(btn => {
      btn.addEventListener('click', () => this.loadAuthorsByLetter(btn.dataset.letter));
    });
  },

  async loadAuthorsByLetter(letter) {
    document.querySelectorAll('.alphabet-btn').forEach(b => b.classList.remove('active'));
    document.querySelector(`[data-letter="${letter}"]`)?.classList.add('active');
    
    const listEl = document.getElementById('authors-list');
    listEl.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    
    try {
      // Don't double-encode - the letter is already a string
      const res = await fetch(`/opds/${this.currentLibrary}/authors/${letter}`);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const entries = xml.querySelectorAll('entry');
      
      if (entries.length === 0) {
        listEl.innerHTML = '<div class="card-body"><p class="text-muted text-center">No authors found</p></div>';
        return;
      }
      
      listEl.innerHTML = `
        <div class="card-body" style="padding:0">
          ${Array.from(entries).map(entry => {
            const title = entry.querySelector('title')?.textContent || '';
            const link = entry.querySelector('link')?.getAttribute('href') || '';
            const id = link.match(/author\/(\d+)/)?.[1];
            if (!id) return '';
            return `
              <a href="#author/${id}" class="list-item">
                <span>${title}</span>
              </a>
            `;
          }).join('')}
        </div>
      `;
    } catch (e) {
      listEl.innerHTML = '<div class="card-body"><p class="text-muted text-center">Failed to load authors</p></div>';
    }
  },

  async loadSeries(container) {
    container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    
    try {
      const res = await fetch(`/opds/${this.currentLibrary}/series`);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const entries = xml.querySelectorAll('entry');
      
      container.innerHTML = `
        <div class="card">
          <div class="card-header">
            <h2 class="card-title">Series (${entries.length})</h2>
          </div>
          <div class="card-body" style="padding:0">
            ${Array.from(entries).map(entry => {
              const title = entry.querySelector('title')?.textContent || '';
              const link = entry.querySelector('link[type*="acquisition"]')?.getAttribute('href') || '';
              const id = link.match(/series\/(\d+)/)?.[1];
              const content = entry.querySelector('content')?.textContent || '';
              const count = content.match(/(\d+)/)?.[1] || '';
              return `
                <a href="#series-books/${id}" class="list-item">
                  <span style="flex:1">${title}</span>
                  <span class="badge">${count} books</span>
                </a>
              `;
            }).join('')}
          </div>
        </div>
      `;
    } catch (e) {
      container.innerHTML = '<div class="card-body"><p class="text-muted text-center">Failed to load series</p></div>';
    }
  },

  async loadGenres(container) {
    const genres = await this.fetchAPI('/api/genres');
    const topLevel = genres.filter(g => g.parent_id === 0);
    
    container.innerHTML = `
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">Genres</h2>
        </div>
        <div class="card-body" style="padding:0">
          ${topLevel.map(genre => {
            const children = genres.filter(g => g.parent_id === genre.id);
            return `
              <div class="list-item" style="flex-direction:column;align-items:flex-start">
                <strong>${genre.name}</strong>
                ${children.length > 0 ? `
                  <div style="display:flex;flex-wrap:wrap;gap:0.5rem;margin-top:0.5rem">
                    ${children.map(c => `
                      <a href="#genre/${c.id}" class="badge badge-primary">${c.name}</a>
                    `).join('')}
                  </div>
                ` : ''}
              </div>
            `;
          }).join('')}
        </div>
      </div>
    `;
  },

  async renderAuthor(authorId) {
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('library', this.currentLibrary)}
        <div class="main">
          ${this.renderHeader('Author')}
          <div class="content">
            <div class="loading"><div class="spinner"></div></div>
          </div>
        </div>
      </div>
    `;

    try {
      const res = await fetch(`/opds/${this.currentLibrary}/author/${authorId}`);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const title = xml.querySelector('feed > title')?.textContent || 'Author';
      const entries = xml.querySelectorAll('entry');
      
      document.querySelector('.header-title').textContent = title;
      document.querySelector('.content').innerHTML = this.renderBookGrid(entries);
    } catch (e) {
      document.querySelector('.content').innerHTML = '<p class="text-muted">Failed to load author</p>';
    }
  },

  async renderSeriesBooks(seriesId) {
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('library', this.currentLibrary)}
        <div class="main">
          ${this.renderHeader('Series')}
          <div class="content">
            <div class="loading"><div class="spinner"></div></div>
          </div>
        </div>
      </div>
    `;

    try {
      const res = await fetch(`/opds/${this.currentLibrary}/series/${seriesId}`);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const title = xml.querySelector('feed > title')?.textContent || 'Series';
      const entries = xml.querySelectorAll('entry');
      
      document.querySelector('.header-title').textContent = title;
      document.querySelector('.content').innerHTML = this.renderBookGrid(entries);
    } catch (e) {
      document.querySelector('.content').innerHTML = '<p class="text-muted">Failed to load series</p>';
    }
  },

  async renderGenreBooks(genreId) {
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('library', this.currentLibrary)}
        <div class="main">
          ${this.renderHeader('Genre')}
          <div class="content">
            <div class="loading"><div class="spinner"></div></div>
          </div>
        </div>
      </div>
    `;

    try {
      const res = await fetch(`/opds/${this.currentLibrary}/genres/${genreId}`);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const title = xml.querySelector('feed > title')?.textContent || 'Genre';
      const entries = xml.querySelectorAll('entry');
      
      document.querySelector('.header-title').textContent = title;
      document.querySelector('.content').innerHTML = this.renderBookGrid(entries);
    } catch (e) {
      document.querySelector('.content').innerHTML = '<p class="text-muted">Failed to load genre</p>';
    }
  },

  async renderSearch() {
    const query = new URLSearchParams(window.location.hash.split('?')[1]).get('q') || '';
    
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('search', this.currentLibrary)}
        <div class="main">
          ${this.renderHeader('Search Results')}
          <div class="content">
            <div class="search-box mb-4" style="max-width:100%">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
              <input type="text" class="form-input" placeholder="Search books..." value="${query}" id="search-input">
            </div>
            <div id="search-results">
              ${query ? '<div class="loading"><div class="spinner"></div></div>' : '<p class="text-muted">Enter a search term</p>'}
            </div>
          </div>
        </div>
      </div>
    `;

    document.getElementById('search-input').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        window.location.hash = `#search/${this.currentLibrary}?q=${encodeURIComponent(e.target.value)}`;
      }
    });

    if (query) {
      this.performSearch(query);
    }
  },

  async performSearch(query) {
    const resultsEl = document.getElementById('search-results');
    
    try {
      const res = await fetch(`/opds/${this.currentLibrary}/search?q=${encodeURIComponent(query)}`);
      const text = await res.text();
      const parser = new DOMParser();
      const xml = parser.parseFromString(text, 'text/xml');
      const entries = xml.querySelectorAll('entry');
      
      if (entries.length === 0) {
        resultsEl.innerHTML = '<p class="text-muted">No books found</p>';
        return;
      }
      
      resultsEl.innerHTML = this.renderBookGrid(entries);
    } catch (e) {
      resultsEl.innerHTML = '<p class="text-muted">Search failed</p>';
    }
  },

  async renderSettings() {
    if (!this.user?.role === 'admin') {
      window.location.hash = '#home';
      return;
    }

    const users = await this.fetchAPI('/api/users');
    
    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('settings')}
        <div class="main">
          ${this.renderHeader('Settings')}
          <div class="content">
            <div class="card mb-4">
              <div class="card-header">
                <h2 class="card-title">Users</h2>
                <button type="button" class="btn btn-primary btn-sm" data-action="showAddUser">Add User</button>
              </div>
              <div class="card-body" style="padding:0">
                <table class="table">
                  <thead>
                    <tr>
                      <th>Username</th>
                      <th>Role</th>
                      <th>Created</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    ${users.map(u => `
                      <tr>
                        <td>${u.username}</td>
                        <td><span class="badge ${u.role === 'admin' ? 'badge-primary' : ''}">${u.role}</span></td>
                        <td class="text-muted">${new Date(u.created_at).toLocaleDateString()}</td>
                        <td style="text-align:right">
                          <button type="button" class="btn btn-sm btn-outline" data-action="showChangePassword" data-id="${u.id}" data-username="${u.username}">Password</button>
                          ${u.id !== this.user.id ? `
                            <button type="button" class="btn btn-sm btn-outline" data-action="showChangeRole" data-id="${u.id}" data-username="${u.username}" data-role="${u.role}">Role</button>
                            <button type="button" class="btn btn-sm btn-danger" data-action="deleteUser" data-id="${u.id}">Delete</button>
                          ` : ''}
                        </td>
                      </tr>
                    `).join('')}
                  </tbody>
                </table>
              </div>
            </div>
            
          </div>
        </div>
      </div>
    `;
  },

  async renderLibraries() {
    if (this.user?.role !== 'admin') {
      window.location.hash = '#home';
      return;
    }

    const libraries = await this.fetchAPI('/api/libraries') || [];
    
    // Fetch stats for each library
    const librariesWithStats = await Promise.all(
      libraries.map(async lib => {
        try {
          const stats = await this.fetchAPI(`/api/libraries/${lib.id}/stats`);
          return { ...lib, ...stats };
        } catch (e) {
          return { ...lib, book_count: 0, author_count: 0, series_count: 0 };
        }
      })
    );

    document.getElementById('app').innerHTML = `
      <div class="app">
        ${this.renderSidebar('libraries')}
        <div class="main">
          ${this.renderHeader('Library Management')}
          <div class="content">
            <div class="card">
              <div class="card-header">
                <h2 class="card-title">Libraries</h2>
              </div>
              <div class="card-body" style="padding:0">
                ${libraries.length === 0 ? `
                  <div class="empty-state">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
                      <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
                    </svg>
                    <h3>No libraries yet</h3>
                    <p>Import a library using the CLI:<br><code>go run . import --inpx /path/to/file.inpx --name "Library Name" --path /path/to/books</code></p>
                  </div>
                ` : `
                  <table class="table">
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>Books</th>
                        <th>OPDS URL</th>
                        <th>Status</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      ${librariesWithStats.map(lib => `
                        <tr>
                          <td>
                            <a href="#library/${lib.id}" class="table-link">${lib.name}</a>
                            <div class="text-muted" style="font-size:0.75rem;max-width:250px;overflow:hidden;text-overflow:ellipsis" title="${lib.path}">${lib.path}</div>
                          </td>
                          <td>${lib.book_count?.toLocaleString() || 0}</td>
                          <td>
                            <code style="background:var(--bg-tertiary);padding:0.25rem 0.5rem;border-radius:var(--radius);font-size:0.8rem;cursor:pointer" 
                                  onclick="navigator.clipboard.writeText('${window.location.origin}/opds/${lib.id}');this.title='Copied!'" 
                                  title="Click to copy">${window.location.origin}/opds/${lib.id}</code>
                          </td>
                          <td>
                            <label class="toggle-switch">
                              <input type="checkbox" ${lib.enabled ? 'checked' : ''} data-action="toggleLibrary" data-id="${lib.id}">
                              <span class="badge ${lib.enabled ? 'badge-success' : ''}">${lib.enabled ? 'Enabled' : 'Disabled'}</span>
                            </label>
                          </td>
                          <td>
                            <div class="flex gap-1">
                              <button class="btn btn-sm btn-outline" data-action="editLibrary" data-id="${lib.id}" data-name="${lib.name}" data-path="${lib.path}">Edit</button>
                              <button class="btn btn-sm btn-outline" data-action="reindexLibrary" data-id="${lib.id}" data-name="${lib.name}">Reindex</button>
                              <button class="btn btn-sm btn-danger" data-action="deleteLibrary" data-id="${lib.id}" data-name="${lib.name}">Delete</button>
                            </div>
                          </td>
                        </tr>
                      `).join('')}
                    </tbody>
                  </table>
                `}
              </div>
            </div>
            
            <div class="card mt-4">
              <div class="card-header">
                <h2 class="card-title">Import New Library</h2>
              </div>
              <div class="card-body">
                <form data-form="submitImportLibrary">
                  <div class="form-group">
                    <label class="form-label">Library Name *</label>
                    <input type="text" name="name" class="form-input" placeholder="My Library" required>
                  </div>
                  <div class="form-group">
                    <label class="form-label">INPX File Path (on server) *</label>
                    <div class="input-with-btn">
                      <input type="text" name="inpx_path" id="inpx-path-input" class="form-input" placeholder="/data/library/books.inpx" required>
                      <button type="button" class="btn btn-outline" data-action="browseInpx">Browse...</button>
                    </div>
                  </div>
                  <div class="form-group">
                    <label class="form-label">Library Path (folder with ZIP files) *</label>
                    <div class="input-with-btn">
                      <input type="text" name="library_path" id="library-path-input" class="form-input" placeholder="/data/library" required>
                      <button type="button" class="btn btn-outline" data-action="browseLibraryPath">Browse...</button>
                    </div>
                  </div>
                  <div class="form-group">
                    <label class="form-label" style="display:flex;align-items:center;gap:0.5rem">
                      <input type="checkbox" name="first_author_only">
                      <span>First author only</span>
                    </label>
                  </div>
                  <div class="mt-2">
                    <button type="submit" class="btn btn-primary" id="import-btn">Import Library</button>
                  </div>
                  <div id="import-progress" style="display:none;margin-top:1rem">
                    <div class="progress-bar">
                      <div class="progress-fill" id="progress-fill" style="width:0%"></div>
                    </div>
                    <div id="import-status" class="text-muted mt-1" style="font-size:0.875rem"></div>
                  </div>
                </form>
              </div>
            </div>
          </div>
        </div>
      </div>
    `;
  },

  async toggleLibrary(btn) {
    const id = btn.dataset.id;
    const enabled = btn.checked;
    
    try {
      await fetch(`/api/libraries/${id}/toggle`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled })
      });
      this.renderLibraries();
    } catch (e) {
      alert('Failed to update library');
    }
  },

  editLibrary(btn) {
    const id = btn.dataset.id;
    const currentName = btn.dataset.name;
    
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
      <div class="modal-dialog">
        <div class="modal-header">
          <h3 class="modal-title">Edit Library</h3>
          <button type="button" class="modal-close" data-action="closeModal">&times;</button>
        </div>
        <div class="modal-body">
          <form data-form="submitEditLibrary" data-library-id="${id}">
            <div class="form-group">
              <label class="form-label">Library Name</label>
              <input type="text" name="name" class="form-control" value="${currentName}" required>
            </div>
            <div id="edit-library-error" class="form-error"></div>
            <div class="modal-actions">
              <button type="button" class="btn btn-outline" data-action="closeModal">Cancel</button>
              <button type="submit" class="btn btn-primary">Save</button>
            </div>
          </form>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
  },

  async submitEditLibrary(form) {
    const id = form.dataset.libraryId;
    const data = new FormData(form);
    const name = data.get('name').trim();
    const errorEl = document.getElementById('edit-library-error');
    
    if (!name) {
      errorEl.textContent = 'Library name is required';
      return;
    }
    
    try {
      const res = await fetch(`/api/libraries/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      });
      
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to update library');
      }
      
      this.closeModal();
      this.renderLibraries();
    } catch (e) {
      errorEl.textContent = e.message;
    }
  },

  reindexLibrary(btn) {
    const id = btn.dataset.id;
    const name = btn.dataset.name;
    
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
      <div class="modal-dialog">
        <div class="modal-header">
          <h3 class="modal-title">Reindex Library: ${name}</h3>
          <button type="button" class="modal-close" data-action="closeModal">&times;</button>
        </div>
        <div class="modal-body">
          <form data-form="submitReindex" data-library-id="${id}">
            <div class="form-group">
              <label class="form-label">Output INPX File Path</label>
              <div class="input-group">
                <input type="text" name="output_path" id="reindex-output-path" class="form-control" placeholder="/path/to/output.inpx" required>
                <button type="button" class="btn btn-outline" data-action="browseReindexOutput">Browse...</button>
              </div>
              <small class="form-help">Full path where the INPX file will be created</small>
            </div>
            <div id="reindex-status" class="form-info" style="display:none;">
              <div class="progress-bar">
                <div id="reindex-progress" class="progress-fill" style="width: 0%"></div>
              </div>
              <div id="reindex-message" class="mt-2"></div>
            </div>
            <div id="reindex-error" class="form-error"></div>
            <div class="modal-actions">
              <button type="button" class="btn btn-outline" data-action="closeModal">Cancel</button>
              <button type="submit" class="btn btn-primary" id="reindex-submit-btn">Start Reindex</button>
            </div>
          </form>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
  },

  async submitReindex(form) {
    const id = form.dataset.libraryId;
    const data = new FormData(form);
    const outputPath = data.get('output_path').trim();
    const errorEl = document.getElementById('reindex-error');
    const statusEl = document.getElementById('reindex-status');
    const messageEl = document.getElementById('reindex-message');
    const progressEl = document.getElementById('reindex-progress');
    const submitBtn = document.getElementById('reindex-submit-btn');
    
    if (!outputPath) {
      errorEl.textContent = 'Output path is required';
      return;
    }
    
    errorEl.textContent = '';
    statusEl.style.display = 'block';
    submitBtn.disabled = true;
    messageEl.textContent = 'Starting reindex...';
    progressEl.style.width = '0%';
    
    try {
      const res = await fetch('/api/libraries/reindex', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          library_id: parseInt(id),
          output_path: outputPath 
        })
      });
      
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to reindex library');
      }
      
      const result = await res.json();
      progressEl.style.width = '100%';
      messageEl.textContent = result.message || 'Reindex completed successfully!';
      
      setTimeout(() => {
        this.closeModal();
      }, 2000);
    } catch (e) {
      errorEl.textContent = e.message;
      submitBtn.disabled = false;
      statusEl.style.display = 'none';
    }
  },

  async deleteLibrary(btn) {
    const id = btn.dataset.id;
    const name = btn.dataset.name;
    
    if (!confirm(`Are you sure you want to delete the library "${name}"?\n\nThis will remove all books, authors, and series from the database. The actual book files will not be deleted.`)) {
      return;
    }
    
    try {
      await fetch(`/api/libraries/${id}`, { method: 'DELETE' });
      this.renderLibraries();
    } catch (e) {
      alert('Failed to delete library');
    }
  },

  browseInpx() {
    this.showFilePicker('inpx-path-input', 'inpx');
  },

  browseLibraryPath() {
    this.showFilePicker('library-path-input', 'dir');
  },

  browseReindexOutput() {
    // Use dir picker, then user can append filename
    this.filePickerCallback = (path) => {
      const input = document.getElementById('reindex-output-path');
      // Append default filename if path is a directory
      const outputPath = path.endsWith('/') ? path + 'library.inpx' : path + '/library.inpx';
      input.value = outputPath;
      this.closeFilePicker();
    };
    this.filePickerType = 'dir';
    this.renderFilePicker('/');
  },

  async submitImportLibrary(form) {
    const data = new FormData(form);
    const statusEl = document.getElementById('import-status');
    const progressEl = document.getElementById('import-progress');
    const progressFill = document.getElementById('progress-fill');
    const btnEl = document.getElementById('import-btn');
    
    btnEl.disabled = true;
    btnEl.textContent = 'Importing...';
    progressEl.style.display = 'block';
    progressFill.style.width = '0%';
    statusEl.textContent = 'Starting import...';
    statusEl.style.color = 'var(--text-muted)';
    
    const params = new URLSearchParams({
      name: data.get('name'),
      inpx_path: data.get('inpx_path'),
      library_path: data.get('library_path'),
      first_author_only: data.get('first_author_only') === 'on' ? 'true' : 'false'
    });

    try {
      console.log('Import request:', `/api/libraries/import?${params}`);
      const response = await fetch(`/api/libraries/import?${params}`, {
        credentials: 'include'
      });
      
      console.log('Response status:', response.status, response.statusText);
      
      if (!response.ok) {
        const text = await response.text();
        console.log('Error response:', text);
        try {
          const err = JSON.parse(text);
          throw new Error(err.error || 'Import failed');
        } catch (e) {
          if (e.message.includes('Import failed') || e.message.includes('authenticated')) {
            throw e;
          }
          throw new Error(`HTTP ${response.status}: ${text || response.statusText}`);
        }
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const progress = JSON.parse(line.slice(6));
            
            if (progress.total > 0) {
              const percent = Math.round((progress.current / progress.total) * 100);
              progressFill.style.width = percent + '%';
            }
            
            statusEl.textContent = progress.message;
            
            if (progress.done) {
              if (progress.error) {
                statusEl.textContent = `✗ ${progress.error}`;
                statusEl.style.color = 'var(--danger)';
                progressFill.style.background = 'var(--danger)';
                btnEl.disabled = false;
                btnEl.textContent = 'Import Library';
              } else {
                statusEl.textContent = `✓ ${progress.message}`;
                statusEl.style.color = 'var(--success)';
                progressFill.style.width = '100%';
                progressFill.style.background = 'var(--success)';
                form.reset();
                setTimeout(() => this.renderLibraries(), 1500);
              }
              return;
            }
          }
        }
      }
    } catch (e) {
      statusEl.textContent = '✗ ' + e.message;
      statusEl.style.color = 'var(--danger)';
      btnEl.disabled = false;
      btnEl.textContent = 'Import Library';
    }
  },

  renderBookGrid(entries) {
    if (entries.length === 0) {
      return '<p class="text-muted">No books found</p>';
    }
    
    return `
      <div class="book-grid">
        ${Array.from(entries).map(entry => {
          const title = entry.querySelector('title')?.textContent || 'Untitled';
          const author = entry.querySelector('author name')?.textContent || '';
          const format = entry.querySelector('format')?.textContent || 'fb2';
          
          // Find acquisition link - try multiple selectors
          let link = '';
          const links = entry.querySelectorAll('link');
          for (const l of links) {
            const rel = l.getAttribute('rel') || '';
            if (rel.includes('acquisition')) {
              link = l.getAttribute('href');
              break;
            }
          }
          
          // Extract book ID from link for cover
          const bookId = link.match(/book\/(\d+)/)?.[1];
          const libId = link.match(/opds\/(\d+)/)?.[1] || '1';
          const coverUrl = bookId ? `/opds/${libId}/covers/${bookId}/cover.jpg` : '';
          
          return `
            <div class="book-card">
              <div class="book-cover" ${coverUrl ? `style="background:none"` : ''}>
                ${coverUrl ? `<img src="${coverUrl}" onerror="this.parentElement.innerHTML='📚';this.parentElement.style.background='linear-gradient(135deg, var(--primary-light), var(--primary-dark))'">` : '📚'}
              </div>
              <div class="book-info">
                <div class="book-title">${title}</div>
                <div class="book-author">${author}</div>
                <div class="flex gap-1 mt-1">
                  ${link ? `<a href="${link}" class="btn btn-sm btn-primary" download>Download ${format.toUpperCase()}</a>` : '<span class="text-muted">No download</span>'}
                </div>
              </div>
            </div>
          `;
        }).join('')}
      </div>
    `;
  },

  renderSidebar(active, libId = null) {
    const isAdmin = this.user?.role === 'admin';
    
    return `
      <aside class="sidebar">
        <div class="sidebar-header">
          <div class="sidebar-logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
            </svg>
            <span>OPDS Server</span>
          </div>
        </div>
        <nav class="sidebar-nav">
          <div class="nav-section">Main</div>
          <a href="#home" class="nav-item ${active === 'home' ? 'active' : ''}">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
              <polyline points="9,22 9,12 15,12 15,22"/>
            </svg>
            <span>Dashboard</span>
          </a>
          
          ${libId ? `
            <div class="nav-section">Library</div>
            <a href="#library/${libId}/authors" class="nav-item ${active === 'library' ? 'active' : ''}">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                <circle cx="9" cy="7" r="4"/>
              </svg>
              <span>Authors</span>
            </a>
            <a href="#library/${libId}/series" class="nav-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
              </svg>
              <span>Series</span>
            </a>
            <a href="#library/${libId}/genres" class="nav-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="12,2 2,7 12,12 22,7"/>
                <polyline points="2,17 12,22 22,17"/>
                <polyline points="2,12 12,17 22,12"/>
              </svg>
              <span>Genres</span>
            </a>
            <a href="#search/${libId}" class="nav-item ${active === 'search' ? 'active' : ''}">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="m21 21-4.35-4.35"/>
              </svg>
              <span>Search</span>
            </a>
          ` : ''}
          
          ${isAdmin ? `
            <div class="nav-section">Admin</div>
            <a href="#libraries" class="nav-item ${active === 'libraries' ? 'active' : ''}">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
                <path d="M8 7h8M8 11h8M8 15h4"/>
              </svg>
              <span>Libraries</span>
            </a>
            <a href="#settings" class="nav-item ${active === 'settings' ? 'active' : ''}">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
              </svg>
              <span>Settings</span>
            </a>
          ` : ''}
        </nav>
        <div class="sidebar-footer">
          <div class="user-info">
            <div class="user-avatar">${this.user?.username?.charAt(0).toUpperCase() || 'U'}</div>
            <div class="user-details">
              <div class="user-name">${this.user?.username || 'User'}</div>
              <div class="user-role">${this.user?.role || ''}</div>
            </div>
            <button class="btn btn-icon btn-outline" data-action="logout" title="Logout">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16,17 21,12 16,7"/>
                <line x1="21" y1="12" x2="9" y2="12"/>
              </svg>
            </button>
          </div>
        </div>
      </aside>
    `;
  },

  // Form Handlers
  async submitSetup(form) {
    const data = new FormData(form);
    const username = data.get('username');
    const password = data.get('password');
    const confirm = data.get('confirm');
    
    if (password !== confirm) {
      document.getElementById('setup-error').textContent = 'Passwords do not match';
      return;
    }

    try {
      const res = await fetch('/api/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      
      const result = await res.json();
      if (result.success) {
        this.user = result.user;
        window.location.hash = '#home';
      } else {
        document.getElementById('setup-error').textContent = result.error || 'Setup failed';
      }
    } catch (e) {
      document.getElementById('setup-error').textContent = 'Setup failed';
    }
  },

  async submitLogin(form) {
    const data = new FormData(form);
    
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: data.get('username'),
          password: data.get('password')
        })
      });
      
      const result = await res.json();
      if (result.success) {
        this.user = result.user;
        window.location.hash = '#home';
      } else {
        document.getElementById('login-error').textContent = result.error || 'Login failed';
      }
    } catch (e) {
      document.getElementById('login-error').textContent = 'Login failed';
    }
  },

  async logout() {
    await fetch('/api/auth/logout', { method: 'POST' });
    this.user = null;
    window.location.hash = '#login';
  },

  async deleteUser(btn) {
    if (!confirm('Are you sure you want to delete this user?')) return;
    
    const id = btn.dataset.id;
    await fetch(`/api/users/${id}`, { method: 'DELETE', credentials: 'include' });
    this.renderSettings();
  },

  showAddUser() {
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
      <div class="modal-dialog">
        <div class="modal-header">
          <h3 class="modal-title">Add User</h3>
          <button type="button" class="modal-close" data-action="closeModal">&times;</button>
        </div>
        <div class="modal-body">
          <form data-form="submitAddUser">
            <div class="form-group">
              <label class="form-label">Username</label>
              <input type="text" name="username" class="form-control" placeholder="Enter username" required autocomplete="username">
            </div>
            <div class="form-group">
              <label class="form-label">Password</label>
              <input type="password" name="password" class="form-control" placeholder="Enter password" required autocomplete="new-password">
            </div>
            <div class="form-group">
              <label class="form-label">Role</label>
              <select name="role" class="form-control">
                <option value="readonly">Read Only</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div id="add-user-error" class="form-error"></div>
            <div class="modal-actions">
              <button type="button" class="btn btn-outline" data-action="closeModal">Cancel</button>
              <button type="submit" class="btn btn-primary">Create User</button>
            </div>
          </form>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
  },

  async submitAddUser(form) {
    const data = new FormData(form);
    const errorEl = document.getElementById('add-user-error');
    
    try {
      const res = await fetch('/api/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          username: data.get('username'),
          password: data.get('password'),
          role: data.get('role')
        })
      });
      
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to create user');
      }
      
      this.closeModal();
      this.renderSettings();
    } catch (e) {
      errorEl.textContent = e.message;
    }
  },

  showChangePassword(btn) {
    const id = btn.dataset.id;
    const username = btn.dataset.username;
    
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
      <div class="modal-dialog">
        <div class="modal-header">
          <h3 class="modal-title">Change Password: ${username}</h3>
          <button type="button" class="modal-close" data-action="closeModal">&times;</button>
        </div>
        <div class="modal-body">
          <form data-form="submitChangePassword" data-user-id="${id}">
            <div class="form-group">
              <label class="form-label">New Password</label>
              <input type="password" name="password" class="form-control" placeholder="Enter new password" required autocomplete="new-password">
            </div>
            <div class="form-group">
              <label class="form-label">Confirm Password</label>
              <input type="password" name="confirm" class="form-control" placeholder="Confirm new password" required autocomplete="new-password">
            </div>
            <div id="change-password-error" class="form-error"></div>
            <div class="modal-actions">
              <button type="button" class="btn btn-outline" data-action="closeModal">Cancel</button>
              <button type="submit" class="btn btn-primary">Update Password</button>
            </div>
          </form>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
  },

  async submitChangePassword(form) {
    const data = new FormData(form);
    const errorEl = document.getElementById('change-password-error');
    const userId = form.dataset.userId;
    
    if (data.get('password') !== data.get('confirm')) {
      errorEl.textContent = 'Passwords do not match';
      return;
    }
    
    try {
      const res = await fetch(`/api/users/${userId}/password`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ password: data.get('password') })
      });
      
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to update password');
      }
      
      this.closeModal();
      this.renderSettings();
    } catch (e) {
      errorEl.textContent = e.message;
    }
  },

  showChangeRole(btn) {
    const id = btn.dataset.id;
    const username = btn.dataset.username;
    const currentRole = btn.dataset.role;
    
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
      <div class="modal-dialog">
        <div class="modal-header">
          <h3 class="modal-title">Change Role: ${username}</h3>
          <button type="button" class="modal-close" data-action="closeModal">&times;</button>
        </div>
        <div class="modal-body">
          <form data-form="submitChangeRole" data-user-id="${id}">
            <div class="form-group">
              <label class="form-label">Role</label>
              <select name="role" class="form-control">
                <option value="readonly" ${currentRole === 'readonly' ? 'selected' : ''}>Read Only</option>
                <option value="admin" ${currentRole === 'admin' ? 'selected' : ''}>Admin</option>
              </select>
            </div>
            <div id="change-role-error" class="form-error"></div>
            <div class="modal-actions">
              <button type="button" class="btn btn-outline" data-action="closeModal">Cancel</button>
              <button type="submit" class="btn btn-primary">Update Role</button>
            </div>
          </form>
        </div>
      </div>
    `;
    document.body.appendChild(modal);
  },

  async submitChangeRole(form) {
    const data = new FormData(form);
    const errorEl = document.getElementById('change-role-error');
    const userId = form.dataset.userId;
    
    try {
      const res = await fetch(`/api/users/${userId}/role`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ role: data.get('role') })
      });
      
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to update role');
      }
      
      this.closeModal();
      this.renderSettings();
    } catch (e) {
      errorEl.textContent = e.message;
    }
  },

  closeModal() {
    const modal = document.querySelector('.modal-overlay');
    if (modal) modal.remove();
  },

  // API Helper
  async fetchAPI(url) {
    try {
      const res = await fetch(url);
      return await res.json();
    } catch (e) {
      console.error('API error:', e);
      return [];
    }
  },

  // File/Directory Picker
  filePickerCallback: null,
  filePickerType: 'dir',

  async showFilePicker(inputId, type = 'dir') {
    this.filePickerCallback = (path) => {
      document.getElementById(inputId).value = path;
      this.closeFilePicker();
    };
    this.filePickerType = type;
    await this.renderFilePicker('/');
  },

  async renderFilePicker(path) {
    const type = this.filePickerType;
    const res = await this.fetchAPI(`/api/browse?path=${encodeURIComponent(path)}&type=${type}`);
    
    const modal = document.getElementById('file-picker-modal') || this.createFilePickerModal();
    
    modal.querySelector('.modal-title').textContent = type === 'inpx' ? 'Select INPX File' : 'Select Directory';
    modal.querySelector('.modal-body').innerHTML = `
      <div class="file-picker">
        <div class="file-picker-path">
          <input type="text" class="form-input" value="${res.path}" id="picker-path-input" style="flex:1">
          <button class="btn btn-outline btn-sm" onclick="App.navigateToPath()">Go</button>
        </div>
        <div class="file-picker-list">
          ${res.parent ? `
            <div class="file-picker-item" onclick="App.renderFilePicker('${res.parent.replace(/'/g, "\\'")}')">
              <span class="file-icon">📁</span>
              <span>..</span>
            </div>
          ` : ''}
          ${res.entries.map(entry => `
            <div class="file-picker-item ${entry.is_dir ? 'is-dir' : 'is-file'}" 
                 onclick="App.${entry.is_dir ? 'renderFilePicker' : 'selectFile'}('${entry.path.replace(/'/g, "\\'")}')">
              <span class="file-icon">${entry.is_dir ? '📁' : '📄'}</span>
              <span>${entry.name}</span>
              ${!entry.is_dir && entry.size ? `<span class="file-size">${this.formatSize(entry.size)}</span>` : ''}
            </div>
          `).join('')}
          ${res.entries.length === 0 ? '<div class="text-muted text-center" style="padding:2rem">No items</div>' : ''}
        </div>
        <div class="file-picker-actions">
          ${type === 'dir' ? `<button class="btn btn-primary" onclick="App.selectCurrentDir()">Select This Directory</button>` : ''}
          <button class="btn btn-outline" onclick="App.closeFilePicker()">Cancel</button>
        </div>
      </div>
    `;
    
    modal.style.display = 'flex';
  },

  createFilePickerModal() {
    const modal = document.createElement('div');
    modal.id = 'file-picker-modal';
    modal.className = 'modal-overlay';
    modal.innerHTML = `
      <div class="modal">
        <div class="modal-header">
          <h3 class="modal-title">Select Directory</h3>
          <button class="btn btn-icon" onclick="App.closeFilePicker()">&times;</button>
        </div>
        <div class="modal-body"></div>
      </div>
    `;
    document.body.appendChild(modal);
    return modal;
  },

  navigateToPath() {
    const path = document.getElementById('picker-path-input').value;
    this.renderFilePicker(path);
  },

  selectFile(path) {
    if (this.filePickerCallback) {
      this.filePickerCallback(path);
    }
  },

  selectCurrentDir() {
    const path = document.getElementById('picker-path-input').value;
    if (this.filePickerCallback) {
      this.filePickerCallback(path);
    }
  },

  closeFilePicker() {
    const modal = document.getElementById('file-picker-modal');
    if (modal) {
      modal.style.display = 'none';
    }
    this.filePickerCallback = null;
  },

  formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
  },

  showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    
    const icons = {
      success: '✅',
      error: '❌',
      warning: '⚠️',
      info: 'ℹ️',
      loading: '⏳'
    };
    
    toast.innerHTML = `
      <span>${icons[type] || icons.info}</span>
      <span>${this.escapeHtml(message)}</span>
    `;
    
    container.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 1500);
  }
};

// Initialize app
document.addEventListener('DOMContentLoaded', () => App.init());

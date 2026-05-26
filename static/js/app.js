    const emojiOptions = [
      '🔗', '🌐', '🏠', '⭐', '📌', '🔍', '💻', '🖥️', '⌨️', '🧑‍💻',
      '🚀', '🛰️', '⚙️', '🛠️', '🧰', '🔧', '📊', '📈', '📉', '🧾',
      '🧠', '📚', '📖', '📝', '📄', '🗂️', '🗄️', '💾', '🧮', '🧪',
      '🎨', '🖌️', '🧩', '🎯', '✅', '📋', '🗓️', '⏱️', '🔔', '📬',
      '🔐', '🛡️', '🔑', '☁️', '📦', '🧭', '💬', '🤖', '⚡', '🔥'
    ];

    const glowOptions = [
      { name: '星蓝', value: 'rgba(96,165,250,.45)' },
      { name: '青 cyan', value: 'rgba(34,211,238,.45)' },
      { name: '紫光', value: 'rgba(168,85,247,.45)' },
      { name: '玫红', value: 'rgba(244,114,182,.45)' },
      { name: '薄荷', value: 'rgba(52,211,153,.45)' },
      { name: '琥珀', value: 'rgba(251,191,36,.45)' },
      { name: '靛蓝', value: 'rgba(129,140,248,.45)' },
      { name: '蓝绿', value: 'rgba(45,212,191,.45)' }
    ];

    const themeOptions = [
      { name: '深空', value: 'dark', swatch: 'oklch(85% .11 205)' },
      { name: '晨光', value: 'morning', swatch: 'oklch(68% .12 78)' },
      { name: '森屿', value: 'forest', swatch: 'oklch(82% .13 132)' },
      { name: '梅雾', value: 'plum', swatch: 'oklch(68% .14 325)' }
    ];
    const themeStorageKey = 'navigation.theme.override';

    const state = {
      sites: [],
      categories: [],
      category: '全部',
      query: '',
      user: null,
      settings: null,
      theme: themeOptions[0].value
    };

    const els = {
      tabs: document.querySelector('#categoryTabs'),
      grid: document.querySelector('#siteGrid'),
      search: document.querySelector('#searchInput'),
      siteCount: document.querySelector('#siteCount'),
      categoryCount: document.querySelector('#categoryCount'),
      coverage: document.querySelector('#coverage'),
      addBtn: document.querySelector('#addSiteBtn'),
      manageCategoriesBtn: document.querySelector('#manageCategoriesBtn'),
      dialog: document.querySelector('#siteDialog'),
      closeBtn: document.querySelector('#closeDialogBtn'),
      cancelBtn: document.querySelector('#cancelBtn'),
      form: document.querySelector('#siteForm'),
      formError: document.querySelector('#formError'),
      dialogTitle: document.querySelector('#dialogTitle'),
      categoryInput: document.querySelector('#category'),
      categoryMenu: document.querySelector('#categoryMenu'),
      categoryToggle: document.querySelector('#categoryToggle'),
      iconInput: document.querySelector('#icon'),
      emojiSelectBtn: document.querySelector('#emojiSelectBtn'),
      emojiPreview: document.querySelector('#emojiPreview'),
      emojiDialog: document.querySelector('#emojiDialog'),
      closeEmojiDialogBtn: document.querySelector('#closeEmojiDialogBtn'),
      emojiGrid: document.querySelector('#emojiGrid'),
      glowInput: document.querySelector('#glow'),
      glowPicker: document.querySelector('#glowPicker'),
      categoryDialog: document.querySelector('#categoryDialog'),
      closeCategoryDialogBtn: document.querySelector('#closeCategoryDialogBtn'),
      categoryList: document.querySelector('#categoryList'),
      authScreen: document.querySelector('#authScreen'),
      loginForm: document.querySelector('#loginForm'),
      loginError: document.querySelector('#loginError'),
      closeLoginBtn: document.querySelector('#closeLoginBtn'),
      currentUsername: document.querySelector('#currentUsername'),
      userMenu: document.querySelector('#userMenu'),
      userMenuBtn: document.querySelector('#userMenuBtn'),
      userDropdown: document.querySelector('#userDropdown'),
      logoutBtn: document.querySelector('#logoutBtn'),
      openAccountBtn: document.querySelector('#openAccountBtn'),
      accountDialog: document.querySelector('#accountDialog'),
      closeAccountDialogBtn: document.querySelector('#closeAccountDialogBtn'),
      accountForm: document.querySelector('#accountForm'),
      accountError: document.querySelector('#accountError'),
      openSettingsBtn: document.querySelector('#openSettingsBtn'),
      settingsDialog: document.querySelector('#settingsDialog'),
      closeSettingsDialogBtn: document.querySelector('#closeSettingsDialogBtn'),
      settingsForm: document.querySelector('#settingsForm'),
      settingsError: document.querySelector('#settingsError'),
      defaultThemeInput: document.querySelector('#defaultThemeInput'),
      heroBadge: document.querySelector('#heroBadge'),
      heroTitle: document.querySelector('#heroTitle'),
      heroSubtitle: document.querySelector('#heroSubtitle'),
      themeSwitcher: document.querySelector('#themeSwitcher'),
      themeToggleBtn: document.querySelector('#themeToggleBtn'),
      themeToggleText: document.querySelector('#themeToggleText'),
      themeMenu: document.querySelector('#themeMenu')
    };

    function currentTheme() {
      return themeOptions.find(theme => theme.value === state.theme) || themeOptions[0];
    }

    function hasTheme(theme) {
      return themeOptions.some(option => option.value === theme);
    }

    function normalizedTheme(theme) {
      return hasTheme(theme) ? theme : themeOptions[0].value;
    }

    function storedTheme() {
      try {
        const value = localStorage.getItem(themeStorageKey);
        return hasTheme(value) ? value : null;
      } catch (error) {
        return null;
      }
    }

    function applyTheme(theme, persist = false) {
      const nextTheme = normalizedTheme(theme);
      state.theme = nextTheme;
      document.body.dataset.theme = nextTheme;
      if (els.themeToggleText) els.themeToggleText.textContent = `主题：${currentTheme().name}`;
      renderThemeMenu();
      if (!persist) return;

      try {
        localStorage.setItem(themeStorageKey, nextTheme);
      } catch (error) {
        // localStorage 不可用时只保留当前会话的主题。
      }
    }

    function renderThemeMenu() {
      els.themeMenu.innerHTML = '';
      themeOptions.forEach(theme => {
        const button = document.createElement('button');
        button.className = `theme-option${theme.value === state.theme ? ' active' : ''}`;
        button.type = 'button';
        button.style.setProperty('--theme-swatch', theme.swatch);
        button.innerHTML = '<span class="theme-option-label"><span class="theme-swatch"></span><span></span></span><span class="theme-check">✓</span>';
        button.querySelector('.theme-option-label span:last-child').textContent = theme.name;
        button.addEventListener('click', async () => {
          closeThemeMenu();
          await saveTheme(theme.value);
        });
        els.themeMenu.appendChild(button);
      });
    }

    function toggleThemeMenu() {
      const open = !els.themeMenu.classList.contains('open');
      els.themeMenu.classList.toggle('open', open);
      els.themeToggleBtn.setAttribute('aria-expanded', String(open));
    }

    function closeThemeMenu() {
      els.themeMenu.classList.remove('open');
      els.themeToggleBtn.setAttribute('aria-expanded', 'false');
    }

    async function requestJSON(url, options = {}) {
      const { authPrompt = true, ...fetchOptions } = options;
      const response = await fetch(url, {
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        ...fetchOptions
      });
      if (response.status === 204) return null;
      const data = await response.json();
      if (response.status === 401 && authPrompt) showLogin();
      if (!response.ok) throw new Error(data.error || '请求失败');
      return data;
    }

    function updateAuthUI() {
      const authenticated = Boolean(state.user);
      els.currentUsername.textContent = authenticated ? state.user.username : '登录';
      els.addBtn.hidden = !authenticated;
      els.manageCategoriesBtn.hidden = !authenticated;
      els.userDropdown.classList.remove('open');
      renderSites(state.sites);
    }

    function setAnonymous() {
      state.user = null;
      document.querySelector('#accountUsername').value = '';
      els.authScreen.classList.add('hidden');
      updateAuthUI();
    }

    function hideLogin(user) {
      state.user = user;
      document.querySelector('#accountUsername').value = user.username;
      els.authScreen.classList.add('hidden');
      updateAuthUI();
    }

    function showLogin() {
      state.user = null;
      updateAuthUI();
      els.authScreen.classList.remove('hidden');
      document.querySelector('#loginPassword').value = '';
      document.querySelector('#loginUsername').focus();
    }

    async function bootstrap() {
      try {
        const user = await requestJSON('/api/session', { authPrompt: false });
        hideLogin(user);
      } catch (error) {
        setAnonymous();
      }
      await loadAll();
    }

    async function login(event) {
      event.preventDefault();
      els.loginError.textContent = '';
      try {
        const user = await requestJSON('/api/login', {
          method: 'POST',
          body: JSON.stringify({
            username: document.querySelector('#loginUsername').value,
            password: document.querySelector('#loginPassword').value
          })
        });
        hideLogin(user);
        await loadAll();
      } catch (error) {
        els.loginError.textContent = error.message;
      }
    }

    function siteQueryURL() {
      const params = new URLSearchParams();
      if (state.category && state.category !== '全部') params.set('category', state.category);
      if (state.query) params.set('q', state.query);
      const suffix = params.toString();
      return suffix ? `/api/sites?${suffix}` : '/api/sites';
    }

    async function loadAll() {
      const [categories, sites, stats, settings] = await Promise.all([
        requestJSON('/api/categories'),
        requestJSON(siteQueryURL()),
        requestJSON('/api/stats'),
        requestJSON('/api/settings')
      ]);
      state.sites = sites;
      state.categories = categories;
      state.settings = settings;
      applySettings(settings);
      renderTabs(categories);
      renderCategoryOptions(categories);
      renderSites(sites);
      renderStats(stats);
    }

    function applySettings(settings) {
      const defaultTheme = normalizedTheme(settings.theme);
      document.title = settings.siteTitle;
      els.heroBadge.textContent = settings.badge;
      els.heroTitle.textContent = settings.heroTitle;
      els.heroSubtitle.textContent = settings.subtitle;
      applyTheme(storedTheme() || defaultTheme);
      document.querySelector('#siteTitleInput').value = settings.siteTitle;
      document.querySelector('#badgeInput').value = settings.badge;
      document.querySelector('#heroTitleInput').value = settings.heroTitle;
      document.querySelector('#subtitleInput').value = settings.subtitle;
      renderDefaultThemeOptions(defaultTheme);
    }

    function renderDefaultThemeOptions(selectedTheme) {
      const activeTheme = normalizedTheme(selectedTheme);
      els.defaultThemeInput.innerHTML = '';
      themeOptions.forEach(theme => {
        const option = document.createElement('option');
        option.value = theme.value;
        option.textContent = theme.name;
        option.selected = theme.value === activeTheme;
        els.defaultThemeInput.appendChild(option);
      });
    }

    function saveTheme(theme) {
      applyTheme(theme, true);
    }

    async function loadSitesOnly() {
      state.sites = await requestJSON(siteQueryURL());
      renderSites(state.sites);
    }

    function renderStats(stats) {
      els.siteCount.textContent = stats.siteCount;
      els.categoryCount.textContent = stats.categoryCount;
      els.coverage.textContent = stats.coverage;
    }

    function renderTabs(categories) {
      els.tabs.innerHTML = '';
      categories.forEach(category => {
        const button = document.createElement('button');
        button.className = `tab${category === state.category ? ' active' : ''}`;
        button.type = 'button';
        button.textContent = category;
        button.addEventListener('click', async () => {
          state.category = category;
          renderTabs(categories);
          await loadSitesOnly();
        });
        els.tabs.appendChild(button);
      });
    }

    function renderCategoryOptions(categories, filter = '') {
      const keyword = filter.trim().toLowerCase();
      const options = categories
        .filter(category => category !== '全部')
        .filter(category => !keyword || category.toLowerCase().includes(keyword));

      els.categoryMenu.innerHTML = '';
      if (!options.length) {
        const empty = document.createElement('div');
        empty.className = 'category-empty';
        empty.textContent = filter ? '没有匹配分类，保存后会创建为新分类。' : '暂无可选分类，可以直接输入新分类。';
        els.categoryMenu.appendChild(empty);
        return;
      }

      options.forEach(category => {
        const option = document.createElement('button');
        option.className = `category-option${category === els.categoryInput.value ? ' active' : ''}`;
        option.type = 'button';
        option.innerHTML = '<span></span><small>已有</small>';
        option.querySelector('span').textContent = category;
        option.addEventListener('mousedown', event => event.preventDefault());
        option.addEventListener('click', () => {
          els.categoryInput.value = category;
          hideCategoryMenu();
          els.categoryInput.focus();
        });
        els.categoryMenu.appendChild(option);
      });
    }

    function showCategoryMenu() {
      renderCategoryOptions(state.categories, els.categoryInput.value);
      els.categoryMenu.classList.add('open');
    }

    function hideCategoryMenu() {
      els.categoryMenu.classList.remove('open');
    }

    function renderEmojiOptions(selectedIcon = emojiOptions[0]) {
      const matched = emojiOptions.includes(selectedIcon);
      const activeIcon = matched ? selectedIcon : emojiOptions[0];
      els.iconInput.value = activeIcon;
      els.emojiPreview.textContent = activeIcon;
      els.emojiGrid.innerHTML = '';

      emojiOptions.forEach(icon => {
        const button = document.createElement('button');
        button.className = `emoji-option${icon === activeIcon ? ' active' : ''}`;
        button.type = 'button';
        button.textContent = icon;
        button.title = icon;
        button.addEventListener('click', () => {
          renderEmojiOptions(icon);
          closeEmojiDialog();
        });
        els.emojiGrid.appendChild(button);
      });
    }

    function openEmojiDialog() {
      renderEmojiOptions(els.iconInput.value || emojiOptions[0]);
      els.emojiDialog.classList.add('open');
      els.emojiDialog.setAttribute('aria-hidden', 'false');
    }

    function closeEmojiDialog() {
      els.emojiDialog.classList.remove('open');
      els.emojiDialog.setAttribute('aria-hidden', 'true');
    }

    function renderGlowOptions(selectedGlow = glowOptions[0].value) {
      const matched = glowOptions.some(option => option.value === selectedGlow);
      const activeGlow = matched ? selectedGlow : glowOptions[0].value;
      els.glowInput.value = activeGlow;
      els.glowPicker.innerHTML = '';

      glowOptions.forEach(option => {
        const button = document.createElement('button');
        button.className = `glow-option${option.value === activeGlow ? ' active' : ''}`;
        button.type = 'button';
        button.textContent = option.name;
        button.style.setProperty('--swatch', option.value);
        button.addEventListener('click', () => renderGlowOptions(option.value));
        els.glowPicker.appendChild(button);
      });
    }

    function renderSites(sites) {
      els.grid.innerHTML = '';
      if (!sites.length) {
        const empty = document.createElement('div');
        empty.className = 'empty';
        empty.textContent = '没有找到匹配的站点';
        els.grid.appendChild(empty);
        return;
      }

      sites.forEach(site => {
        const card = document.createElement('a');
        card.className = 'card';
        card.href = site.url;
        card.target = '_blank';
        card.rel = 'noopener noreferrer';
        card.style.setProperty('--glow', site.glow || 'rgba(96,165,250,.45)');
        card.innerHTML = `
          ${state.user ? `<div class="card-actions">
            <button class="icon-btn" type="button" data-action="edit" title="编辑">✎</button>
            <button class="icon-btn" type="button" data-action="delete" title="删除">×</button>
          </div>` : ''}
          <div class="icon"></div>
          <h3></h3>
          <p></p>
          <span class="arrow">→</span>
        `;
        card.querySelector('.icon').textContent = site.icon || '🔗';
        card.querySelector('h3').textContent = site.name;
        card.querySelector('p').textContent = site.description || site.category;
        if (state.user) {
          card.querySelector('[data-action="edit"]').addEventListener('click', event => {
            event.preventDefault();
            openDialog(site);
          });
          card.querySelector('[data-action="delete"]').addEventListener('click', async event => {
            event.preventDefault();
            await deleteSite(site);
          });
        }
        els.grid.appendChild(card);
      });
    }

    function openDialog(site = null) {
      if (!state.user) {
        showLogin();
        return;
      }
      els.form.reset();
      els.formError.textContent = '';
      els.dialogTitle.textContent = site ? '编辑站点' : '新增站点';
      document.querySelector('#siteId').value = site?.id || '';
      document.querySelector('#name').value = site?.name || '';
      els.categoryInput.value = site?.category || '';
      document.querySelector('#url').value = site?.url || '';
      renderEmojiOptions(site?.icon || emojiOptions[0]);
      document.querySelector('#sort').value = site?.sort || '';
      renderGlowOptions(site?.glow || glowOptions[0].value);
      document.querySelector('#description').value = site?.description || '';
      renderCategoryOptions(state.categories, els.categoryInput.value);
      hideCategoryMenu();
      els.dialog.classList.add('open');
      els.dialog.setAttribute('aria-hidden', 'false');
      document.querySelector('#name').focus();
    }

    function closeDialog() {
      els.dialog.classList.remove('open');
      els.dialog.setAttribute('aria-hidden', 'true');
    }

    async function openCategoryDialog() {
      if (!state.user) {
        showLogin();
        return;
      }
      els.categoryList.innerHTML = '<div class="empty">正在加载分类...</div>';
      els.categoryDialog.classList.add('open');
      els.categoryDialog.setAttribute('aria-hidden', 'false');

      try {
        const categories = await requestJSON('/api/category-stats');
        renderCategoryManager(categories);
      } catch (error) {
        els.categoryList.innerHTML = `<div class="empty">${error.message}</div>`;
      }
    }

    function closeCategoryDialog() {
      els.categoryDialog.classList.remove('open');
      els.categoryDialog.setAttribute('aria-hidden', 'true');
    }

    function openAccountDialog() {
      if (!state.user) {
        showLogin();
        return;
      }
      els.accountError.textContent = '';
      document.querySelector('#accountUsername').value = state.user?.username || '';
      document.querySelector('#currentPassword').value = '';
      document.querySelector('#newPassword').value = '';
      els.accountDialog.classList.add('open');
      els.accountDialog.setAttribute('aria-hidden', 'false');
      els.userDropdown.classList.remove('open');
    }

    function closeAccountDialog() {
      els.accountDialog.classList.remove('open');
      els.accountDialog.setAttribute('aria-hidden', 'true');
    }

    function openSettingsDialog() {
      if (!state.user) {
        showLogin();
        return;
      }
      els.settingsError.textContent = '';
      if (state.settings) applySettings(state.settings);
      els.settingsDialog.classList.add('open');
      els.settingsDialog.setAttribute('aria-hidden', 'false');
      els.userDropdown.classList.remove('open');
    }

    function closeSettingsDialog() {
      els.settingsDialog.classList.remove('open');
      els.settingsDialog.setAttribute('aria-hidden', 'true');
    }

    function renderCategoryManager(categories) {
      els.categoryList.innerHTML = '';
      if (!categories.length) {
        els.categoryList.innerHTML = '<div class="empty">暂无分类</div>';
        return;
      }

      categories.forEach(category => {
        const row = document.createElement('div');
        row.className = 'category-row';
        row.innerHTML = `
          <div class="category-info">
            <b></b>
            <small></small>
          </div>
          <div class="category-actions">
            <button class="ghost-btn" type="button" data-action="rename">编辑</button>
            <button class="danger-btn" type="button" data-action="delete">删除</button>
          </div>
        `;
        row.querySelector('b').textContent = category.name;
        row.querySelector('small').textContent = `${category.count} 个站点`;
        row.querySelector('[data-action="rename"]').addEventListener('click', () => renameCategory(category));
        row.querySelector('[data-action="delete"]').addEventListener('click', () => deleteCategory(category));
        els.categoryList.appendChild(row);
      });
    }

    async function saveSite(event) {
      event.preventDefault();
      if (!state.user) {
        showLogin();
        return;
      }
      els.formError.textContent = '';
      const id = document.querySelector('#siteId').value;
      const payload = {
        name: document.querySelector('#name').value,
        category: els.categoryInput.value,
        url: document.querySelector('#url').value,
        icon: els.iconInput.value,
        sort: Number(document.querySelector('#sort').value || 0),
        glow: els.glowInput.value,
        description: document.querySelector('#description').value
      };

      try {
        await requestJSON(id ? `/api/sites/${id}` : '/api/sites', {
          method: id ? 'PUT' : 'POST',
          body: JSON.stringify(payload)
        });
        closeDialog();
        await loadAll();
      } catch (error) {
        els.formError.textContent = error.message;
      }
    }

    async function deleteSite(site) {
      if (!state.user) {
        showLogin();
        return;
      }
      if (!confirm(`确定删除「${site.name}」吗？`)) return;
      await requestJSON(`/api/sites/${site.id}`, { method: 'DELETE' });
      await loadAll();
    }

    async function deleteCategory(category) {
      if (!state.user) {
        showLogin();
        return;
      }
      const message = `确定删除「${category.name}」分类吗？该分类下的 ${category.count} 个站点会保留，但分类会被清空。`;
      if (!confirm(message)) return;

      try {
        await requestJSON(`/api/categories/${encodeURIComponent(category.name)}`, { method: 'DELETE' });
        if (state.category === category.name) state.category = '全部';
        await loadAll();
        await openCategoryDialog();
      } catch (error) {
        alert(error.message);
      }
    }

    async function renameCategory(category) {
      if (!state.user) {
        showLogin();
        return;
      }

      const nextName = prompt('请输入新的分类名称', category.name);
      if (nextName === null) return;

      const normalizedName = nextName.trim();
      if (!normalizedName || normalizedName === category.name) return;

      try {
        const result = await requestJSON(`/api/categories/${encodeURIComponent(category.name)}`, {
          method: 'PUT',
          body: JSON.stringify({ name: normalizedName })
        });
        if (state.category === category.name) state.category = result.name;
        await loadAll();
        await openCategoryDialog();
      } catch (error) {
        alert(error.message);
      }
    }

    async function saveAccount(event) {
      event.preventDefault();
      els.accountError.textContent = '';
      try {
        const user = await requestJSON('/api/account', {
          method: 'PUT',
          body: JSON.stringify({
            username: document.querySelector('#accountUsername').value,
            currentPassword: document.querySelector('#currentPassword').value,
            newPassword: document.querySelector('#newPassword').value
          })
        });
        hideLogin(user);
        closeAccountDialog();
      } catch (error) {
        els.accountError.textContent = error.message;
      }
    }

    async function saveSettings(event) {
      event.preventDefault();
      els.settingsError.textContent = '';
      try {
        const settings = await requestJSON('/api/settings', {
          method: 'PUT',
          body: JSON.stringify({
            siteTitle: document.querySelector('#siteTitleInput').value,
            badge: document.querySelector('#badgeInput').value,
            heroTitle: document.querySelector('#heroTitleInput').value,
            subtitle: document.querySelector('#subtitleInput').value,
            theme: els.defaultThemeInput.value
          })
        });
        state.settings = settings;
        applySettings(settings);
        closeSettingsDialog();
      } catch (error) {
        els.settingsError.textContent = error.message;
      }
    }

    async function logout() {
      await requestJSON('/api/logout', { method: 'POST' }).catch(() => null);
      els.userDropdown.classList.remove('open');
      setAnonymous();
    }

    function debounce(fn, delay = 250) {
      let timer = null;
      return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => fn(...args), delay);
      };
    }

    els.search.addEventListener('input', debounce(async event => {
      state.query = event.target.value.trim();
      await loadSitesOnly();
    }));
    els.categoryInput.addEventListener('focus', showCategoryMenu);
    els.categoryInput.addEventListener('input', () => showCategoryMenu());
    els.categoryToggle.addEventListener('click', () => {
      if (els.categoryMenu.classList.contains('open')) {
        hideCategoryMenu();
      } else {
        els.categoryInput.focus();
        showCategoryMenu();
      }
    });
    els.addBtn.addEventListener('click', () => openDialog());
    els.emojiSelectBtn.addEventListener('click', openEmojiDialog);
    els.manageCategoriesBtn.addEventListener('click', openCategoryDialog);
    els.loginForm.addEventListener('submit', login);
    els.closeLoginBtn.addEventListener('click', setAnonymous);
    els.userMenuBtn.addEventListener('click', () => {
      if (!state.user) {
        showLogin();
        return;
      }
      els.userDropdown.classList.toggle('open');
    });
    els.openAccountBtn.addEventListener('click', openAccountDialog);
    els.openSettingsBtn.addEventListener('click', openSettingsDialog);
    els.themeToggleBtn.addEventListener('click', toggleThemeMenu);
    els.logoutBtn.addEventListener('click', logout);
    els.closeAccountDialogBtn.addEventListener('click', closeAccountDialog);
    els.closeSettingsDialogBtn.addEventListener('click', closeSettingsDialog);
    els.accountForm.addEventListener('submit', saveAccount);
    els.settingsForm.addEventListener('submit', saveSettings);
    els.closeBtn.addEventListener('click', closeDialog);
    els.cancelBtn.addEventListener('click', closeDialog);
    els.closeEmojiDialogBtn.addEventListener('click', closeEmojiDialog);
    els.closeCategoryDialogBtn.addEventListener('click', closeCategoryDialog);
    els.dialog.addEventListener('click', event => {
      if (event.target === els.dialog) closeDialog();
    });
    els.emojiDialog.addEventListener('click', event => {
      if (event.target === els.emojiDialog) closeEmojiDialog();
    });
    els.categoryDialog.addEventListener('click', event => {
      if (event.target === els.categoryDialog) closeCategoryDialog();
    });
    els.accountDialog.addEventListener('click', event => {
      if (event.target === els.accountDialog) closeAccountDialog();
    });
    els.settingsDialog.addEventListener('click', event => {
      if (event.target === els.settingsDialog) closeSettingsDialog();
    });
    document.addEventListener('click', event => {
      if (!event.target.closest('#categoryPicker')) hideCategoryMenu();
      if (!event.target.closest('#userMenu')) els.userDropdown.classList.remove('open');
      if (!event.target.closest('#themeSwitcher')) closeThemeMenu();
    });
    els.form.addEventListener('submit', saveSite);

    applyTheme(storedTheme() || themeOptions[0].value);
    bootstrap().catch(error => {
      els.grid.innerHTML = `<div class="empty">${error.message}</div>`;
    });

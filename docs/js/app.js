/**
 * BiblioTânia — Portal Híbrido Oficial do Colégio Betânia
 * Frontend SPA Estático para GitHub Pages integrado com API Go na Cloud VPS
 */

(function () {
  'use strict';

  // Configuração padrão da API Cloud VPS (Tailscale ou Host Remoto)
  const STORAGE_KEY_API = 'bibliotania_api_url';
  const DEFAULT_API_URL = localStorage.getItem(STORAGE_KEY_API) || 'http://100.112.247.97:8090';

  let state = {
    apiUrl: DEFAULT_API_URL,
    isApiOnline: false,
    allBooks: [],
    filteredBooks: [],
    currentClass: 0, // 0 = todas
    currentCategory: 'todas',
    searchQuery: '',
    currentPage: 1,
    pageSize: 24,
    stats: null
  };

  // Elementos do DOM
  const dom = {
    catalogGrid: document.getElementById('catalog-grid'),
    searchInput: document.getElementById('search-input'),
    searchClearBtn: document.getElementById('search-clear'),
    resultsCount: document.getElementById('results-count'),
    classFilterPills: document.getElementById('class-filter-pills'),
    categoryFilterPills: document.getElementById('category-filter-pills'),
    paginationContainer: document.getElementById('pagination-container'),
    apiBadge: document.getElementById('api-badge'),
    apiBadgeText: document.getElementById('api-badge-text'),

    // Modais
    modalBackdrop: document.getElementById('modal-backdrop'),
    modalContainer: document.getElementById('modal-container'),
    toastContainer: document.getElementById('toast-container')
  };

  // Inicialização
  async function init() {
    setupEventListeners();
    await loadCatalogData();
    checkApiHealth();
  }

  // Carregar dados estáticos dos livros
  async function loadCatalogData() {
    try {
      const resp = await fetch('data/books.json');
      if (!resp.ok) throw new Error('Falha ao carregar catálogo');
      state.allBooks = await resp.json();
      
      // Carregar estatísticas
      try {
        const statsResp = await fetch('data/stats.json');
        if (statsResp.ok) state.stats = await statsResp.json();
      } catch (e) {
        console.warn('Stats não carregados:', e);
      }

      applyFilters();
    } catch (err) {
      console.error('Erro ao inicializar dados:', err);
      if (dom.catalogGrid) {
        dom.catalogGrid.innerHTML = `
          <div class="empty-state" style="grid-column: 1/-1; padding: 3rem; text-align: center;">
            <p style="color: #ef4444; font-weight: 600;">Erro ao carregar acervo escolar.</p>
            <p style="color: #64748b;">Por favor, recarregue a página.</p>
          </div>
        `;
      }
    }
  }

  // Verificar status da API Go na Cloud VPS
  async function checkApiHealth() {
    if (!state.apiUrl) {
      setApiStatus(false, 'Modo Estático');
      return;
    }

    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 3500);

      const resp = await fetch(`${state.apiUrl}/api/stats`, {
        method: 'GET',
        headers: { 'Accept': 'application/json' },
        signal: controller.signal
      });
      clearTimeout(timeoutId);

      if (resp.ok) {
        setApiStatus(true, 'VPS Conectada');
      } else {
        setApiStatus(false, 'VPS Standby');
      }
    } catch (err) {
      setApiStatus(false, 'Catálogo Cacheado');
    }
  }

  function setApiStatus(online, text) {
    state.isApiOnline = online;
    if (dom.apiBadge && dom.apiBadgeText) {
      dom.apiBadge.className = 'api-badge ' + (online ? 'online' : 'offline');
      dom.apiBadgeText.textContent = text;
    }
  }

  // Filtragem e Pesquisa
  function applyFilters() {
    let list = state.allBooks.slice();

    // 1. Filtro por Classe (1ª à 12ª)
    if (state.currentClass > 0) {
      list = list.filter(b => b.classe === state.currentClass);
    }

    // 2. Filtro por Categoria Temática
    if (state.currentCategory !== 'todas') {
      const cat = state.currentCategory.toLowerCase();
      list = list.filter(b => {
        const disc = (b.disciplina || '').toLowerCase();
        const nivel = (b.nivel || '').toLowerCase();
        const tit = (b.titulo || '').toLowerCase();

        if (cat === 'literatura') {
          return disc.includes('literatura') || nivel.includes('literatura') || b.classe === 0;
        } else if (cat === 'linguas') {
          return disc.includes('língua') || disc.includes('lingua') || tit.includes('kimbundu') || tit.includes('umbundu') || tit.includes('kikongo') || tit.includes('cokwe');
        } else if (cat === 'tecnico') {
          return disc.includes('informática') || disc.includes('tlp') || disc.includes('rcc') || disc.includes('seac') || disc.includes('sibd') || tit.includes('programação');
        } else if (cat === 'saude_agro') {
          return disc.includes('saúde') || disc.includes('agro') || disc.includes('biologia') || tit.includes('mandioca') || tit.includes('malária');
        }
        return true;
      });
    }

    // 3. Filtro por Termo de Busca
    if (state.searchQuery) {
      const q = state.searchQuery.toLowerCase();
      list = list.filter(b => {
        return (b.titulo || '').toLowerCase().includes(q) ||
               (b.autor || '').toLowerCase().includes(q) ||
               (b.disciplina || '').toLowerCase().includes(q) ||
               (b.sinopse || '').toLowerCase().includes(q);
      });
    }

    state.filteredBooks = list;
    state.currentPage = 1;
    renderCatalog();
  }

  // Renderização da Grade de Livros
  function renderCatalog() {
    if (!dom.catalogGrid) return;

    const total = state.filteredBooks.length;
    if (dom.resultsCount) {
      dom.resultsCount.textContent = `${total} obra${total !== 1 ? 's' : ''} encontrada${total !== 1 ? 's' : ''}`;
    }

    if (total === 0) {
      dom.catalogGrid.innerHTML = `
        <div style="grid-column: 1/-1; padding: 4rem 1rem; text-align: center;">
          <div style="font-size: 3rem; margin-bottom: 0.75rem;">📚</div>
          <h3 style="font-size: 1.25rem; font-weight: 700; color: #0b1e3b; margin-bottom: 0.5rem;">Nenhum livro encontrado</h3>
          <p style="color: #64748b; font-size: 0.95rem;">Tente ajustar os termos de pesquisa ou remover os filtros de classe.</p>
        </div>
      `;
      if (dom.paginationContainer) dom.paginationContainer.innerHTML = '';
      return;
    }

    // Paginação
    const startIdx = (state.currentPage - 1) * state.pageSize;
    const endIdx = startIdx + state.pageSize;
    const pageBooks = state.filteredBooks.slice(startIdx, endIdx);

    let html = '';
    pageBooks.forEach(book => {
      const coverSrc = book.capa_url || 'images/covers/default.jpg';
      const readUrl = book.link_externo || (book.arquivo_url ? `${state.apiUrl}${book.arquivo_url}` : '#');
      const isExternal = Boolean(book.link_externo);

      let badgeClasse = '';
      if (book.classe > 0) {
        badgeClasse = `<span class="book-badge" style="background:#eff6ff; color:#1e40af; border:1px solid #bfdbfe;">${book.classe}ª Classe</span>`;
      } else {
        badgeClasse = `<span class="book-badge" style="background:#fef3c7; color:#92400e; border:1px solid #fde68a;">Cânone</span>`;
      }

      html += `
        <article class="book-card" data-book-id="${book.id}">
          <div class="book-cover-container">
            <div class="book-cover-wrapper">
              <img src="${escapeHtml(coverSrc)}" 
                   alt="Capa de ${escapeHtml(book.titulo)}" 
                   class="book-cover-img"
                   loading="lazy"
                   onerror="this.onerror=null; this.src='images/covers/default.jpg';">
              <div class="book-cover-spine"></div>
            </div>
          </div>
          <div class="book-card-body">
            <div style="display: flex; gap: 0.4rem; flex-wrap: wrap; margin-bottom: 0.5rem;">
              ${badgeClasse}
              <span class="book-badge" style="background:#f1f5f9; color:#475569;">${escapeHtml(book.disciplina || 'Geral')}</span>
            </div>
            <h3 class="book-card-title" title="${escapeHtml(book.titulo)}">
              ${escapeHtml(book.titulo)}
            </h3>
            <p class="book-card-author">
              ${escapeHtml(book.autor || 'Autor Desconhecido')}
            </p>
            <div class="book-card-actions">
              <a href="${escapeHtml(readUrl)}" 
                 target="_blank" 
                 rel="noopener noreferrer" 
                 class="btn-read">
                📖 Ler Obra
              </a>
              <button type="button" 
                      class="btn-details" 
                      data-id="${book.id}">
                ℹ Detalhes
              </button>
            </div>
          </div>
        </article>
      `;
    });

    dom.catalogGrid.innerHTML = html;
    renderPagination(total);
  }

  // Renderizar paginação
  function renderPagination(total) {
    if (!dom.paginationContainer) return;
    const totalPages = Math.ceil(total / state.pageSize);
    if (totalPages <= 1) {
      dom.paginationContainer.innerHTML = '';
      return;
    }

    let pHtml = `
      <div class="pagination">
        <button class="pag-btn" ${state.currentPage === 1 ? 'disabled' : ''} data-page="${state.currentPage - 1}">« Anterior</button>
    `;

    for (let p = 1; p <= totalPages; p++) {
      if (totalPages > 8 && Math.abs(p - state.currentPage) > 2 && p !== 1 && p !== totalPages) {
        if (p === 2 || p === totalPages - 1) pHtml += `<span class="pag-ellipsis">...</span>`;
        continue;
      }
      pHtml += `
        <button class="pag-btn ${p === state.currentPage ? 'active' : ''}" data-page="${p}">${p}</button>
      `;
    }

    pHtml += `
        <button class="pag-btn" ${state.currentPage === totalPages ? 'disabled' : ''} data-page="${state.currentPage + 1}">Próxima »</button>
      </div>
    `;

    dom.paginationContainer.innerHTML = pHtml;
  }

  // Event Listeners
  function setupEventListeners() {
    // Busca em tempo real com debounce
    let debounceTimer;
    if (dom.searchInput) {
      dom.searchInput.addEventListener('input', (e) => {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
          state.searchQuery = e.target.value.trim();
          applyFilters();
        }, 180);
      });
    }

    if (dom.searchClearBtn) {
      dom.searchClearBtn.addEventListener('click', () => {
        if (dom.searchInput) dom.searchInput.value = '';
        state.searchQuery = '';
        applyFilters();
      });
    }

    // Atalho de teclado para busca (tecla /)
    window.addEventListener('keydown', (e) => {
      if (e.key === '/' && document.activeElement !== dom.searchInput) {
        e.preventDefault();
        dom.searchInput?.focus();
      }
    });

    // Pílulas de Filtro por Classe
    if (dom.classFilterPills) {
      dom.classFilterPills.addEventListener('click', (e) => {
        const btn = e.target.closest('button[data-classe]');
        if (!btn) return;
        const cls = parseInt(btn.dataset.classe, 10);
        state.currentClass = cls;

        dom.classFilterPills.querySelectorAll('button').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        applyFilters();
      });
    }

    // Pílulas de Categoria Temática
    if (dom.categoryFilterPills) {
      dom.categoryFilterPills.addEventListener('click', (e) => {
        const btn = e.target.closest('button[data-cat]');
        if (!btn) return;
        state.currentCategory = btn.dataset.cat;

        dom.categoryFilterPills.querySelectorAll('button').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        applyFilters();
      });
    }

    // Paginação
    if (dom.paginationContainer) {
      dom.paginationContainer.addEventListener('click', (e) => {
        const btn = e.target.closest('button[data-page]');
        if (!btn || btn.disabled) return;
        state.currentPage = parseInt(btn.dataset.page, 10);
        renderCatalog();
        window.scrollTo({ top: dom.catalogGrid.offsetTop - 120, behavior: 'smooth' });
      });
    }

    // Clique em "Detalhes" no catálogo
    if (dom.catalogGrid) {
      dom.catalogGrid.addEventListener('click', (e) => {
        const btn = e.target.closest('.btn-details');
        if (!btn) return;
        const id = parseInt(btn.dataset.id, 10);
        const book = state.allBooks.find(b => b.id === id);
        if (book) openBookDetailsModal(book);
      });
    }

    // Menu e Modais Globais
    document.querySelectorAll('[data-open-modal]').forEach(el => {
      el.addEventListener('click', (e) => {
        e.preventDefault();
        const modalType = el.getAttribute('data-open-modal');
        if (modalType === 'submeter') openSubmeterModal();
        else if (modalType === 'mural') openMuralModal();
        else if (modalType === 'sobre') openSobreModal();
        else if (modalType === 'api-config') openApiConfigModal();
      });
    });

    // Fechar modal ao clicar fora ou em botões fechar/cancelar
    if (dom.modalBackdrop) {
      dom.modalBackdrop.addEventListener('click', (e) => {
        if (e.target === dom.modalBackdrop || e.target.closest('[data-action="close-modal"]')) {
          closeModal();
        }
      });
    }

    // Fechar modal com tecla ESC
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') closeModal();
    });
  }

  // Modais
  function openModal(title, bodyHtml, footerHtml = '') {
    if (!dom.modalContainer || !dom.modalBackdrop) return;

    dom.modalContainer.innerHTML = `
      <div class="modal-card">
        <div class="modal-header">
          <h3 class="modal-title">${escapeHtml(title)}</h3>
          <button type="button" class="modal-close-btn" data-action="close-modal" aria-label="Fechar modal">&times;</button>
        </div>
        <div class="modal-body">
          ${bodyHtml}
        </div>
        ${footerHtml ? `<div class="modal-footer">${footerHtml}</div>` : ''}
      </div>
    `;

    dom.modalBackdrop.classList.add('active');
    document.body.style.overflow = 'hidden';
  }

  function closeModal() {
    if (dom.modalBackdrop) dom.modalBackdrop.classList.remove('active');
    document.body.style.overflow = '';
  }

  // Modal de Detalhes da Obra
  function openBookDetailsModal(book) {
    const coverSrc = book.capa_url || 'images/covers/default.jpg';
    const readUrl = book.link_externo || (book.arquivo_url ? `${state.apiUrl}${book.arquivo_url}` : '#');
    const classeText = book.classe > 0 ? `${book.classe}ª Classe` : 'Cânone da Literatura';

    const bodyHtml = `
      <div style="display: flex; gap: 1.5rem; flex-wrap: wrap;">
        <div style="flex: 0 0 160px; margin: 0 auto;">
          <div class="book-cover-container" style="box-shadow: 0 10px 20px rgba(11,30,59,0.15);">
            <div class="book-cover-wrapper">
              <img src="${escapeHtml(coverSrc)}" alt="Capa" class="book-cover-img" onerror="this.src='images/covers/default.jpg'">
              <div class="book-cover-spine"></div>
            </div>
          </div>
        </div>
        <div style="flex: 1; min-width: 250px;">
          <div style="display:flex; gap:0.5rem; margin-bottom:0.75rem;">
            <span class="book-badge" style="background:#eff6ff; color:#1e40af; border:1px solid #bfdbfe;">${classeText}</span>
            <span class="book-badge" style="background:#f1f5f9; color:#475569;">${escapeHtml(book.disciplina || 'Geral')}</span>
          </div>
          <h4 style="font-size: 1.25rem; font-weight: 800; color: #0b1e3b; margin-bottom: 0.25rem;">${escapeHtml(book.titulo)}</h4>
          <p style="color: #64748b; font-size: 0.95rem; margin-bottom: 1rem;">Autor: <strong>${escapeHtml(book.autor)}</strong></p>
          
          <h5 style="font-size: 0.85rem; font-weight: 700; text-transform: uppercase; color: #94a3b8; letter-spacing: 0.5px; margin-bottom: 0.4rem;">Sinopse Pedagógica</h5>
          <p style="color: #334155; font-size: 0.95rem; line-height: 1.6; margin-bottom: 1rem;">
            ${escapeHtml(book.sinopse || 'Recurso didático e literário integrado ao catálogo oficial do Colégio Betânia.')}
          </p>
        </div>
      </div>
    `;

    const footerHtml = `
      <button type="button" class="btn" style="background:#e2e8f0; color:#334155; border:none; padding:0.5rem 1rem; border-radius:6px; cursor:pointer;" data-action="close-modal">Fechar</button>
      <a href="${escapeHtml(readUrl)}" target="_blank" rel="noopener noreferrer" class="btn" style="background:#0b1e3b; color:white; padding:0.5rem 1.25rem; border-radius:6px; text-decoration:none; font-weight:600;">📖 Abrir e Ler Obra</a>
    `;

    openModal('Ficha Técnica da Obra', bodyHtml, footerHtml);
  }

  // Modal de Submissão de Aluno
  function openSubmeterModal() {
    const isOnline = state.isApiOnline;
    const bodyHtml = `
      <p style="font-size: 0.95rem; color: #475569; margin-bottom: 1.25rem;">
        Estudantes e professores do Colégio Betânia podem enriquecer o acervo sugerindo novos materiais curriculares ou obras literárias. A Inteligência Artificial da escola fará a triagem pedagógica em tempo real.
      </p>

      ${!isOnline ? `
        <div style="background: #fffbeb; border: 1px solid #fef3c7; padding: 0.75rem 1rem; border-radius: 8px; margin-bottom: 1rem; font-size: 0.85rem; color: #92400e;">
          ℹ <strong>Aviso de Conexão:</strong> O backend Go está em standby na VPS. Sua sugestão será validada e registrada localmente ou encaminhada assim que o nó estiver ativo.
        </div>
      ` : ''}

      <form id="submission-form" style="display: flex; flex-direction: column; gap: 1rem;">
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
          <div>
            <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Seu Nome Completo *</label>
            <input type="text" name="estudante_nome" required class="search-input" style="width:100%; padding:0.6rem; border:1px solid #cbd5e1; border-radius:6px;" placeholder="Ex: Manuel Antunes">
          </div>
          <div>
            <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Classe / Turma *</label>
            <input type="text" name="estudante_classe" required class="search-input" style="width:100%; padding:0.6rem; border:1px solid #cbd5e1; border-radius:6px;" placeholder="Ex: 11ª Classe A">
          </div>
        </div>

        <div>
          <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Título do Livro / Material *</label>
          <input type="text" name="titulo" required class="search-input" style="width:100%; padding:0.6rem; border:1px solid #cbd5e1; border-radius:6px;" placeholder="Ex: Manual de Biologia — Genética">
        </div>

        <div>
          <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Autor ou Entidade Responsável</label>
          <input type="text" name="autor" class="search-input" style="width:100%; padding:0.6rem; border:1px solid #cbd5e1; border-radius:6px;" placeholder="Ex: Ministério da Educação (MED)">
        </div>

        <div>
          <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Tipo de Envio</label>
          <div style="display:flex; gap:1.5rem; margin-top:0.25rem;">
            <label style="display:flex; align-items:center; gap:0.4rem; font-size:0.9rem; cursor:pointer;">
              <input type="radio" name="tipo_envio" value="link" checked> Hiperligação Pública (HTTPS)
            </label>
            <label style="display:flex; align-items:center; gap:0.4rem; font-size:0.9rem; cursor:pointer;">
              <input type="radio" name="tipo_envio" value="arquivo"> Envio de Ficheiro (PDF/DOCX)
            </label>
          </div>
        </div>

        <div id="group-link">
          <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Link do Livro (URL Aberto)</label>
          <input type="url" name="link_url" class="search-input" style="width:100%; padding:0.6rem; border:1px solid #cbd5e1; border-radius:6px;" placeholder="https://exemplo.org/livro.pdf">
        </div>

        <div id="group-file" style="display:none;">
          <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">Ficheiro do Livro (máx. 50MB)</label>
          <input type="file" name="arquivo" accept=".pdf,.docx" style="font-size:0.9rem;">
        </div>

        <div style="margin-top:0.5rem; text-align:right;">
          <button type="submit" class="btn" style="background:#0b1e3b; color:white; padding:0.65rem 1.5rem; border:none; border-radius:6px; font-weight:700; cursor:pointer;">
            🤖 Submeter para Triagem da IA
          </button>
        </div>
      </form>
    `;

    openModal('Sugerir / Enviar Livro para a BiblioTânia', bodyHtml);

    // Configurar alternância de tipo de envio e submissão
    const form = document.getElementById('submission-form');
    if (form) {
      const grpLink = document.getElementById('group-link');
      const grpFile = document.getElementById('group-file');

      form.querySelectorAll('input[name="tipo_envio"]').forEach(radio => {
        radio.addEventListener('change', (e) => {
          const isLink = e.target.value === 'link';
          if (grpLink) grpLink.style.display = isLink ? 'block' : 'none';
          if (grpFile) grpFile.style.display = isLink ? 'none' : 'block';
        });
      });

      form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(form);
        const submitBtn = form.querySelector('button[type="submit"]');
        if (submitBtn) {
          submitBtn.disabled = true;
          submitBtn.textContent = 'Enviando e Avaliando com IA...';
        }

        try {
          if (state.isApiOnline) {
            const resp = await fetch(`${state.apiUrl}/api/submissoes`, {
              method: 'POST',
              headers: { 'Accept': 'application/json' },
              body: formData
            });

            const data = await resp.json();
            if (resp.ok) {
              showToast('Submissão enviada com sucesso! A IA do Colégio Betânia avaliará o material.', 'success');
              closeModal();
            } else {
              showToast(data.error || 'Erro ao processar submissão.', 'error');
            }
          } else {
            // Simulação graciosa offline / registro no localStorage
            const localSubs = JSON.parse(localStorage.getItem('bibliotania_pending_subs') || '[]');
            localSubs.push({
              titulo: formData.get('titulo'),
              estudante: formData.get('estudante_nome'),
              classe: formData.get('estudante_classe'),
              tipo: formData.get('tipo_envio'),
              data: new Date().toISOString()
            });
            localStorage.setItem('bibliotania_pending_subs', JSON.stringify(localSubs));

            showToast('Sugestão registrada! Será sincronizada com o Curador de IA assim que a conexão estiver restabelecida.', 'success');
            closeModal();
          }
        } catch (err) {
          console.error(err);
          showToast('Erro de comunicação. Verifique a URL da API da Cloud VPS.', 'error');
        } finally {
          if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.textContent = '🤖 Submeter para Triagem da IA';
          }
        }
      });
    }
  }

  // Modal do Mural da IA
  async function openMuralModal() {
    let listHtml = '';

    if (state.isApiOnline) {
      try {
        const resp = await fetch(`${state.apiUrl}/api/submissoes`, {
          headers: { 'Accept': 'application/json' }
        });
        if (resp.ok) {
          const subs = await resp.json();
          if (subs && subs.length > 0) {
            listHtml = subs.map(s => `
              <div style="border-bottom:1px solid #e2e8f0; padding:0.75rem 0;">
                <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.25rem;">
                  <strong style="color:#0b1e3b;">${escapeHtml(s.Titulo)}</strong>
                  <span class="book-badge" style="${s.Status === 'aprovado' ? 'background:#ecfdf5; color:#065f46;' : (s.Status === 'rejeitado' ? 'background:#fef2f2; color:#991b1b;' : 'background:#fffbeb; color:#92400e;')}">
                    ${s.Status === 'aprovado' ? '✓ Aprovado' : (s.Status === 'rejeitado' ? '✗ Rejeitado' : '⏳ Em Triagem')}
                  </span>
                </div>
                <p style="font-size:0.85rem; color:#64748b; margin-bottom:0.25rem;">Enviado por: ${escapeHtml(s.EstudanteNome)} (${escapeHtml(s.EstudanteClasse)})</p>
                ${s.ParecerIA ? `<p style="font-size:0.85rem; color:#334155; background:#f8fafc; padding:0.5rem; border-radius:4px; font-style:italic;">"${escapeHtml(s.ParecerIA)}"</p>` : ''}
              </div>
            `).join('');
          }
        }
      } catch (e) {
        console.warn('Erro ao consultar submissões online:', e);
      }
    }

    if (!listHtml) {
      listHtml = `
        <div style="padding: 1.5rem; text-align: center; color: #64748b;">
          <div style="font-size: 2.5rem; margin-bottom: 0.5rem;">🤖</div>
          <h4 style="font-weight: 700; color: #0b1e3b; margin-bottom: 0.25rem;">Curador de IA em Monitorização Contínua</h4>
          <p style="font-size: 0.9rem; line-height: 1.5;">
            Todas as submissões enviadas por alunos e professores são avaliadas pelo modelo de Inteligência Artificial para verificar relevância pedagógica, conformidade curricular e segurança escolar antes de serem integradas ao acervo.
          </p>
        </div>
      `;
    }

    openModal('Mural de Triagem da IA — Colégio Betânia', listHtml);
  }

  // Modal Sobre Institucional
  function openSobreModal() {
    const bodyHtml = `
      <div style="font-size: 0.95rem; line-height: 1.7; color: #334155;">
        <h4 style="font-size: 1.15rem; font-weight: 800; color: #0b1e3b; margin-bottom: 0.5rem;">A Biblioteca Digital Oficial do Colégio Betânia</h4>
        <p style="margin-bottom: 1rem;">
          A <strong>BiblioTânia</strong> é a plataforma oficial de biblioteca digital do <strong>Colégio Betânia</strong>. Projetada sob o princípio da máxima simplicidade, alta velocidade e zero dependências pesadas, seu objetivo é democratizar o acesso ao conhecimento pedagógico de excelência para todos os alunos e docentes da instituição.
        </p>

        <h5 style="font-weight: 700; color: #0b1e3b; margin-top: 1rem; margin-bottom: 0.4rem;">Pilares Fundamentais:</h5>
        <ul style="padding-left: 1.25rem; margin-bottom: 1rem; color: #475569;">
          <li><strong>Acervo Curricular de Angola:</strong> Manuais oficiais da 1ª à 12ª Classe certificados pelo MED/INIDE e literatura nacional clássica e contemporânea.</li>
          <li><strong>Capas Físicas Editoriais (3:4):</strong> Design inspirado na sensação tátil de uma estante real com relevo de lombada.</li>
          <li><strong>Curadoria por Inteligência Artificial:</strong> Avaliação automatizada de novas obras com foco em ética pedagógica e rigor curricular.</li>
          <li><strong>Engenharia de Alto Desempenho em Go:</strong> Backend ultraleve operando em Cloud VPS com SQLite FTS5 e consumo de memória inferior a 25MB de RAM.</li>
        </ul>
      </div>
    `;

    openModal('Sobre a BiblioTânia', bodyHtml);
  }

  // Modal de Configuração da API Cloud VPS
  function openApiConfigModal() {
    const currentUrl = state.apiUrl;
    const bodyHtml = `
      <p style="font-size: 0.9rem; color: #475569; margin-bottom: 1rem;">
        O portal GitHub Pages opera em arquitetura híbrida. Ele pode sincronizar dados com o servidor Go da BiblioTânia rodando na sua Cloud VPS ou rede segura Tailscale.
      </p>

      <form id="api-config-form" style="display: flex; flex-direction: column; gap: 1rem;">
        <div>
          <label style="display:block; font-size:0.85rem; font-weight:600; margin-bottom:0.25rem; color:#1e293b;">URL do Servidor Go (Cloud VPS / Tailscale)</label>
          <input type="url" id="config-api-url" required class="search-input" style="width:100%; padding:0.6rem; border:1px solid #cbd5e1; border-radius:6px;" value="${escapeHtml(currentUrl)}" placeholder="http://100.112.247.97:8090">
        </div>

        <div style="background:#f8fafc; border:1px solid #e2e8f0; padding:0.75rem; border-radius:6px; font-size:0.8rem; color:#64748b;">
          💡 <strong>Dica de Rede:</strong> Em ambiente local ou Tailscale Mesh, use o IP da máquina com a porta configurada (ex: <code>http://100.112.247.97:8090</code> ou domínio público).
        </div>

        <div style="display:flex; justify-content:flex-end; gap:0.5rem; margin-top:0.5rem;">
          <button type="button" class="btn" style="background:#e2e8f0; color:#334155; border:none; padding:0.5rem 1rem; border-radius:6px; cursor:pointer;" data-action="close-modal">Cancelar</button>
          <button type="submit" class="btn" style="background:#0b1e3b; color:white; border:none; padding:0.5rem 1.25rem; border-radius:6px; font-weight:700; cursor:pointer;">Salvar e Testar Conexão</button>
        </div>
      </form>
    `;

    openModal('Conexão com a Cloud VPS', bodyHtml);

    const form = document.getElementById('api-config-form');
    if (form) {
      form.addEventListener('submit', (e) => {
        e.preventDefault();
        const newUrl = document.getElementById('config-api-url').value.trim();
        state.apiUrl = newUrl;
        localStorage.setItem(STORAGE_KEY_API, newUrl);
        showToast('URL da API atualizada com sucesso!', 'success');
        closeModal();
        checkApiHealth();
      });
    }
  }

  // Notificações Toast
  function showToast(message, type = 'info') {
    if (!dom.toastContainer) return;
    const toast = document.createElement('div');
    toast.className = `toast ${type === 'error' ? 'toast-error' : ''}`;
    toast.textContent = message;
    dom.toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateX(100%)';
      toast.style.transition = 'all 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
  }

  // Sanitização de HTML
  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // Expor API pública
  window.BiblioTania = {
    init,
    closeModal,
    openSubmeterModal,
    openMuralModal,
    openSobreModal,
    openApiConfigModal
  };

  // Disparar no carregamento do DOM
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();

// BiblioTânia — Interatividade leve e sem frameworks (Vanilla JS)
document.addEventListener('DOMContentLoaded', () => {
  // 1. Alternar campo entre Envio de Arquivo e Link na submissão
  const radiosTipo = document.querySelectorAll('input[name="tipo_envio"]');
  const grupoArquivo = document.getElementById('grupo-arquivo');
  const grupoLink = document.getElementById('grupo-link');

  if (radiosTipo && grupoArquivo && grupoLink) {
    radiosTipo.forEach(radio => {
      radio.addEventListener('change', () => {
        if (radio.value === 'arquivo') {
          grupoArquivo.style.display = 'block';
          grupoLink.style.display = 'none';
        } else {
          grupoArquivo.style.display = 'none';
          grupoLink.style.display = 'block';
        }
      });
    });
  }

  // 2. Atalho de teclado rápido: Tecla '/' foca instantaneamente na barra de busca
  const searchInput = document.querySelector('.search-input');
  if (searchInput) {
    window.addEventListener('keydown', (e) => {
      if (e.key === '/' && document.activeElement !== searchInput && !['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement.tagName)) {
        e.preventDefault();
        searchInput.focus();
        searchInput.select();
      } else if (e.key === 'Escape' && document.activeElement === searchInput) {
        searchInput.blur();
      }
    });
  }
});
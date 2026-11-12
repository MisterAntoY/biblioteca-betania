// BiblioTânia — Interatividade leve e sem frameworks (Vanilla JS)
document.addEventListener('DOMContentLoaded', () => {
  // Alternar campo entre Envio de Arquivo e Link na submissão
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
});

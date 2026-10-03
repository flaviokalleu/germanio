// Browser test of the action forms (merge, copy, move), driven by
// TestNavegadorAcoes. Usage: node acoes.js <base-url> <chromium-executable>
const { chromium } = require('playwright-core');
const [base, exe] = process.argv.slice(2);

function fail(msg) { console.error('FALHA: ' + msg); process.exit(1); }

(async () => {
  const browser = await chromium.launch({ executablePath: exe });
  // the audit context bypasses the CSP only to inject axe-core
  const ctx = await browser.newContext({ bypassCSP: true });
  const p = await ctx.newPage();
  await p.goto(base + '/entrar');
  await p.fill('input[name=login]', 'ana@x.com');
  await p.fill('input[name=senha]', 'senha-segura-1');
  await Promise.all([p.waitForNavigation(), p.click('form button')]);

  async function audit(path, what) {
    await p.goto(base + path);
    if (!(await p.locator(what).count())) fail(path + ' sem ' + what);
    await p.addScriptTag({ path: require.resolve('axe-core/axe.min.js') });
    const result = await p.evaluate(async () => await axe.run(document, { resultTypes: ['violations'] }));
    const bad = result.violations.filter(v => v.impact === 'serious' || v.impact === 'critical');
    if (bad.length) fail('acessibilidade em ' + path + ': ' + bad.map(v => v.id + ' (' + v.nodes.length + ')').join(', '));
  }
  await audit('/livros/1/revisoes/1', 'button[name=mesclar_quando_passar]');
  await audit('/livros/1', 'form[action="/livros/1/acao/copiar"]');
  await audit('/livros/1/tarefas/1', 'form[action="/livros/1/tarefas/1/acao/mudar"]');

  // merge when it passes, by clicking (the build waits: no executor runs it)
  await p.goto(base + '/livros/1/revisoes/1');
  await p.check('input[name=juntar_commits]');
  await Promise.all([p.waitForNavigation(), p.click('button[name=mesclar_quando_passar]')]);
  if (!(await p.locator('text=Mesclagem agendada').count())) fail('mesclar quando passar não agendou');
  await Promise.all([p.waitForNavigation(), p.click('form[action$="/acao/cancelar_mesclagem"] button')]);
  if (!(await p.locator('button[name=mesclar_quando_passar]').count())) fail('cancelar a mesclagem não voltou ao formulário');

  // move the task by choosing the destination
  await p.goto(base + '/livros/1/tarefas/1');
  await p.selectOption('form[action$="/acao/mudar"] select', { label: 'mapa' });
  await Promise.all([p.waitForNavigation(), p.click('form[action$="/acao/mudar"] button')]);
  if (!p.url().includes('/livros/2/tarefas/1')) fail('mudar levou a ' + p.url());

  // copy with a new name
  await p.goto(base + '/livros/1');
  await p.fill('form[action="/livros/1/acao/copiar"] input[name=nome]', 'atlas 2');
  await Promise.all([p.waitForNavigation(), p.click('form[action="/livros/1/acao/copiar"] button')]);
  if (!p.url().includes('/livros/3')) fail('copiar levou a ' + p.url());

  console.log('ok');
  await browser.close();
})().catch(e => fail(e.message));

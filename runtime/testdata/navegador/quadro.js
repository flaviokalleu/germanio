// Browser test of live pages and boards (GEPs 0020 and 0023), driven by
// TestNavegador. Usage: node quadro.js <base-url> <chromium-executable>
const { chromium } = require('playwright-core');
const [base, exe] = process.argv.slice(2);

function fail(msg) { console.error('FALHA: ' + msg); process.exit(1); }

(async () => {
  const browser = await chromium.launch({ executablePath: exe });
  const a = await browser.newContext();
  const pa = await a.newPage();
  // sign up and create a board with one card, all through the pages
  await pa.goto(base + '/cadastro');
  await pa.fill('input[name=nome]', 'Ana');
  await pa.fill('input[name=email]', 'ana@x.com');
  await pa.fill('input[name=senha]', 'senha-segura-1');
  await Promise.all([pa.waitForNavigation(), pa.click('form button')]);
  await pa.goto(base + '/quadros');
  await pa.fill('form[action="/quadros/novo"] input[name=nome]', 'Produto');
  await Promise.all([pa.waitForNavigation(), pa.click('form[action="/quadros/novo"] button')]);
  if (!pa.url().includes('/quadros/1')) fail('criar quadro levou a ' + pa.url());
  await pa.fill('form[action="/quadros/1/cartoes/novo"] input[name=titulo]', 'Login');
  await Promise.all([pa.waitForNavigation(), pa.click('form[action="/quadros/1/cartoes/novo"] button')]);

  // a second tab of the same person, looking at the board
  const b = await browser.newContext();
  const pb = await b.newPage();
  await pb.goto(base + '/entrar');
  await pb.fill('input[name=login]', 'ana@x.com');
  await pb.fill('input[name=senha]', 'senha-segura-1');
  await Promise.all([pb.waitForNavigation(), pb.click('form button')]);
  await pb.goto(base + '/quadros/1');
  await pb.waitForSelector('.coluna[data-estado=pendente] .cartao >> text=Login');

  // drag the card to "concluido" in the first tab
  await pa.goto(base + '/quadros/1');
  await pa.locator('.coluna[data-estado=pendente] .cartao').first().dragTo(pa.locator('.coluna[data-estado=concluido]'));
  // the second tab follows without reloading
  await pb.waitForSelector('.coluna[data-estado=concluido] .cartao >> text=Login', { timeout: 8000 })
    .catch(() => fail('a outra aba não viu o cartão ir para concluido'));

  // the second tab goes offline; a card is created; back online it appears
  await b.setOffline(true);
  await pa.fill('form[action="/quadros/1/cartoes/novo"] input[name=titulo]', 'Durante a queda');
  await Promise.all([pa.waitForNavigation(), pa.click('form[action="/quadros/1/cartoes/novo"] button')]);
  await pb.waitForTimeout(500);
  await b.setOffline(false);
  await pb.waitForSelector('.cartao >> text=Durante a queda', { timeout: 15000 })
    .catch(() => fail('depois de voltar, a aba não recuperou o que perdeu'));

  // keyboard: → moves the focused card to the next column; Ctrl+Z undoes it
  await pa.goto(base + '/quadros/1');
  const card = pa.locator('.coluna[data-estado=pendente] .cartao', { hasText: 'Durante a queda' });
  await card.focus();
  await pa.keyboard.press('ArrowRight');
  await pb.waitForSelector('.coluna[data-estado=comecado] .cartao >> text=Durante a queda', { timeout: 8000 })
    .catch(() => fail('a seta não moveu o cartão (ou o servidor não registrou)'));
  const announced = await pa.textContent('[data-quadro-aviso]');
  if (!/movido para/.test(announced)) fail('o movimento não foi anunciado: ' + announced);
  if (await pa.locator('[data-desfazer]').isHidden()) fail('o botão Desfazer não apareceu');
  await pa.keyboard.press('Control+z');
  await pb.waitForSelector('.coluna[data-estado=pendente] .cartao >> text=Durante a queda', { timeout: 8000 })
    .catch(() => fail('Ctrl+Z não desfez o movimento'));

  // the card opens in a panel; editing its title there updates the board
  await pa.locator('.cartao a', { hasText: 'Login' }).click();
  await pa.waitForSelector('dialog.painel[open] h1, dialog.painel[open] h2', { timeout: 5000 })
    .catch(() => fail('o cartão não abriu num painel'));
  if (!pa.url().endsWith('/quadros/1')) fail('abrir o painel saiu do quadro: ' + pa.url());
  await pa.fill('dialog.painel form[action$="/editar"] input[name=titulo]', 'Login novo');
  await pa.click('dialog.painel form[action$="/editar"] button');
  await pa.waitForSelector('.quadro .cartao >> text=Login novo', { timeout: 8000 })
    .catch(() => fail('editar no painel não atualizou o quadro'));
  await pa.keyboard.press('Escape');
  if (await pa.locator('dialog.painel[open]').count() !== 0) fail('Esc não fechou o painel');

  // selection: two cards selected with Ctrl+click move together; undo brings both back
  await pa.goto(base + '/quadros/1');
  await pa.fill('form[action="/quadros/1/cartoes/novo"] input[name=titulo]', 'Terceiro');
  await Promise.all([pa.waitForNavigation(), pa.click('form[action="/quadros/1/cartoes/novo"] button')]);
  await pa.goto(base + '/quadros/1'); // creating a card leads to the card's page
  const pend = (t) => pa.locator('.coluna[data-estado=pendente] .cartao', { hasText: t });
  await pend('Durante a queda').click({ modifiers: ['Control'], position: { x: 2, y: 2 } });
  await pend('Terceiro').click({ modifiers: ['Control'], position: { x: 2, y: 2 } });
  if (await pa.locator('.cartao.selecionado').count() !== 2) fail('Ctrl+clique não selecionou dois cartões');
  await pend('Terceiro').focus();
  await pa.keyboard.press('ArrowRight');
  await pb.waitForFunction(() => document.querySelectorAll('.coluna[data-estado=comecado] .cartao').length === 2, null, { timeout: 8000 })
    .catch(() => fail('a seleção não se moveu junta'));
  await pa.click('[data-desfazer]');
  await pb.waitForFunction(() => document.querySelectorAll('.coluna[data-estado=comecado] .cartao').length === 0, null, { timeout: 8000 })
    .catch(() => fail('desfazer não devolveu os dois cartões'));

  // local filter hides what does not match, without a request
  await pa.goto(base + '/quadros/1');
  let requests = [];
  await pa.waitForTimeout(300); // the page's own live subscription opens after loading
  pa.on('request', (r) => { if (!r.url().includes('/_ge/atualiza')) requests.push(r.url()); });
  await pa.fill('[data-filtro-quadro]', 'terceiro');
  if (await pa.locator('.quadro .cartao:visible').count() !== 1) fail('o filtro local não escondeu os outros cartões');
  if (requests.length !== 0) fail('o filtro local pediu algo ao servidor: ' + requests.join(', '));

  console.log('ok');
  await browser.close();
})().catch(e => fail(e.message));

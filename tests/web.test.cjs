const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

// Small DOM adapter for exercising application events without browser dependencies.
class Element {
  constructor() {
    this.children = [];
    this.listeners = {};
    this.attributes = {};
    this.dataset = {};
    this.value = '';
    this.checked = false;
    this.hidden = false;
    this.classList = { add() {}, remove() {} };
  }
  setAttribute(name, value) { this.attributes[name] = value; }
  removeAttribute(name) { delete this[name]; delete this.attributes[name]; }
  addEventListener(name, handler) { this.listeners[name] = handler; }
  replaceChildren(...children) { this.children = children; }
  append(...children) { this.children.push(...children); }
  get childElementCount() { return this.children.length; }
  async fire(name, properties = {}) {
    return this.listeners[name]?.({ target: this, preventDefault() {}, ...properties });
  }
}

async function app({ loadingError = false } = {}) {
  const html = fs.readFileSync(path.join(__dirname, '../web/index.html'), 'utf8');
  const elements = new Map([...html.matchAll(/id="([^"]+)"/g)].map(match => [match[1], new Element()]));
  const calls = [];
  const revoked = [];
  const context = vm.createContext({
    document: {
      getElementById: id => {
        assert.ok(elements.has(id), `Missing element: ${id}`);
        return elements.get(id);
      },
      createElement: () => new Element(),
    },
    Go: class { importObject = {}; run() {} },
    WebAssembly: { instantiateStreaming: async () => {
      if (loadingError) throw new Error('Unavailable');
      return { instance: {} };
    } },
    fetch: async () => ({}),
    Blob,
    URL: {
      createObjectURL: () => 'blob:download',
      revokeObjectURL: url => revoked.push(url),
    },
    convertPhoenixToKoinly: (...args) => { calls.push(['phoenix', ...args]); return 'Date,Sent Amount\n'; },
    convertXapoToKoinly: (...args) => { calls.push(['xapo', ...args]); return 'Date,Sent Amount\n'; },
  });
  vm.runInContext(fs.readFileSync(path.join(__dirname, '../web/app.js'), 'utf8'), context);
  await new Promise(resolve => setImmediate(resolve));
  const el = id => elements.get(id);
  const add = files => el('file-input').fire('change', { target: { files, value: '' } });
  return { el, add, calls, revoked, context };
}

function file(name, contents = 'csv') {
  return { name, size: contents.length, text: async () => contents };
}

test('Phoenix conversion requires files and invalidates download when rounding changes or a file is removed', async () => {
  const { el, add, calls, revoked } = await app();
  assert.equal(el('convert').disabled, true);
  await add([file('phoenix.csv')]);
  assert.equal(el('convert').disabled, false);
  await el('convert').fire('click');
  assert.deepEqual(calls[0], ['phoenix', 'csv', false]);
  assert.equal(el('result').hidden, false);
  el('rounding').checked = true;
  await el('rounding').fire('change');
  assert.equal(el('result').hidden, true);
  assert.equal(revoked.length, 1);
  await el('convert').fire('click');
  assert.deepEqual(calls[1], ['phoenix', 'csv', true]);
  await el('file-list').children[0].children[2].fire('click');
  assert.equal(el('result').hidden, true);
  assert.equal(el('convert').disabled, true);
});

test('Xapo accepts multiple exports, retains filenames and clears Phoenix state', async () => {
  const { el, add, calls } = await app();
  await add([file('phoenix.csv')]);
  await el('source-list').children[1].fire('click');
  assert.equal(el('file-input').multiple, true);
  assert.equal(el('rounding-options').hidden, true);
  assert.equal(el('file-list').children.length, 0);
  await add([file('USD_account.csv', 'account'), file('BTC_interest.csv', 'interest')]);
  await el('convert').fire('click');
  assert.equal(calls[0][0], 'xapo');
  assert.deepEqual(JSON.parse(JSON.stringify(calls[0][1])), [
    { name: 'USD_account.csv', contents: 'account' },
    { name: 'BTC_interest.csv', contents: 'interest' },
  ]);
  await el('source-list').children[0].fire('click');
  assert.equal(el('result').hidden, true);
  assert.equal(el('file-input').multiple, false);
});

test('changing source during a file read discards the pending conversion', async () => {
  const { el, add, calls } = await app();
  let complete;
  await add([{ name: 'slow.csv', size: 12, text: () => new Promise(resolve => { complete = resolve; }) }]);
  const pending = el('convert').fire('click');
  await el('source-list').children[1].fire('click');
  complete('old file');
  await pending;
  assert.equal(calls.length, 0);
  assert.equal(el('result').hidden, true);
  assert.equal(el('convert').disabled, true);
});

test('search filters sources and invalid files produce an error', async () => {
  const { el, add } = await app();
  el('source-search').value = 'bank';
  await el('source-search').fire('input');
  assert.equal(el('source-list').children.length, 1);
  el('source-search').value = 'unknown';
  await el('source-search').fire('input');
  assert.equal(el('no-sources').hidden, false);
  await add([file('not-a-csv.txt')]);
  assert.equal(el('status').dataset.kind, 'error');
  assert.equal(el('convert').disabled, true);
  await add([file('a.csv'), file('b.csv')]);
  assert.equal(el('status').dataset.kind, 'error');
  assert.equal(el('file-list').children.length, 0);
});

test('converter load failure keeps conversion disabled and reports the error', async () => {
  const { el, add } = await app({ loadingError: true });
  assert.equal(el('status').dataset.kind, 'error');
  await add([file('phoenix.csv')]);
  assert.equal(el('convert').disabled, true);
  assert.equal(el('status').dataset.kind, 'error');
});

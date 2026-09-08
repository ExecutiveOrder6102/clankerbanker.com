const sources = {
  phoenix: {
    dropTitle: 'Drop your Phoenix CSV here',
    rounding: true,
    name: 'Phoenix Wallet', category: 'Wallet', symbol: '↯', multiple: false,
    description: 'Bitcoin & Lightning',
    help: 'Choose one transaction export from Phoenix Wallet.',
    convert: (files, rounding) => convertPhoenixToKoinly(files[0].contents, rounding),
  },
  xapo: {
    dropTitle: 'Drop your Xapo CSVs here',
    rounding: false,
    name: 'Xapo Bank', category: 'Bank', symbol: 'x', multiple: true,
    description: 'BTC, USD & interest',
    help: 'Add all account and interest exports for the period. Keep their original filenames.',
    convert: files => convertXapoToKoinly(files),
  },
};

const $ = id => document.getElementById(id);
let sourceId = 'phoenix';
let selectedFiles = [];
let ready = false;
let loadFailed = false;
let revision = 0;
let downloadUrl;
const go = new Go();

function status(message, kind = '') {
  if (loadFailed) {
    message = 'The converter could not load. Reload the page and try again.';
    kind = 'error';
  }
  $('status').textContent = message;
  $('status').dataset.kind = kind;
}

function clearResult() {
  revision++;
  $('result').hidden = true;
  if (downloadUrl) URL.revokeObjectURL(downloadUrl);
  downloadUrl = undefined;
  $('download').removeAttribute('href');
}

function renderSources() {
  $('source-count').textContent = `${Object.keys(sources).length} available`;
  const query = $('source-search').value.trim().toLowerCase();
  $('source-list').replaceChildren();
  for (const [id, source] of Object.entries(sources)) {
    if (!`${source.name} ${source.category} ${source.description}`.toLowerCase().includes(query)) continue;
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'source-card';
    button.setAttribute('aria-pressed', String(id === sourceId));
    button.innerHTML = `<span class="source-icon ${id}">${source.symbol}</span><span class="source-copy"><strong>${source.name}</strong><small>${source.description}</small></span><span class="source-check" aria-hidden="true">${id === sourceId ? '✓' : '›'}</span>`;
    button.addEventListener('click', () => {
      if (id !== sourceId) selectSource(id);
    });
    $('source-list').append(button);
  }
  $('no-sources').hidden = $('source-list').childElementCount > 0;
}

function selectSource(id) {
  sourceId = id;
  selectedFiles = [];
  $('file-input').value = '';
  $('file-input').multiple = sources[id].multiple;
  $('source-help').textContent = sources[id].help;
  $('selected-source').textContent = sources[id].name;
  $('rounding-options').hidden = !sources[id].rounding;
  $('drop-title').textContent = sources[id].dropTitle;
  clearResult();
  renderSources();
  renderFiles();
  status(ready ? 'Ready when you are.' : 'Loading local converter…');
}

function renderFiles() {
  $('file-list').replaceChildren();
  selectedFiles.forEach((file, index) => {
    const row = document.createElement('li');
    const name = document.createElement('span');
    name.textContent = file.name;
    const size = document.createElement('small');
    size.textContent = `${Math.max(1, Math.ceil(file.size / 1024))} KB`;
    const remove = document.createElement('button');
    remove.type = 'button';
    remove.textContent = '×';
    remove.setAttribute('aria-label', `Remove ${file.name}`);
    remove.addEventListener('click', () => {
      selectedFiles.splice(index, 1);
      clearResult();
      renderFiles();
      status('File removed.');
    });
    row.append(name, size, remove);
    $('file-list').append(row);
  });
  $('convert').disabled = !ready || !selectedFiles.length;
}

function addFiles(fileList) {
  const files = Array.from(fileList);
  if (!files.length) return;
  clearResult();
  if (files.some(file => !file.name.toLowerCase().endsWith('.csv'))) {
    status('Choose CSV files to continue.', 'error');
    return;
  }
  if (!sources[sourceId].multiple && files.length !== 1) {
    status('Phoenix accepts one export at a time.', 'error');
    return;
  }
  if (sources[sourceId].multiple) {
    for (const file of files) {
      const existing = selectedFiles.findIndex(item => item.name === file.name);
      if (existing >= 0) selectedFiles[existing] = file;
      else selectedFiles.push(file);
    }
  } else selectedFiles = files;
  renderFiles();
  status(`${selectedFiles.length} file${selectedFiles.length === 1 ? '' : 's'} ready to convert.`);
}

$('source-search').addEventListener('input', renderSources);
$('file-input').addEventListener('change', event => {
  addFiles(event.target.files);
  event.target.value = '';
});
$('drop-zone').addEventListener('click', () => $('file-input').click());
for (const eventName of ['dragenter', 'dragover']) {
  $('drop-zone').addEventListener(eventName, event => {
    event.preventDefault();
    $('drop-zone').classList.add('dragover');
  });
}
$('drop-zone').addEventListener('dragleave', () => $('drop-zone').classList.remove('dragover'));
$('drop-zone').addEventListener('drop', event => {
  event.preventDefault();
  $('drop-zone').classList.remove('dragover');
  addFiles(event.dataTransfer.files);
});
$('rounding').addEventListener('change', () => {
  clearResult();
  status(selectedFiles.length ? 'Options updated. Ready to convert.' : 'Ready when you are.');
});
$('convert').addEventListener('click', async () => {
  if (!ready || !selectedFiles.length) return;
  clearResult();
  const currentRevision = revision;
  const source = sources[sourceId];
  const rounding = $('rounding').checked;
  $('convert').disabled = true;
  status('Converting on your device…');
  try {
    const files = await Promise.all(selectedFiles.map(async file => ({ name: file.name, contents: await file.text() })));
    if (currentRevision !== revision) return;
    const output = source.convert(files, rounding);
    if (output.startsWith('Error')) throw new Error(output);
    downloadUrl = URL.createObjectURL(new Blob([output], { type: 'text/csv' }));
    $('download').href = downloadUrl;
    $('result').hidden = false;
    status('Conversion complete. Your Koinly CSV is ready.', 'success');
  } catch (error) {
    if (currentRevision === revision) status(error.message, 'error');
  } finally {
    renderFiles();
  }
});

selectSource(sourceId);
WebAssembly.instantiateStreaming(fetch('main.wasm'), go.importObject).then(result => {
  go.run(result.instance);
  ready = true;
  $('engine-state').textContent = 'Local converter ready';
  status(selectedFiles.length ? 'Your files are ready to convert.' : 'Ready when you are.');
  renderFiles();
}).catch(() => {
  loadFailed = true;
  $('engine-state').textContent = 'Converter unavailable';
  status('The converter could not load. Reload the page and try again.', 'error');
});

const go = new Go();
let wasmLoaded = false;

// Load WASM
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    wasmLoaded = true;
    console.log("WASM Loaded");
}).catch(err => {
    console.error("Failed to load WASM:", err);
    showStatus("Failed to load WASM core. Please ensure main.wasm is present.", "error");
});

const dropZone = document.getElementById('drop-zone');
const fileInput = document.getElementById('file-input');
const statusDiv = document.getElementById('status');
const resultArea = document.getElementById('result-area');
const downloadBtn = document.getElementById('download-btn');
const xapoSource = document.getElementById('xapo-source');
const roundingCheckbox = document.getElementById('rounding-checkbox');
const uploadPrompt = document.getElementById('upload-prompt');

dropZone.addEventListener('click', () => fileInput.click());

dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
});

dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
});

dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    if (e.dataTransfer.files.length) {
        handleFiles(e.dataTransfer.files);
    }
});

fileInput.addEventListener('change', (e) => {
    if (e.target.files.length) {
        handleFiles(e.target.files);
    }
});

document.querySelectorAll('input[name="source"]').forEach((input) => {
    input.addEventListener('change', updateSourceControls);
});

function updateSourceControls() {
    const isXapo = xapoSource.checked;
    fileInput.multiple = isXapo;
    roundingCheckbox.disabled = isXapo;
    uploadPrompt.innerHTML = isXapo
        ? 'Drag & Drop your <strong>Xapo account and interest CSVs</strong> here'
        : 'Drag & Drop your <strong>Phoenix CSV</strong> here';
}

async function handleFiles(files) {
    if (!wasmLoaded) {
        showStatus("WASM not loaded yet. Please wait...", "error");
        return;
    }

    const isXapo = xapoSource.checked;
    if (!isXapo && files.length !== 1) {
        showStatus("Phoenix conversion accepts one CSV export at a time.", "error");
        return;
    }

    try {
        const contents = await Promise.all(Array.from(files, readFileAsText));
        const output = isXapo
            ? convertXapoToKoinly(Array.from(files, (file, index) => ({
                name: file.name,
                contents: contents[index],
            })))
            : convertPhoenixToKoinly(contents[0], roundingCheckbox.checked);

        if (output.startsWith("Error")) {
            showStatus(output, "error");
            resultArea.classList.add('hidden');
        } else {
            const success = isXapo
                ? `Consolidated ${files.length} Xapo statement${files.length === 1 ? '' : 's'} successfully!`
                : "Conversion successful!";
            showStatus(success, "success");
            setupDownload(output);
        }
    } catch (err) {
        showStatus("Error during conversion: " + err, "error");
    }
}

function readFileAsText(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => resolve(e.target.result);
        reader.onerror = () => reject(reader.error || new Error(`Could not read ${file.name}`));
        reader.readAsText(file);
    });
}

function showStatus(msg, type) {
    statusDiv.textContent = msg;
    statusDiv.className = 'status ' + type;
}

function setupDownload(csvContent) {
    resultArea.classList.remove('hidden');

    downloadBtn.onclick = () => {
        const blob = new Blob([csvContent], { type: 'text/csv' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'koinly.csv';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };
}

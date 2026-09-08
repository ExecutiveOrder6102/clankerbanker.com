const go = new Go();
let wasmLoaded = false;

// Load WASM
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);
    wasmLoaded = true;
    showStatus("Ready for your Phoenix CSV.", "success");
}).catch(err => {
    console.error("Failed to load WASM:", err);
    showStatus("The converter could not load. Please reload the page or check your local build.", "error");
});

const dropZone = document.getElementById('drop-zone');
const fileInput = document.getElementById('file-input');
const statusDiv = document.getElementById('status');
const resultArea = document.getElementById('result-area');
const downloadBtn = document.getElementById('download-btn');

dropZone.addEventListener('click', () => fileInput.click());
dropZone.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        fileInput.click();
    }
});

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
        handleFile(e.dataTransfer.files[0]);
    }
});

fileInput.addEventListener('change', (e) => {
    if (e.target.files.length) {
        handleFile(e.target.files[0]);
    }
});

let conversionVersion = 0;

function handleFile(file) {
    const version = ++conversionVersion;
    resultArea.classList.add('hidden');
    downloadBtn.onclick = null;
    if (!wasmLoaded) {
        showStatus("The converter is still loading. Please try again shortly.", "error");
        return;
    }

    const reader = new FileReader();
    showStatus("Reading and converting your CSV…", "");
    reader.onload = async (e) => {
        if (version !== conversionVersion) return;
        const content = e.target.result;
        const addRoundingCost = document.getElementById('rounding-checkbox').checked;
        try {
            const output = convertPhoenixToKoinly(content, addRoundingCost);

            if (output.startsWith("Error")) {
                showStatus(output, "error");
                resultArea.classList.add('hidden');
            } else {
                showStatus("Conversion successful!", "success");
                setupDownload(output);
            }
        } catch (err) {
            showStatus("Error during conversion: " + err, "error");
        }
    };
    reader.onerror = () => {
        if (version === conversionVersion) {
            showStatus("Could not read the selected file. Please try again.", "error");
        }
    };
    reader.readAsText(file);
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

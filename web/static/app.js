const term = new Terminal({
  cursorBlink: true,
  fontFamily: '"JetBrains Mono", "Fira Code", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
  fontSize: 14,
  lineHeight: 1.12,
  theme: {
    background: '#15191c',
    foreground: '#e8edf0',
    cursor: '#68d8c1',
    selectionBackground: '#314149',
  },
  allowProposedApi: true,
});

const fitAddon = new FitAddon.FitAddon();
term.loadAddon(fitAddon);

if (window.WebLinksAddon?.WebLinksAddon) {
  term.loadAddon(new WebLinksAddon.WebLinksAddon());
}

if (window.ImageAddon?.ImageAddon) {
  term.loadAddon(new ImageAddon.ImageAddon());
}

term.open(document.getElementById('terminal'));
fitAddon.fit();
term.focus();

const status = document.getElementById('connection');
const statusbar = document.querySelector('.statusbar');
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const socket = new WebSocket(`${protocol}//${window.location.host}/ws`);
socket.binaryType = 'arraybuffer';

function resize() {
  fitAddon.fit();
  if (socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({
      type: 'resize',
      cols: term.cols,
      rows: term.rows,
    }));
  }
}

socket.addEventListener('open', () => {
  status.textContent = 'Connected';
  statusbar.classList.add('is-ready');
  resize();
});

socket.addEventListener('message', async (event) => {
  if (event.data instanceof ArrayBuffer) {
    term.write(new Uint8Array(event.data));
    return;
  }
  if (event.data instanceof Blob) {
    term.write(new Uint8Array(await event.data.arrayBuffer()));
    return;
  }
  term.write(event.data);
});

socket.addEventListener('close', () => {
  status.textContent = 'Disconnected - reload to start a new session';
  statusbar.classList.remove('is-ready');
});

socket.addEventListener('error', () => {
  status.textContent = 'Connection error';
  statusbar.classList.remove('is-ready');
});

term.onData((data) => {
  if (socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type: 'input', data }));
  }
});

window.addEventListener('resize', resize);

async function bootWASM() {
  const el = document.getElementById('wasm-status');
  if (!window.Go) {
    el.textContent = 'WASM helper unavailable';
    return;
  }
  try {
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(fetch('/weather-web.wasm'), go.importObject);
    go.run(result.instance);
    const info = window.weatherCLIInfo?.();
    el.textContent = info ? `${info.name} ${info.version}` : 'WASM helper ready';
  } catch (error) {
    el.textContent = 'WASM helper failed';
    console.error(error);
  }
}

bootWASM();

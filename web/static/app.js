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
window.weatherTerminal = term;

if (window.WebLinksAddon?.WebLinksAddon) {
  term.loadAddon(new WebLinksAddon.WebLinksAddon());
}

if (window.ImageAddon?.ImageAddon) {
  term.loadAddon(new ImageAddon.ImageAddon({
    sixelSupport: true,
    sixelScrolling: true,
    iipSupport: true,
  }));
}

const terminalElement = document.getElementById('terminal');
term.open(terminalElement);
fitAddon.fit();
term.focus();

const status = document.getElementById('connection');
const statusbar = document.querySelector('.statusbar');
const radarImage = document.getElementById('radar-image');
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
let socket;
let imageObjectUrl;
let terminalBuffer = '';
const decoder = new TextDecoder();

function resize() {
  fitAddon.fit();
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({
      type: 'resize',
      cols: term.cols,
      rows: term.rows,
    }));
  }
}

function connect(params = new URLSearchParams()) {
  const query = params.toString();
  socket = new WebSocket(`${protocol}//${window.location.host}/ws${query ? `?${query}` : ''}`);
  socket.binaryType = 'arraybuffer';

  socket.addEventListener('open', () => {
    status.textContent = params.has('lat') ? 'Connected · browser location' : 'Connected';
    statusbar.classList.add('is-ready');
    resize();
  });

  socket.addEventListener('message', async (event) => {
    if (event.data instanceof ArrayBuffer) {
      writeTerminalText(decoder.decode(event.data, { stream: true }));
      return;
    }
    if (event.data instanceof Blob) {
      writeTerminalText(decoder.decode(await event.data.arrayBuffer(), { stream: true }));
      return;
    }
    writeTerminalText(event.data);
  });

  socket.addEventListener('close', () => {
    status.textContent = 'Disconnected - reload to start a new session';
    statusbar.classList.remove('is-ready');
  });

  socket.addEventListener('error', () => {
    status.textContent = 'Connection error';
    statusbar.classList.remove('is-ready');
  });
}

window.addEventListener('resize', resize);
new ResizeObserver(resize).observe(document.querySelector('.terminal-frame'));
document.fonts?.ready.then(resize);

term.onData((data) => {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type: 'input', data }));
  }
});

function writeTerminalText(text) {
  if (!text) {
    return;
  }
  terminalBuffer += text;
  let cursor = 0;
  while (cursor < terminalBuffer.length) {
    const clearAt = terminalBuffer.indexOf('\x1b[!p', cursor);
    const imageAt = terminalBuffer.indexOf('\x1b]1337;File=', cursor);
    const nextAt = [clearAt, imageAt].filter((idx) => idx >= 0).sort((a, b) => a - b)[0];
    if (nextAt === undefined) {
      term.write(terminalBuffer.slice(cursor));
      terminalBuffer = '';
      return;
    }
    if (nextAt > cursor) {
      term.write(terminalBuffer.slice(cursor, nextAt));
    }
    if (nextAt === clearAt) {
      hideRadarImage();
      term.write('\x1b[!p');
      cursor = nextAt + 4;
      continue;
    }
    const endAt = terminalBuffer.indexOf('\x07', nextAt);
    if (endAt < 0) {
      terminalBuffer = terminalBuffer.slice(nextAt);
      return;
    }
    showInlineImage(terminalBuffer.slice(nextAt, endAt + 1));
    cursor = endAt + 1;
  }
  terminalBuffer = '';
}

function showInlineImage(sequence) {
  const match = sequence.match(/^\x1b\]1337;File=([^:]*):([A-Za-z0-9+/=]+)\x07$/);
  if (!match) {
    return;
  }
  const params = Object.fromEntries(match[1].split(';').map((entry) => {
    const [key, value = ''] = entry.split('=');
    return [key, value];
  }));
  const bytes = Uint8Array.from(atob(match[2]), (char) => char.charCodeAt(0));
  const blob = new Blob([bytes], { type: 'image/png' });
  if (imageObjectUrl) {
    URL.revokeObjectURL(imageObjectUrl);
  }
  imageObjectUrl = URL.createObjectURL(blob);

  const screen = document.querySelector('.xterm-screen')?.getBoundingClientRect();
  const cellWidth = screen && term.cols ? screen.width / term.cols : 8;
  const cellHeight = screen && term.rows ? screen.height / term.rows : 16;
  const width = Number.parseInt(params.width, 10);
  const height = Number.parseInt(params.height, 10);
  radarImage.src = imageObjectUrl;
  radarImage.style.left = '0px';
  radarImage.style.top = `${Math.round(cellHeight * 4)}px`;
  radarImage.style.width = `${Math.round((Number.isFinite(width) ? width : 80) * cellWidth)}px`;
  radarImage.style.height = `${Math.round((Number.isFinite(height) ? height : 24) * cellHeight)}px`;
  radarImage.style.display = 'block';
}

function hideRadarImage() {
  radarImage.style.display = 'none';
  radarImage.removeAttribute('src');
  if (imageObjectUrl) {
    URL.revokeObjectURL(imageObjectUrl);
    imageObjectUrl = undefined;
  }
}

async function browserLocationParams() {
  if (!navigator.geolocation) {
    return new URLSearchParams();
  }
  status.textContent = 'Requesting browser location...';
  const location = new Promise((resolve) => {
    navigator.geolocation.getCurrentPosition(
      (position) => resolve(position),
      () => resolve(null),
      { enableHighAccuracy: false, timeout: 2500, maximumAge: 15 * 60 * 1000 },
    );
  });
  const timeout = new Promise((resolve) => setTimeout(() => resolve(null), 3000));
  const position = await Promise.race([location, timeout]);
  const params = new URLSearchParams();
  if (position?.coords) {
    params.set('lat', String(position.coords.latitude));
    params.set('lon', String(position.coords.longitude));
  }
  return params;
}

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
browserLocationParams().then(connect);

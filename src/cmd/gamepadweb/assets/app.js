let ws;
let timer = null;
let monitorTimer = null;
const statusEl = document.getElementById("status");
const padEl = document.getElementById("pad");
const gpapiEl = document.getElementById("gpapi");
const outEl = document.getElementById("out");
const axesPanel = document.getElementById("axesPanel");
const buttonsPanel = document.getElementById("buttonsPanel");
const startBtn = document.getElementById("start");
const stopBtn = document.getElementById("stop");

function log(obj) {
  outEl.textContent = JSON.stringify(obj, null, 2) + "\n" + outEl.textContent;
}

function connect() {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  ws = new WebSocket(proto + "://" + location.host + "/ws");
  ws.onopen = () => {
    statusEl.textContent = "WebSocket: connected";
  };
  ws.onclose = () => {
    statusEl.textContent = "WebSocket: closed";
    setTimeout(connect, 1500);
  };
  ws.onerror = () => {
    statusEl.textContent = "WebSocket: error";
  };
  ws.onmessage = (ev) => {
    try {
      log(JSON.parse(ev.data));
    } catch {
      log({ raw: ev.data });
    }
  };
}

function updateGamepadApiStatus() {
  const hasApi = typeof navigator.getGamepads === "function" || typeof navigator.webkitGetGamepads === "function";
  const secure = !!window.isSecureContext;
  gpapiEl.textContent = hasApi
    ? "Gamepad API: available (" + (secure ? "secure" : "insecure") + ")"
    : "Gamepad API: unavailable";
}

function readPads() {
  try {
    if (typeof navigator.getGamepads === "function") {
      return navigator.getGamepads() || [];
    }
    if (typeof navigator.webkitGetGamepads === "function") {
      return navigator.webkitGetGamepads() || [];
    }
  } catch {
    return [];
  }
  return [];
}

function getState() {
  const pads = readPads();
  let p = null;
  if (pads && pads.length) {
    for (const candidate of pads) {
      if (candidate) {
        p = candidate;
        break;
      }
    }
  }
  if (!p) {
    const count = pads ? Array.from(pads).filter(Boolean).length : 0;
    padEl.textContent = "Gamepad: none (connected=" + count + ", hint=tab focus + button press)";
    return null;
  }
  padEl.textContent = "Gamepad[" + p.index + "]: " + p.id;
  return {
    axes: Array.from(p.axes || []),
    buttons: Array.from(p.buttons || []).map((b) => ({
      pressed: !!b.pressed,
      value: Number(b.value || 0),
    })),
  };
}

function renderState(state) {
  renderAxes(state ? state.axes : []);
  renderButtons(state ? state.buttons : []);
}

function renderAxes(axes) {
  if (!axes.length) {
    axesPanel.innerHTML = '<div class="mono">no axis input</div>';
    return;
  }
  const pairs = Math.ceil(axes.length / 2);
  let gridHtml = "";
  for (let i = 0; i < pairs; i++) {
    const xi = i * 2;
    const yi = i * 2 + 1;
    const x = Math.max(-1, Math.min(1, Number(axes[xi] || 0)));
    const y = Math.max(-1, Math.min(1, Number(axes[yi] || 0)));
    const leftPct = ((x + 1) / 2) * 100;
    const topPct = ((y + 1) / 2) * 100;
    gridHtml +=
      '<div class="xy-card">' +
      '<div class="xy-title mono">axis[' +
      xi +
      "] / axis[" +
      yi +
      "]</div>" +
      '<div class="xy-plane"><div class="xy-dot" style="left:' +
      leftPct.toFixed(2) +
      "%;top:" +
      topPct.toFixed(2) +
      '%"></div></div>' +
      '<div class="xy-values mono">x=' +
      x.toFixed(3) +
      " y=" +
      y.toFixed(3) +
      "</div>" +
      "</div>";
  }
  const raw = axes
    .map((v, i) => "a[" + i + "]=" + Math.max(-1, Math.min(1, Number(v || 0))).toFixed(3))
    .join("  ");
  axesPanel.innerHTML =
    '<div class="xy-grid">' + gridHtml + '</div><div class="axes-raw mono">' + raw + "</div>";
}

function renderButtons(buttons) {
  if (!buttons.length) {
    buttonsPanel.innerHTML = '<div class="mono">no button input</div>';
    return;
  }
  buttonsPanel.innerHTML = buttons
    .map((b, i) => {
      const val = Math.max(0, Math.min(1, Number((b && b.value) || 0)));
      const pressed = !!(b && b.pressed);
      const cls = "btn-card" + (pressed ? " active" : "");
      return (
        '<div class="' +
        cls +
        '">' +
        '<div class="btn-title mono">btn[' +
        i +
        "]</div>" +
        '<div class="mono">pressed: ' +
        (pressed ? "1" : "0") +
        "</div>" +
        '<div class="mono">value: ' +
        val.toFixed(3) +
        "</div>" +
        "</div>"
      );
    })
    .join("");
}

function startMonitor() {
  if (monitorTimer) {
    return;
  }
  monitorTimer = setInterval(() => {
    updateGamepadApiStatus();
    const state = getState();
    renderState(state);
  }, 50);
}

startBtn.onclick = () => {
  if (timer) {
    return;
  }
  timer = setInterval(() => {
    if (!ws || ws.readyState !== 1) {
      return;
    }
    const state = getState();
    renderState(state);
    if (!state) {
      return;
    }
    ws.send(JSON.stringify(state));
  }, 100);
};

stopBtn.onclick = () => {
  clearInterval(timer);
  timer = null;
};

window.addEventListener("gamepadconnected", (e) => log({ event: "gamepadconnected", id: e.gamepad.id }));
window.addEventListener("gamepaddisconnected", (e) => log({ event: "gamepaddisconnected", id: e.gamepad.id }));

updateGamepadApiStatus();
startMonitor();
connect();

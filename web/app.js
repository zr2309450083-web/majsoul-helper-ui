"use strict";

const $ = (id) => document.getElementById(id);
const heroLabel = $("hero-label");
const heroTile = $("hero-tile");
const heroMeta = $("hero-meta");
const heroBadge = $("hero-badge");
const callEl = $("call");
const callTitle = $("call-title");
const callVerdict = $("call-verdict");
const callReason = $("call-reason");
const callYaku = $("call-yaku");
const callPass = $("call-pass");
const callOptions = $("call-options");
const connDot = $("conn");
const roundEl = $("round");
const candidatesEl = $("candidates");
const defenseEl = $("defense");
const handEl = $("hand");
const meldsEl = $("melds");
const doraEl = $("dora");
const historyEl = $("history");
const waitingEl = $("waiting");
const updatedEl = $("updated");

let socket = null;
let retryMs = 500;

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined && text !== null) node.textContent = String(text);
  return node;
}

function suitClass(tile) {
  return tile && tile.suit ? tile.suit : "honor";
}

function tileEl(tile, extra) {
  const node = el("span", "tile " + suitClass(tile) + (extra ? " " + extra : ""));
  node.textContent = tile && tile.name ? tile.name : "?";
  if (tile && tile.red) node.classList.add("red");
  return node;
}

function fmtRate(value) {
  return value > 0 ? value.toFixed(1) + "%" : "—";
}

function fmtRisk(value, known) {
  if (!known) return "—";
  return value.toFixed(0);
}

function riskTone(value, known) {
  if (!known) return "safe";
  if (value >= 50) return "danger";
  if (value >= 20) return "warn";
  return "safe";
}

function setConn(on) {
  connDot.classList.toggle("on", on);
}

function setRound(state) {
  const round = state.round || {};
  const pieces = [];
  if (round.roundText) pieces.push(round.roundText);
  if (round.honbaText) pieces.push(round.honbaText);
  if (round.windText) pieces.push(round.windText);
  if (round.selfWind) pieces.push("自风" + round.selfWind);
  if (state.turn > 0) pieces.push("第" + state.turn + "巡");
  roundEl.textContent = pieces.length ? pieces.join(" · ") : "等待数据";
}

function chooseBest(state) {
  if (state.attack && state.attack.tile) {
    const candidate = (state.candidates || []).find(
      (item) => item.tile && item.tile.tile === state.attack.tile.tile
    );
    if (candidate) return candidate;
    return {
      tile: state.attack.tile,
      risk: state.attack.risk,
      riskKnown: state.attack.riskKnown,
      waitsCount: 0,
      agariRate: 0,
      improveCount: 0,
      isDora: false,
      isIncShanten: false,
    };
  }
  return (state.candidates || [])[0] || null;
}

function renderHero(state) {
  const call = state.call;
  const isCall = !!(call && call.available);

  if (isCall && !call.shouldCall) {
    heroLabel.textContent = "鸣牌判断";
    heroTile.textContent = "跳过";
    heroTile.classList.remove("empty");
    const pass = call.pass || {};
    heroMeta.textContent = "不鸣：" + (pass.shantenText || "—") + " · 进张 " + (pass.waitsCount || 0);
    heroBadge.textContent = "";
    heroBadge.appendChild(el("span", "badge danger", call.reason || "建议跳过"));
    if (pass.isClosed) heroBadge.appendChild(el("span", "badge", "门清可立直"));
    (pass.yaku || []).forEach((name) => heroBadge.appendChild(el("span", "badge accent", name)));
    return;
  }

  const best = isCall && call.shouldCall ? call.best || chooseBest(state) : chooseBest(state);
  heroLabel.textContent = isCall && call.shouldCall
    ? "建议" + (call.kind || "鸣") + "，打"
    : "推荐打哪张牌";
  heroTile.textContent = best && best.tile ? best.tile.name : "--";
  heroTile.classList.toggle("empty", !best || !best.tile);

  if (best && best.tile) {
    const parts = [];
    parts.push("进张 " + (best.waitsCount || 0));
    if (best.agariRate > 0) parts.push("和率 " + best.agariRate.toFixed(1) + "%");
    if (best.improveCount > 0) parts.push("改良 " + best.improveCount);
    parts.push("危险度 " + fmtRisk(best.risk, best.riskKnown));
    heroMeta.textContent = parts.join(" · ");
  } else if (state.shantenText) {
    heroMeta.textContent = "当前" + state.shantenText;
  } else {
    heroMeta.textContent = "";
  }

  heroBadge.textContent = "";
  if (isCall && call.shouldCall) {
    heroBadge.appendChild(el("span", "badge safe", call.verdict || "建议鸣牌"));
    (call.yaku || []).forEach((name) => heroBadge.appendChild(el("span", "badge accent", name)));
  }
  if (state.isRiichi) heroBadge.appendChild(el("span", "badge accent", "立直"));
  if (best && best.isDora) heroBadge.appendChild(el("span", "badge accent", "宝牌"));
  if (best && best.isIncShanten) heroBadge.appendChild(el("span", "badge", "退向听"));
  if (best && best.meldText) heroBadge.appendChild(el("span", "badge", best.meldText));
  if (best && best.tile) {
    const tone = riskTone(best.risk, best.riskKnown);
    const label = "危险度 " + fmtRisk(best.risk, best.riskKnown);
    heroBadge.appendChild(el("span", "badge " + tone, label));
  }
  if (state.mode === "analysis") heroBadge.appendChild(el("span", "badge", "静态分析"));
}

function renderCall(state) {
  const call = state.call;
  if (!call || !call.available) {
    callEl.classList.add("hidden");
    return;
  }
  callEl.classList.remove("hidden");
  callEl.classList.toggle("yes", !!call.shouldCall);
  callEl.classList.toggle("no", !call.shouldCall);

  callTitle.textContent = "可" + (call.kind || "鸣") + " " + (call.tile ? call.tile.name : "");
  callVerdict.textContent = call.verdict || (call.shouldCall ? "建议鸣牌" : "建议跳过");
  callVerdict.className = "call-verdict " + (call.shouldCall ? "yes" : "no");
  callReason.textContent = call.reason || "";

  callYaku.textContent = "";
  if (call.hasYaku && call.yaku && call.yaku.length) {
    call.yaku.forEach((name) => callYaku.appendChild(el("span", "badge accent", name)));
  } else {
    callYaku.appendChild(el("span", "badge danger", "无役"));
  }
  if (call.furiten) callYaku.appendChild(el("span", "badge danger", "振听"));

  const pass = call.pass || {};
  const passBits = [];
  passBits.push("不鸣：" + (pass.shantenText || "—"));
  passBits.push("进张 " + (pass.waitsCount || 0));
  if (pass.value > 0) passBits.push("打点 " + Math.round(pass.value));
  if (pass.yaku && pass.yaku.length) {
    passBits.push(pass.yaku.join("、"));
  } else if (pass.isClosed) {
    passBits.push("可立直");
  }
  callPass.textContent = passBits.join(" · ");

  callOptions.textContent = "";
  (call.candidates || []).slice(0, 4).forEach((item) => {
    const isBest =
      call.best && item.tile && call.best.tile && item.tile.tile === call.best.tile.tile;
    const row = el("div", "call-option" + (isBest ? " best" : ""));
    row.appendChild(el("span", "discard", "打 " + (item.tile ? item.tile.name : "?")));

    const metaBits = ["进张 " + (item.waitsCount || 0)];
    if (item.agariRate > 0) metaBits.push("和率 " + item.agariRate.toFixed(1) + "%");
    if (item.meldText) metaBits.push(item.meldText);
    if (item.riskKnown) metaBits.push("危险度 " + fmtRisk(item.risk, item.riskKnown));
    if (item.furitenRate >= 1) metaBits.push("振听");
    row.appendChild(el("span", "meta", metaBits.join(" · ")));

    const tag = el("span", "tag" + (item.hasYaku ? "" : " no-yaku"));
    tag.textContent =
      item.hasYaku && item.yaku && item.yaku.length ? item.yaku.join("、") : "无役";
    row.appendChild(tag);
    callOptions.appendChild(row);
  });
}

function renderCandidates(state) {
  candidatesEl.textContent = "";
  if (state.call && state.call.available) {
    return;
  }
  const list = (state.candidates || []).slice(0, 5);
  if (!list.length) {
    if (state.hand && state.hand.length) {
      const note = el("div", "candidate");
      note.appendChild(el("span", "cand-rank", " "));
      note.appendChild(el("span", "cand-tile honor", "待"));
      note.appendChild(el("span", "cand-meta", "当前" + (state.shantenText || "") + "，等待摸牌"));
      note.appendChild(el("span", "cand-right", ""));
      candidatesEl.appendChild(note);
    }
    return;
  }

  list.forEach((item, index) => {
    const row = el("div", "candidate" + (index === 0 ? " best" : ""));
    row.appendChild(el("span", "cand-rank", "#" + (item.rank || index + 1)));
    row.appendChild(el("span", "cand-tile " + suitClass(item.tile), item.tile ? item.tile.name : "?"));

    const meta = el("span", "cand-meta");
    const bits = ["进张 " + (item.waitsCount || 0)];
    if (item.improveCount > 0) bits.push("改良 " + item.improveCount);
    if (item.meldText) bits.push(item.meldText);
    if (item.isIncShanten) bits.push("退向听");
    if (item.isDora) bits.push("宝牌");
    meta.textContent = bits.join(" · ");
    row.appendChild(meta);

    const right = el("span", "cand-right");
    right.appendChild(el("span", "cand-rate", fmtRate(item.agariRate)));
    const bar = el("span", "risk-bar");
    const fill = el("i");
    fill.style.width = Math.max(0, Math.min(100, item.riskKnown ? item.risk : 0)) + "%";
    fill.style.background =
      riskTone(item.risk, item.riskKnown) === "danger"
        ? "var(--danger)"
        : riskTone(item.risk, item.riskKnown) === "warn"
        ? "var(--warn)"
        : "var(--safe)";
    bar.appendChild(fill);
    right.appendChild(bar);
    row.appendChild(right);
    candidatesEl.appendChild(row);
  });
}

function renderDefense(state) {
  const defense = state.defense || {};
  const hasTile = defense.tile && defense.tile.name;
  const text = defense.text || "";
  if (!hasTile && !text) {
    defenseEl.classList.add("hidden");
    return;
  }
  defenseEl.classList.remove("hidden");
  defenseEl.textContent = "";
  defenseEl.appendChild(el("span", "label", "防守推荐"));
  const value = el("span", "value");
  value.textContent = hasTile ? "切 " + defense.tile.name : text;
  defenseEl.appendChild(value);
  defenseEl.appendChild(
    el("span", "risk", hasTile ? "危险度 " + fmtRisk(defense.risk, defense.riskKnown) : "")
  );
}

function renderHand(state) {
  handEl.textContent = "";
  const hand = state.hand || [];
  const drawn = state.drawn && typeof state.drawn.tile === "number" ? state.drawn.tile : -1;
  let drawnMarked = false;
  hand.forEach((tile) => {
    let extra = "";
    if (!drawnMarked && drawn >= 0 && tile.tile === drawn) {
      extra = "drawn";
      drawnMarked = true;
    }
    handEl.appendChild(tileEl(tile, extra));
  });

  meldsEl.textContent = "";
  (state.melds || []).forEach((meld) => {
    const box = el("div", "meld");
    box.appendChild(el("span", "meld-type", meld.typeText || "鸣"));
    (meld.tiles || []).forEach((tile) => box.appendChild(tileEl(tile, "small")));
    meldsEl.appendChild(box);
  });

  const dora = state.doraTiles || [];
  doraEl.classList.toggle("hidden", dora.length === 0);
  doraEl.textContent = "";
  if (dora.length) {
    doraEl.appendChild(el("span", "", "宝牌"));
    dora.forEach((tile) => doraEl.appendChild(el("span", "chip", tile.name)));
  }
}

function renderHistory(state) {
  historyEl.textContent = "";
  const history = (state.history || []).slice(-12);
  history.forEach((item) => {
    const box = el("div", "history-item");
    box.appendChild(el("div", "history-turn", "第" + item.turn + "巡"));
    if (item.meldText) {
      box.appendChild(el("div", "history-line", item.meldText));
    }
    const self = el("div", "history-line", "打 ");
    self.appendChild(el("b", "", item.selfDiscard ? item.selfDiscard.name : "—"));
    box.appendChild(self);
    const attack = el("div", "history-line attack", "攻 ");
    attack.appendChild(el("b", "", item.aiAttack ? item.aiAttack.name : "—"));
    box.appendChild(attack);
    const defense = el("div", "history-line defence", "守 ");
    defense.appendChild(el("b", "", item.aiDefence ? item.aiDefence.name : "—"));
    box.appendChild(defense);
    historyEl.appendChild(box);
  });
  historyEl.scrollLeft = historyEl.scrollWidth;
}

function renderWaiting(state) {
  const ready = state.status === "ready" && state.hand && state.hand.length > 0;
  waitingEl.classList.toggle("hidden", ready);
  if (!ready) {
    const title = waitingEl.querySelector("div");
    if (title) title.textContent = state.message || "等待牌局数据…";
  }
}

function render(state) {
  if (!state || typeof state !== "object") return;
  setRound(state);
  renderCall(state);
  renderHero(state);
  renderCandidates(state);
  renderDefense(state);
  renderHand(state);
  renderHistory(state);
  renderWaiting(state);
  if (state.updatedAt) {
    updatedEl.textContent = new Date(state.updatedAt).toLocaleTimeString();
  }
  document.title = state.round && state.round.roundText
    ? "雀魂助手 · " + state.round.roundText
    : "雀魂助手";
}

function connect() {
  const protocol = location.protocol === "https:" ? "wss://" : "ws://";
  socket = new WebSocket(protocol + location.host + "/ws");

  socket.onopen = () => {
    retryMs = 500;
    setConn(true);
  };
  socket.onmessage = (event) => {
    try {
      render(JSON.parse(event.data));
    } catch (err) {
      // Ignore malformed frames; the next state supersedes them.
    }
  };
  socket.onclose = () => {
    setConn(false);
    window.setTimeout(connect, retryMs);
    retryMs = Math.min(retryMs * 2, 5000);
  };
  socket.onerror = () => {
    if (socket) socket.close();
  };
}

connect();

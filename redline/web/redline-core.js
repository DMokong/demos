/* redline core — pure logic, no DOM globals, no app state.
   Loaded as a plain script so web/test.html can run it over file://. */
(function(global){
"use strict";

function buildTextIndex(doc){
  const nodes = [], walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT);
  let text = "", n;
  while ((n = walker.nextNode())){
    // `end` is redundant with start+length but is part of the index shape
    // app.html's globalPos() has always read; keep it.
    nodes.push({ node: n, start: text.length, end: text.length + n.nodeValue.length });
    text += n.nodeValue;
  }
  return { text, nodes };
}

function nearestIndexOf(hay, needle, near){
  let best = -1, bestD = Infinity, i = hay.indexOf(needle);
  while (i >= 0){
    const d = Math.abs(i - near);
    if (d < bestD){ bestD = d; best = i; }
    i = hay.indexOf(needle, i + 1);
  }
  return best;
}

function rangeInElement(doc, el, start, end){
  const walker = doc.createTreeWalker(el, NodeFilter.SHOW_TEXT);
  let pos = 0, n, range = doc.createRange(), started = false;
  while ((n = walker.nextNode())){
    const len = n.nodeValue.length;
    if (!started && pos + len >= start){ range.setStart(n, start - pos); started = true; }
    if (started && pos + len >= end){ range.setEnd(n, end - pos); return range; }
    pos += len;
  }
  return null;
}

function rangeFromGlobal(doc, idx, gs, ge){
  const find = p => {
    for (const e of idx.nodes){
      if (p >= e.start && p <= e.start + e.node.nodeValue.length) return { node: e.node, off: p - e.start };
    }
    return null;
  };
  const a = find(gs), b = find(ge);
  if (!a || !b) return null;
  const range = doc.createRange();
  range.setStart(a.node, a.off);
  range.setEnd(b.node, b.off);
  return range;
}

function countOccurrences(hay, needle){
  if (!needle) return 0;
  let c = 0, i = hay.indexOf(needle);
  while (i >= 0){ c++; i = hay.indexOf(needle, i + 1); }
  return c;
}

/* Same ladder as resolveAnchor, but reports which rung landed and refuses
   to guess when the quote is ambiguous (SKILL.md §4 step 4, client-side). */
function resolve(doc, idx, a){
  const miss = { status: "missing", range: null, rung: 0 };
  if (!doc) return miss;
  let el = null;
  try { el = doc.querySelector(a.selector); } catch(_){}
  if (el && el.textContent.slice(a.start_offset, a.end_offset) === a.quoted_text){
    const r = rangeInElement(doc, el, a.start_offset, a.end_offset);
    if (r) return { status: "resolved", range: r, rung: 1 };
  }
  const T = idx ? idx.text : "";
  if (a.prefix || a.suffix){
    const j = T.indexOf((a.prefix||"") + a.quoted_text + (a.suffix||""));
    if (j >= 0){
      const i = j + (a.prefix||"").length;
      const r = rangeFromGlobal(doc, idx, i, i + a.quoted_text.length);
      if (r) return { status: "resolved", range: r, rung: 2 };
    }
  }
  const n = countOccurrences(T, a.quoted_text);
  if (n === 0) return miss;
  if (n > 1) return { status: "ambiguous", range: null, rung: 0 };
  const i = T.indexOf(a.quoted_text);
  const r = rangeFromGlobal(doc, idx, i, i + a.quoted_text.length);
  return r ? { status: "resolved", range: r, rung: 3 } : miss;
}

/* Resolution order mirrors SKILL.md exactly: selector+offsets, then
   prefix+quote+suffix, then the quote nearest its recorded position. */
function resolveAnchor(doc, idx, a){ return resolve(doc, idx, a).range; }

const TRACKING = /^(utm_[a-z_]+|fbclid|gclid|mc_eid|ref)$/i;

/* Stable identity for an annotated document. Deliberately NOT a content
   hash: a hash changes exactly when the page is edited, which is the
   moment continuity matters most. */
function normalizeKey(raw){
  const s = String(raw || "").trim();
  if (!/^https?:/i.test(s)) return s;              // file:, sample:, anything else verbatim
  let u;
  try { u = new URL(s); } catch(_){ return s; }
  u.hash = "";
  for (const k of Array.from(u.searchParams.keys())){
    if (TRACKING.test(k)) u.searchParams.delete(k);
  }
  let out = u.origin + u.pathname.replace(/\/+$/, "");
  const q = u.searchParams.toString();
  return q ? out + "?" + q : out;
}

/* Spec §6. Two independent facts — does the anchor resolve, and did the
   agent respond — plus the user's dismissal, which outranks both. */
function deriveState({ resolution, response, dismissed }){
  if (dismissed) return "dismissed";
  const status = resolution ? resolution.status : "missing";
  if (status === "ambiguous") return "needs-reanchor";
  const kind = response ? response.kind : null;
  if (status === "resolved"){
    if (kind === "replied") return "answered";
    if (kind === "applied") return "applied-unverified";
    return "open";
  }
  if (kind === "applied") return "done";
  if (kind === "replied") return "answered-moved";
  return "stale";
}

const LIVE_STATES = ["open", "answered", "applied-unverified", "needs-reanchor"];
function isLive(state){ return LIVE_STATES.indexOf(state) >= 0; }

const BLOCK_RE = /<script\s+type="application\/redline\+json"\s+id="redline-state">([\s\S]*?)<\/script>/i;

function parseBundle(html){
  const m = BLOCK_RE.exec(String(html || ""));
  if (!m) return null;
  try { return JSON.parse(m[1]); } catch(_){ return null; }
}

/* Inert in a normal browser; meaningful only to redline. Replaces any
   existing block so a bundle can round-trip repeatedly. */
function serializeBundle(html, state){
  const json = JSON.stringify(state, null, 2).replace(/<\/script/gi, "<\\/script");
  const block = '<script type="application/redline+json" id="redline-state">\n' + json + '\n</script>';
  const s = String(html || "");
  if (BLOCK_RE.test(s)) return s.replace(BLOCK_RE, block);
  if (/<\/body>/i.test(s)) return s.replace(/<\/body>/i, block + "\n</body>");
  return s + "\n" + block;
}

global.RedlineCore = { buildTextIndex, nearestIndexOf, rangeInElement, rangeFromGlobal, resolveAnchor, resolve, countOccurrences, normalizeKey, deriveState, isLive, LIVE_STATES, parseBundle, serializeBundle };
})(window);

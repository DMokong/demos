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

/* Resolution order mirrors SKILL.md exactly: selector+offsets, then
   prefix+quote+suffix, then the quote nearest its recorded position. */
function resolveAnchor(doc, idx, a){
  if (!doc) return null;
  let el = null;
  try { el = doc.querySelector(a.selector); } catch(_){}
  if (el && el.textContent.slice(a.start_offset, a.end_offset) === a.quoted_text){
    return rangeInElement(doc, el, a.start_offset, a.end_offset);
  }
  const T = idx ? idx.text : "";
  let i = -1;
  if (a.prefix || a.suffix){
    const j = T.indexOf((a.prefix||"") + a.quoted_text + (a.suffix||""));
    if (j >= 0) i = j + (a.prefix||"").length;
  }
  if (i < 0) i = nearestIndexOf(T, a.quoted_text, a.document_start_offset || 0);
  if (i < 0) return null;
  return rangeFromGlobal(doc, idx, i, i + a.quoted_text.length);
}

global.RedlineCore = { buildTextIndex, nearestIndexOf, rangeInElement, rangeFromGlobal, resolveAnchor };
})(window);

/* redline store — IndexedDB persistence, one record per document key.
   Deliberately not a generic ORM: one store, five operations. */
(function(global){
"use strict";

const DB_NAME = "redline", DB_VERSION = 1, STORE = "documents";
let dbp = null;

function open(){
  if (dbp) return dbp;
  dbp = new Promise((res, rej) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE, { keyPath: "key" });
    };
    req.onsuccess = () => res(req.result);
    req.onerror = () => rej(req.error);
  });
  return dbp;
}

function tx(mode, fn){
  return open().then(db => new Promise((res, rej) => {
    const t = db.transaction(STORE, mode);
    const out = fn(t.objectStore(STORE));
    t.oncomplete = () => res(out && out.result !== undefined ? out.result : undefined);
    t.onerror = () => rej(t.error);
  }));
}

function blank(key){
  return { key, round: 0, annotations: [], responses: {}, dismissed: {}, draw_history: [], updated_at: null };
}

function loadDoc(key){
  return tx("readonly", s => s.get(key)).then(r => r || blank(key));
}

function saveDoc(key, doc){
  const rec = Object.assign(blank(key), doc, { key, updated_at: new Date().toISOString() });
  return tx("readwrite", s => s.put(rec)).then(() => undefined);
}

function listDocs(){
  return tx("readonly", s => s.getAll()).then(rows =>
    (rows || []).map(r => ({ key: r.key, round: r.round, updated_at: r.updated_at }))
                .sort((a, b) => String(b.updated_at).localeCompare(String(a.updated_at))));
}

/* Merge by annotation id. A stale bundle must never clobber newer local
   state, so this only adds or overwrites the ids it carries. */
function mergeResponses(key, responses){
  return loadDoc(key).then(doc => {
    doc.responses = Object.assign({}, doc.responses, responses || {});
    return saveDoc(key, doc);
  });
}

global.RedlineStore = { open, loadDoc, saveDoc, listDocs, mergeResponses };
})(window);

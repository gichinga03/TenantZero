import React, { useState } from 'react';
import type { Theme } from '../BloodHound';
import { SAVED_QUERIES } from '../data/queries';

interface Props { theme: Theme; }

export default function CypherEditor({ theme: t }: Props) {
  const [query, setQuery]   = useState(SAVED_QUERIES[0].cypher);
  const [result, setResult] = useState<string | null>(null);
  const [running, setRunning] = useState(false);

  const run = () => {
    setRunning(true);
    setResult(null);
    setTimeout(() => {
      setRunning(false);
      setResult(`Returned 3 paths in 42ms\n\n• JSMITH → MemberOf → IT SUPPORT\n• IT SUPPORT → AdminTo → WKSTN01\n• WKSTN01 → HasSession ← JSMITH`);
    }, 900);
  };

  return (
    <div style={{ display:'flex', flexDirection:'column', height:'100%', padding:16, gap:12 }}>
      <div style={{ fontSize:12, fontWeight:600, color: t.text }}>Cypher Query Editor</div>
      <div style={{ fontSize:11, color: t.textFaint }}>Run raw Cypher queries against the BloodHound graph database.</div>

      {/* Quick templates */}
      <div style={{ display:'flex', gap:6, flexWrap:'wrap' }}>
        {SAVED_QUERIES.slice(0, 5).map(q => (
          <button key={q.name} onClick={() => setQuery(q.cypher)} style={{ background: t.bg2, border:`1px solid ${t.border}`, borderRadius:5, padding:'3px 9px', fontSize:11, color: t.textMuted, cursor:'pointer', fontFamily:'inherit' }}>
            {q.name}
          </button>
        ))}
      </div>

      {/* Editor area */}
      <textarea
        value={query}
        onChange={e => setQuery(e.target.value)}
        spellCheck={false}
        style={{
          flex:1, minHeight:160, background: t.bg0, border:`1px solid ${t.border}`, borderRadius:6,
          padding:'10px 12px', color: t.text, fontSize:12, fontFamily:'Consolas,"Courier New",monospace',
          resize:'none', outline:'none', lineHeight:1.6,
        }}
      />

      <div style={{ display:'flex', gap:8, alignItems:'center' }}>
        <button onClick={run} disabled={running} style={{ background: t.accent, border:'none', borderRadius:6, color:'#fff', fontSize:12, padding:'7px 16px', cursor:'pointer', fontFamily:'inherit', fontWeight:600 }}>
          {running ? 'Running...' : '▶ Run Query'}
        </button>
        <button onClick={() => { setQuery(''); setResult(null); }} style={{ background: t.bg2, border:`1px solid ${t.border}`, borderRadius:6, color: t.textMuted, fontSize:12, padding:'7px 12px', cursor:'pointer', fontFamily:'inherit' }}>
          Clear
        </button>
        <span style={{ fontSize:11, color: t.textFaint, marginLeft:'auto' }}>Shift+Enter to run</span>
      </div>

      {result && (
        <pre style={{ background: t.bg0, border:`1px solid ${t.border}`, borderRadius:6, padding:'10px 12px', fontSize:11, color: t.text, fontFamily:'Consolas,"Courier New",monospace', overflowX:'auto', whiteSpace:'pre-wrap' }}>
          {result}
        </pre>
      )}
    </div>
  );
}
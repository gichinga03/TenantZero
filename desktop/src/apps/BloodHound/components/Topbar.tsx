import React, { useState, useCallback } from 'react';
import type { Theme } from '../BloodHound';
import { NODES } from '../data/nodes';
import type { NodeDef } from '../types';

interface Props {
  theme:        Theme;
  darkMode:     boolean;
  onToggleDark: () => void;
  searchVal:    string;
  onSearch:     (v: string) => void;
  onFindPath:   () => void;
  onClear:      () => void;
}

export default function Topbar({ theme: t, darkMode, onToggleDark, searchVal, onSearch, onFindPath, onClear }: Props) {
  const [showDrop, setShowDrop] = useState(false);
  const [results, setResults]   = useState<NodeDef[]>([]);

  const handleInput = useCallback((v: string) => {
    onSearch(v);
    const hits = NODES.filter(n => n.label.toLowerCase().includes(v.toLowerCase()) || n.type.toLowerCase().includes(v.toLowerCase()));
    setResults(hits.slice(0, 8));
    setShowDrop(hits.length > 0 && v.length > 0);
  }, [onSearch]);

  return (
    <div style={{ height:48, background: t.bg1, borderBottom:`1px solid ${t.border}`, display:'flex', alignItems:'center', padding:'0 12px', gap:8, flexShrink:0, boxShadow:`0 1px 3px ${t.shadow}` }}>

      {/* Logo */}
      <div style={{ display:'flex', alignItems:'center', gap:6, marginRight:4 }}>
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <circle cx="12" cy="12" r="10" fill="#cf222e" opacity="0.15"/>
          <circle cx="12" cy="12" r="6"  fill="#cf222e" opacity="0.4"/>
          <circle cx="12" cy="12" r="3"  fill="#cf222e"/>
          <line x1="12" y1="2"  x2="12" y2="22" stroke="#cf222e" strokeWidth="1.2" opacity="0.4"/>
          <line x1="2"  y1="12" x2="22" y2="12" stroke="#cf222e" strokeWidth="1.2" opacity="0.4"/>
        </svg>
        <span style={{ fontWeight:700, fontSize:15, color:'#cf222e', letterSpacing:-0.3 }}>BloodHound</span>
        <span style={{ fontSize:10, color: t.textFaint, background: t.bg2, border:`1px solid ${t.border}`, borderRadius:4, padding:'1px 5px' }}>CE</span>
      </div>

      <div style={{ width:1, height:24, background: t.border, margin:'0 4px' }}/>

      {/* Menu buttons */}
      {['File','View','Workspace','Help'].map(m => (
        <button key={m} style={{ background:'transparent', border:'none', color: t.textMuted, fontSize:12, padding:'5px 10px', borderRadius:6, cursor:'pointer', fontFamily:'inherit' }}>{m}</button>
      ))}

      {/* Search */}
      <div style={{ flex:1, display:'flex', justifyContent:'center', padding:'0 20px' }}>
        <div style={{ position:'relative', width:300 }}>
          <svg style={{ position:'absolute', left:9, top:'50%', transform:'translateY(-50%)', pointerEvents:'none' }} viewBox="0 0 16 16" width="14" height="14" fill={t.textFaint}>
            <path d="M10.68 11.74a6 6 0 0 1-7.922-8.982 6 6 0 0 1 8.982 7.922l3.04 3.04a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215ZM11.5 7a4.499 4.499 0 1 0-8.997 0A4.499 4.499 0 0 0 11.5 7Z"/>
          </svg>
          <input
            style={{ width:'100%', background: t.bg2, border:`1px solid ${t.border}`, borderRadius:6, padding:'6px 10px 6px 30px', color: t.text, fontSize:12, outline:'none', fontFamily:'inherit', boxSizing:'border-box' }}
            placeholder="Search nodes, types, SIDs..."
            value={searchVal}
            onChange={e => handleInput(e.target.value)}
            onFocus={() => results.length > 0 && setShowDrop(true)}
            onBlur={() => setTimeout(() => setShowDrop(false), 150)}
          />
          {showDrop && (
            <div style={{ position:'absolute', top:'calc(100% + 4px)', left:0, right:0, background: t.bg1, border:`1px solid ${t.border}`, borderRadius:8, zIndex:50, boxShadow:`0 8px 24px ${t.shadow}`, overflow:'hidden' }}>
              {results.map(n => (
                <div key={n.id} style={{ display:'flex', alignItems:'center', gap:10, padding:'8px 12px', cursor:'pointer' }} onMouseDown={() => onSearch(n.label)}>
                  <div style={{ width:9, height:9, borderRadius:'50%', background:'#4e9af1', flexShrink:0 }}/>
                  <div>
                    <div style={{ fontSize:12, color: t.text, fontWeight:500 }}>{n.label.split('@')[0]}</div>
                    <div style={{ fontSize:10, color: t.textFaint }}>{n.type} · {n.domain}</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Right controls */}
      <button style={{ background: t.bg2, border:`1px solid ${t.border}`, borderRadius:6, color: t.textMuted, fontSize:12, padding:'5px 12px', cursor:'pointer', fontFamily:'inherit' }} onClick={onClear}>Clear</button>
      <button style={{ background: t.bg2, border:`1px solid ${t.border}`, borderRadius:6, color: t.textMuted, fontSize:12, padding:'5px 12px', cursor:'pointer', fontFamily:'inherit' }} onClick={onFindPath}>⚡ Pathfinding</button>

      {/* Dark mode toggle */}
      <button onClick={onToggleDark} title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
        style={{ background: t.bg2, border:`1px solid ${t.border}`, borderRadius:6, color: t.text, fontSize:16, padding:'4px 10px', cursor:'pointer', lineHeight:1 }}>
        {darkMode ? '☀️' : '🌙'}
      </button>

      {/* Avatar */}
      <div style={{ width:30, height:30, borderRadius:'50%', background:'#0969da', color:'#fff', fontSize:11, fontWeight:700, display:'flex', alignItems:'center', justifyContent:'center', flexShrink:0 }}>AD</div>
    </div>
  );
}
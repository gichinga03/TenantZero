import React from 'react';
import type { Theme } from '../BloodHound';
import { NODE_COLORS } from '../data/nodeStyles';

interface Props { theme: Theme; darkMode: boolean; }

export default function Legend({ theme: t }: Props) {
  return (
    <div style={{ position:'absolute', bottom:50, right:14, background: t.legendBg, border:`1px solid ${t.border}`, borderRadius:8, padding:'10px 12px', fontSize:11, color: t.textMuted, boxShadow:`0 2px 8px ${t.shadow}`, zIndex:10 }}>
      <div style={{ fontSize:10, fontWeight:600, textTransform:'uppercase', letterSpacing:0.5, color: t.textMuted, marginBottom:7 }}>Legend</div>
      {Object.entries(NODE_COLORS).slice(0,7).map(([type, col]) => (
        <div key={type} style={{ display:'flex', alignItems:'center', gap:7, marginBottom:5 }}>
          <div style={{ width:10, height:10, borderRadius:'50%', background: col.bg, border:`1.5px solid ${col.border}`, flexShrink:0 }}/>
          <span>{type}</span>
        </div>
      ))}
      <div style={{ borderTop:`1px solid ${t.border}`, margin:'6px 0 4px' }}/>
      <div style={{ display:'flex', alignItems:'center', gap:7, marginBottom:5 }}>
        <div style={{ width:10, height:10, borderRadius:'50%', border:'2px solid #d4a017', background:'transparent', flexShrink:0 }}/>
        <span>High Value</span>
      </div>
      <div style={{ display:'flex', alignItems:'center', gap:7 }}>
        <div style={{ width:10, height:10, borderRadius:'50%', border:'2px solid #cf222e', background:'transparent', flexShrink:0 }}/>
        <span>Attack Path</span>
      </div>
    </div>
  );
}
import React from 'react';
import type { Theme } from '../BloodHound';

interface Props { theme: Theme; cy: any; }

export default function ZoomControls({ theme: t, cy }: Props) {
  const btn: React.CSSProperties = {
    width:30, height:28, background:'transparent', border:'none',
    cursor:'pointer', fontSize:16, color: t.textMuted,
    display:'flex', alignItems:'center', justifyContent:'center', fontFamily:'inherit',
  };
  return (
    <div style={{ position:'absolute', top:14, right:14, background: t.bg1, border:`1px solid ${t.border}`, borderRadius:8, overflow:'hidden', display:'flex', flexDirection:'column', boxShadow:`0 2px 8px ${t.shadow}`, zIndex:10 }}>
      <button style={btn} onClick={() => cy?.zoom(cy.zoom() * 1.25)}>+</button>
      <div style={{ height:1, background: t.border }}/>
      <button style={btn} onClick={() => cy?.zoom(cy.zoom() * 0.8)}>−</button>
      <div style={{ height:1, background: t.border }}/>
      <button style={btn} title="Fit" onClick={() => cy?.fit(undefined, 50)}>⤢</button>
    </div>
  );
}
import React from 'react';
import type { Theme } from '../BloodHound';
import { NODES, } from '../data/nodes';
import { EDGES }  from '../data/edges';

interface Props { theme: Theme; pathActive: boolean; queryStatus: string; }

export default function Statusbar({ theme: t, pathActive, queryStatus }: Props) {
  return (
    <div style={{ height:24, background: t.bg1, borderTop:`1px solid ${t.border}`, display:'flex', alignItems:'center', padding:'0 14px', gap:10, fontSize:11, color: t.textMuted, flexShrink:0 }}>
      <div style={{ width:7, height:7, borderRadius:'50%', background:'#1a7f37', flexShrink:0 }}/>
      <span>Connected — meridian.local</span>
      <Sep theme={t}/>
      <span>Nodes: {NODES.length}</span>
      <span style={{ color: t.border }}>·</span>
      <span>Edges: {EDGES.length}</span>
      <Sep theme={t}/>
      <span style={{ color: pathActive ? '#cf222e' : t.textFaint }}>
        {pathActive ? '⚡ Attack path active' : 'No path selected'}
      </span>
      <div style={{ flex:1 }}/>
      <span style={{ color: queryStatus.startsWith('Running') ? t.warning : t.textFaint }}>{queryStatus}</span>
      <Sep theme={t}/>
      <span>BloodHound CE v6.3.0</span>
    </div>
  );
}

function Sep({ theme: t }: { theme: Theme }) {
  return <div style={{ width:1, height:14, background: t.border, margin:'0 2px' }}/>;
}
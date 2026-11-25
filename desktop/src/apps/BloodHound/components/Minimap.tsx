import React from 'react';
import type { Theme } from '../BloodHound';
import type { NodeDef } from '../types';
import { NODE_COLORS } from '../data/nodeStyles';

function hashCode(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = Math.imul(31, h) + s.charCodeAt(i) | 0;
  return h;
}

interface Props { theme: Theme; nodes: NodeDef[]; }

export default function Minimap({ theme: t, nodes }: Props) {
  return (
    <div style={{ position:'absolute', bottom:50, left:14, background: t.legendBg, border:`1px solid ${t.border}`, borderRadius:8, padding:8, boxShadow:`0 2px 8px ${t.shadow}`, width:130, zIndex:10 }}>
      <div style={{ fontSize:9, color: t.textFaint, textTransform:'uppercase', letterSpacing:0.5, marginBottom:4 }}>Overview</div>
      <div style={{ position:'relative', width:'100%', height:70, background: t.bg2, borderRadius:4, overflow:'hidden' }}>
        {nodes.map(n => {
          const c = NODE_COLORS[n.type];
          return (
            <div key={n.id} title={n.label} style={{
              position:'absolute',
              width:  n.type === 'Domain' ? 8 : 5,
              height: n.type === 'Domain' ? 8 : 5,
              borderRadius:'50%',
              background: c?.bg ?? '#8b949e',
              left: `${Math.abs(hashCode(n.id))       % 85 + 5}%`,
              top:  `${Math.abs(hashCode(n.id + 'y')) % 80 + 5}%`,
            }}/>
          );
        })}
      </div>
    </div>
  );
}
import React, { useEffect, useRef } from 'react';
import type { Theme } from '../BloodHound';
import type { NodeDef } from '../types';

interface Props {
  theme:        Theme;
  x:            number;
  y:            number;
  node:         NodeDef;
  onClose:      () => void;
  onSelectNode: (n: NodeDef) => void;
}

export default function ContextMenu({ theme: t, x, y, node, onClose, onSelectNode }: Props) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [onClose]);

  const items = [
    { label: '🔍 View node info',         fn: () => { onSelectNode(node); onClose(); } },
    { label: '⚡ Shortest paths from here', fn: () => { onClose(); } },
    { label: '⚡ Shortest paths to here',   fn: () => { onClose(); } },
    { label: '───', fn: null },
    { label: '🏴 Mark as owned',            fn: () => { onClose(); } },
    { label: '★ Mark as high value',       fn: () => { onClose(); } },
    { label: '───', fn: null },
    { label: '📋 Copy name',               fn: () => { navigator.clipboard.writeText(node.label); onClose(); } },
    { label: '🔑 Copy SID',               fn: () => { node.sid && navigator.clipboard.writeText(node.sid); onClose(); } },
    { label: '📐 Copy GUID',              fn: () => { node.guid && navigator.clipboard.writeText(node.guid); onClose(); } },
    { label: '───', fn: null },
    { label: '⬇ Export JSON',             fn: () => {
      const blob = new Blob([JSON.stringify(node, null, 2)], { type:'application/json' });
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = `${node.id}.json`;
      a.click();
      onClose();
    }},
  ];

  return (
    <div ref={ref} style={{
      position:'fixed', left: x, top: y, zIndex:100,
      background: t.bg1, border:`1px solid ${t.border}`,
      borderRadius:8, padding:'4px 0', minWidth:220,
      boxShadow:`0 8px 24px ${t.shadow}`, fontSize:12,
    }}>
      <div style={{ padding:'6px 12px 4px', borderBottom:`1px solid ${t.border}`, marginBottom:4 }}>
        <div style={{ fontWeight:600, color: t.text, fontSize:12 }}>{node.label.split('@')[0]}</div>
        <div style={{ fontSize:10, color: t.textFaint }}>{node.type}</div>
      </div>
      {items.map((item, i) =>
        item.label === '───' ? (
          <div key={i} style={{ height:1, background: t.border, margin:'4px 0' }}/>
        ) : (
          <div key={i} onClick={item.fn ?? undefined} style={{
            padding:'6px 14px', cursor:'pointer', color: t.text,
            display:'flex', alignItems:'center', gap:8,
          }}
            onMouseEnter={e => (e.currentTarget.style.background = t.bg2)}
            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
          >{item.label}</div>
        )
      )}
    </div>
  );
}
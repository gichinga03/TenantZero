import React from 'react';
import type { Theme } from '../BloodHound';
import { NODES } from '../data/nodes';
import { ATTACK_PATH } from '../data/edges';
import { SAVED_QUERIES } from '../data/queries';
import { NODE_COLORS } from '../data/nodeStyles';

interface Props {
  theme:       Theme;
  ownedNodes:  Set<string>;
  activeQuery: string | null;
  pathActive:  boolean;
  onRunQuery:  (q: string) => void;
}

const DB_STATS = [
  { label:'Users',      count:847,  color:'#4e9af1' },
  { label:'Computers',  count:312,  color:'#f0883e' },
  { label:'Groups',     count:203,  color:'#a371f7' },
  { label:'Domains',    count:2,    color:'#3fb950' },
  { label:'GPOs',       count:28,   color:'#d4a017' },
  { label:'OUs',        count:64,   color:'#79c0ff' },
  { label:'Containers', count:11,   color:'#8b949e' },
];

const GROUPED = SAVED_QUERIES.reduce<Record<string, typeof SAVED_QUERIES>>((acc, q) => {
  (acc[q.group] ??= []).push(q);
  return acc;
}, {});

export default function Sidebar({ theme: t, ownedNodes, activeQuery, pathActive, onRunQuery }: Props) {
  return (
    <div style={{ width:230, background: t.bg1, borderRight:`1px solid ${t.border}`, display:'flex', flexDirection:'column', flexShrink:0, overflowY:'auto' }}>

      {/* DB Stats */}
      <SbSection title="Database Statistics" theme={t}>
        {DB_STATS.map(r => (
          <div key={r.label} style={{ display:'flex', alignItems:'center', gap:7, padding:'4px 4px', borderRadius:4 }}>
            <div style={{ width:8, height:8, borderRadius:'50%', background:r.color, flexShrink:0 }}/>
            <span style={{ fontSize:12, color: t.text, flex:1 }}>{r.label}</span>
            <span style={{ fontSize:11, color: t.textFaint, fontVariantNumeric:'tabular-nums' }}>{r.count.toLocaleString()}</span>
          </div>
        ))}
      </SbSection>

      {/* Owned principals */}
      <SbSection title="Owned Principals" theme={t} badge={ownedNodes.size}>
        {ownedNodes.size === 0
          ? <div style={{ fontSize:11, color: t.textFaint }}>No principals marked as owned.</div>
          : [...ownedNodes].map(id => {
              const n = NODES.find(x => x.id === id);
              const c = NODE_COLORS[n?.type ?? 'User'];
              return n ? (
                <div key={id} style={{ display:'flex', alignItems:'center', gap:7, padding:'3px 4px' }}>
                  <div style={{ width:8, height:8, borderRadius:'50%', background: c?.bg, flexShrink:0 }}/>
                  <span style={{ fontSize:11, color: t.text }}>{n.label.split('@')[0]}</span>
                </div>
              ) : null;
            })
        }
      </SbSection>

      {/* Queries grouped */}
      {Object.entries(GROUPED).map(([group, queries]) => (
        <SbSection key={group} title={group} theme={t}>
          {queries.map(q => (
            <div key={q.name} onClick={() => onRunQuery(q.name)}
              style={{ display:'flex', alignItems:'center', gap:7, padding:'5px 6px', borderRadius:5, cursor:'pointer', fontSize:12, marginBottom:1,
                background: activeQuery === q.name ? t.accentBg : 'transparent',
                color: activeQuery === q.name ? t.accent : t.textMuted,
              }}>
              <svg width="10" height="10" viewBox="0 0 16 16" fill={activeQuery === q.name ? t.accent : t.textFaint} style={{ flexShrink:0 }}>
                <path d="M8 0a8 8 0 1 1 0 16A8 8 0 0 1 8 0ZM1.5 8a6.5 6.5 0 1 0 13 0 6.5 6.5 0 0 0-13 0Zm4.879-2.773 4.264 2.559a.25.25 0 0 1 0 .428l-4.264 2.559A.25.25 0 0 1 6 10.559V5.442a.25.25 0 0 1 .379-.215Z"/>
              </svg>
              <span style={{ lineHeight:1.3 }}>{q.name}</span>
            </div>
          ))}
        </SbSection>
      ))}

      {/* Active path */}
      {pathActive && (
        <SbSection title="Active Attack Path" theme={t} danger>
          <div style={{ display:'flex', flexDirection:'column', gap:2 }}>
            {ATTACK_PATH.map((id, i) => {
              const n = NODES.find(x => x.id === id);
              const c = NODE_COLORS[n?.type ?? 'User'];
              return (
                <React.Fragment key={id}>
                  <div style={{ display:'flex', alignItems:'center', gap:6, padding:'3px 4px' }}>
                    <div style={{ width:8, height:8, borderRadius:'50%', background: c?.bg, flexShrink:0 }}/>
                    <span style={{ fontSize:11, color: t.text }}>{n?.label.split('@')[0] ?? id}</span>
                  </div>
                  {i < ATTACK_PATH.length - 1 && <div style={{ fontSize:10, color:'#cf222e', paddingLeft:14 }}>↓</div>}
                </React.Fragment>
              );
            })}
          </div>
        </SbSection>
      )}
    </div>
  );
}

function SbSection({ title, theme: t, badge, danger, children }: {
  title:    string;
  theme:    Theme;
  badge?:   number;
  danger?:  boolean;
  children: React.ReactNode;
}) {
  return (
    <div style={{ borderBottom:`1px solid ${t.border}`, padding:'12px 10px 10px' }}>
      <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between', marginBottom:8 }}>
        <span style={{ fontSize:11, fontWeight:600, color: danger ? '#cf222e' : t.textMuted, textTransform:'uppercase', letterSpacing:0.6 }}>{title}</span>
        {badge !== undefined && <span style={{ fontSize:10, background: t.accentBg, color: t.accent, borderRadius:9, padding:'1px 6px', fontWeight:600 }}>{badge}</span>}
      </div>
      {children}
    </div>
  );
}
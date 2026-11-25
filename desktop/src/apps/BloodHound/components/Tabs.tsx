import React from 'react';
import type { Theme } from '../BloodHound';

interface Props {
  theme:    Theme;
  activeTab:number;
  onTab:    (i: number) => void;
}

const TABS = ['Graph', 'Pre-Built Queries', 'Owned Principals', 'High Value Targets', 'Cypher Query', 'Azure'];

export default function Tabs({ theme: t, activeTab, onTab }: Props) {
  return (
    <div style={{ display:'flex', alignItems:'center', background: t.bg1, borderBottom:`1px solid ${t.border}`, padding:'0 12px', flexShrink:0, height:36 }}>
      {TABS.map((tab, i) => (
        <div key={tab} onClick={() => onTab(i)} style={{
          padding:'0 14px', height:'100%', display:'flex', alignItems:'center',
          fontSize:12, cursor:'pointer', marginBottom:-1, userSelect:'none',
          color:       activeTab === i ? t.accent : t.textMuted,
          borderBottom:`2px solid ${activeTab === i ? t.accent : 'transparent'}`,
          fontWeight:  activeTab === i ? 500 : 400,
        }}>{tab}</div>
      ))}
      <div style={{ flex:1 }}/>
      <span style={{ fontSize:11, color:'#1a7f37', marginRight:12 }}>● Synced</span>
      <span style={{ color: t.textFaint, fontSize:11 }}>meridian.local · Last ingested 2 min ago</span>
    </div>
  );
}
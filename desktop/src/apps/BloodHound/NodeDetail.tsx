import React from 'react';
import type { Theme } from './BloodHound';
import type { NodeDef } from './types';
import { EDGES, ACLS } from './data/edges';
import { NODES } from './data/nodes';
import { NODE_COLORS, EDGE_COLORS } from './data/nodeStyles';

interface Props {
  theme:        Theme;
  node:         NodeDef | null;
  ownedNodes:   Set<string>;
  onClose:      () => void;
  onMarkOwned:  (id: string) => void;
  onFindPath:   () => void;
  onSelectNode: (n: NodeDef) => void;
}

export default function NodeDetail({ theme: t, node, ownedNodes, onClose, onMarkOwned, onFindPath, onSelectNode }: Props) {
  if (!node) return <EmptyPanel theme={t} />;

  const outEdges = EDGES.filter(e => e.source === node.id);
  const inEdges  = EDGES.filter(e => e.target === node.id);
  const acls     = ACLS[node.id] ?? [];
  const isOwned  = ownedNodes.has(node.id);
  const col      = NODE_COLORS[node.type];

  const rows: Array<[string, React.ReactNode]> = [
    ['Object ID', <code style={{ fontSize:10, fontFamily:'monospace', wordBreak:'break-all' }}>{node.id}</code>],
    node.sid  ? ['SID',  <code style={{ fontSize:9, fontFamily:'monospace', wordBreak:'break-all', color: t.textMuted }}>{node.sid}</code>] : null,
    node.guid ? ['GUID', <code style={{ fontSize:9, fontFamily:'monospace', wordBreak:'break-all', color: t.textMuted }}>{node.guid}</code>] : null,
    ['Domain', node.domain],
    node.os              ? ['OS', node.os] : null,
    node.enabled         !== undefined ? ['Enabled', node.enabled  ? <span style={{ color: t.success }}>True</span> : <span style={{ color: t.danger }}>False</span>] : null,
    node.hasspn          !== undefined ? ['Has SPN', node.hasspn   ? <span style={{ color: t.warning }}>True</span> : 'False'] : null,
    node.dontreqpreauth  !== undefined ? ['AS-REP Roastable', node.dontreqpreauth ? <span style={{ color: t.warning }}>True</span> : 'False'] : null,
    node.pwdneverexpires !== undefined ? ['Pwd Never Expires', node.pwdneverexpires ? <span style={{ color: t.warning }}>True</span> : 'False'] : null,
    node.unconstraineddelegation ? ['Unconstrained Delegation', <span style={{ color: t.danger }}>True</span>] : null,
    node.adminCount      !== undefined ? ['AdminCount', node.adminCount ? '1' : '0'] : null,
    node.sensitive       !== undefined ? ['Sensitive', node.sensitive ? 'True' : 'False'] : null,
    node.pwdlastset   ? ['Pwd Last Set', node.pwdlastset] : null,
    node.lastlogon    ? ['Last Logon',   node.lastlogon]  : null,
    node.functionallevel ? ['Functional Level', node.functionallevel] : null,
    node.members      !== undefined ? ['Member Count', String(node.members)] : null,
    node.description  ? ['Description', <span style={{ color: t.textMuted, fontSize:11 }}>{node.description}</span>] : null,
  ].filter(Boolean) as Array<[string, React.ReactNode]>;

  return (
    <div style={{ width:290, background: t.bg1, borderLeft:`1px solid ${t.border}`, display:'flex', flexDirection:'column', flexShrink:0, overflowY:'auto' }}>

      {/* Header */}
      <div style={{ padding:'10px 14px', borderBottom:`1px solid ${t.border}`, display:'flex', alignItems:'center', justifyContent:'space-between', flexShrink:0 }}>
        <span style={{ fontSize:12, fontWeight:600, color: t.text }}>Node Info</span>
        <button onClick={onClose} style={{ background:'transparent', border:'none', color: t.textFaint, cursor:'pointer', fontSize:16, lineHeight:1 }}>✕</button>
      </div>

      <div style={{ flex:1, overflowY:'auto' }}>

        {/* Node header */}
        <div style={{ display:'flex', alignItems:'center', gap:12, padding:'14px 14px 10px' }}>
          <div style={{ width:46, height:46, borderRadius:10, background: col?.bg ?? '#8b949e', display:'flex', alignItems:'center', justifyContent:'center', flexShrink:0 }}>
            <svg viewBox="0 0 24 24" width="22" height="22" fill="#ffffff">
              <path d="M12 12c2.7 0 4.8-2.1 4.8-4.8S14.7 2.4 12 2.4 7.2 4.5 7.2 7.2 9.3 12 12 12zm0 2.4c-3.2 0-9.6 1.6-9.6 4.8v2.4h19.2v-2.4c0-3.2-6.4-4.8-9.6-4.8z"/>
            </svg>
          </div>
          <div style={{ minWidth:0 }}>
            <div style={{ fontSize:13, fontWeight:600, color: t.text, wordBreak:'break-all', lineHeight:1.3 }}>{node.label.split('@')[0]}</div>
            <div style={{ fontSize:11, color: t.textFaint, marginTop:2 }}>{node.domain}</div>
          </div>
        </div>

        {/* Badges */}
        <div style={{ display:'flex', flexWrap:'wrap', gap:5, padding:'0 14px 10px' }}>
          <Badge label={node.type}            bg={t.bg2}           border={t.border}       color={t.textMuted}  />
          {node.highvalue    && <Badge label="★ High Value"      bg={t.warningBg}      border={t.warning}      color={t.warning}  />}
          {node.adminCount   && <Badge label="AdminCount=1"      bg={t.dangerBg}       border={t.danger}       color={t.danger}   />}
          {node.hasspn       && <Badge label="SPN"               bg={t.warningBg}      border={t.warning}      color={t.warning}  />}
          {node.dontreqpreauth && <Badge label="AS-REP"          bg={t.warningBg}      border={t.warning}      color={t.warning}  />}
          {node.unconstraineddelegation && <Badge label="Unconstrained Deleg." bg={t.dangerBg} border={t.danger} color={t.danger} />}
          {isOwned           && <Badge label="🏴 Owned"           bg={t.dangerBg}       border={t.danger}       color={t.danger}   />}
        </div>

        {/* Properties */}
        <div style={{ borderTop:`1px solid ${t.border}`, borderBottom:`1px solid ${t.border}`, marginBottom:10 }}>
          {rows.map(([k, v]) => (
            <div key={k} style={{ display:'flex', borderBottom:`1px solid ${t.border}`, minHeight:28 }}>
              <div style={{ width:120, padding:'5px 10px 5px 14px', fontSize:11, color: t.textMuted, background: t.bg2, flexShrink:0, fontWeight:500, display:'flex', alignItems:'center' }}>{k}</div>
              <div style={{ flex:1, padding:'5px 10px', fontSize:11, color: t.text, display:'flex', alignItems:'center', wordBreak:'break-all' }}>{v}</div>
            </div>
          ))}
        </div>

        {/* ACL */}
        {acls.length > 0 && (
          <Section title={`ACL Entries (${acls.length})`} theme={t}>
            {acls.map((a, i) => (
              <div key={i} style={{ display:'flex', alignItems:'center', gap:7, padding:'4px 7px', borderRadius:4, fontSize:11, color: t.textMuted }}>
                <span style={{ color: t.text, fontWeight:500 }}>{a.principal}</span>
                <span style={{ color: EDGE_COLORS[a.right] ?? t.accent, fontSize:10 }}>{a.right}</span>
                <span style={{ marginLeft:'auto', fontSize:10, color: t.textFaint }}>{a.inherited ? 'inherited' : 'explicit'}</span>
              </div>
            ))}
          </Section>
        )}

        {/* Outbound */}
        {outEdges.length > 0 && (
          <Section title={`Outbound (${outEdges.length})`} theme={t}>
            {outEdges.map((e, i) => {
              const target = NODES.find(n => n.id === e.target);
              const tc = NODE_COLORS[target?.type ?? 'User'];
              return (
                <div key={i} style={{ display:'flex', alignItems:'center', gap:7, padding:'5px 7px', borderRadius:4, cursor:'pointer' }}
                  onClick={() => target && onSelectNode(target)}>
                  <div style={{ width:8, height:8, borderRadius:'50%', background: tc?.bg, flexShrink:0 }}/>
                  <span style={{ fontSize:11, color: t.text, flex:1, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>{target?.label.split('@')[0] ?? e.target}</span>
                  <span style={{ fontSize:10, fontWeight:500, color: EDGE_COLORS[e.label] ?? t.accent, flexShrink:0 }}>{e.label}</span>
                </div>
              );
            })}
          </Section>
        )}

        {/* Inbound */}
        {inEdges.length > 0 && (
          <Section title={`Inbound (${inEdges.length})`} theme={t}>
            {inEdges.map((e, i) => {
              const source = NODES.find(n => n.id === e.source);
              const sc = NODE_COLORS[source?.type ?? 'User'];
              return (
                <div key={i} style={{ display:'flex', alignItems:'center', gap:7, padding:'5px 7px', borderRadius:4, cursor:'pointer' }}
                  onClick={() => source && onSelectNode(source)}>
                  <div style={{ width:8, height:8, borderRadius:'50%', background: sc?.bg, flexShrink:0 }}/>
                  <span style={{ fontSize:11, color: t.text, flex:1, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>{source?.label.split('@')[0] ?? e.source}</span>
                  <span style={{ fontSize:10, fontWeight:500, color: EDGE_COLORS[e.label] ?? t.accent, flexShrink:0 }}>{e.label}</span>
                </div>
              );
            })}
          </Section>
        )}

        {/* Actions */}
        <div style={{ padding:'10px 14px 16px' }}>
          <div style={{ fontSize:10, fontWeight:600, color: t.textMuted, textTransform:'uppercase', letterSpacing:0.5, marginBottom:7 }}>Actions</div>
          {[
            { label:'⚡ Find shortest paths from here', fn: onFindPath },
            { label: isOwned ? '🏴 Unmark as owned' : '🏴 Mark as owned', fn: () => onMarkOwned(node.id) },
            { label:'📋 Copy object name',   fn: () => navigator.clipboard.writeText(node.label) },
            { label:'🔑 Copy SID',           fn: () => node.sid && navigator.clipboard.writeText(node.sid) },
            { label:'⬇ Export node (JSON)',  fn: () => {
              const blob = new Blob([JSON.stringify(node, null, 2)], { type:'application/json' });
              const a = document.createElement('a');
              a.href = URL.createObjectURL(blob);
              a.download = `${node.id}.json`;
              a.click();
            }},
          ].map(({ label, fn }) => (
            <button key={label} onClick={fn} style={{
              width:'100%', padding:'6px 10px', background: t.bg2,
              border:`1px solid ${t.border}`, borderRadius:6, color: t.text,
              fontSize:12, cursor:'pointer', textAlign:'left', marginBottom:5, fontFamily:'inherit',
            }}>{label}</button>
          ))}
        </div>
      </div>
    </div>
  );
}

function EmptyPanel({ theme: t }: { theme: Theme }) {
  return (
    <div style={{ width:290, background: t.bg1, borderLeft:`1px solid ${t.border}`, display:'flex', flexDirection:'column', alignItems:'center', justifyContent:'center', padding:24, textAlign:'center', flexShrink:0 }}>
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" style={{ opacity:0.2 }}>
        <circle cx="12" cy="12" r="10" stroke={t.text} strokeWidth="1.5"/>
        <circle cx="12" cy="12" r="4"  stroke={t.text} strokeWidth="1.5"/>
        <line x1="12" y1="2"  x2="12" y2="8"  stroke={t.text} strokeWidth="1.5"/>
        <line x1="12" y1="16" x2="12" y2="22" stroke={t.text} strokeWidth="1.5"/>
        <line x1="2"  y1="12" x2="8"  y2="12" stroke={t.text} strokeWidth="1.5"/>
        <line x1="16" y1="12" x2="22" y2="12" stroke={t.text} strokeWidth="1.5"/>
      </svg>
      <div style={{ fontSize:13, color: t.textMuted, marginTop:12 }}>Select a node to view properties</div>
      <div style={{ fontSize:11, color: t.textFaint, marginTop:4 }}>Click any node in the graph to inspect its properties, relationships, and attack paths.</div>
    </div>
  );
}

function Badge({ label, bg, border, color }: { label:string; bg:string; border:string; color:string }) {
  return (
    <span style={{ fontSize:10, borderRadius:9, padding:'2px 7px', fontWeight:500, background: bg, border:`1px solid ${border}`, color }}>{label}</span>
  );
}

function Section({ title, theme: t, children }: { title:string; theme:Theme; children:React.ReactNode }) {
  return (
    <div style={{ padding:'8px 14px' }}>
      <div style={{ fontSize:10, fontWeight:600, color: t.textMuted, textTransform:'uppercase', letterSpacing:0.5, marginBottom:6 }}>{title}</div>
      {children}
    </div>
  );
}
import React, { useState, useEffect, useRef } from 'react';
import cytoscape from 'cytoscape';

interface BloodHoundProps {}

interface NodeDef {
  id: string;
  label: string;
  type: 'User' | 'Computer' | 'Group' | 'Domain';
  domain: string;
}

interface EdgeDef {
  source: string;
  target: string;
  label: string;
}

const NODES: NodeDef[] = [
  { id: 'jsmith',     label: 'jsmith',        type: 'User',     domain: 'MERIDIAN' },
  { id: 'awalker',    label: 'awalker',        type: 'User',     domain: 'MERIDIAN' },
  { id: 'krbtgt',     label: 'krbtgt',         type: 'User',     domain: 'MERIDIAN' },
  { id: 'WKSTN01',    label: 'WKSTN01',        type: 'Computer', domain: 'MERIDIAN' },
  { id: 'DC01',       label: 'DC01',           type: 'Computer', domain: 'MERIDIAN' },
  { id: 'IT_Support', label: 'IT Support',     type: 'Group',    domain: 'MERIDIAN' },
  { id: 'DA',         label: 'Domain Admins',  type: 'Group',    domain: 'MERIDIAN' },
  { id: 'meridian',   label: 'meridian.local', type: 'Domain',   domain: 'MERIDIAN' },
];

const EDGES: EdgeDef[] = [
  { source: 'jsmith',     target: 'WKSTN01',    label: 'HasSession' },
  { source: 'awalker',    target: 'IT_Support',  label: 'MemberOf'  },
  { source: 'awalker',    target: 'meridian',    label: 'WriteDACL'  },
  { source: 'IT_Support', target: 'WKSTN01',     label: 'AdminTo'   },
  { source: 'DA',         target: 'DC01',        label: 'AdminTo'   },
];

const NODE_COLORS: Record<string, string> = {
  User:     '#17539e',
  Computer: '#d35400',
  Group:    '#7d3c98',
  Domain:   '#1e8449',
};

interface SelectedNode {
  id: string;
  label: string;
  type: string;
  domain: string;
}

export default function BloodHound(_props: BloodHoundProps) {
  const cyRef = useRef<HTMLDivElement>(null);
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);

  useEffect(() => {
    if (!cyRef.current) return;

    const cy = cytoscape({
      container: cyRef.current,
      elements: [
        ...NODES.map(n => ({
          data: { id: n.id, label: n.label, type: n.type, domain: n.domain },
        })),
        ...EDGES.map(e => ({
          data: {
            id: `${e.source}-${e.target}`,
            source: e.source,
            target: e.target,
            label: e.label,
          },
        })),
      ],
      style: [
        {
          selector: 'node',
          style: {
            'background-color': (ele: cytoscape.NodeSingular) =>
              NODE_COLORS[ele.data('type')] ?? '#444',
            'border-width': 2,
            'border-color': 'rgba(255,255,255,0.2)',
            'label': 'data(label)',
            'color': '#ffffff',
            'font-size': 11,
            'text-valign': 'bottom' as const,
            'text-halign': 'center' as const,
            'text-margin-y': 4,
            'width': 44,
            'height': 44,
          },
        },
        {
          selector: 'edge',
          style: {
            'line-color': '#666',
            'width': 1.5,
            'target-arrow-color': '#666',
            'target-arrow-shape': 'triangle' as const,
            'curve-style': 'bezier' as const,
            'label': 'data(label)',
            'font-size': 9,
            'color': '#999',
            'text-background-color': '#0d1117',
            'text-background-opacity': 1,
            'text-background-padding': '2px',
          },
        },
        {
          selector: 'node:selected',
          style: {
            'border-color': '#58a6ff',
            'border-width': 3,
          },
        },
      ],
      layout: { name: 'cose', padding: 40, animate: false } as cytoscape.LayoutOptions,
    });

    cy.on('tap', 'node', evt => {
      const node = evt.target;
      setSelectedNode({
        id: node.id(),
        label: node.data('label'),
        type: node.data('type'),
        domain: node.data('domain'),
      });
    });

    return () => cy.destroy();
  }, []);

  const s: Record<string, React.CSSProperties> = {
    root: {
      display: 'flex', flexDirection: 'column',
      width: '100%', height: '100%',
      background: '#0d1117', position: 'relative',
    },
    toolbar: {
      height: 40, background: '#161b22',
      borderBottom: '1px solid #30363d',
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      padding: '0 12px', gap: 12, flexShrink: 0,
    },
    toolBtn: {
      background: '#21262d', border: '1px solid #30363d',
      borderRadius: 4, color: '#8b949e', fontSize: 12,
      padding: '4px 10px', cursor: 'default', fontFamily: 'inherit',
    },
    infoPanel: {
      position: 'absolute', left: 12, top: 52,
      width: 220,
      background: 'rgba(22,27,34,0.95)',
      border: '1px solid #30363d', borderRadius: 6,
      padding: 12, fontSize: 12, color: '#c9d1d9',
      zIndex: 10,
    },
    typeBadge: (type: string): React.CSSProperties => ({
      display: 'inline-block',
      background: NODE_COLORS[type] ?? '#444',
      color: '#fff', fontSize: 10, borderRadius: 3,
      padding: '1px 6px', marginTop: 4,
    }),
    canvas: {
      flex: 1, background: '#0d1117',
    },
  };

  return (
    <div style={s.root}>
      {/* Toolbar */}
      <div style={s.toolbar}>
        <span style={{ color: '#ce2027', fontWeight: 700, fontSize: 14, marginRight: 'auto', fontFamily: 'inherit' }}>
          BloodHound
        </span>
        <input
          style={{
            background: '#21262d', border: '1px solid #30363d',
            borderRadius: 4, padding: '4px 10px', color: '#c9d1d9',
            fontSize: 12, width: 220, outline: 'none', fontFamily: 'inherit',
          }}
          placeholder="🔍 Search"
          readOnly
        />
        <button style={s.toolBtn}>Reload Graph</button>
        <button style={s.toolBtn}>Clear</button>
      </div>

      {/* Node info panel */}
      <div style={s.infoPanel}>
        {selectedNode ? (
          <>
            <div style={{ color: '#ffffff', fontSize: 14, fontWeight: 600 }}>{selectedNode.label}</div>
            <div style={s.typeBadge(selectedNode.type)}>{selectedNode.type}</div>
            <div style={{ marginTop: 8, color: '#8b949e' }}>Domain: {selectedNode.domain}</div>
            <div style={{ marginTop: 4, color: '#8b949e' }}>ID: {selectedNode.id}</div>
          </>
        ) : (
          <span style={{ color: '#8b949e' }}>Click a node to view details</span>
        )}
      </div>

      {/* Cytoscape canvas */}
      <div ref={cyRef} style={s.canvas} />
    </div>
  );
}
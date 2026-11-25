import React, { useEffect, useRef, useCallback } from 'react';
import cytoscape from 'cytoscape';
import type { Theme } from './BloodHound';
import type { NodeDef } from './types';
import { NODES } from './data/nodes';
import { EDGES, ATTACK_PATH } from './data/edges';
import { NODE_COLORS, EDGE_COLORS } from './data/nodeStyles';
import ContextMenu from './components/ContextMenu';
import Minimap from './components/Minimap';
import ZoomControls from './components/ZoomControls';
import Legend from './components/Legend';

interface Props {
  theme:        Theme;
  darkMode:     boolean;
  selectedNode: NodeDef | null;
  pathActive:   boolean;
  ownedNodes:   Set<string>;
  onSelectNode: (n: NodeDef | null) => void;
  onCyReady:    (cy: any) => void;
}

export default function GraphCanvas({ theme: t, darkMode, selectedNode, pathActive, ownedNodes, onSelectNode, onCyReady }: Props) {
  const cyRef      = useRef<HTMLDivElement>(null);
  const cyInstance = useRef<cytoscape.Core | null>(null);
  const [ctxMenu, setCtxMenu] = React.useState<{ x: number; y: number; node: NodeDef } | null>(null);

  const buildStyle = useCallback((): cytoscape.Stylesheet[] => {
    const col = (n: cytoscape.NodeSingular) => {
      const c = NODE_COLORS[n.data('type')];
      return darkMode ? (c?.darkBg ?? '#484f58') : (c?.bg ?? '#8b949e');
    };
    const bord = (n: cytoscape.NodeSingular) => {
      const c = NODE_COLORS[n.data('type')];
      return darkMode ? (c?.darkBorder ?? '#8b949e') : (c?.border ?? '#555');
    };
    const eCol = (e: cytoscape.EdgeSingular) => EDGE_COLORS[e.data('label')] ?? (darkMode ? '#484f58' : '#8b949e');

    return [
      {
        selector: 'node',
        style: {
          'background-color':   col,
          'border-width':       2,
          'border-color':       bord,
          'label':              (n: cytoscape.NodeSingular) => n.data('label').split('@')[0],
          'color':              darkMode ? '#e6edf3' : '#1a1a2e',
          'font-size':          10,
          'font-family':        '"Inter","Segoe UI",Arial,sans-serif',
          'font-weight':        '500' as any,
          'text-valign':        'bottom' as const,
          'text-halign':        'center' as const,
          'text-margin-y':      6,
          'text-background-color':   darkMode ? '#161b22' : '#ffffff',
          'text-background-opacity': 0.85,
          'text-background-padding': '2px' as any,
          'width':  46,
          'height': 46,
        },
      },
      {
        selector: 'node[?highvalue]',
        style: { 'border-width': 3, 'border-color': '#d4a017', 'width': 54, 'height': 54 },
      },
      {
        selector: 'edge',
        style: {
          'line-color':           eCol,
          'width':                1.5,
          'target-arrow-color':   eCol,
          'target-arrow-shape':   'triangle' as const,
          'curve-style':          'bezier' as const,
          'label':                'data(label)',
          'font-size':            8,
          'font-family':          '"Inter","Segoe UI",Arial,sans-serif',
          'color':                darkMode ? '#8b949e' : '#555',
          'text-background-color':   darkMode ? '#161b22' : '#f6f8fa',
          'text-background-opacity': 1,
          'text-background-padding': '2px' as any,
        },
      },
      {
        selector: 'node:selected',
        style: { 'border-color': darkMode ? '#58a6ff' : '#0969da', 'border-width': 3 },
      },
      {
        selector: '.attack-path-node',
        style: { 'border-color': '#cf222e', 'border-width': 4 },
      },
      {
        selector: 'edge.attack-path-edge',
        style: {
          'line-color':         '#cf222e',
          'target-arrow-color': '#cf222e',
          'width':              3,
          'line-style':         'dashed' as const,
          'line-dash-pattern':  [8, 4] as any,
        },
      },
    ];
  }, [darkMode]);

  useEffect(() => {
    if (!cyRef.current) return;

    const cy = cytoscape({
      container: cyRef.current,
      elements: [
        ...NODES.map(n => ({ data: { id: n.id, label: n.label, type: n.type, ...n } })),
        ...EDGES.map(e => ({ data: { id: `${e.source}__${e.target}__${e.label}`, source: e.source, target: e.target, label: e.label } })),
      ],
      style: buildStyle(),
      layout: {
        name: 'cose', padding: 60, animate: false,
        nodeRepulsion: () => 10000,
        idealEdgeLength: () => 130,
        edgeElasticity: () => 100,
      } as cytoscape.LayoutOptions,
      minZoom: 0.15, maxZoom: 4,
    });

    // Click node
    cy.on('tap', 'node', evt => {
      const def = NODES.find(n => n.id === evt.target.id());
      onSelectNode(def ?? null);
    });

    // Click background
    cy.on('tap', evt => { if (evt.target === cy) onSelectNode(null); });

    // Right-click context menu
    cy.on('cxttap', 'node', evt => {
      const def = NODES.find(n => n.id === evt.target.id());
      if (!def) return;
      const pos = evt.renderedPosition ?? { x: 0, y: 0 };
      const wrap = cyRef.current!.getBoundingClientRect();
      setCtxMenu({ x: pos.x + wrap.left, y: pos.y + wrap.top, node: def });
    });

    // Edge hover highlight
    cy.on('mouseover', 'edge', evt => {
      const e = evt.target;
      e.source().addClass('edge-hover');
      e.target().addClass('edge-hover');
    });
    cy.on('mouseout', 'edge', evt => {
      const e = evt.target;
      e.source().removeClass('edge-hover');
      e.target().removeClass('edge-hover');
    });

    // Tooltip on node hover
    cy.on('mouseover', 'node', evt => {
      const node = evt.target;
      const def = NODES.find(n => n.id === node.id());
      if (!def) return;
      showTooltip(def, evt.renderedPosition.x, evt.renderedPosition.y);
    });
    cy.on('mouseout', 'node', () => hideTooltip());

    cyInstance.current = cy;
    onCyReady(cy);
    return () => cy.destroy();
  }, []);

  // Update style when dark mode changes
  useEffect(() => {
    cyInstance.current?.setStyle(buildStyle());
  }, [darkMode, buildStyle]);

  // Apply attack path
  useEffect(() => {
    const cy = cyInstance.current;
    if (!cy) return;
    cy.elements().removeClass('attack-path-node attack-path-edge');
    if (!pathActive) return;
    ATTACK_PATH.forEach(id => cy.$(`#${id}`).addClass('attack-path-node'));
    for (let i = 0; i < ATTACK_PATH.length - 1; i++) {
      cy.edges(`[source="${ATTACK_PATH[i]}"][target="${ATTACK_PATH[i + 1]}"]`).addClass('attack-path-edge');
    }
  }, [pathActive]);

  return (
    <div style={{ flex:1, position:'relative', overflow:'hidden', background: t.graphBg }}>
      {/* Grid background */}
      <svg style={{ position:'absolute', inset:0, width:'100%', height:'100%', pointerEvents:'none', opacity: 0.4 }}>
        <defs>
          <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
            <path d="M 40 0 L 0 0 0 40" fill="none" stroke={t.border} strokeWidth="0.5"/>
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#grid)"/>
      </svg>

      {/* Cytoscape */}
      <div ref={cyRef} style={{ width:'100%', height:'100%' }} />

      {/* Hover tooltip */}
      <div id="cy-tooltip" style={{
        position:'absolute', display:'none', pointerEvents:'none',
        background: t.bg1, border:`1px solid ${t.border}`,
        borderRadius:8, padding:'8px 12px', fontSize:11,
        color: t.text, zIndex:40, minWidth:160,
        boxShadow:`0 4px 16px ${t.shadow}`,
      }}/>

      {/* Context menu */}
      {ctxMenu && (
        <ContextMenu
          theme={t}
          x={ctxMenu.x}
          y={ctxMenu.y}
          node={ctxMenu.node}
          onClose={() => setCtxMenu(null)}
          onSelectNode={onSelectNode}
        />
      )}

      <Legend theme={t} darkMode={darkMode} />
      <Minimap theme={t} nodes={NODES} />
      <ZoomControls theme={t} cy={cyInstance.current} />
    </div>
  );
}

function showTooltip(n: NodeDef, x: number, y: number) {
  const el = document.getElementById('cy-tooltip');
  if (!el) return;
  const lines = [
    `<strong>${n.label.split('@')[0]}</strong>`,
    `Type: ${n.type}`,
    n.sid ? `SID: <span style="font-family:monospace;font-size:10px">${n.sid}</span>` : '',
    n.hasspn  ? `<span style="color:#d4a017">⚑ SPN set (Kerberoastable)</span>` : '',
    n.dontreqpreauth ? `<span style="color:#d4a017">⚑ AS-REP Roastable</span>` : '',
    n.unconstraineddelegation ? `<span style="color:#cf222e">⚠ Unconstrained Delegation</span>` : '',
    n.adminCount ? `<span style="color:#cf222e">★ AdminCount=1</span>` : '',
  ].filter(Boolean).join('<br>');
  el.innerHTML = lines;
  el.style.display = 'block';
  el.style.left = `${x + 14}px`;
  el.style.top  = `${y - 10}px`;
}

function hideTooltip() {
  const el = document.getElementById('cy-tooltip');
  if (el) el.style.display = 'none';
}
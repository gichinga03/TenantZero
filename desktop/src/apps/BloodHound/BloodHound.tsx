import React, { useState, useCallback } from 'react';
import Topbar from './components/Topbar';
import Tabs from './components/Tabs';
import Sidebar from './components/Sidebar';
import GraphCanvas from './GraphCanvas';
import NodeDetail from './NodeDetail';
import Statusbar from './components/Statusbar';
import { ATTACK_PATH } from './data/edges';
import { NODES } from './data/nodes';
import type { NodeDef } from './types';

export default function BloodHound() {
  const [selectedNode, setSelectedNode]   = useState<NodeDef | null>(null);
  const [pathActive,   setPathActive]     = useState(false);
  const [ownedNodes,   setOwnedNodes]     = useState<Set<string>>(new Set());
  const [queryStatus,  setQueryStatus]    = useState('Ready');
  const [activeQuery,  setActiveQuery]    = useState<string | null>(null);
  const [activeTab,    setActiveTab]      = useState(0);
  const [darkMode,     setDarkMode]       = useState(false);
  const [searchVal,    setSearchVal]      = useState('');
  const [cyRef,        setCyRef]          = useState<any>(null);

  const handleFindPath = useCallback(() => setPathActive(true),  []);
  const handleClear    = useCallback(() => {
    setSelectedNode(null);
    setPathActive(false);
    setActiveQuery(null);
    cyRef?.elements().removeClass('attack-path-node attack-path-edge');
  }, [cyRef]);

  const handleMarkOwned = useCallback((id: string) => {
    setOwnedNodes(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
    const n = NODES.find(x => x.id === id);
    setQueryStatus(`Marked ${n?.label.split('@')[0] ?? id} as owned`);
    setTimeout(() => setQueryStatus('Ready'), 2000);
  }, []);

  const handleRunQuery = useCallback((q: string) => {
    setActiveQuery(q);
    setQueryStatus(`Running: ${q}...`);
    setTimeout(() => setQueryStatus(`Done — ${q}`), 1100);
  }, []);

  const handleSearch = useCallback((val: string) => {
    setSearchVal(val);
    if (!val.trim() || !cyRef) return;
    const hit = cyRef.nodes().find((n: any) =>
      n.data('label').toLowerCase().includes(val.toLowerCase())
    );
    if (hit) {
      cyRef.nodes().unselect();
      hit.select();
      cyRef.animate({ center: { eles: hit }, zoom: 1.6 }, { duration: 400 });
      const def = NODES.find(x => x.id === hit.id());
      if (def) setSelectedNode(def);
    }
  }, [cyRef]);

  const t = darkMode ? dark : light;

  return (
    <div style={{ ...s.root, background: t.bg0, color: t.text }}>
      <Topbar
        theme={t}
        darkMode={darkMode}
        onToggleDark={() => setDarkMode(d => !d)}
        searchVal={searchVal}
        onSearch={handleSearch}
        onFindPath={handleFindPath}
        onClear={handleClear}
      />
      <Tabs
        theme={t}
        activeTab={activeTab}
        onTab={setActiveTab}
      />
      <div style={s.main}>
        <Sidebar
          theme={t}
          ownedNodes={ownedNodes}
          activeQuery={activeQuery}
          pathActive={pathActive}
          onRunQuery={handleRunQuery}
        />
        <GraphCanvas
          theme={t}
          darkMode={darkMode}
          selectedNode={selectedNode}
          pathActive={pathActive}
          ownedNodes={ownedNodes}
          onSelectNode={setSelectedNode}
          onCyReady={setCyRef}
        />
        <NodeDetail
          theme={t}
          node={selectedNode}
          ownedNodes={ownedNodes}
          onClose={() => setSelectedNode(null)}
          onMarkOwned={handleMarkOwned}
          onFindPath={handleFindPath}
          onSelectNode={setSelectedNode}
        />
      </div>
      <Statusbar
        theme={t}
        pathActive={pathActive}
        queryStatus={queryStatus}
      />
    </div>
  );
}

const s: Record<string, React.CSSProperties> = {
  root: { display:'flex', flexDirection:'column', width:'100%', height:'100vh', fontFamily:'"Inter","Segoe UI",Arial,sans-serif', overflow:'hidden', fontSize:13 },
  main: { display:'flex', flex:1, overflow:'hidden' },
};

export const light = {
  bg0:'#f6f8fa', bg1:'#ffffff', bg2:'#f6f8fa', bg3:'#eaeef2',
  text:'#1a1a2e', textMuted:'#57606a', textFaint:'#8c959f',
  border:'#d0d7de', borderStrong:'#adb5bd',
  accent:'#0969da', accentBg:'#ddf4ff',
  danger:'#cf222e', dangerBg:'#fff0f0',
  success:'#1a7f37',
  warning:'#d4a017', warningBg:'#fff8c5',
  shadow:'rgba(0,0,0,0.08)',
  graphBg:'#f6f8fa',
  nodeLabel:'#1a1a2e',
  edgeLabel:'#555',
  edgeLabelBg:'#f6f8fa',
  legendBg:'#ffffff',
};

export const dark = {
  bg0:'#0d1117', bg1:'#161b22', bg2:'#21262d', bg3:'#30363d',
  text:'#e6edf3', textMuted:'#8b949e', textFaint:'#6e7681',
  border:'#30363d', borderStrong:'#484f58',
  accent:'#58a6ff', accentBg:'#1f3a5a',
  danger:'#f85149', dangerBg:'#3d1515',
  success:'#3fb950',
  warning:'#e6a817', warningBg:'#3d2e00',
  shadow:'rgba(0,0,0,0.3)',
  graphBg:'#0d1117',
  nodeLabel:'#e6edf3',
  edgeLabel:'#8b949e',
  edgeLabelBg:'#161b22',
  legendBg:'#161b22',
};

export type Theme = typeof light;
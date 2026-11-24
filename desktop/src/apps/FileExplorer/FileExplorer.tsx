import React, { useState, useRef, useCallback, useEffect } from 'react';

interface FileExplorerProps {
  onOpenFile?: (filePath: string) => void;
}

type EntryType = 'drive' | 'folder' | 'file';

interface VFSEntry {
  name: string;
  type: EntryType;
  path: string;
  size?: string;
  modified?: string;
  hidden?: boolean;
}

const VFS: Record<string, VFSEntry[]> = {
  'This PC': [
    { name: 'Local Disk (C:)', type: 'drive', path: 'C:\\', size: '237 GB free of 476 GB', modified: '' },
  ],
  'C:\\': [
    { name: 'PerfLogs',   type: 'folder', path: 'C:\\PerfLogs',   size: '',  modified: '7/14/2023 09:12 AM', hidden: true },
    { name: 'Program Files', type: 'folder', path: 'C:\\Program Files', size: '', modified: '11/18/2026 03:44 PM' },
    { name: 'Program Files (x86)', type: 'folder', path: 'C:\\Program Files (x86)', size: '', modified: '11/18/2026 03:44 PM' },
    { name: 'tools',      type: 'folder', path: 'C:\\tools',      size: '',  modified: '11/20/2026 08:30 AM' },
    { name: 'Users',      type: 'folder', path: 'C:\\Users',      size: '',  modified: '11/20/2026 08:31 AM' },
    { name: 'Windows',    type: 'folder', path: 'C:\\Windows',    size: '',  modified: '7/14/2023 09:12 AM' },
  ],
  'C:\\PerfLogs': [],
  'C:\\Program Files': [
    { name: 'Common Files', type: 'folder', path: 'C:\\Program Files\\Common Files', size: '', modified: '7/14/2023 10:00 AM' },
    { name: 'Windows Defender', type: 'folder', path: 'C:\\Program Files\\Windows Defender', size: '', modified: '11/18/2026 03:44 PM' },
  ],
  'C:\\Program Files\\Common Files': [],
  'C:\\Program Files\\Windows Defender': [],
  'C:\\Program Files (x86)': [
    { name: 'Common Files', type: 'folder', path: 'C:\\Program Files (x86)\\Common Files', size: '', modified: '7/14/2023 10:00 AM' },
  ],
  'C:\\Program Files (x86)\\Common Files': [],
  'C:\\tools': [
    { name: 'mimikatz',    type: 'folder', path: 'C:\\tools\\mimikatz',    size: '', modified: '11/20/2026 07:15 AM' },
    { name: 'PowerSploit', type: 'folder', path: 'C:\\tools\\PowerSploit', size: '', modified: '11/20/2026 07:16 AM' },
    { name: 'Rubeus.exe',  type: 'file',   path: 'C:\\tools\\Rubeus.exe',  size: '387 KB', modified: '11/20/2026 07:14 AM' },
    { name: 'SharpHound.exe', type: 'file', path: 'C:\\tools\\SharpHound.exe', size: '1.02 MB', modified: '11/20/2026 07:14 AM' },
    { name: 'PowerView.ps1', type: 'file', path: 'C:\\tools\\PowerView.ps1', size: '764 KB', modified: '11/20/2026 07:15 AM' },
    { name: 'ADModule',    type: 'folder', path: 'C:\\tools\\ADModule',    size: '', modified: '11/20/2026 07:16 AM' },
  ],
  'C:\\tools\\mimikatz': [
    { name: 'mimikatz.exe', type: 'file', path: 'C:\\tools\\mimikatz\\mimikatz.exe', size: '1.33 MB', modified: '11/20/2026 07:15 AM' },
    { name: 'mimilib.dll',  type: 'file', path: 'C:\\tools\\mimikatz\\mimilib.dll',  size: '411 KB',  modified: '11/20/2026 07:15 AM' },
    { name: 'mimidrv.sys',  type: 'file', path: 'C:\\tools\\mimikatz\\mimidrv.sys',  size: '39 KB',   modified: '11/20/2026 07:15 AM' },
  ],
  'C:\\tools\\PowerSploit': [
    { name: 'Recon',          type: 'folder', path: 'C:\\tools\\PowerSploit\\Recon',          size: '', modified: '11/20/2026 07:16 AM' },
    { name: 'Exfiltration',   type: 'folder', path: 'C:\\tools\\PowerSploit\\Exfiltration',   size: '', modified: '11/20/2026 07:16 AM' },
    { name: 'Privesc',        type: 'folder', path: 'C:\\tools\\PowerSploit\\Privesc',        size: '', modified: '11/20/2026 07:16 AM' },
    { name: 'CodeExecution',  type: 'folder', path: 'C:\\tools\\PowerSploit\\CodeExecution',  size: '', modified: '11/20/2026 07:16 AM' },
  ],
  'C:\\tools\\PowerSploit\\Recon': [
    { name: 'PowerView.ps1', type: 'file', path: 'C:\\tools\\PowerSploit\\Recon\\PowerView.ps1', size: '764 KB', modified: '11/20/2026 07:16 AM' },
  ],
  'C:\\tools\\PowerSploit\\Exfiltration': [],
  'C:\\tools\\PowerSploit\\Privesc': [],
  'C:\\tools\\PowerSploit\\CodeExecution': [],
  'C:\\tools\\ADModule': [
    { name: 'ActiveDirectory', type: 'folder', path: 'C:\\tools\\ADModule\\ActiveDirectory', size: '', modified: '11/20/2026 07:16 AM' },
    { name: 'Import-ActiveDirectory.ps1', type: 'file', path: 'C:\\tools\\ADModule\\Import-ActiveDirectory.ps1', size: '2 KB', modified: '11/20/2026 07:16 AM' },
  ],
  'C:\\tools\\ADModule\\ActiveDirectory': [],
  'C:\\Users': [
    { name: 'jsmith',  type: 'folder', path: 'C:\\Users\\jsmith',  size: '', modified: '11/20/2026 08:31 AM' },
    { name: 'Public',  type: 'folder', path: 'C:\\Users\\Public',  size: '', modified: '7/14/2023 09:12 AM' },
    { name: 'Default', type: 'folder', path: 'C:\\Users\\Default', size: '', modified: '7/14/2023 09:12 AM', hidden: true },
  ],
  'C:\\Users\\Public': [],
  'C:\\Users\\Default': [],
  'C:\\Users\\jsmith': [
    { name: '.ssh',       type: 'folder', path: 'C:\\Users\\jsmith\\.ssh',       size: '', modified: '11/20/2026 08:35 AM', hidden: true },
    { name: 'AppData',    type: 'folder', path: 'C:\\Users\\jsmith\\AppData',    size: '', modified: '11/20/2026 09:00 AM', hidden: true },
    { name: 'Desktop',    type: 'folder', path: 'C:\\Users\\jsmith\\Desktop',    size: '', modified: '11/20/2026 10:14 AM' },
    { name: 'Documents',  type: 'folder', path: 'C:\\Users\\jsmith\\Documents',  size: '', modified: '11/20/2026 10:14 AM' },
    { name: 'Downloads',  type: 'folder', path: 'C:\\Users\\jsmith\\Downloads',  size: '', modified: '11/20/2026 08:31 AM' },
    { name: 'Favorites',  type: 'folder', path: 'C:\\Users\\jsmith\\Favorites',  size: '', modified: '11/20/2026 08:31 AM', hidden: true },
    { name: 'Music',      type: 'folder', path: 'C:\\Users\\jsmith\\Music',      size: '', modified: '11/20/2026 08:31 AM' },
    { name: 'Pictures',   type: 'folder', path: 'C:\\Users\\jsmith\\Pictures',   size: '', modified: '11/20/2026 08:31 AM' },
    { name: 'Videos',     type: 'folder', path: 'C:\\Users\\jsmith\\Videos',     size: '', modified: '11/20/2026 08:31 AM' },
    { name: 'NTUSER.DAT', type: 'file',   path: 'C:\\Users\\jsmith\\NTUSER.DAT', size: '1.57 MB', modified: '11/20/2026 10:14 AM', hidden: true },
  ],
  'C:\\Users\\jsmith\\.ssh': [
    { name: 'known_hosts', type: 'file', path: 'C:\\Users\\jsmith\\.ssh\\known_hosts', size: '1 KB', modified: '11/20/2026 08:35 AM' },
  ],
  'C:\\Users\\jsmith\\AppData': [
    { name: 'Local',    type: 'folder', path: 'C:\\Users\\jsmith\\AppData\\Local',    size: '', modified: '11/20/2026 09:00 AM' },
    { name: 'Roaming',  type: 'folder', path: 'C:\\Users\\jsmith\\AppData\\Roaming',  size: '', modified: '11/20/2026 09:00 AM' },
    { name: 'LocalLow', type: 'folder', path: 'C:\\Users\\jsmith\\AppData\\LocalLow', size: '', modified: '11/20/2026 09:00 AM' },
  ],
  'C:\\Users\\jsmith\\AppData\\Local': [],
  'C:\\Users\\jsmith\\AppData\\Roaming': [],
  'C:\\Users\\jsmith\\AppData\\LocalLow': [],
  'C:\\Users\\jsmith\\Desktop': [
    { name: 'itwiki.txt',       type: 'file', path: 'C:\\Users\\jsmith\\Desktop\\itwiki.txt',       size: '3 KB',  modified: '11/19/2026 02:45 PM' },
    { name: 'shortcuts.lnk',   type: 'file', path: 'C:\\Users\\jsmith\\Desktop\\shortcuts.lnk',   size: '1 KB',  modified: '11/20/2026 08:31 AM' },
  ],
  'C:\\Users\\jsmith\\Documents': [
    { name: 'VPN Config',       type: 'folder', path: 'C:\\Users\\jsmith\\Documents\\VPN Config',       size: '', modified: '11/18/2026 04:12 PM' },
    { name: 'meeting_notes.txt',type: 'file',   path: 'C:\\Users\\jsmith\\Documents\\meeting_notes.txt',size: '6 KB', modified: '11/19/2026 11:30 AM' },
    { name: 'passwords_old.txt',type: 'file',   path: 'C:\\Users\\jsmith\\Documents\\passwords_old.txt',size: '1 KB', modified: '10/02/2026 09:00 AM' },
  ],
  'C:\\Users\\jsmith\\Documents\\VPN Config': [
    { name: 'meridian-vpn.ovpn', type: 'file', path: 'C:\\Users\\jsmith\\Documents\\VPN Config\\meridian-vpn.ovpn', size: '5 KB', modified: '11/18/2026 04:12 PM' },
  ],
  'C:\\Users\\jsmith\\Downloads': [
    { name: 'AnyDesk.exe',       type: 'file', path: 'C:\\Users\\jsmith\\Downloads\\AnyDesk.exe',       size: '4.1 MB', modified: '11/15/2026 10:22 AM' },
    { name: 'vcredist_x64.exe',  type: 'file', path: 'C:\\Users\\jsmith\\Downloads\\vcredist_x64.exe',  size: '14.3 MB',modified: '11/18/2026 03:44 PM' },
  ],
  'C:\\Users\\jsmith\\Favorites': [],
  'C:\\Users\\jsmith\\Music': [],
  'C:\\Users\\jsmith\\Pictures': [],
  'C:\\Users\\jsmith\\Videos': [],
  'C:\\Windows': [
    { name: 'System32',  type: 'folder', path: 'C:\\Windows\\System32',  size: '', modified: '11/18/2026 03:44 PM' },
    { name: 'SysWOW64',  type: 'folder', path: 'C:\\Windows\\SysWOW64',  size: '', modified: '11/18/2026 03:44 PM' },
    { name: 'Temp',      type: 'folder', path: 'C:\\Windows\\Temp',      size: '', modified: '11/20/2026 10:14 AM' },
    { name: 'Logs',      type: 'folder', path: 'C:\\Windows\\Logs',      size: '', modified: '11/20/2026 10:00 AM' },
  ],
  'C:\\Windows\\System32': [
    { name: 'cmd.exe',        type: 'file', path: 'C:\\Windows\\System32\\cmd.exe',        size: '289 KB', modified: '7/14/2023 09:12 AM' },
    { name: 'powershell.exe', type: 'file', path: 'C:\\Windows\\System32\\powershell.exe', size: '449 KB', modified: '7/14/2023 09:12 AM' },
    { name: 'lsass.exe',      type: 'file', path: 'C:\\Windows\\System32\\lsass.exe',      size: '57 KB',  modified: '7/14/2023 09:12 AM' },
    { name: 'ntds.dit',       type: 'file', path: 'C:\\Windows\\System32\\ntds.dit',       size: '18.4 MB',modified: '11/20/2026 10:14 AM' },
    { name: 'drivers',        type: 'folder', path: 'C:\\Windows\\System32\\drivers',      size: '',       modified: '11/18/2026 03:44 PM' },
  ],
  'C:\\Windows\\System32\\drivers': [],
  'C:\\Windows\\SysWOW64': [],
  'C:\\Windows\\Temp': [
    { name: 'bloodhound_20261120.zip', type: 'file', path: 'C:\\Windows\\Temp\\bloodhound_20261120.zip', size: '2.1 MB', modified: '11/20/2026 10:12 AM' },
  ],
  'C:\\Windows\\Logs': [
    { name: 'CBS',   type: 'folder', path: 'C:\\Windows\\Logs\\CBS',   size: '', modified: '11/18/2026 03:44 PM' },
    { name: 'DISM',  type: 'folder', path: 'C:\\Windows\\Logs\\DISM',  size: '', modified: '11/18/2026 03:44 PM' },
  ],
  'C:\\Windows\\Logs\\CBS': [],
  'C:\\Windows\\Logs\\DISM': [],
};

function fileIcon(name: string, type: EntryType): string {
  if (type === 'drive') return '💽';
  if (type === 'folder') return '📁';
  if (name.endsWith('.exe')) return '⚙️';
  if (name.endsWith('.ps1')) return '📜';
  if (name.endsWith('.txt')) return '📝';
  if (name.endsWith('.lnk')) return '🔗';
  if (name.endsWith('.zip')) return '🗜️';
  if (name.endsWith('.dll')) return '🔧';
  if (name.endsWith('.sys')) return '🔩';
  if (name.endsWith('.dat') || name.endsWith('.DAT')) return '🗄️';
  if (name.endsWith('.ovpn')) return '🌐';
  if (name.endsWith('.log')) return '📋';
  return '📄';
}

function parentPath(path: string): string {
  if (path === 'This PC' || path === 'C:\\') return 'This PC';
  const parts = path.replace(/\\$/, '').split('\\');
  if (parts.length === 1) return 'This PC';
  parts.pop();
  const result = parts.join('\\');
  return result === 'C:' ? 'C:\\' : result || 'This PC';
}

function formatBreadcrumbs(path: string): { label: string; path: string }[] {
  if (path === 'This PC') return [{ label: 'This PC', path: 'This PC' }];
  const crumbs: { label: string; path: string }[] = [{ label: 'This PC', path: 'This PC' }];
  if (path === 'C:\\') {
    crumbs.push({ label: 'Local Disk (C:)', path: 'C:\\' });
    return crumbs;
  }
  crumbs.push({ label: 'Local Disk (C:)', path: 'C:\\' });
  const rest = path.replace(/^C:\\/, '');
  const parts = rest.split('\\');
  let built = 'C:\\';
  for (const part of parts) {
    built = built === 'C:\\' ? `C:\\${part}` : `${built}\\${part}`;
    crumbs.push({ label: part, path: built });
  }
  return crumbs;
}

type ViewMode = 'icons' | 'details' | 'list';
type SortKey = 'name' | 'modified' | 'size' | 'type';

const SIDEBAR_GROUPS = [
  {
    label: 'Quick access',
    items: [
      { icon: '📌', label: 'Desktop',    path: 'C:\\Users\\jsmith\\Desktop' },
      { icon: '⬇️', label: 'Downloads',  path: 'C:\\Users\\jsmith\\Downloads' },
      { icon: '📄', label: 'Documents',  path: 'C:\\Users\\jsmith\\Documents' },
    ],
  },
  {
    label: 'This PC',
    items: [
      { icon: '🖥️', label: 'This PC',    path: 'This PC' },
      { icon: '💽', label: 'Local Disk (C:)', path: 'C:\\' },
      { icon: '📁', label: 'tools',      path: 'C:\\tools' },
      { icon: '📁', label: 'Windows',    path: 'C:\\Windows' },
    ],
  },
];

const TOOLBAR_TABS = ['Home', 'Share', 'View'];

export default function FileExplorer({ onOpenFile }: FileExplorerProps) {
  const [currentPath, setCurrentPath] = useState('C:\\Users\\jsmith\\Desktop');
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [history, setHistory] = useState<string[]>([]);
  const [forwardHistory, setForwardHistory] = useState<string[]>([]);
  const [viewMode, setViewMode] = useState<ViewMode>('details');
  const [sortKey, setSortKey] = useState<SortKey>('name');
  const [sortAsc, setSortAsc] = useState(true);
  const [showHidden, setShowHidden] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState('Home');
  const [hoveredSidebar, setHoveredSidebar] = useState<string | null>(null);
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);
  const [hoveredBtn, setHoveredBtn] = useState<string | null>(null);
  const [statusMsg, setStatusMsg] = useState('');
  const contentRef = useRef<HTMLDivElement>(null);

  const navigate = useCallback((path: string) => {
    setHistory(h => [...h, currentPath]);
    setForwardHistory([]);
    setCurrentPath(path);
    setSelectedItems(new Set());
    setSearchQuery('');
  }, [currentPath]);

  const navigateBack = () => {
    if (!history.length) return;
    const prev = history[history.length - 1];
    setForwardHistory(f => [...f, currentPath]);
    setHistory(h => h.slice(0, -1));
    setCurrentPath(prev);
    setSelectedItems(new Set());
  };

  const navigateForward = () => {
    if (!forwardHistory.length) return;
    const next = forwardHistory[forwardHistory.length - 1];
    setHistory(h => [...h, currentPath]);
    setForwardHistory(f => f.slice(0, -1));
    setCurrentPath(next);
    setSelectedItems(new Set());
  };

  const navigateUp = () => navigate(parentPath(currentPath));

  const rawEntries = VFS[currentPath] ?? [];
  let entries = rawEntries.filter(e => showHidden || !e.hidden);
  if (searchQuery.trim()) {
    entries = entries.filter(e => e.name.toLowerCase().includes(searchQuery.toLowerCase()));
  }
  entries = [...entries].sort((a, b) => {
    // folders first
    if (a.type !== b.type) {
      if (a.type === 'folder' || a.type === 'drive') return -1;
      if (b.type === 'folder' || b.type === 'drive') return 1;
    }
    let cmp = 0;
    if (sortKey === 'name') cmp = a.name.localeCompare(b.name);
    else if (sortKey === 'size') cmp = (a.size || '').localeCompare(b.size || '');
    else if (sortKey === 'modified') cmp = (a.modified || '').localeCompare(b.modified || '');
    else if (sortKey === 'type') cmp = a.type.localeCompare(b.type);
    return sortAsc ? cmp : -cmp;
  });

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) setSortAsc(a => !a);
    else { setSortKey(key); setSortAsc(true); }
  };

  const selectedList = entries.filter(e => selectedItems.has(e.path));
  useEffect(() => {
    if (selectedItems.size === 0) setStatusMsg(`${entries.length} items`);
    else if (selectedItems.size === 1) {
      const e = entries.find(x => selectedItems.has(x.path));
      setStatusMsg(e ? `${e.name}${e.size ? '   |   ' + e.size : ''}   |   ${e.modified}` : `${selectedItems.size} item selected`);
    } else {
      setStatusMsg(`${selectedItems.size} items selected`);
    }
  }, [selectedItems, entries]);

  const c: Record<string, React.CSSProperties> = {
    root: {
      display: 'flex', flexDirection: 'column', width: '100%', height: '100%',
      background: '#202020', color: '#e8e8e8',
      fontFamily: '"Segoe UI", system-ui, sans-serif', fontSize: 13,
      userSelect: 'none',
    },
    // ── Ribbon tabs ─────────────────────────────────────────────────────────
    tabRow: {
      height: 28, display: 'flex', flexDirection: 'row', alignItems: 'flex-end',
      background: '#2a2a2a', paddingLeft: 8,
      borderBottom: '1px solid #3a3a3a', flexShrink: 0,
    },
    tab: (active: boolean): React.CSSProperties => ({
      padding: '5px 14px 4px', fontSize: 12, cursor: 'default',
      color: active ? '#e8e8e8' : 'rgba(232,232,232,0.6)',
      borderBottom: active ? '2px solid #0078D4' : '2px solid transparent',
      marginRight: 2,
    }),
    // ── Ribbon toolbar ───────────────────────────────────────────────────────
    ribbon: {
      height: 48, background: '#272727', display: 'flex', flexDirection: 'row',
      alignItems: 'center', padding: '0 8px', gap: 2,
      borderBottom: '1px solid #3a3a3a', flexShrink: 0,
    },
    ribbonSep: {
      width: 1, height: 32, background: 'rgba(255,255,255,0.1)', margin: '0 6px',
    },
    ribbonBtn: (id: string): React.CSSProperties => ({
      display: 'flex', flexDirection: 'column', alignItems: 'center',
      padding: '4px 8px', borderRadius: 4, cursor: 'default', minWidth: 48,
      background: hoveredBtn === id ? 'rgba(255,255,255,0.08)' : 'transparent',
      border: 'none', color: '#e8e8e8',
    }),
    ribbonBtnIcon: { fontSize: 18, lineHeight: 1 },
    ribbonBtnLabel: { fontSize: 10, marginTop: 2, color: 'rgba(232,232,232,0.8)', whiteSpace: 'nowrap' as const },
    viewBtn: (mode: ViewMode): React.CSSProperties => ({
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      width: 28, height: 28, borderRadius: 4, cursor: 'default', border: 'none',
      background: viewMode === mode ? 'rgba(0,120,212,0.35)' : hoveredBtn === `view-${mode}` ? 'rgba(255,255,255,0.08)' : 'transparent',
      color: '#e8e8e8', fontSize: 15,
    }),
    // ── Nav bar ──────────────────────────────────────────────────────────────
    navbar: {
      height: 36, background: '#2a2a2a', display: 'flex', flexDirection: 'row',
      alignItems: 'center', padding: '0 8px', gap: 4,
      borderBottom: '1px solid #3a3a3a', flexShrink: 0,
    },
    navBtn: (disabled: boolean, id: string): React.CSSProperties => ({
      width: 28, height: 28, border: 'none', borderRadius: 4,
      background: hoveredBtn === id && !disabled ? 'rgba(255,255,255,0.08)' : 'transparent',
      color: disabled ? 'rgba(232,232,232,0.3)' : '#e8e8e8',
      fontSize: 14, cursor: 'default',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }),
    breadcrumbBar: {
      flex: 1, background: 'rgba(255,255,255,0.05)',
      border: '1px solid rgba(255,255,255,0.12)', borderRadius: 4,
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      padding: '0 8px', height: 26, overflow: 'hidden',
    },
    crumb: (last: boolean): React.CSSProperties => ({
      color: last ? '#e8e8e8' : 'rgba(232,232,232,0.6)',
      fontSize: 12, whiteSpace: 'nowrap' as const, cursor: last ? 'default' : 'default',
    }),
    crumbSep: {
      color: 'rgba(232,232,232,0.4)', fontSize: 12, padding: '0 3px',
    },
    searchInput: {
      width: 180, background: 'rgba(255,255,255,0.06)',
      border: '1px solid rgba(255,255,255,0.12)', borderRadius: 4,
      padding: '3px 8px', fontSize: 12, color: '#e8e8e8', outline: 'none',
      height: 26, boxSizing: 'border-box' as const,
    },
    // ── Body ─────────────────────────────────────────────────────────────────
    body: { display: 'flex', flexDirection: 'row', flex: 1, overflow: 'hidden', minHeight: 0 },
    sidebar: {
      width: 196, background: '#1e1e1e', borderRight: '1px solid #333',
      overflowY: 'auto' as const, flexShrink: 0, padding: '6px 0',
    },
    sidebarGroupLabel: {
      fontSize: 11, fontWeight: 600, color: 'rgba(232,232,232,0.5)',
      padding: '10px 12px 3px 12px', letterSpacing: '0.03em',
      textTransform: 'uppercase' as const,
    },
    sidebarItem: (path: string): React.CSSProperties => ({
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      gap: 7, padding: '5px 10px 5px 20px',
      background: hoveredSidebar === path
        ? 'rgba(255,255,255,0.07)'
        : currentPath === path
        ? 'rgba(0,120,212,0.25)'
        : 'transparent',
      cursor: 'default', borderRadius: 4, margin: '0 4px',
    }),
    // ── Content ───────────────────────────────────────────────────────────────
    contentArea: {
      flex: 1, overflowY: 'auto' as const, overflowX: 'hidden' as const,
      background: '#202020',
    },
    // details view
    detailsTable: {
      width: '100%', borderCollapse: 'collapse' as const, fontSize: 13,
    },
    detailsHead: {
      position: 'sticky' as const, top: 0, zIndex: 1,
      background: '#2a2a2a', height: 26,
    },
    detailsHeadCell: (key: SortKey): React.CSSProperties => ({
      textAlign: 'left' as const, padding: '0 8px', fontWeight: 500,
      fontSize: 12, color: sortKey === key ? '#0078D4' : 'rgba(232,232,232,0.7)',
      cursor: 'default', whiteSpace: 'nowrap' as const,
      borderBottom: '1px solid #3a3a3a', borderRight: '1px solid #333',
    }),
    detailsRow: (path: string): React.CSSProperties => ({
      height: 24,
      background: selectedItems.has(path)
        ? 'rgba(0,120,212,0.35)'
        : hoveredItem === path
        ? 'rgba(255,255,255,0.05)'
        : 'transparent',
      cursor: 'default',
    }),
    detailsCell: {
      padding: '0 8px', overflow: 'hidden' as const,
      textOverflow: 'ellipsis' as const, whiteSpace: 'nowrap' as const,
      borderBottom: '1px solid rgba(255,255,255,0.03)',
      maxWidth: 300,
    },
    // icons view
    iconGrid: {
      display: 'flex', flexWrap: 'wrap' as const,
      alignContent: 'flex-start', gap: 4, padding: 12,
    },
    iconItem: (path: string): React.CSSProperties => ({
      width: 80, height: 90, display: 'flex', flexDirection: 'column',
      alignItems: 'center', padding: 6, borderRadius: 4,
      cursor: 'default', boxSizing: 'border-box' as const,
      background: selectedItems.has(path)
        ? 'rgba(0,120,212,0.35)'
        : hoveredItem === path
        ? 'rgba(255,255,255,0.06)'
        : 'transparent',
      border: selectedItems.has(path)
        ? '1px solid rgba(0,120,212,0.6)'
        : '1px solid transparent',
    }),
    iconEmoji: { fontSize: 38, lineHeight: 1 },
    iconLabel: {
      fontSize: 11, color: '#e8e8e8', textAlign: 'center' as const,
      wordBreak: 'break-word' as const, maxWidth: 76,
      marginTop: 4, lineHeight: 1.3, overflow: 'hidden' as const,
      display: '-webkit-box' as const, WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' as const,
    },
    // list view
    listItem: (path: string): React.CSSProperties => ({
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      gap: 6, padding: '3px 12px', height: 22,
      background: selectedItems.has(path)
        ? 'rgba(0,120,212,0.35)'
        : hoveredItem === path
        ? 'rgba(255,255,255,0.05)'
        : 'transparent',
      cursor: 'default', borderRadius: 2,
    }),
    // ── Status bar ────────────────────────────────────────────────────────────
    statusBar: {
      height: 22, background: '#1e1e1e', borderTop: '1px solid #333',
      padding: '0 12px', fontSize: 11, color: 'rgba(232,232,232,0.5)',
      display: 'flex', alignItems: 'center', flexShrink: 0,
      gap: 8,
    },
  };

  const handleItemClick = (e: React.MouseEvent, path: string) => {
    e.stopPropagation();
    if (e.ctrlKey) {
      setSelectedItems(prev => {
        const next = new Set(prev);
        next.has(path) ? next.delete(path) : next.add(path);
        return next;
      });
    } else if (e.shiftKey && selectedItems.size > 0) {
      const paths = entries.map(x => x.path);
      const lastSelected = [...selectedItems].pop()!;
      const a = paths.indexOf(lastSelected);
      const b = paths.indexOf(path);
      const range = paths.slice(Math.min(a, b), Math.max(a, b) + 1);
      setSelectedItems(new Set(range));
    } else {
      setSelectedItems(new Set([path]));
    }
  };

  const handleItemDblClick = (entry: VFSEntry) => {
    if (entry.type === 'folder' || entry.type === 'drive') {
      navigate(entry.path);
    } else {
      onOpenFile?.(entry.path);
    }
  };

  const clearSelection = () => setSelectedItems(new Set());

  const sortArrow = (key: SortKey) =>
    sortKey === key ? (sortAsc ? ' ▲' : ' ▼') : '';

  const crumbs = formatBreadcrumbs(currentPath);

  return (
    <div style={c.root} onClick={clearSelection}>
      {/* Tab row */}
      <div style={c.tabRow}>
        {TOOLBAR_TABS.map(tab => (
          <div key={tab} style={c.tab(activeTab === tab)} onClick={() => setActiveTab(tab)}>
            {tab}
          </div>
        ))}
      </div>

      {/* Ribbon */}
      <div style={c.ribbon}>
        {activeTab === 'Home' && (<>
          <button style={c.ribbonBtn('pin')}
            onMouseOver={() => setHoveredBtn('pin')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>📌</span>
            <span style={c.ribbonBtnLabel}>Pin to Quick<br/>access</span>
          </button>
          <button style={c.ribbonBtn('copy')}
            onMouseOver={() => setHoveredBtn('copy')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>📋</span>
            <span style={c.ribbonBtnLabel}>Copy</span>
          </button>
          <button style={c.ribbonBtn('paste')}
            onMouseOver={() => setHoveredBtn('paste')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>📄</span>
            <span style={c.ribbonBtnLabel}>Paste</span>
          </button>
          <div style={c.ribbonSep} />
          <button style={c.ribbonBtn('move')}
            onMouseOver={() => setHoveredBtn('move')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>✂️</span>
            <span style={c.ribbonBtnLabel}>Move to</span>
          </button>
          <button style={c.ribbonBtn('delete')}
            onMouseOver={() => setHoveredBtn('delete')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>🗑️</span>
            <span style={c.ribbonBtnLabel}>Delete</span>
          </button>
          <button style={c.ribbonBtn('rename')}
            onMouseOver={() => setHoveredBtn('rename')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>✏️</span>
            <span style={c.ribbonBtnLabel}>Rename</span>
          </button>
          <div style={c.ribbonSep} />
          <button style={c.ribbonBtn('prop')}
            onMouseOver={() => setHoveredBtn('prop')} onMouseOut={() => setHoveredBtn(null)}>
            <span style={c.ribbonBtnIcon}>🛈</span>
            <span style={c.ribbonBtnLabel}>Properties</span>
          </button>
        </>)}
        {activeTab === 'View' && (<>
          <span style={{ fontSize: 11, color: 'rgba(232,232,232,0.5)', marginRight: 6 }}>Layout</span>
          {(['icons','list','details'] as ViewMode[]).map(m => (
            <button key={m} style={c.viewBtn(m)}
              onMouseOver={() => setHoveredBtn(`view-${m}`)} onMouseOut={() => setHoveredBtn(null)}
              onClick={() => setViewMode(m)}
              title={m.charAt(0).toUpperCase() + m.slice(1)}
            >
              {m === 'icons' ? '⊞' : m === 'list' ? '☰' : '⊟'}
            </button>
          ))}
          <div style={c.ribbonSep} />
          <span style={{ fontSize: 11, color: 'rgba(232,232,232,0.5)', marginRight: 4 }}>Show/hide</span>
          <button style={c.ribbonBtn('hidden')}
            onMouseOver={() => setHoveredBtn('hidden')} onMouseOut={() => setHoveredBtn(null)}
            onClick={() => setShowHidden(h => !h)}
          >
            <span style={c.ribbonBtnIcon}>{showHidden ? '👁' : '🚫'}</span>
            <span style={c.ribbonBtnLabel}>Hidden<br/>items</span>
          </button>
        </>)}
        {activeTab === 'Share' && (
          <span style={{ fontSize: 12, color: 'rgba(232,232,232,0.4)', padding: '0 8px' }}>
            Select items to share.
          </span>
        )}
      </div>

      {/* Nav bar */}
      <div style={c.navbar}>
        <button style={c.navBtn(!history.length, 'back')}
          onMouseOver={() => setHoveredBtn('back')} onMouseOut={() => setHoveredBtn(null)}
          onClick={navigateBack} disabled={!history.length}>
          ‹
        </button>
        <button style={c.navBtn(!forwardHistory.length, 'fwd')}
          onMouseOver={() => setHoveredBtn('fwd')} onMouseOut={() => setHoveredBtn(null)}
          onClick={navigateForward} disabled={!forwardHistory.length}>
          ›
        </button>
        <button style={c.navBtn(currentPath === 'This PC', 'up')}
          onMouseOver={() => setHoveredBtn('up')} onMouseOut={() => setHoveredBtn(null)}
          onClick={navigateUp} disabled={currentPath === 'This PC'} title="Up one level">
          ↑
        </button>

        <div style={c.breadcrumbBar}>
          {crumbs.map((crumb, i) => (
            <React.Fragment key={crumb.path}>
              {i > 0 && <span style={c.crumbSep}>›</span>}
              <span
                style={c.crumb(i === crumbs.length - 1)}
                onClick={i < crumbs.length - 1 ? () => navigate(crumb.path) : undefined}
                onMouseOver={e => { if (i < crumbs.length - 1) (e.currentTarget.style.color = '#0078D4'); }}
                onMouseOut={e => { (e.currentTarget.style.color = i === crumbs.length - 1 ? '#e8e8e8' : 'rgba(232,232,232,0.6)'); }}
              >
                {crumb.label}
              </span>
            </React.Fragment>
          ))}
        </div>

        <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
          <span style={{ position: 'absolute', left: 8, fontSize: 12, color: 'rgba(232,232,232,0.4)', pointerEvents: 'none' }}>🔍</span>
          <input
            style={{ ...c.searchInput, paddingLeft: 26 }}
            placeholder={`Search ${crumbs[crumbs.length - 1]?.label}`}
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            onClick={e => e.stopPropagation()}
          />
        </div>
      </div>

      {/* Body */}
      <div style={c.body} onClick={clearSelection}>
        {/* Sidebar */}
        <div style={c.sidebar}>
          {SIDEBAR_GROUPS.map(group => (
            <div key={group.label}>
              <div style={c.sidebarGroupLabel}>{group.label}</div>
              {group.items.map(item => (
                <div
                  key={item.path}
                  style={c.sidebarItem(item.path)}
                  onMouseOver={() => setHoveredSidebar(item.path)}
                  onMouseOut={() => setHoveredSidebar(null)}
                  onClick={e => { e.stopPropagation(); navigate(item.path); }}
                >
                  <span style={{ fontSize: 14 }}>{item.icon}</span>
                  <span style={{ fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {item.label}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>

        {/* Content */}
        <div
          ref={contentRef}
          style={c.contentArea}
          onClick={e => { e.stopPropagation(); clearSelection(); }}
        >
          {entries.length === 0 && !searchQuery && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'rgba(232,232,232,0.3)', fontSize: 13 }}>
              This folder is empty.
            </div>
          )}
          {entries.length === 0 && searchQuery && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'rgba(232,232,232,0.3)', fontSize: 13 }}>
              No items match your search.
            </div>
          )}

          {/* Details view */}
          {viewMode === 'details' && entries.length > 0 && (
            <table style={c.detailsTable}>
              <thead style={c.detailsHead}>
                <tr>
                  <th style={{ ...c.detailsHeadCell('name'), width: '45%' }}
                    onClick={() => toggleSort('name')}>
                    Name{sortArrow('name')}
                  </th>
                  <th style={{ ...c.detailsHeadCell('modified'), width: '28%' }}
                    onClick={() => toggleSort('modified')}>
                    Date modified{sortArrow('modified')}
                  </th>
                  <th style={{ ...c.detailsHeadCell('type'), width: '15%' }}
                    onClick={() => toggleSort('type')}>
                    Type{sortArrow('type')}
                  </th>
                  <th style={{ ...c.detailsHeadCell('size'), width: '12%' }}
                    onClick={() => toggleSort('size')}>
                    Size{sortArrow('size')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {entries.map(entry => (
                  <tr
                    key={entry.path}
                    style={c.detailsRow(entry.path)}
                    onClick={e => handleItemClick(e, entry.path)}
                    onDoubleClick={() => handleItemDblClick(entry)}
                    onMouseOver={() => setHoveredItem(entry.path)}
                    onMouseOut={() => setHoveredItem(null)}
                  >
                    <td style={c.detailsCell}>
                      <span style={{ marginRight: 7, fontSize: 14 }}>{fileIcon(entry.name, entry.type)}</span>
                      <span style={{ opacity: entry.hidden ? 0.5 : 1 }}>{entry.name}</span>
                    </td>
                    <td style={{ ...c.detailsCell, color: 'rgba(232,232,232,0.65)' }}>{entry.modified}</td>
                    <td style={{ ...c.detailsCell, color: 'rgba(232,232,232,0.65)' }}>
                      {entry.type === 'folder' ? 'File folder' :
                       entry.type === 'drive'  ? 'Local Disk' :
                       entry.name.includes('.') ? entry.name.split('.').pop()!.toUpperCase() + ' File' : 'File'}
                    </td>
                    <td style={{ ...c.detailsCell, color: 'rgba(232,232,232,0.65)', textAlign: 'right' as const }}>
                      {entry.type === 'folder' || entry.type === 'drive' ? '' : entry.size}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {/* Icons view */}
          {viewMode === 'icons' && entries.length > 0 && (
            <div style={c.iconGrid} onClick={e => e.stopPropagation()}>
              {entries.map(entry => (
                <div
                  key={entry.path}
                  style={c.iconItem(entry.path)}
                  onClick={e => handleItemClick(e, entry.path)}
                  onDoubleClick={() => handleItemDblClick(entry)}
                  onMouseOver={() => setHoveredItem(entry.path)}
                  onMouseOut={() => setHoveredItem(null)}
                >
                  <span style={{ ...c.iconEmoji, opacity: entry.hidden ? 0.45 : 1 }}>
                    {fileIcon(entry.name, entry.type)}
                  </span>
                  <span style={c.iconLabel}>{entry.name}</span>
                </div>
              ))}
            </div>
          )}

          {/* List view */}
          {viewMode === 'list' && entries.length > 0 && (
            <div style={{ padding: '4px 0' }} onClick={e => e.stopPropagation()}>
              {entries.map(entry => (
                <div
                  key={entry.path}
                  style={c.listItem(entry.path)}
                  onClick={e => handleItemClick(e, entry.path)}
                  onDoubleClick={() => handleItemDblClick(entry)}
                  onMouseOver={() => setHoveredItem(entry.path)}
                  onMouseOut={() => setHoveredItem(null)}
                >
                  <span style={{ fontSize: 15, opacity: entry.hidden ? 0.45 : 1 }}>
                    {fileIcon(entry.name, entry.type)}
                  </span>
                  <span style={{ opacity: entry.hidden ? 0.5 : 1 }}>{entry.name}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Status bar */}
      <div style={c.statusBar}>
        <span>{statusMsg}</span>
      </div>
    </div>
  );
}

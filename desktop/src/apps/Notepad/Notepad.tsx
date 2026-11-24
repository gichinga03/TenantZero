import React from 'react';

interface NotepadProps {
  filePath?: string;
}

const FILE_CONTENTS: Record<string, string> = {
  'C:\\Users\\jsmith\\Desktop\\itwiki.txt': `IT Wiki — Internal Use Only
===========================

Server Access Credentials (IT Support rotation — update quarterly)
-------------------------------------------------------------------
Host        : FILESVR01.meridian.local
Account     : MERIDIAN\\awalker
Password    : Autumn2024!
Purpose     : File share administration, backup verification

Note: rotate before Q1 2025 audit window.`,
  'C:\\tools\\PowerView.ps1': `# PowerView — AD Enumeration Script
# Import with: Import-Module .\\PowerView.ps1
# Then use Get-DomainUser, Get-ObjectAcl, etc.
# (Simulated — commands run through TenantZero engine)`,
};

function getFilename(filePath?: string): string {
  if (!filePath) return 'Untitled';
  const parts = filePath.split('\\');
  return parts[parts.length - 1] || 'Untitled';
}

export default function Notepad({ filePath }: NotepadProps) {
  let content = '';
  if (filePath) {
    content = filePath in FILE_CONTENTS
      ? FILE_CONTENTS[filePath]
      : `File not found: ${filePath}`;
  }

  const filename = getFilename(filePath);

  const s: Record<string, React.CSSProperties> = {
    root: {
      display: 'flex', flexDirection: 'column',
      width: '100%', height: '100%',
      background: '#1e1e1e', color: '#d4d4d4',
      fontFamily: "'Consolas', 'Courier New', monospace",
    },
    menuBar: {
      height: 30, background: '#2b2b2b',
      borderBottom: '1px solid rgba(255,255,255,0.08)',
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      padding: '0 4px', gap: 2, flexShrink: 0,
    },
    menuItem: {
      padding: '4px 8px', fontSize: 13,
      color: 'rgba(255,255,255,0.8)', cursor: 'default',
      borderRadius: 3, userSelect: 'none' as const,
    },
    textArea: {
      flex: 1, padding: '12px 16px',
      fontFamily: "'Consolas', 'Courier New', monospace",
      fontSize: 14, lineHeight: 1.6,
      color: '#d4d4d4', background: '#1e1e1e',
      whiteSpace: 'pre-wrap' as const,
      overflowY: 'auto' as const,
      wordBreak: 'break-word' as const,
      margin: 0,
    },
    statusBar: {
      height: 24, background: '#007acc',
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      padding: '0 12px', justifyContent: 'space-between',
      fontSize: 12, color: '#ffffff', flexShrink: 0,
      userSelect: 'none' as const,
    },
  };

  const MENU_ITEMS = ['File', 'Edit', 'View', 'Help'];

  return (
    <div style={s.root}>
      {/* Menu bar */}
      <div style={s.menuBar}>
        {MENU_ITEMS.map(item => (
          <span
            key={item}
            style={s.menuItem}
            onMouseOver={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.1)')}
            onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
          >
            {item}
          </span>
        ))}
      </div>

      {/* Content */}
      <pre style={s.textArea}>{content}</pre>

      {/* Status bar */}
      <div style={s.statusBar}>
        <span>Ln 1, Col 1</span>
        <span>{filename}&nbsp;&nbsp;UTF-8&nbsp;&nbsp;Windows (CRLF)</span>
      </div>
    </div>
  );
}
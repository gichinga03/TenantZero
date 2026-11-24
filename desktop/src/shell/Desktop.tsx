import React, { useState, useEffect } from 'react';

interface DesktopProps {
  onOpenApp: (appId: string) => void;
}

interface IconDef {
  id: string;
  label: string;
  icon: string;        // emoji OR '/path.ext'
}

const ICONS: IconDef[] = [
  { id: 'fileexplorer', label: 'File Explorer', icon: '📁' },
  { id: 'alloynet',     label: 'AlloyNet C2',   icon: '/Alloynet.jpeg' },
  { id: 'bloodhound',   label: 'BloodHound',    icon: '🐕' },
];

const MENU_ITEMS = ['View', 'Sort by', 'Refresh', null, 'Display settings', 'Personalize'];

function DesktopIcon({ icon }: { icon: string }) {
  const isImg = icon.startsWith('/') || icon.startsWith('http');
  if (isImg) {
    return (
      <img
        src={icon}
        draggable={false}
        style={{
          width: 44, height: 44,
          objectFit: 'contain',
          borderRadius: 4,
          pointerEvents: 'none',
          display: 'block',
        }}
      />
    );
  }
  return (
    <span style={{ fontSize: 36, lineHeight: 1, pointerEvents: 'none' }}>
      {icon}
    </span>
  );
}

export default function Desktop({ onOpenApp }: DesktopProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const [ctx, setCtx] = useState<{ x: number; y: number } | null>(null);

  useEffect(() => {
    const close = () => setCtx(null);
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, []);

  return (
    <div
      style={{
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 48,
        backgroundImage: 'url("/Walpaper.png")',
        backgroundSize: 'cover', backgroundPosition: 'center',
        overflow: 'hidden',
      }}
      onContextMenu={e => { e.preventDefault(); setSelected(null); setCtx({ x: e.clientX, y: e.clientY }); }}
      onClick={() => { setSelected(null); setCtx(null); }}
    >
      <div style={{
        position: 'absolute', top: 12, left: 12,
        display: 'flex', flexDirection: 'column', gap: 4,
      }}>
        {ICONS.map(def => (
          <div
            key={def.id}
            onClick={e => { e.stopPropagation(); setCtx(null); setSelected(def.id); }}
            onDoubleClick={e => { e.stopPropagation(); onOpenApp(def.id); }}
            style={{
              width: 76, height: 86,
              display: 'flex', flexDirection: 'column',
              alignItems: 'center', justifyContent: 'flex-start',
              paddingTop: 8,
              borderRadius: 6, cursor: 'default',
              background: selected === def.id
                ? 'rgba(0,120,212,0.35)'
                : 'rgba(0,0,0,0.0)',
              border: selected === def.id
                ? '1px solid rgba(0,120,212,0.7)'
                : '1px solid transparent',
              backdropFilter: selected === def.id ? 'blur(4px)' : 'none',
              boxSizing: 'border-box' as const,
              userSelect: 'none' as const,
              transition: 'background 0.1s',
            }}
            onMouseEnter={e => {
              if (selected !== def.id)
                e.currentTarget.style.background = 'rgba(255,255,255,0.08)';
            }}
            onMouseLeave={e => {
              if (selected !== def.id)
                e.currentTarget.style.background = 'rgba(0,0,0,0.0)';
            }}
          >
            <DesktopIcon icon={def.icon} />
            <span style={{
              color: '#ffffff', fontSize: 11,
              textAlign: 'center' as const,
              marginTop: 5, lineHeight: '1.2',
              maxWidth: 68, overflow: 'hidden',
              wordBreak: 'break-word' as const,
              textShadow: '0 1px 4px rgba(0,0,0,0.95)',
              fontFamily: 'Segoe UI, sans-serif',
              fontWeight: 500,
              pointerEvents: 'none',
            }}>
              {def.label}
            </span>
          </div>
        ))}
      </div>

      {ctx && (
        <div
          onMouseDown={e => e.stopPropagation()}
          style={{
            position: 'fixed', left: ctx.x, top: ctx.y,
            background: 'rgba(28,28,28,0.97)',
            backdropFilter: 'blur(20px)',
            border: '1px solid rgba(255,255,255,0.12)',
            borderRadius: 8, padding: '4px 0',
            minWidth: 200, zIndex: 9999,
            fontFamily: 'Segoe UI, sans-serif',
          }}
        >
          {MENU_ITEMS.map((item, i) =>
            item === null
              ? <div key={i} style={{ height: 1, background: 'rgba(255,255,255,0.1)', margin: '4px 0' }} />
              : (
                <div key={item}
                  style={{ padding: '7px 16px', fontSize: 13, color: '#fff', cursor: 'default', whiteSpace: 'nowrap' as const }}
                  onMouseOver={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.1)')}
                  onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
                >
                  {item}
                </div>
              )
          )}
        </div>
      )}
    </div>
  );
}

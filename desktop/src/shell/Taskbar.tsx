import React, { useState, useEffect } from 'react';

interface TaskbarProps {
  openWindows: Array<{ id: string; title: string; minimized: boolean }>;
  activeWindowId: string | null;
  onWindowClick: (id: string) => void;
  onStartClick: () => void;
}

function WindowsLogo() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="0"  y="0"  width="8" height="8" fill="#F25022" />
      <rect x="10" y="0"  width="8" height="8" fill="#7FBA00" />
      <rect x="0"  y="10" width="8" height="8" fill="#00A4EF" />
      <rect x="10" y="10" width="8" height="8" fill="#FFB900" />
    </svg>
  );
}

function WifiIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M8 12a1 1 0 1 1 0 2 1 1 0 0 1 0-2z" fill="rgba(255,255,255,0.7)" />
      <path d="M5.2 9.8a4 4 0 0 1 5.6 0" stroke="rgba(255,255,255,0.7)" strokeWidth="1.4" strokeLinecap="round" fill="none" />
      <path d="M2.8 7.4a7 7 0 0 1 10.4 0" stroke="rgba(255,255,255,0.7)" strokeWidth="1.4" strokeLinecap="round" fill="none" />
    </svg>
  );
}

function VolumeIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M3 6h2l3-3v10l-3-3H3V6z" fill="rgba(255,255,255,0.7)" />
      <path d="M11 5a4 4 0 0 1 0 6" stroke="rgba(255,255,255,0.7)" strokeWidth="1.4" strokeLinecap="round" fill="none" />
    </svg>
  );
}

export default function Taskbar({ openWindows, activeWindowId, onWindowClick, onStartClick }: TaskbarProps) {
  const [now, setNow] = useState(new Date());

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  const timeStr = now.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
  const dateStr = now.toLocaleDateString('en-US', { month: '2-digit', day: '2-digit', year: 'numeric' });

  const styles: Record<string, React.CSSProperties> = {
    bar: {
      position: 'fixed', bottom: 0, left: 0, right: 0, height: 48,
      background: 'rgba(32,32,32,0.85)',
      backdropFilter: 'blur(20px)',
      borderTop: '1px solid rgba(255,255,255,0.08)',
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      zIndex: 10000,
    },
    left: {
      flex: '0 0 auto', display: 'flex', alignItems: 'center', paddingLeft: 4,
    },
    startBtn: {
      width: 40, height: 40, borderRadius: 4,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'transparent', border: 'none', cursor: 'default',
    },
    center: {
      position: 'absolute', left: '50%', transform: 'translateX(-50%)',
      display: 'flex', flexDirection: 'row', alignItems: 'center',
    },
    winBtn: (id: string): React.CSSProperties => ({
      position: 'relative',
      height: 40, minWidth: 48, padding: '0 8px', margin: '0 2px',
      background: id === activeWindowId ? 'rgba(255,255,255,0.15)' : 'transparent',
      borderRadius: 4, border: 'none', cursor: 'default',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      color: '#ffffff',
    }),
    winTitle: {
      fontSize: 12, maxWidth: 120,
      overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' as const,
      fontFamily: 'Segoe UI, sans-serif',
    },
    indicator: (id: string): React.CSSProperties => ({
      position: 'absolute', bottom: 2,
      left: '50%', transform: 'translateX(-50%)',
      width: 3, height: 3, borderRadius: '2px 2px 0 0',
      background: id === activeWindowId ? '#0078D4' : 'transparent',
    }),
    right: {
      position: 'absolute', right: 0,
      display: 'flex', flexDirection: 'row', alignItems: 'center',
      paddingRight: 8, gap: 8, height: 48,
    },
    clock: {
      display: 'flex', flexDirection: 'column', alignItems: 'flex-end',
      paddingRight: 8,
    },
    clockTime: {
      color: '#ffffff', fontSize: 12,
      fontWeight: 500, fontFamily: 'Segoe UI, sans-serif',
      lineHeight: 1.3,
    },
    clockDate: {
      color: 'rgba(255,255,255,0.70)', fontSize: 11,
      fontFamily: 'Segoe UI, sans-serif', lineHeight: 1.3,
    },
  };

  return (
    <div style={styles.bar}>
      <div style={styles.left}>
        <button
          style={styles.startBtn}
          onClick={onStartClick}
          onMouseOver={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.1)')}
          onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
        >
          <WindowsLogo />
        </button>
      </div>

      <div style={styles.center}>
        {openWindows.map(w => (
          <button
            key={w.id}
            style={styles.winBtn(w.id)}
            onClick={() => onWindowClick(w.id)}
            onMouseOver={e => {
              if (w.id !== activeWindowId)
                e.currentTarget.style.background = 'rgba(255,255,255,0.1)';
            }}
            onMouseOut={e => {
              if (w.id !== activeWindowId)
                e.currentTarget.style.background = 'transparent';
            }}
          >
            <span style={styles.winTitle}>{w.title}</span>
            <span style={styles.indicator(w.id)} />
          </button>
        ))}
      </div>

      <div style={styles.right}>
        <WifiIcon />
        <VolumeIcon />
        <div style={styles.clock}>
          <span style={styles.clockTime}>{timeStr}</span>
          <span style={styles.clockDate}>{dateStr}</span>
        </div>
      </div>
    </div>
  );
}
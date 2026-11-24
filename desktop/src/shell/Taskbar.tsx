import React, { useState, useEffect, useRef } from 'react';

interface TaskbarProps {
  openWindows: Array<{ id: string; title: string; minimized: boolean }>;
  activeWindowId: string | null;
  onWindowClick: (id: string) => void;
  onStartClick: () => void;
  onLock: () => void;
  onOpenApp: (appId: string) => void;
}

// ── SVG icons ────────────────────────────────────────────────────
function WindowsLogo() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
      <rect x="0"  y="0"  width="8" height="8" fill="#F25022" />
      <rect x="10" y="0"  width="8" height="8" fill="#7FBA00" />
      <rect x="0"  y="10" width="8" height="8" fill="#00A4EF" />
      <rect x="10" y="10" width="8" height="8" fill="#FFB900" />
    </svg>
  );
}
function WifiIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      <path d="M8 12a1 1 0 1 1 0 2 1 1 0 0 1 0-2z" fill="rgba(255,255,255,0.85)" />
      <path d="M5.2 9.8a4 4 0 0 1 5.6 0" stroke="rgba(255,255,255,0.85)" strokeWidth="1.4" strokeLinecap="round" fill="none" />
      <path d="M2.8 7.4a7 7 0 0 1 10.4 0" stroke="rgba(255,255,255,0.85)" strokeWidth="1.4" strokeLinecap="round" fill="none" />
    </svg>
  );
}
function VolumeIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
      <path d="M3 6h2l3-3v10l-3-3H3V6z" fill="rgba(255,255,255,0.85)" />
      <path d="M11 5a4 4 0 0 1 0 6" stroke="rgba(255,255,255,0.85)" strokeWidth="1.4" strokeLinecap="round" fill="none" />
    </svg>
  );
}
function BatteryIcon() {
  return (
    <svg width="18" height="16" viewBox="0 0 18 16" fill="none">
      <rect x="1" y="4" width="14" height="8" rx="1.5" stroke="rgba(255,255,255,0.85)" strokeWidth="1.3" />
      <rect x="2.5" y="5.5" width="10" height="5" rx="0.5" fill="rgba(255,255,255,0.85)" />
      <path d="M15.5 7v2" stroke="rgba(255,255,255,0.85)" strokeWidth="1.3" strokeLinecap="round" />
    </svg>
  );
}

// ── Popup shell ───────────────────────────────────────────────────
function Popup({ children, style }: { children: React.ReactNode; style?: React.CSSProperties }) {
  return (
    <div style={{
      position: 'fixed',
      background: 'rgba(28,28,28,0.97)',
      backdropFilter: 'blur(24px)',
      border: '1px solid rgba(255,255,255,0.1)',
      borderRadius: 10,
      boxShadow: '0 8px 32px rgba(0,0,0,0.6)',
      zIndex: 20000,
      fontFamily: 'Segoe UI, sans-serif',
      ...style,
    }}>
      {children}
    </div>
  );
}

// ── Wifi popup ────────────────────────────────────────────────────
const WIFI_NETWORKS = [
  { ssid: 'MERIDIAN-CORP',  strength: 4, secured: true,  connected: true  },
  { ssid: 'MERIDIAN-GUEST', strength: 3, secured: true,  connected: false },
  { ssid: 'HP-PRINT-5F',    strength: 2, secured: false, connected: false },
  { ssid: 'AndroidAP_7E2',  strength: 1, secured: true,  connected: false },
];

function WifiStrengthIcon({ bars }: { bars: number }) {
  return (
    <svg width="18" height="14" viewBox="0 0 18 14" fill="none">
      {[1,2,3,4].map((b, i) => (
        <rect key={b}
          x={i * 4.5} y={14 - b * 3.2}
          width="3" height={b * 3.2}
          rx="1"
          fill={b <= bars ? 'rgba(255,255,255,0.9)' : 'rgba(255,255,255,0.2)'}
        />
      ))}
    </svg>
  );
}

function WifiPopup({ onClose }: { onClose: () => void }) {
  return (
    <Popup style={{ bottom: 56, right: 12, width: 320, padding: '14px 0 8px' }}>
      <div style={{ padding: '0 16px 10px', fontSize: 13, fontWeight: 600, color: '#fff' }}>
        Wi-Fi
      </div>
      {WIFI_NETWORKS.map(net => (
        <div key={net.ssid} style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '8px 16px', cursor: 'default',
        }}
          onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.07)'}
          onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
        >
          <WifiStrengthIcon bars={net.strength} />
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 13, color: net.connected ? '#60cdff' : '#cccccc' }}>
              {net.ssid} {net.connected && <span style={{ fontSize: 10, color: '#4ec9b0', marginLeft: 6 }}>Connected</span>}
            </div>
            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.4)', marginTop: 1 }}>
              {net.secured ? '🔒 Secured' : 'Open'}
            </div>
          </div>
        </div>
      ))}
      <div style={{ margin: '8px 16px 4px', height: 1, background: 'rgba(255,255,255,0.08)' }} />
      <div style={{ padding: '6px 16px', fontSize: 12, color: 'rgba(255,255,255,0.5)', cursor: 'default' }}>
        Network &amp; Internet settings
      </div>
    </Popup>
  );
}

// ── Volume popup ──────────────────────────────────────────────────
function VolumePopup() {
  const [vol, setVol] = useState(72);
  return (
    <Popup style={{ bottom: 56, right: 12, width: 300, padding: '16px 20px' }}>
      <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.5)', marginBottom: 10 }}>Volume</div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <VolumeIcon />
        <input type="range" min={0} max={100} value={vol}
          onChange={e => setVol(Number(e.target.value))}
          style={{ flex: 1, accentColor: '#60cdff', cursor: 'default' }}
        />
        <span style={{ fontSize: 12, color: '#cccccc', width: 28, textAlign: 'right' }}>{vol}</span>
      </div>
      <div style={{ marginTop: 14, height: 1, background: 'rgba(255,255,255,0.08)' }} />
      <div style={{ marginTop: 10, fontSize: 12, color: 'rgba(255,255,255,0.5)' }}>Output</div>
      <div style={{ marginTop: 6, fontSize: 13, color: '#cccccc' }}>Speakers (Realtek HD Audio)</div>
    </Popup>
  );
}

// ── Clock / calendar popup ────────────────────────────────────────
function CalendarPopup({ now }: { now: Date }) {
  const year  = now.getFullYear();
  const month = now.getMonth();
  const today = now.getDate();

  const firstDay = new Date(year, month, 1).getDay();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const cells: (number | null)[] = [
    ...Array(firstDay).fill(null),
    ...Array.from({ length: daysInMonth }, (_, i) => i + 1),
  ];
  while (cells.length % 7 !== 0) cells.push(null);

  const monthName = now.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
  const DOW = ['Su','Mo','Tu','We','Th','Fr','Sa'];

  return (
    <Popup style={{ bottom: 56, right: 8, width: 300, padding: '16px' }}>
      {/* Large time */}
      <div style={{ fontSize: 48, fontWeight: 200, color: '#fff', lineHeight: 1, letterSpacing: -1 }}>
        {now.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })}
      </div>
      <div style={{ fontSize: 14, color: 'rgba(255,255,255,0.6)', marginTop: 4, marginBottom: 16 }}>
        {now.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })}
      </div>

      <div style={{ height: 1, background: 'rgba(255,255,255,0.08)', marginBottom: 14 }} />

      {/* Calendar */}
      <div style={{ fontSize: 13, color: '#fff', fontWeight: 600, marginBottom: 10 }}>{monthName}</div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7,1fr)', gap: 2, textAlign: 'center' }}>
        {DOW.map(d => (
          <div key={d} style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', paddingBottom: 4 }}>{d}</div>
        ))}
        {cells.map((day, i) => (
          <div key={i} style={{
            fontSize: 12, padding: '4px 2px', borderRadius: 4,
            color: day === today ? '#000' : day ? '#ccc' : 'transparent',
            background: day === today ? '#60cdff' : 'transparent',
            fontWeight: day === today ? 700 : 400,
          }}>{day ?? ''}</div>
        ))}
      </div>
    </Popup>
  );
}

// ── Start menu ────────────────────────────────────────────────────
const START_PINNED = [
  { id: 'fileexplorer', label: 'File Explorer', icon: '📁' },
  { id: 'alloynet',     label: 'AlloyNet C2',   icon: '/Alloynet.jpeg', isImg: true },
  { id: 'bloodhound',   label: 'BloodHound',    icon: '🐕' },
  { id: 'notepad',      label: 'Notepad',       icon: '📝' },
  { id: 'settings',     label: 'Settings',      icon: '⚙️' },
  { id: 'terminal',     label: 'Terminal',      icon: '🖥️' },
];
const START_RECOMMENDED = [
  { label: 'itwiki.txt',          icon: '📝', sub: 'Desktop · 2 days ago' },
  { label: 'meeting_notes.txt',   icon: '📝', sub: 'Documents · 3 days ago' },
  { label: 'passwords_old.txt',   icon: '📝', sub: 'Documents · 7 weeks ago' },
  { label: 'meridian-vpn.ovpn',   icon: '🌐', sub: 'VPN Config · 6 days ago' },
];

function StartMenu({
  onOpenApp,
  onLock,
  onClose,
  user,
}: {
  onOpenApp: (id: string) => void;
  onLock: () => void;
  onClose: () => void;
  user: string;
}) {
  return (
    <Popup style={{
      bottom: 56, left: '50%', transform: 'translateX(-50%)',
      width: 620, padding: 0, overflow: 'hidden',
    }}>
      {/* Search bar */}
      <div style={{ padding: '20px 24px 14px' }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          background: 'rgba(255,255,255,0.07)',
          border: '1px solid rgba(255,255,255,0.12)',
          borderRadius: 6, padding: '7px 14px',
        }}>
          <span style={{ color: 'rgba(255,255,255,0.4)', fontSize: 15 }}>🔍</span>
          <span style={{ fontSize: 13, color: 'rgba(255,255,255,0.3)', fontFamily: 'Segoe UI, sans-serif' }}>
            Search for apps, files, settings…
          </span>
        </div>
      </div>

      {/* Pinned */}
      <div style={{ padding: '0 24px' }}>
        <div style={{ fontSize: 12, fontWeight: 600, color: 'rgba(255,255,255,0.6)', marginBottom: 12, fontFamily: 'Segoe UI, sans-serif' }}>
          Pinned
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6,1fr)', gap: 4, marginBottom: 20 }}>
          {START_PINNED.map(app => (
            <div key={app.id}
              onClick={() => { onOpenApp(app.id); onClose(); }}
              style={{
                display: 'flex', flexDirection: 'column', alignItems: 'center',
                padding: '10px 4px', borderRadius: 6, cursor: 'default',
                fontFamily: 'Segoe UI, sans-serif',
              }}
              onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.08)'}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
            >
              {app.isImg
                ? <img src={app.icon} style={{ width: 32, height: 32, objectFit: 'contain', borderRadius: 4, marginBottom: 6 }} draggable={false} />
                : <span style={{ fontSize: 28, lineHeight: 1, marginBottom: 6 }}>{app.icon}</span>
              }
              <span style={{ fontSize: 11, color: '#cccccc', textAlign: 'center', lineHeight: 1.3, maxWidth: 72, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {app.label}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Recommended */}
      <div style={{ padding: '0 24px', borderTop: '1px solid rgba(255,255,255,0.07)', paddingTop: 14 }}>
        <div style={{ fontSize: 12, fontWeight: 600, color: 'rgba(255,255,255,0.6)', marginBottom: 10, fontFamily: 'Segoe UI, sans-serif' }}>
          Recommended
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, marginBottom: 14 }}>
          {START_RECOMMENDED.map(item => (
            <div key={item.label}
              style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '8px 10px', borderRadius: 6, cursor: 'default',
                fontFamily: 'Segoe UI, sans-serif',
              }}
              onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.07)'}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
            >
              <span style={{ fontSize: 22 }}>{item.icon}</span>
              <div>
                <div style={{ fontSize: 12, color: '#cccccc' }}>{item.label}</div>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.4)', marginTop: 1 }}>{item.sub}</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Footer — user + power */}
      <div style={{
        display: 'flex', alignItems: 'center',
        padding: '12px 24px',
        background: 'rgba(0,0,0,0.25)',
        borderTop: '1px solid rgba(255,255,255,0.07)',
        fontFamily: 'Segoe UI, sans-serif',
      }}>
        {/* Avatar */}
        <div style={{
          width: 32, height: 32, borderRadius: '50%',
          background: '#0078D4', display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontSize: 14, fontWeight: 600, flexShrink: 0,
        }}>
          {user.replace(/.*\\/,'').charAt(0).toUpperCase()}
        </div>
        <span style={{ fontSize: 13, color: '#cccccc', marginLeft: 10, flex: 1 }}>
          {user.replace(/.*\\/,'')}
        </span>

        {/* Power menu */}
        <div style={{ display: 'flex', gap: 4 }}>
          {[
            { label: '🔒', title: 'Lock',     action: () => { onLock(); onClose(); } },
            { label: '⏽',  title: 'Sleep',    action: onClose },
            { label: '⏻',  title: 'Shut down', action: onClose },
            { label: '↺',  title: 'Restart',   action: onClose },
          ].map(btn => (
            <button key={btn.title} onClick={btn.action} title={btn.title} style={{
              width: 32, height: 32, borderRadius: 4,
              background: 'transparent', border: 'none',
              color: 'rgba(255,255,255,0.7)', fontSize: 16, cursor: 'pointer',
            }}
              onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
            >{btn.label}</button>
          ))}
        </div>
      </div>
    </Popup>
  );
}

// ── Main Taskbar ──────────────────────────────────────────────────
type Panel = 'start' | 'wifi' | 'volume' | 'clock' | null;

export default function Taskbar({
  openWindows, activeWindowId, onWindowClick, onStartClick, onLock, onOpenApp,
}: TaskbarProps) {
  const [now, setNow]       = useState(new Date());
  const [panel, setPanel]   = useState<Panel>(null);
  const barRef              = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  // Win+L shortcut
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'l' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        onLock();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onLock]);

  // Click outside → close panel
  useEffect(() => {
    if (!panel) return;
    const handler = (e: MouseEvent) => {
      if (barRef.current && !barRef.current.contains(e.target as Node)) {
        setPanel(null);
      }
    };
    // slight delay so the opening click doesn't immediately close
    const tid = setTimeout(() => document.addEventListener('mousedown', handler), 50);
    return () => { clearTimeout(tid); document.removeEventListener('mousedown', handler); };
  }, [panel]);

  const toggle = (p: Panel) => setPanel(prev => prev === p ? null : p);

  const timeStr = now.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
  const dateStr = now.toLocaleDateString('en-US', { month: '2-digit', day: '2-digit', year: 'numeric' });

  const iconBtn = (content: React.ReactNode, p: Panel, title?: string): React.CSSProperties => ({
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    padding: '0 8px', height: 48, cursor: 'default', border: 'none',
    background: panel === p ? 'rgba(255,255,255,0.12)' : 'transparent',
    borderRadius: 4,
  });

  return (
    <div ref={barRef}>
      {/* ── Popups ── */}
      {panel === 'start'  && (
        <StartMenu
          onOpenApp={id => { onOpenApp(id); setPanel(null); }}
          onLock={onLock}
          onClose={() => setPanel(null)}
          user="MERIDIAN\\jsmith"
        />
      )}
      {panel === 'wifi'   && <WifiPopup onClose={() => setPanel(null)} />}
      {panel === 'volume' && <VolumePopup />}
      {panel === 'clock'  && <CalendarPopup now={now} />}

      {/* ── Bar ── */}
      <div style={{
        position: 'fixed', bottom: 0, left: 0, right: 0, height: 48,
        background: 'rgba(32,32,32,0.88)',
        backdropFilter: 'blur(20px)',
        borderTop: '1px solid rgba(255,255,255,0.08)',
        display: 'flex', flexDirection: 'row', alignItems: 'center',
        zIndex: 10000,
      }}>
        {/* Start button */}
        <button
          onClick={() => { toggle('start'); onStartClick(); }}
          style={{
            width: 44, height: 44, borderRadius: 4, marginLeft: 4,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: panel === 'start' ? 'rgba(255,255,255,0.12)' : 'transparent',
            border: 'none', cursor: 'default',
          }}
          onMouseEnter={e => { if (panel !== 'start') e.currentTarget.style.background = 'rgba(255,255,255,0.1)'; }}
          onMouseLeave={e => { if (panel !== 'start') e.currentTarget.style.background = 'transparent'; }}
        >
          <WindowsLogo />
        </button>

        {/* Open windows — centred */}
        <div style={{
          position: 'absolute', left: '50%', transform: 'translateX(-50%)',
          display: 'flex', flexDirection: 'row', alignItems: 'center',
        }}>
          {openWindows.map(w => (
            <button key={w.id}
              onClick={() => onWindowClick(w.id)}
              style={{
                position: 'relative',
                height: 40, minWidth: 48, padding: '0 10px', margin: '0 2px',
                background: w.id === activeWindowId ? 'rgba(255,255,255,0.15)' : 'transparent',
                borderRadius: 4, border: 'none', cursor: 'default',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: '#ffffff',
              }}
              onMouseEnter={e => { if (w.id !== activeWindowId) e.currentTarget.style.background = 'rgba(255,255,255,0.1)'; }}
              onMouseLeave={e => { if (w.id !== activeWindowId) e.currentTarget.style.background = 'transparent'; }}
            >
              <span style={{ fontSize: 12, maxWidth: 120, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontFamily: 'Segoe UI, sans-serif' }}>
                {w.title}
              </span>
              <span style={{
                position: 'absolute', bottom: 2, left: '50%', transform: 'translateX(-50%)',
                width: w.id === activeWindowId ? 16 : 4, height: 3,
                borderRadius: 2, transition: 'width 0.15s',
                background: w.id === activeWindowId ? '#60cdff' : 'rgba(255,255,255,0.4)',
              }} />
            </button>
          ))}
        </div>

        {/* Right tray */}
        <div style={{
          position: 'absolute', right: 0,
          display: 'flex', flexDirection: 'row', alignItems: 'center',
          height: 48, paddingRight: 4, gap: 2,
        }}>
          {/* Battery */}
          <button style={iconBtn(null, null)} title="Battery"
            onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.08)'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
          >
            <BatteryIcon />
          </button>

          {/* Wifi */}
          <button
            onClick={() => toggle('wifi')}
            style={{
              ...iconBtn(null, 'wifi'),
              background: panel === 'wifi' ? 'rgba(255,255,255,0.12)' : 'transparent',
            }}
            onMouseEnter={e => { if (panel !== 'wifi') e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; }}
            onMouseLeave={e => { if (panel !== 'wifi') e.currentTarget.style.background = 'transparent'; }}
            title="Wi-Fi"
          >
            <WifiIcon />
          </button>

          {/* Volume */}
          <button
            onClick={() => toggle('volume')}
            style={{
              ...iconBtn(null, 'volume'),
              background: panel === 'volume' ? 'rgba(255,255,255,0.12)' : 'transparent',
            }}
            onMouseEnter={e => { if (panel !== 'volume') e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; }}
            onMouseLeave={e => { if (panel !== 'volume') e.currentTarget.style.background = 'transparent'; }}
            title="Volume"
          >
            <VolumeIcon />
          </button>

          {/* Clock — clicks open calendar */}
          <button
            onClick={() => toggle('clock')}
            style={{
              display: 'flex', flexDirection: 'column', alignItems: 'flex-end',
              justifyContent: 'center', padding: '0 10px 0 6px', height: 48,
              background: panel === 'clock' ? 'rgba(255,255,255,0.12)' : 'transparent',
              border: 'none', cursor: 'default', borderRadius: 4,
            }}
            onMouseEnter={e => { if (panel !== 'clock') e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; }}
            onMouseLeave={e => { if (panel !== 'clock') e.currentTarget.style.background = 'transparent'; }}
          >
            <span style={{ color: '#ffffff', fontSize: 12, fontWeight: 500, fontFamily: 'Segoe UI, sans-serif', lineHeight: 1.3 }}>{timeStr}</span>
            <span style={{ color: 'rgba(255,255,255,0.65)', fontSize: 11, fontFamily: 'Segoe UI, sans-serif', lineHeight: 1.3 }}>{dateStr}</span>
          </button>

          {/* Show desktop strip */}
          <div style={{
            width: 4, height: 48, cursor: 'default',
            borderLeft: '1px solid rgba(255,255,255,0.08)',
          }}
            onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.1)'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
            title="Show desktop"
          />
        </div>
      </div>
    </div>
  );
}

import React, { useState, useEffect, useRef } from 'react';

interface LoginScreenProps {
  labUser: string;
  labDomain: string;
  onLogin: (username: string, password: string) => void;
  loginError: string;
}

function getSAMAccountName(labUser: string): string {
  const parts = labUser.split('\\');
  return parts.length > 1 ? parts[1] : labUser;
}

export default function LoginScreen({ labUser, labDomain, onLogin, loginError }: LoginScreenProps) {
  const [password, setPassword] = useState('');
  const [now, setNow] = useState(new Date());
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  const timeStr = now.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  const dateStr = now.toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
  const sam = getSAMAccountName(labUser);
  const initial = sam.charAt(0).toUpperCase();

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') onLogin(sam, password);
  };

  const styles: Record<string, React.CSSProperties> = {
    root: {
      position: 'fixed', inset: 0,
      background: 'radial-gradient(ellipse at center, #1a3a5c 0%, #0a0f1a 100%)',
      display: 'flex', flexDirection: 'column',
      alignItems: 'center', justifyContent: 'center',
      overflow: 'hidden', userSelect: 'none',
    },
    timeBlock: {
      display: 'flex', flexDirection: 'column', alignItems: 'center',
      marginBottom: 48,
    },
    time: {
      fontSize: 96, fontWeight: 100, color: '#ffffff',
      lineHeight: 1, letterSpacing: -2, fontFamily: 'Segoe UI, sans-serif',
    },
    date: {
      fontSize: 18, color: 'rgba(255,255,255,0.80)',
      marginTop: 8, fontFamily: 'Segoe UI, sans-serif', fontWeight: 400,
    },
    card: {
      backdropFilter: 'blur(20px)',
      background: 'rgba(255,255,255,0.08)',
      border: '1px solid rgba(255,255,255,0.15)',
      borderRadius: 12,
      padding: '40px 48px',
      width: 380,
      display: 'flex', flexDirection: 'column', alignItems: 'center',
      boxSizing: 'border-box',
    },
    avatar: {
      width: 64, height: 64, borderRadius: '50%',
      background: '#0078D4',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      color: '#fff', fontSize: 28, fontWeight: 600,
      fontFamily: 'Segoe UI, sans-serif',
      flexShrink: 0,
    },
    username: {
      color: '#ffffff', fontSize: 18, fontWeight: 600,
      marginTop: 16, fontFamily: 'Segoe UI, sans-serif',
    },
    domain: {
      color: 'rgba(255,255,255,0.60)', fontSize: 13,
      marginTop: 4, fontFamily: 'Segoe UI, sans-serif',
    },
    input: {
      width: '100%', marginTop: 24,
      background: 'rgba(255,255,255,0.10)',
      border: '1px solid rgba(255,255,255,0.30)',
      borderRadius: 4,
      color: '#ffffff', padding: '10px 14px',
      fontSize: 14, fontFamily: 'Segoe UI, sans-serif',
      outline: 'none', boxSizing: 'border-box',
    },
    error: {
      color: '#ff6b6b', fontSize: 13,
      marginTop: 8, alignSelf: 'flex-start',
      fontFamily: 'Segoe UI, sans-serif',
    },
    arrowRow: {
      width: '100%', display: 'flex',
      justifyContent: 'flex-end', marginTop: 16,
    },
    arrowBtn: {
      width: 40, height: 40, borderRadius: '50%',
      background: '#0078D4', border: 'none',
      color: '#fff', fontSize: 18,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      cursor: 'pointer',
    },
    bottomCenter: {
      position: 'absolute', bottom: 32,
      left: '50%', transform: 'translateX(-50%)',
      color: 'rgba(255,255,255,0.60)', fontSize: 13,
      fontFamily: 'Segoe UI, sans-serif', cursor: 'pointer',
      whiteSpace: 'nowrap',
    },
    bottomRight: {
      position: 'absolute', bottom: 24, right: 24,
      display: 'flex', gap: 16, alignItems: 'center',
    },
    iconCircle: {
      width: 20, height: 20, borderRadius: '50%',
      border: '1px solid rgba(255,255,255,0.4)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      cursor: 'default',
    },
  };

  return (
    <div style={styles.root}>
      <div style={styles.timeBlock}>
        <span style={styles.time}>{timeStr}</span>
        <span style={styles.date}>{dateStr}</span>
      </div>

      <div style={styles.card}>
        <div style={styles.avatar}>{initial}</div>
        <span style={styles.username}>{sam}</span>
        <span style={styles.domain}>{labDomain}</span>
        <input
          ref={inputRef}
          type="password"
          placeholder="Password"
          value={password}
          onChange={e => setPassword(e.target.value)}
          onKeyDown={handleKeyDown}
          style={styles.input}
        />
        {loginError && <span style={styles.error}>{loginError}</span>}
        <div style={styles.arrowRow}>
          <button
            style={styles.arrowBtn}
            onClick={() => onLogin(sam, password)}
            onMouseOver={e => (e.currentTarget.style.background = '#106EBE')}
            onMouseOut={e => (e.currentTarget.style.background = '#0078D4')}
          >
            →
          </button>
        </div>
      </div>

      <span style={styles.bottomCenter}>Sign-in options</span>

      <div style={styles.bottomRight}>
        <div style={styles.iconCircle}>
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <circle cx="6" cy="6" r="4" stroke="rgba(255,255,255,0.6)" strokeWidth="1.5" />
          </svg>
        </div>
        <div style={styles.iconCircle}>
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <circle cx="6" cy="6" r="4" stroke="rgba(255,255,255,0.6)" strokeWidth="1.5" />
          </svg>
        </div>
      </div>
    </div>
  );
}
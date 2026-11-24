import React, { useState, useRef, useEffect } from 'react'

const C = {
  bg:      '#1a1a1a',
  panel:   '#252526',
  panel2:  '#1f1f1f',
  rowEven: '#2d2d2d',
  rowOdd:  '#272727',
  rowSel:  '#094771',
  rowHov:  '#2a2d2e',
  border:  '#3c3c3c',
  blue:    '#007acc',
  text:    '#cccccc',
  dim:     '#858585',
  mono:    '#9cdcfe',
  green:   '#4ec9b0',
  yellow:  '#dcdcaa',
  red:     '#f44747',
  orange:  '#ce9178',
  toolbar: '#333333',
}

type Format = 'dll' | 'shellcode' | 'exe'
type Mode   = 'view' | 'create' | 'edit'

interface Profile {
  id:        string
  name:      string
  isSystem:  boolean
  active:    boolean
  toml:      string
  createdAt: string
}

interface Build {
  buildId:   string
  profileId: string
  format:    Format
  filename:  string
  size:      string
  createdAt: string
}

const DEFAULT_TOML = `# AlloyNet Malleable HTTP Profile
# ─────────────────────────────────────────────
[agent]
sleep  = 60      # seconds between callbacks
jitter = 20      # ± percent

[http]
user_agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
uri        = "/jquery-3.3.1.slim.min.js"

[http.headers]
Host            = "cdn.jquery.com"
Accept          = "*/*"
Accept-Language = "en-US,en;q=0.5"
Cache-Control   = "no-cache"

[staging]
uri = "/api/v1/data"
`

const TEAMS_TOML = `# AlloyNet Malleable HTTP Profile — Teams CDN
# ─────────────────────────────────────────────
[agent]
sleep  = 300
jitter = 15

[http]
user_agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SkypeTeams/1.00.517013"
uri        = "/v2/users/me/chats/messages"

[http.headers]
Host           = "teams.microsoft.com"
Accept         = "application/json"
Authorization  = "Bearer {{TOKEN}}"
x-ms-client-type = "win32"

[staging]
uri = "/v1/auth/deviceauth"
`

const SEED_PROFILES: Profile[] = [
  {
    id: 'sys-jquery',
    name: 'jquery-cdn',
    isSystem: true,
    active: true,
    toml: DEFAULT_TOML,
    createdAt: '2026-11-01T08:00:00Z',
  },
  {
    id: 'sys-teams',
    name: 'teams-mimicry',
    isSystem: true,
    active: false,
    toml: TEAMS_TOML,
    createdAt: '2026-11-01T08:00:00Z',
  },
]

const SEED_BUILDS: Build[] = [
  { buildId: 'a1b2c3d4e5f6', profileId: 'sys-jquery', format: 'dll',       filename: 'beacon-a1b2c3d4.dll',       size: '124 KB', createdAt: '2026-11-20T07:14:00Z' },
  { buildId: '0f1e2d3c4b5a', profileId: 'sys-jquery', format: 'shellcode', filename: 'beacon-0f1e2d3c.bin',       size: '88 KB',  createdAt: '2026-11-20T09:30:00Z' },
  { buildId: 'deadbeef1234', profileId: 'sys-teams',  format: 'exe',       filename: 'beacon-deadbeef.exe',       size: '316 KB', createdAt: '2026-11-21T11:00:00Z' },
]

function uid() { return Math.random().toString(36).slice(2, 14) }

// ── Shared sub-components ─────────────────────────────────────────
const labelStyle: React.CSSProperties = {
  display: 'block', fontSize: 9, color: C.dim,
  textTransform: 'uppercase', letterSpacing: '0.07em', marginBottom: 5,
}
const inputStyle: React.CSSProperties = {
  width: '100%', boxSizing: 'border-box',
  background: '#1e1e1e', border: `1px solid ${C.border}`,
  color: C.text, padding: '5px 8px', fontSize: 12,
  borderRadius: 2, outline: 'none', fontFamily: 'inherit',
}
const tomlStyle: React.CSSProperties = {
  flex: 1, background: '#1e1e1e', border: 'none', outline: 'none',
  color: C.mono, fontFamily: '"Consolas","Courier New",monospace',
  fontSize: 12, lineHeight: 1.75, padding: '14px 18px',
  resize: 'none', overflowY: 'auto',
}

function TomlBar({ label }: { label: string }) {
  return (
    <div style={{
      padding: '3px 12px', flexShrink: 0,
      background: '#1e1e1e', borderBottom: `1px solid ${C.border}`, borderTop: `1px solid ${C.border}`,
      fontSize: 9, color: C.dim, letterSpacing: '0.07em', textTransform: 'uppercase',
    }}>{label}</div>
  )
}

function PanelBar({ title, tags, right }: {
  title: string
  tags?: { label: string; color: string }[]
  right?: React.ReactNode
}) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0,
      padding: '0 12px', height: 34, background: C.panel,
      borderBottom: `1px solid ${C.border}`,
    }}>
      <span style={{ fontSize: 12, color: C.text, fontWeight: 600 }}>{title}</span>
      {tags?.map(t => (
        <span key={t.label} style={{
          fontSize: 8, fontWeight: 700, color: t.color,
          border: `1px solid ${t.color}44`, padding: '1px 5px', borderRadius: 2,
        }}>{t.label}</span>
      ))}
      <div style={{ flex: 1 }} />
      {right}
    </div>
  )
}

function BarBtn({ label, onClick, primary, yellow, disabled }: {
  label: string; onClick?: () => void
  primary?: boolean; yellow?: boolean; disabled?: boolean
}) {
  const bg  = primary ? C.blue : 'transparent'
  const col = primary ? '#fff' : yellow ? C.yellow : C.dim
  const brd = primary ? C.blue : yellow ? `${C.yellow}44` : C.border
  return (
    <button onClick={onClick} disabled={disabled} style={{
      background: bg, border: `1px solid ${brd}`, color: col,
      cursor: disabled ? 'wait' : 'pointer',
      padding: '2px 10px', fontSize: 11, borderRadius: 2,
      opacity: disabled ? 0.5 : 1, fontFamily: 'inherit',
    }}>{label}</button>
  )
}

// ── Main ──────────────────────────────────────────────────────────
export default function BeaconGen() {
  const [profiles,    setProfiles]    = useState<Profile[]>(SEED_PROFILES)
  const [builds,      setBuilds]      = useState<Build[]>(SEED_BUILDS)
  const [selected,    setSelected]    = useState<Profile | null>(SEED_PROFILES[0])
  const [format,      setFormat]      = useState<Format>('dll')
  const [mode,        setMode]        = useState<Mode>('view')
  const [notice,      setNotice]      = useState('')
  const [error,       setError]       = useState('')
  const [generating,  setGenerating]  = useState(false)
  const [saving,      setSaving]      = useState(false)
  const [showBuilds,  setShowBuilds]  = useState(false)
  const [fName,       setFName]       = useState('')
  const [fTOML,       setFTOML]       = useState(DEFAULT_TOML)

  const nameRef      = useRef<HTMLInputElement>(null)
  const tomlRef      = useRef<HTMLTextAreaElement>(null)
  const noticeTimer  = useRef<ReturnType<typeof setTimeout>>()

  function flash(msg: string, isErr = false) {
    clearTimeout(noticeTimer.current)
    isErr ? (setError(msg), setNotice('')) : (setNotice(msg), setError(''))
    noticeTimer.current = setTimeout(() => { setError(''); setNotice('') }, 4000)
  }

  function pick(p: Profile) { setSelected(p); setMode('view') }

  function openCreate() {
    setSelected(null); setFName(''); setFTOML(DEFAULT_TOML); setMode('create')
    setTimeout(() => nameRef.current?.focus(), 50)
  }

  function openEdit(p: Profile) {
    setSelected(p); setFName(p.name); setFTOML(p.toml); setMode('edit')
    setTimeout(() => tomlRef.current?.focus(), 50)
  }

  function createProfile() {
    if (!fName.trim())  { flash('Profile name required', true); return }
    if (!fTOML.trim())  { flash('TOML cannot be empty', true); return }
    setSaving(true)
    setTimeout(() => {
      const p: Profile = {
        id: uid(), name: fName.trim(), isSystem: false, active: false,
        toml: fTOML.trim(), createdAt: new Date().toISOString(),
      }
      setProfiles(prev => [...prev, p])
      setSelected(p); setMode('view')
      flash(`Profile "${p.name}" created`)
      setSaving(false)
    }, 400)
  }

  function updateProfile() {
    if (!selected) return
    setSaving(true)
    setTimeout(() => {
      setProfiles(prev => prev.map(p =>
        p.id === selected.id ? { ...p, name: fName.trim(), toml: fTOML.trim() } : p
      ))
      setSelected(prev => prev ? { ...prev, name: fName.trim(), toml: fTOML.trim() } : prev)
      flash(`"${fName}" saved`)
      setMode('view')
      setSaving(false)
    }, 400)
  }

  function setActive(p: Profile) {
    setProfiles(prev => prev.map(x => ({ ...x, active: x.id === p.id })))
    flash(`"${p.name}" is now active`)
  }

  function deleteProfile(p: Profile) {
    setProfiles(prev => prev.filter(x => x.id !== p.id))
    if (selected?.id === p.id) { setSelected(null); setMode('view') }
    flash(`"${p.name}" deleted`)
  }

  async function generate() {
    if (!selected) return
    setGenerating(true)
    await new Promise(r => setTimeout(r, 1800))
    const id  = uid()
    const ext = format === 'shellcode' ? 'bin' : format
    const b: Build = {
      buildId:   id,
      profileId: selected.id,
      format,
      filename:  `beacon-${id.slice(0,8)}.${ext}`,
      size:      format === 'exe' ? '316 KB' : format === 'dll' ? '124 KB' : '88 KB',
      createdAt: new Date().toISOString(),
    }
    setBuilds(prev => [b, ...prev])
    setShowBuilds(true)
    flash(`Build ${id.slice(0,8)} complete`)
    setGenerating(false)
  }

  const sysProfs  = profiles.filter(p => p.isSystem)
  const projProfs = profiles.filter(p => !p.isSystem)
  const activeProf = profiles.find(p => p.active)

  const TH: React.CSSProperties = {
    padding: '4px 10px', textAlign: 'left', fontSize: 9, fontWeight: 600,
    color: C.dim, background: C.panel, borderBottom: `1px solid ${C.border}`,
    textTransform: 'uppercase', letterSpacing: '0.06em',
    position: 'sticky', top: 0, whiteSpace: 'nowrap',
  }
  const TD: React.CSSProperties = {
    padding: '5px 10px', fontSize: 11, color: C.text, borderBottom: '1px solid #222',
  }

  function ProfileRow({ p, idx }: { p: Profile; idx: number }) {
    const sel = selected?.id === p.id
    const bg  = sel ? C.rowSel : idx % 2 === 0 ? C.rowEven : C.rowOdd
    return (
      <div onClick={() => pick(p)} style={{
        display: 'flex', alignItems: 'center', gap: 6,
        padding: '8px 10px', cursor: 'pointer',
        background: bg, borderBottom: '1px solid #222',
        borderLeft: sel ? `2px solid ${C.blue}` : '2px solid transparent',
      }}
        onMouseEnter={e => { if (!sel) e.currentTarget.style.background = C.rowHov }}
        onMouseLeave={e => { if (!sel) e.currentTarget.style.background = bg }}
      >
        {p.isSystem && <span style={{ fontSize: 9, color: C.dim }}>🔒</span>}
        <span style={{
          flex: 1, fontSize: 12, color: sel ? '#fff' : C.text, fontWeight: sel ? 600 : 400,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>{p.name}</span>
        {p.active && (
          <span style={{
            fontSize: 8, fontWeight: 700, color: C.green,
            border: `1px solid ${C.green}55`, padding: '1px 4px', borderRadius: 2,
          }}>LIVE</span>
        )}
        <div style={{ display: 'flex', gap: 3 }} onClick={e => e.stopPropagation()}>
          {!p.active && (
            <button onClick={() => setActive(p)} style={{
              background: 'transparent', border: `1px solid ${C.border}`,
              color: C.dim, cursor: 'pointer', fontSize: 9, padding: '1px 5px', borderRadius: 2,
            }}
              onMouseEnter={e => { e.currentTarget.style.color = C.blue; e.currentTarget.style.borderColor = C.blue }}
              onMouseLeave={e => { e.currentTarget.style.color = C.dim;  e.currentTarget.style.borderColor = C.border }}
            >USE</button>
          )}
          {!p.isSystem && <>
            <button onClick={() => openEdit(p)} title="Edit" style={{
              background: 'transparent', border: `1px solid ${C.border}`,
              color: C.dim, cursor: 'pointer', fontSize: 11, padding: '1px 5px', borderRadius: 2,
            }}
              onMouseEnter={e => { e.currentTarget.style.color = C.yellow; e.currentTarget.style.borderColor = C.yellow }}
              onMouseLeave={e => { e.currentTarget.style.color = C.dim;    e.currentTarget.style.borderColor = C.border }}
            >✎</button>
            <button onClick={() => deleteProfile(p)} title="Delete" style={{
              background: 'transparent', border: 'none', color: C.red,
              cursor: 'pointer', fontSize: 12, padding: '1px 4px', borderRadius: 2, opacity: 0.35,
            }}
              onMouseEnter={e => { e.currentTarget.style.opacity='1'; e.currentTarget.style.background='#3a1212' }}
              onMouseLeave={e => { e.currentTarget.style.opacity='0.35'; e.currentTarget.style.background='transparent' }}
            >✕</button>
          </>}
        </div>
      </div>
    )
  }

  return (
    <div style={{
      display: 'flex', flexDirection: 'column', height: '100%',
      background: C.bg, fontFamily: '"Consolas","Courier New",monospace',
      color: C.text, overflow: 'hidden',
    }}>

      {/* ── Toolbar ───────────────────────────────────────────── */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0,
        padding: '0 12px', height: 36, background: C.toolbar,
        borderBottom: `1px solid ${C.border}`,
      }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: C.blue, letterSpacing: '0.08em', marginRight: 4 }}>
          ⚡ BEACON GEN
        </span>
        <div style={{ width: 1, height: 16, background: C.border }} />
        {activeProf && (
          <span style={{ fontSize: 11, color: C.dim }}>
            active: <span style={{ color: C.green }}>{activeProf.name}</span>
          </span>
        )}
        <div style={{ flex: 1 }} />
        <button onClick={() => setShowBuilds(v => !v)} style={{
          background: showBuilds ? `${C.blue}22` : 'transparent',
          border: `1px solid ${showBuilds ? C.blue : C.border}`,
          color: showBuilds ? C.blue : C.dim,
          cursor: 'pointer', padding: '2px 10px', fontSize: 11, borderRadius: 2,
        }}>
          History {builds.length > 0 ? `(${builds.length})` : ''}
        </button>
        <button onClick={openCreate} style={{
          background: C.blue, border: 'none', color: '#fff',
          cursor: 'pointer', padding: '3px 12px', fontSize: 11, borderRadius: 2, fontWeight: 600,
        }}>+ New Profile</button>
      </div>

      {/* Notice bar */}
      {(error || notice) && (
        <div style={{
          padding: '5px 12px', flexShrink: 0,
          background: error ? '#2d1117' : '#1a2e1a',
          borderBottom: `1px solid ${error ? C.red : C.green}`,
          color: error ? C.red : C.green, fontSize: 11,
          display: 'flex', alignItems: 'center', gap: 8,
        }}>
          <span>{error || notice}</span>
          <button onClick={() => { setError(''); setNotice('') }} style={{
            marginLeft: 'auto', background: 'transparent', border: 'none',
            color: error ? C.red : C.green, cursor: 'pointer', fontSize: 13,
          }}>✕</button>
        </div>
      )}

      {/* ── Body ──────────────────────────────────────────────── */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 0 }}>

        {/* Left sidebar */}
        <div style={{
          width: 220, flexShrink: 0, display: 'flex', flexDirection: 'column',
          borderRight: `1px solid ${C.border}`, overflow: 'hidden', background: C.panel2,
        }}>
          <div style={{
            padding: '5px 10px', fontSize: 9, fontWeight: 700, color: C.dim,
            letterSpacing: '0.08em', textTransform: 'uppercase',
            borderBottom: `1px solid ${C.border}`, background: C.panel, flexShrink: 0,
          }}>System</div>
          <div style={{ flexShrink: 0 }}>
            {sysProfs.map((p, i) => <ProfileRow key={p.id} p={p} idx={i} />)}
          </div>

          <div style={{
            padding: '5px 10px', fontSize: 9, fontWeight: 700, color: C.dim,
            letterSpacing: '0.08em', textTransform: 'uppercase',
            borderBottom: `1px solid ${C.border}`, borderTop: `1px solid ${C.border}`,
            background: C.panel, flexShrink: 0,
          }}>Project</div>
          <div style={{ flex: 1, overflowY: 'auto' }}>
            {projProfs.length === 0 ? (
              <div style={{ padding: '20px 12px', fontSize: 11, color: C.dim, textAlign: 'center', lineHeight: 1.6 }}>
                <div style={{ fontSize: 20, opacity: 0.15, marginBottom: 8 }}>⚡</div>
                No custom profiles<br/>
                <span style={{ color: C.blue, cursor: 'pointer' }} onClick={openCreate}>+ Create one</span>
              </div>
            ) : projProfs.map((p, i) => <ProfileRow key={p.id} p={p} idx={i} />)}
          </div>
        </div>

        {/* Right panel */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>

          {/* CREATE */}
          {mode === 'create' && (
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              <PanelBar title="New Profile" right={
                <div style={{ display: 'flex', gap: 6 }}>
                  <BarBtn label="Cancel" onClick={() => setMode('view')} />
                  <BarBtn label={saving ? 'Creating…' : 'Create'} primary disabled={saving} onClick={createProfile} />
                </div>
              }/>
              <div style={{ padding: '10px 14px', borderBottom: `1px solid ${C.border}`, flexShrink: 0 }}>
                <label style={labelStyle}>Profile Name</label>
                <input ref={nameRef} value={fName} onChange={e => setFName(e.target.value)}
                  placeholder="e.g. jquery-cdn, teams-mimicry…"
                  onKeyDown={e => e.key === 'Enter' && tomlRef.current?.focus()}
                  style={inputStyle}
                  onFocus={e => e.currentTarget.style.borderColor = C.blue}
                  onBlur={e => e.currentTarget.style.borderColor = C.border}
                />
              </div>
              <TomlBar label="Malleable Profile TOML" />
              <textarea ref={tomlRef} value={fTOML} onChange={e => setFTOML(e.target.value)}
                spellCheck={false} style={tomlStyle} />
            </div>
          )}

          {/* EDIT */}
          {mode === 'edit' && selected && (
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              <PanelBar title={`Editing: ${selected.name}`} right={
                <div style={{ display: 'flex', gap: 6 }}>
                  <BarBtn label="Cancel" onClick={() => setMode('view')} />
                  <BarBtn label={saving ? 'Saving…' : 'Save'} primary disabled={saving} onClick={updateProfile} />
                </div>
              }/>
              <div style={{ padding: '10px 14px', borderBottom: `1px solid ${C.border}`, flexShrink: 0 }}>
                <label style={labelStyle}>Profile Name</label>
                <input ref={nameRef} value={fName} onChange={e => setFName(e.target.value)}
                  style={inputStyle}
                  onFocus={e => e.currentTarget.style.borderColor = C.blue}
                  onBlur={e => e.currentTarget.style.borderColor = C.border}
                />
              </div>
              <TomlBar label="Malleable Profile TOML" />
              <textarea ref={tomlRef} value={fTOML} onChange={e => setFTOML(e.target.value)}
                spellCheck={false} style={tomlStyle} />
            </div>
          )}

          {/* VIEW */}
          {mode === 'view' && (
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
              {selected ? <>
                <PanelBar
                  title={selected.name}
                  tags={[
                    selected.isSystem ? { label: 'SYSTEM', color: C.blue }  : null,
                    selected.active   ? { label: 'ACTIVE', color: C.green } : null,
                  ].filter(Boolean) as any}
                  right={
                    <div style={{ display: 'flex', gap: 6 }}>
                      {!selected.active && <BarBtn label="Set Active" onClick={() => setActive(selected)} />}
                      {!selected.isSystem && <BarBtn label="✎ Edit" yellow onClick={() => openEdit(selected)} />}
                    </div>
                  }
                />
                <TomlBar label={`Profile TOML${selected.isSystem ? ' · read-only' : ''}`} />
                <div style={{
                  flex: 1, overflowY: 'auto', padding: '14px 18px',
                  fontFamily: 'monospace', fontSize: 12, lineHeight: 1.75,
                  color: C.mono, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
                  background: '#1e1e1e',
                }}>
                  {selected.toml || <span style={{ color: C.dim, fontStyle: 'italic' }}>No TOML stored for this profile</span>}
                </div>

                {/* Generate bar */}
                <div style={{
                  flexShrink: 0, background: C.panel,
                  borderTop: `1px solid ${C.border}`, padding: '10px 14px',
                }}>
                  <div style={{
                    fontSize: 9, color: C.dim, letterSpacing: '0.08em',
                    textTransform: 'uppercase', marginBottom: 10,
                  }}>Generate Beacon</div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                    {/* Format toggle */}
                    <div style={{ display: 'flex', gap: 0 }}>
                      {(['dll','shellcode','exe'] as Format[]).map((f, i) => (
                        <button key={f} onClick={() => setFormat(f)} style={{
                          background: format === f ? C.blue : '#1e1e1e',
                          border: `1px solid ${format === f ? C.blue : C.border}`,
                          borderLeft: i === 0 ? undefined : 'none',
                          borderRadius: i === 0 ? '2px 0 0 2px' : i === 2 ? '0 2px 2px 0' : '0',
                          color: format === f ? '#fff' : C.dim,
                          cursor: 'pointer', padding: '4px 12px',
                          fontSize: 10, fontWeight: 700,
                          letterSpacing: '0.05em', textTransform: 'uppercase',
                        }}>{f}</button>
                      ))}
                    </div>
                    {/* Summary */}
                    <div style={{ fontSize: 11, color: C.dim, flex: 1 }}>
                      <span style={{ fontFamily: 'monospace', color: C.mono }}>{selected.name}</span>
                      <span style={{ margin: '0 6px', color: C.border }}>→</span>
                      <span style={{ color: C.yellow }}>beacon.{format === 'shellcode' ? 'bin' : format}</span>
                    </div>
                    <button onClick={generate} disabled={generating} style={{
                      background: generating ? 'transparent' : C.green,
                      border: `1px solid ${generating ? C.border : C.green}`,
                      color: generating ? C.dim : '#000',
                      cursor: generating ? 'wait' : 'pointer',
                      padding: '6px 20px', fontSize: 11, fontWeight: 700,
                      borderRadius: 2, whiteSpace: 'nowrap', letterSpacing: '0.04em',
                    }}>
                      {generating ? '⟳  Building…' : '⚡  Generate'}
                    </button>
                  </div>
                </div>
              </> : (
                <div style={{
                  flex: 1, display: 'flex', flexDirection: 'column',
                  alignItems: 'center', justifyContent: 'center',
                  color: C.dim, fontSize: 12, gap: 12,
                }}>
                  <span style={{ fontSize: 40, opacity: 0.07 }}>⚡</span>
                  <span>Select a profile from the left panel</span>
                  <span style={{ fontSize: 11 }}>
                    or{' '}
                    <span style={{ color: C.blue, cursor: 'pointer' }} onClick={openCreate}>
                      create a new one
                    </span>
                  </span>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* ── Build History drawer ──────────────────────────────── */}
      {showBuilds && builds.length > 0 && (
        <div style={{ flexShrink: 0, maxHeight: 180, overflow: 'auto', borderTop: `2px solid ${C.blue}` }}>
          <div style={{
            position: 'sticky', top: 0, padding: '4px 12px',
            background: C.panel, borderBottom: `1px solid ${C.border}`,
            fontSize: 9, color: C.dim, letterSpacing: '0.08em',
            textTransform: 'uppercase', display: 'flex', alignItems: 'center',
          }}>
            <span>Build History — {builds.length} artifacts</span>
            <button onClick={() => setShowBuilds(false)} style={{
              marginLeft: 'auto', background: 'transparent', border: 'none',
              color: C.dim, cursor: 'pointer', fontSize: 12,
            }}>✕</button>
          </div>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                {['Build ID','Profile','Format','Filename','Size','Date'].map((h, i) => (
                  <th key={i} style={TH}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {builds.map((b, i) => {
                const prof = profiles.find(p => p.id === b.profileId)
                return (
                  <tr key={b.buildId} style={{ background: i % 2 === 0 ? C.rowEven : C.rowOdd }}>
                    <td style={{ ...TD, fontFamily: 'monospace', color: C.mono }}>{b.buildId.slice(0,13)}…</td>
                    <td style={{ ...TD, color: C.dim }}>{prof?.name ?? b.profileId.slice(0,13)}</td>
                    <td style={{ ...TD, color: C.yellow, fontWeight: 700, fontSize: 10, textTransform: 'uppercase' }}>{b.format}</td>
                    <td style={{ ...TD, fontFamily: 'monospace', color: C.dim }}>{b.filename}</td>
                    <td style={{ ...TD, color: C.dim }}>{b.size}</td>
                    <td style={{ ...TD, color: C.dim }}>{new Date(b.createdAt).toLocaleDateString()}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

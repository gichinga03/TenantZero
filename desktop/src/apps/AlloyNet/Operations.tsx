import React, { useState, useEffect, useRef, useCallback } from 'react'
import { useBeaconSocket } from '../../hooks/useBeaconSocket'

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

interface Beacon {
  id:           string
  hostname:     string
  username:     string
  internalIp:   string
  os:           string
  architecture: string
  sleep:        number
  jitter:       number
  lastSeen:     string
  status:       'active' | 'idle' | 'dead'
  pid:          number
  integrity:    'Medium' | 'High' | 'SYSTEM'
}

interface ProcessEntry { pid: number; ppid: number; name: string; path: string }
interface Line { type: 'cmd' | 'out' | 'err' | 'info'; text: string }
interface CtxMenu { x: number; y: number; beacon: Beacon }

const SEED_BEACONS: Beacon[] = [
  {
    id:           'b1a2c3d4-e5f6-7890-abcd-ef1234567890',
    hostname:     'WKSTN-JSMITH',
    username:     'MERIDIAN\\jsmith',
    internalIp:   '10.10.10.21',
    os:           'Windows 10',
    architecture: 'x64',
    sleep:        60,
    jitter:       20,
    lastSeen:     new Date(Date.now() - 42000).toISOString(),
    status:       'active',
    pid:          4821,
    integrity:    'Medium',
  },
]

const FAKE_PROCS: ProcessEntry[] = [
  { pid: 4,    ppid: 0,    name: 'System',               path: '' },
  { pid: 376,  ppid: 4,    name: 'smss.exe',              path: '\\SystemRoot\\System32\\smss.exe' },
  { pid: 568,  ppid: 548,  name: 'csrss.exe',             path: 'C:\\Windows\\System32\\csrss.exe' },
  { pid: 644,  ppid: 636,  name: 'winlogon.exe',          path: 'C:\\Windows\\System32\\winlogon.exe' },
  { pid: 720,  ppid: 712,  name: 'services.exe',          path: 'C:\\Windows\\System32\\services.exe' },
  { pid: 728,  ppid: 712,  name: 'lsass.exe',             path: 'C:\\Windows\\System32\\lsass.exe' },
  { pid: 836,  ppid: 720,  name: 'svchost.exe',           path: 'C:\\Windows\\System32\\svchost.exe' },
  { pid: 912,  ppid: 720,  name: 'svchost.exe',           path: 'C:\\Windows\\System32\\svchost.exe' },
  { pid: 1024, ppid: 720,  name: 'svchost.exe',           path: 'C:\\Windows\\System32\\svchost.exe' },
  { pid: 1104, ppid: 720,  name: 'svchost.exe',           path: 'C:\\Windows\\System32\\svchost.exe' },
  { pid: 1288, ppid: 720,  name: 'spoolsv.exe',           path: 'C:\\Windows\\System32\\spoolsv.exe' },
  { pid: 1356, ppid: 720,  name: 'msdtc.exe',             path: 'C:\\Windows\\System32\\msdtc.exe' },
  { pid: 1504, ppid: 720,  name: 'VGAuthService.exe',     path: 'C:\\Program Files\\VMware\\VMware Tools\\VMware VGAuth\\VGAuthService.exe' },
  { pid: 1548, ppid: 720,  name: 'vm3dservice.exe',       path: 'C:\\Windows\\System32\\vm3dservice.exe' },
  { pid: 1620, ppid: 720,  name: 'vmtoolsd.exe',          path: 'C:\\Program Files\\VMware\\VMware Tools\\vmtoolsd.exe' },
  { pid: 1788, ppid: 836,  name: 'WmiPrvSE.exe',          path: 'C:\\Windows\\System32\\wbem\\WmiPrvSE.exe' },
  { pid: 2124, ppid: 836,  name: 'SearchIndexer.exe',     path: 'C:\\Windows\\System32\\SearchIndexer.exe' },
  { pid: 2672, ppid: 1024, name: 'RuntimeBroker.exe',     path: 'C:\\Windows\\System32\\RuntimeBroker.exe' },
  { pid: 2960, ppid: 836,  name: 'MsMpEng.exe',           path: 'C:\\ProgramData\\Microsoft\\Windows Defender\\Platform\\MsMpEng.exe' },
  { pid: 3220, ppid: 644,  name: 'userinit.exe',          path: 'C:\\Windows\\System32\\userinit.exe' },
  { pid: 3348, ppid: 3220, name: 'explorer.exe',          path: 'C:\\Windows\\explorer.exe' },
  { pid: 3812, ppid: 3348, name: 'vmtoolsd.exe',          path: 'C:\\Program Files\\VMware\\VMware Tools\\vmtoolsd.exe' },
  { pid: 4096, ppid: 3348, name: 'OneDrive.exe',          path: 'C:\\Users\\jsmith\\AppData\\Local\\Microsoft\\OneDrive\\OneDrive.exe' },
  { pid: 4312, ppid: 3348, name: 'Teams.exe',             path: 'C:\\Users\\jsmith\\AppData\\Local\\Microsoft\\Teams\\Teams.exe' },
  { pid: 4512, ppid: 3348, name: 'Outlook.exe',           path: 'C:\\Program Files\\Microsoft Office\\root\\Office16\\OUTLOOK.EXE' },
  { pid: 4821, ppid: 4512, name: 'rundll32.exe',          path: 'C:\\Windows\\System32\\rundll32.exe' },
  { pid: 5100, ppid: 836,  name: 'conhost.exe',           path: 'C:\\Windows\\System32\\conhost.exe' },
]

function fakeResponse(cmd: string, beacon: Beacon): { delay: number; lines: Line[] } {
  const c = cmd.trim().toLowerCase()
  if (c === 'pwd' || c === 'getwd') return { delay: 600, lines: [{ type: 'out', text: `C:\\Users\\${beacon.username.split('\\')[1] || 'user'}\\Desktop` }] }
  if (c === 'whoami') return { delay: 500, lines: [{ type: 'out', text: beacon.username.toLowerCase() }] }
  if (c === 'whoami /priv' || c === 'whoami/priv') return { delay: 700, lines: [{ type: 'out', text: `PRIVILEGES INFORMATION\n----------------------\n\nPrivilege Name                Description                    State\n============================= ============================== =======\nSeShutdownPrivilege           Shut down the system           Disabled\nSeChangeNotifyPrivilege       Bypass traverse checking       Enabled\nSeUndockPrivilege             Remove computer from docking   Disabled\nSeIncreaseWorkingSetPrivilege Increase a process working set Disabled` }] }
  if (c === 'ps' || c === 'getps') return { delay: 1200, lines: [{ type: 'info', text: `[*] Process list — ${FAKE_PROCS.length} entries` }, { type: 'out', text: FAKE_PROCS.map(p => `${String(p.pid).padStart(5)}  ${String(p.ppid).padStart(5)}  ${p.name.padEnd(28)} ${p.path}`).join('\n') }] }
  if (c === 'processbrowser') return { delay: 1100, lines: [{ type: 'out', text: JSON.stringify(FAKE_PROCS) }] }
  if (c === 'ipconfig') return { delay: 700, lines: [{ type: 'out', text: `Windows IP Configuration\n\nEthernet adapter Ethernet0:\n   Connection-specific DNS Suffix  . : meridian.local\n   IPv4 Address. . . . . . . . . . . : ${beacon.internalIp}\n   Subnet Mask . . . . . . . . . . . : 255.255.255.0\n   Default Gateway . . . . . . . . . : 10.10.10.1` }] }
  if (c === 'hostname') return { delay: 400, lines: [{ type: 'out', text: beacon.hostname }] }
  if (c.startsWith('sleep ')) { const n = parseInt(c.split(' ')[1]) || 60; return { delay: 300, lines: [{ type: 'info', text: `[*] Sleep set to ${n}s` }] } }
  if (c === 'getuid' || c === 'id') return { delay: 450, lines: [{ type: 'out', text: `[*] You are ${beacon.username} (Integrity: ${beacon.integrity})` }] }
  if (c.startsWith('shell ') || c.startsWith('cmd ')) { const sub = cmd.replace(/^shell |^cmd /i, '').trim(); return { delay: 900, lines: [{ type: 'out', text: `'${sub}' is not recognized as an internal or external command,\noperable program or batch file.` }] } }
  if (c.startsWith('upload ')) return { delay: 1500, lines: [{ type: 'info', text: '[*] Upload task queued — file will transfer on next callback' }] }
  if (c.startsWith('download ')) return { delay: 1500, lines: [{ type: 'info', text: '[*] Download task queued — file will transfer on next callback' }] }
  if (c === 'help' || c === '?') return { delay: 200, lines: [{ type: 'out', text: `AlloyNet Beacon Commands\n========================\npwd / getwd          Current working directory\nwhoami               Current user\nwhoami /priv         Token privileges\nps / getps           Process list\nprocessbrowser       Interactive process browser\nipconfig             Network interfaces\nhostname             Machine hostname\ngetuid               User + integrity level\nsleep <sec>          Set callback interval\nshell <cmd>          Run via cmd.exe\nupload <src> <dst>   Upload file to target\ndownload <path>      Download file from target\nkill                 Terminate beacon\nclear                Clear console` }] }
  if (c === 'kill') return { delay: 300, lines: [{ type: 'err', text: '[!] Kill task queued — beacon will exit on next callback' }] }
  if (c === 'clear') return { delay: 0, lines: [] }
  return { delay: 800, lines: [{ type: 'err', text: `[-] Unknown command: ${cmd.split(' ')[0]}\n    Type 'help' for available commands` }] }
}

function ago(iso: string): string {
  if (!iso) return '—'
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (s < 60)    return `${s}s`
  if (s < 3600)  return `${Math.floor(s / 60)}m`
  if (s < 86400) return `${Math.floor(s / 3600)}h`
  return `${Math.floor(s / 86400)}d`
}

function statusDot(b: Beacon) {
  const col = b.status === 'active' ? C.green : b.status === 'idle' ? C.yellow : C.dim
  const sym = b.status === 'active' ? '●' : b.status === 'idle' ? '◐' : '○'
  return <span style={{ color: col, fontSize: 14 }} title={b.status}>{sym}</span>
}

function ProcessTable({ raw }: { raw: string }) {
  const [filter,  setFilter]  = useState('')
  const [sortCol, setSortCol] = useState<'pid' | 'ppid' | 'name' | 'path'>('pid')
  const [sortAsc, setSortAsc] = useState(true)

  let rows: ProcessEntry[] = []
  try { rows = JSON.parse(raw) } catch { return <span style={{ color: C.red }}>[!] malformed process JSON</span> }

  const filtered = rows
    .filter(r => !filter || r.name.toLowerCase().includes(filter.toLowerCase()) || r.path.toLowerCase().includes(filter.toLowerCase()) || String(r.pid).includes(filter))
    .sort((a, b) => {
      const av = String(a[sortCol]), bv = String(b[sortCol])
      return sortAsc ? av.localeCompare(bv, undefined, { numeric: true }) : bv.localeCompare(av, undefined, { numeric: true })
    })

  const PTH: React.CSSProperties = {
    padding: '3px 8px', textAlign: 'left', fontSize: 10, fontWeight: 600,
    background: '#1e1e1e', borderBottom: `1px solid ${C.border}`,
    borderRight: `1px solid ${C.border}`, textTransform: 'uppercase',
    letterSpacing: '0.05em', whiteSpace: 'nowrap', position: 'sticky', top: 0,
  }
  function colHdr(label: string, key: typeof sortCol) {
    const active = sortCol === key
    return (
      <th onClick={() => { active ? setSortAsc(a => !a) : (setSortCol(key), setSortAsc(true)) }}
        style={{ ...PTH, color: active ? C.blue : C.dim, cursor: 'pointer', userSelect: 'none' }}>
        {label} {active ? (sortAsc ? '↑' : '↓') : ''}
      </th>
    )
  }

  return (
    <div style={{ margin: '6px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <span style={{ color: C.blue, fontSize: 11 }}>[processbrowser]</span>
        <span style={{ color: C.dim, fontSize: 11 }}>{filtered.length}/{rows.length} processes</span>
        <input value={filter} onChange={e => setFilter(e.target.value)} placeholder="filter…"
          style={{ background: '#2a2a2a', border: `1px solid ${C.border}`, color: C.text, fontSize: 11, padding: '1px 6px', borderRadius: 2, fontFamily: 'monospace', outline: 'none', width: 160 }} />
      </div>
      <div style={{ maxHeight: 300, overflowY: 'auto', border: `1px solid ${C.border}` }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', tableLayout: 'fixed' }}>
          <colgroup>
            <col style={{ width: 58 }} /><col style={{ width: 58 }} />
            <col style={{ width: 170 }} /><col />
          </colgroup>
          <thead>
            <tr>{colHdr('PID','pid')}{colHdr('PPID','ppid')}{colHdr('Name','name')}{colHdr('Path','path')}</tr>
          </thead>
          <tbody>
            {filtered.map((r, i) => (
              <tr key={r.pid} style={{ background: i % 2 === 0 ? C.rowEven : C.rowOdd }}
                onMouseEnter={e => { e.currentTarget.style.background = C.rowHov }}
                onMouseLeave={e => { e.currentTarget.style.background = i % 2 === 0 ? C.rowEven : C.rowOdd }}>
                <td style={{ padding: '2px 8px', fontSize: 11, color: C.yellow, fontFamily: 'monospace', borderRight: '1px solid #222' }}>{r.pid}</td>
                <td style={{ padding: '2px 8px', fontSize: 11, color: C.dim,    fontFamily: 'monospace', borderRight: '1px solid #222' }}>{r.ppid}</td>
                <td style={{ padding: '2px 8px', fontSize: 11, color: C.green,  fontFamily: 'monospace', borderRight: '1px solid #222', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.name}</td>
                <td style={{ padding: '2px 8px', fontSize: 11, color: C.dim,    fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={r.path}>{r.path || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

interface OperationsProps {
  token: string
}

export default function Operations({ token }: OperationsProps) {
  const { connected, lastMessage, wsSend } = useBeaconSocket(token || null, 'op-chainlink')

  const [beacons,       setBeacons]       = useState<Beacon[]>(SEED_BEACONS)
  const [activeBeacon,  setActiveBeacon]  = useState<Beacon | null>(null)
  const [splitPct,      setSplitPct]      = useState(42)
  const [dragging,      setDragging]      = useState(false)
  const [ctxMenu,       setCtxMenu]       = useState<CtxMenu | null>(null)
  const [command,       setCommand]       = useState('')
  const [sending,       setSending]       = useState(false)
  const [cmdHistory,    setCmdHistory]    = useState<string[]>([])
  const [histIdx,       setHistIdx]       = useState(-1)
  const [lines,         setLines]         = useState<Line[]>([
    { type: 'info', text: '[*] AlloyNet C2 — operator console ready' },
    { type: 'info', text: '[*] Select a session above to begin interacting' },
    { type: 'info', text: '' },
  ])

  const containerRef  = useRef<HTMLDivElement>(null)
  const consoleEndRef = useRef<HTMLDivElement>(null)
  const inputRef      = useRef<HTMLInputElement>(null)

  // ── lastMessage effect (real WS responses) ───────────────────
  useEffect(() => {
    if (!lastMessage) return
    if (lastMessage.type === 'output') {
      const out: string    = lastMessage.payload?.output ?? ''
      const isErr: boolean = lastMessage.payload?.isError ?? false
      setLines(prev => [
        ...prev,
        { type: isErr ? 'err' : 'out', text: out },
        { type: 'info', text: '' },
      ])
      setSending(false)
    }
    if (lastMessage.type === 'error') {
      setLines(prev => [
        ...prev,
        { type: 'err', text: `[!] Server error: ${lastMessage.payload?.message ?? 'unknown'}` },
        { type: 'info', text: '' },
      ])
      setSending(false)
    }
  }, [lastMessage])

  // live ticker
  useEffect(() => { const id = setInterval(() => setBeacons(bs => [...bs]), 10000); return () => clearInterval(id) }, [])

  // auto-scroll
  useEffect(() => { consoleEndRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [lines])

  // drag-resize
  useEffect(() => {
    if (!dragging) return
    const onMove = (e: MouseEvent) => {
      const el = containerRef.current; if (!el) return
      const pct = ((e.clientY - el.getBoundingClientRect().top) / el.clientHeight) * 100
      setSplitPct(Math.min(78, Math.max(18, pct)))
    }
    const onUp = () => setDragging(false)
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => { window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp) }
  }, [dragging])

  // dismiss context menu
  useEffect(() => {
    if (!ctxMenu) return
    const h = () => setCtxMenu(null)
    window.addEventListener('click', h)
    return () => window.removeEventListener('click', h)
  }, [ctxMenu])

  function selectBeacon(b: Beacon) {
    setActiveBeacon(b)
    setLines([
      { type: 'info', text: `[*] ─────────────────────────────────────────────────────` },
      { type: 'info', text: `[*] Session   ${b.id.slice(0, 13)}…` },
      { type: 'info', text: `[*] Host      ${b.hostname}` },
      { type: 'info', text: `[*] User      ${b.username}  (${b.integrity})` },
      { type: 'info', text: `[*] Internal  ${b.internalIp}` },
      { type: 'info', text: `[*] OS        ${b.os} / ${b.architecture}  PID: ${b.pid}` },
      { type: 'info', text: `[*] Sleep     ${b.sleep}s ±${b.jitter}%` },
      { type: 'info', text: `[*] ─────────────────────────────────────────────────────` },
      { type: 'info', text: '' },
    ])
    inputRef.current?.focus()
  }

  function killSession(b: Beacon) {
    setCtxMenu(null)
    setBeacons(prev => prev.filter(x => x.id !== b.id))
    if (activeBeacon?.id === b.id) {
      setActiveBeacon(null)
      setLines(prev => [...prev, { type: 'err', text: `[!] Session ${b.id.slice(0,13)} terminated` }])
    }
  }

  async function sendCommand() {
    const cmd = command.trim()
    if (!cmd || !activeBeacon || sending) return
    if (cmd === 'clear') { setLines([]); setCommand(''); return }

    setCommand('')
    setSending(true)
    setCmdHistory(h => [cmd, ...h.slice(0, 99)])
    setHistIdx(-1)
    setLines(prev => [...prev, { type: 'cmd', text: `beacon> ${cmd}` }])

    if (!connected) {
      const { delay, lines: resp } = fakeResponse(cmd, activeBeacon)
      await new Promise(r => setTimeout(r, delay))
      setLines(prev => [...prev, { type: 'info', text: '[*] (offline) task simulated' }, ...resp, { type: 'info', text: '' }])
      setSending(false)
      return
    }
    wsSend(cmd, activeBeacon.id)
    // setSending(false) is called in the lastMessage effect on 'output'/'error'
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') { sendCommand(); return }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      const next = Math.min(histIdx + 1, cmdHistory.length - 1)
      setHistIdx(next); setCommand(cmdHistory[next] ?? '')
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = histIdx - 1
      if (next < 0) { setHistIdx(-1); setCommand(''); return }
      setHistIdx(next); setCommand(cmdHistory[next] ?? '')
    }
    if (e.key === 'Tab') {
      e.preventDefault()
      const cmds = ['pwd','whoami','whoami /priv','ps','processbrowser','ipconfig','hostname','getuid','sleep ','shell ','upload ','download ','kill','help','clear']
      const match = cmds.find(c => c.startsWith(command))
      if (match) setCommand(match)
    }
  }

  function prefill(pre: string) { if (!activeBeacon) return; setCommand(pre); inputRef.current?.focus() }

  const TH: React.CSSProperties = {
    padding: '4px 8px', textAlign: 'left', fontSize: 10, fontWeight: 600,
    color: C.dim, background: C.panel, borderBottom: `1px solid ${C.border}`,
    borderRight: `1px solid ${C.border}`, textTransform: 'uppercase',
    letterSpacing: '0.05em', userSelect: 'none', whiteSpace: 'nowrap',
    position: 'sticky', top: 0, zIndex: 1,
  }
  const TD: React.CSSProperties = {
    padding: '3px 8px', fontSize: 11, color: C.text,
    borderBottom: '1px solid #222', borderRight: '1px solid #222',
    whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
  }

  const TOOLBAR_BTNS = [
    { label: 'Interact',      icon: '⌨', pre: '' },
    { label: 'Shell',         icon: '$',  pre: 'shell ' },
    { label: 'Sleep',         icon: '⏱', pre: 'sleep ' },
    { label: 'Upload',        icon: '↑',  pre: 'upload ' },
    { label: 'Download',      icon: '↓',  pre: 'download ' },
    { label: 'PS',            icon: '⟳',  pre: 'ps' },
    { label: 'Proc Browser',  icon: '⊞',  pre: 'processbrowser' },
    { label: 'Pwd',           icon: '⌂',  pre: 'pwd' },
    { label: 'Whoami',        icon: '👤', pre: 'whoami' },
    { label: 'Ipconfig',      icon: '🌐', pre: 'ipconfig' },
  ]

  const CTX_ITEMS = [
    { label: 'Interact',      icon: '⌨', danger: false, pre: '' },
    { label: 'Shell',         icon: '$',  danger: false, pre: 'shell ' },
    { label: 'Sleep…',        icon: '⏱', danger: false, pre: 'sleep ' },
    { label: 'Upload…',       icon: '↑',  danger: false, pre: 'upload ' },
    { label: 'Download…',     icon: '↓',  danger: false, pre: 'download ' },
    { label: 'Pwd',           icon: '⌂',  danger: false, pre: 'pwd' },
    { label: 'Process List',  icon: '⟳',  danger: false, pre: 'ps' },
    { label: 'Proc Browser',  icon: '⊞',  danger: false, pre: 'processbrowser' },
    null,
    { label: 'Kill Session',  icon: '✕',  danger: true,  pre: null },
  ]

  return (
    <div ref={containerRef} style={{
      display: 'flex', flexDirection: 'column', height: '100%',
      background: C.bg, fontFamily: '"Consolas","Courier New",monospace',
      color: C.text, userSelect: dragging ? 'none' : 'auto', overflow: 'hidden',
    }}>

      {/* ── Toolbar ───────────────────────────────────────────── */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 1,
        padding: '0 8px', background: C.toolbar,
        borderBottom: `1px solid ${C.border}`, height: 34, flexShrink: 0,
      }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: C.blue, marginRight: 12, letterSpacing: '0.08em' }}>
          ⬡ ALLOYNET
        </span>

        {TOOLBAR_BTNS.map(btn => (
          <button key={btn.label}
            onClick={() => prefill(btn.pre)}
            disabled={!activeBeacon}
            title={btn.label}
            style={{
              background: 'transparent', border: '1px solid transparent',
              color: activeBeacon ? C.text : C.dim, cursor: activeBeacon ? 'pointer' : 'default',
              padding: '2px 8px', fontSize: 11, borderRadius: 2,
              display: 'flex', alignItems: 'center', gap: 4,
              opacity: activeBeacon ? 1 : 0.35,
            }}
            onMouseEnter={e => { if (activeBeacon) e.currentTarget.style.background = '#444' }}
            onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
          >
            <span>{btn.icon}</span>
            <span style={{ display: window.innerWidth < 1200 ? 'none' : 'inline' }}>{btn.label}</span>
          </button>
        ))}

        <div style={{ flex: 1 }} />
        <span style={{ fontSize: 11, color: C.dim }}>
          sessions: <span style={{ color: C.green, marginRight: 12 }}>{beacons.length}</span>
          {activeBeacon && <span style={{ color: C.yellow }}>active: {activeBeacon.hostname}</span>}
        </span>
        <span style={{ fontSize: 11, color: connected ? C.green : C.red, marginLeft: 8 }}>
          {connected ? '● live' : '○ offline'}
        </span>
      </div>

      {/* ── Beacon Table ──────────────────────────────────────── */}
      <div style={{ height: `${splitPct}%`, overflow: 'auto', flexShrink: 0 }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', tableLayout: 'fixed' }}>
          <colgroup>
            <col style={{ width: 130 }} /><col style={{ width: 50  }} /><col style={{ width: 150 }} />
            <col style={{ width: 170 }} /><col style={{ width: 110 }} /><col style={{ width: 90  }} />
            <col style={{ width: 90  }} /><col style={{ width: 60  }} /><col style={{ width: 60  }} />
            <col style={{ width: 55  }} /><col style={{ width: 50  }} /><col style={{ width: 36  }} />
          </colgroup>
          <thead>
            <tr>
              {['ID','','Hostname','User','Internal IP','OS','Arch','Sleep','Last','PID','Integ',''].map((h, i) => (
                <th key={i} style={TH}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {beacons.length === 0 ? (
              <tr><td colSpan={12} style={{ ...TD, textAlign: 'center', padding: '48px 0', color: C.dim }}>No active sessions</td></tr>
            ) : beacons.map((b, i) => {
              const sel   = activeBeacon?.id === b.id
              const rowBg = sel ? C.rowSel : i % 2 === 0 ? C.rowEven : C.rowOdd
              const old   = ago(b.lastSeen).endsWith('d')
              return (
                <tr key={b.id}
                  onClick={() => selectBeacon(b)}
                  onContextMenu={e => { e.preventDefault(); setCtxMenu({ x: e.clientX, y: e.clientY, beacon: b }) }}
                  style={{ background: rowBg, cursor: 'pointer' }}
                  onMouseEnter={e => { if (!sel) e.currentTarget.style.background = C.rowHov }}
                  onMouseLeave={e => { e.currentTarget.style.background = rowBg }}
                >
                  <td style={{ ...TD, fontFamily: 'monospace', color: C.mono, fontSize: 10 }}>{b.id.slice(0, 13)}…</td>
                  <td style={{ ...TD, textAlign: 'center', padding: '3px 4px' }}>{statusDot(b)}</td>
                  <td style={{ ...TD, color: '#fff', fontWeight: 600 }}>{b.hostname}</td>
                  <td style={{ ...TD, color: C.text }}>{b.username}</td>
                  <td style={{ ...TD, fontFamily: 'monospace', color: C.mono }}>{b.internalIp}</td>
                  <td style={{ ...TD, color: C.dim, fontSize: 10 }}>{b.os.replace('Windows ', 'Win ')}</td>
                  <td style={{ ...TD, color: C.dim }}>{b.architecture}</td>
                  <td style={{ ...TD, textAlign: 'center', color: C.yellow }}>{b.sleep}s</td>
                  <td style={{ ...TD, textAlign: 'center', color: old ? C.red : C.green }}>{ago(b.lastSeen)}</td>
                  <td style={{ ...TD, textAlign: 'center', color: C.dim, fontFamily: 'monospace', fontSize: 10 }}>{b.pid}</td>
                  <td style={{ ...TD, textAlign: 'center', fontSize: 10,
                    color: b.integrity === 'SYSTEM' ? C.red : b.integrity === 'High' ? C.orange : C.dim }}>
                    {b.integrity}
                  </td>
                  <td style={{ ...TD, textAlign: 'center', padding: '0 4px', borderRight: 'none' }}>
                    <button onClick={e => { e.stopPropagation(); killSession(b) }}
                      title="Kill session"
                      style={{ background: 'transparent', border: 'none', color: C.red, cursor: 'pointer', fontSize: 13, padding: '1px 5px', borderRadius: 2, opacity: 0.4 }}
                      onMouseEnter={e => { e.currentTarget.style.opacity='1'; e.currentTarget.style.background='#3a1212' }}
                      onMouseLeave={e => { e.currentTarget.style.opacity='0.4'; e.currentTarget.style.background='transparent' }}
                    >✕</button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* ── Drag Handle ───────────────────────────────────────── */}
      <div
        onMouseDown={e => { e.preventDefault(); setDragging(true) }}
        style={{ height: 4, background: dragging ? C.blue : C.border, cursor: 'row-resize', flexShrink: 0 }}
        onMouseEnter={e => { e.currentTarget.style.background = C.blue }}
        onMouseLeave={e => { if (!dragging) e.currentTarget.style.background = C.border }}
      />

      {/* ── Console pane ──────────────────────────────────────── */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>

        <div style={{
          padding: '4px 12px', background: C.panel, borderBottom: `1px solid ${C.border}`,
          fontSize: 11, color: C.dim, flexShrink: 0,
          display: 'flex', alignItems: 'center', gap: 8,
        }}>
          <span style={{ color: C.blue, fontWeight: 600 }}>Beacon Console</span>
          {activeBeacon && <>
            <span style={{ color: C.border }}>│</span>
            <span style={{ color: C.green }}>{activeBeacon.hostname}</span>
            <span style={{ color: C.border }}>│</span>
            <span style={{ color: C.text }}>{activeBeacon.username}</span>
            <span style={{ color: C.border }}>│</span>
            <span style={{ color: C.mono }}>{activeBeacon.internalIp}</span>
            <span style={{ color: C.border }}>│</span>
            <span style={{ color: C.orange }}>{activeBeacon.integrity}</span>
            <span style={{ color: C.border }}>│</span>
            <span style={{ color: C.yellow }}>sleep {activeBeacon.sleep}s ±{activeBeacon.jitter}%</span>
          </>}
          <div style={{ flex: 1 }} />
          <span style={{ color: C.dim, fontSize: 10 }}>Tab=autocomplete  ↑↓=history</span>
          <span style={{ color: C.border }}>│</span>
          <button onClick={() => setLines([])}
            style={{ background: 'transparent', border: 'none', color: C.dim, cursor: 'pointer', fontSize: 11 }}>
            clear
          </button>
        </div>

        <div
          style={{ flex: 1, overflowY: 'auto', padding: '8px 12px', fontSize: 12, lineHeight: 1.65 }}
          onClick={() => inputRef.current?.focus()}
        >
          {lines.map((l, i) => {
            const isJSON = l.type === 'out' && l.text.trimStart().startsWith('[{')
            if (isJSON) return <ProcessTable key={i} raw={l.text.trim()} />
            return (
              <div key={i} style={{
                fontFamily: 'monospace', whiteSpace: 'pre-wrap', wordBreak: 'break-all',
                color: l.type === 'cmd'  ? C.yellow
                     : l.type === 'err'  ? C.red
                     : l.type === 'info' ? C.dim
                     : C.text,
              }}>{l.text}</div>
            )
          })}
          <div ref={consoleEndRef} />
        </div>

        <div style={{
          display: 'flex', alignItems: 'center', padding: '5px 12px',
          borderTop: `1px solid ${C.border}`, background: C.panel, gap: 8, flexShrink: 0,
        }}>
          <span style={{ color: activeBeacon ? C.green : C.dim, fontSize: 12, whiteSpace: 'nowrap' }}>
            {activeBeacon ? `${activeBeacon.hostname}\\${activeBeacon.username.split('\\')[1] ?? '?'}> ` : 'no session> '}
          </span>
          <input
            ref={inputRef}
            value={command}
            onChange={e => setCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={!activeBeacon || sending}
            placeholder={activeBeacon ? 'enter command… (help for list)' : 'select a session above'}
            style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', color: C.yellow, fontSize: 12, fontFamily: 'monospace', caretColor: C.yellow }}
          />
          {sending && <span style={{ color: C.dim, fontSize: 11, animation: 'none' }}>…</span>}
          <button onClick={sendCommand}
            disabled={!activeBeacon || !command.trim() || sending}
            style={{
              background: activeBeacon && command.trim() && !sending ? C.blue : 'transparent',
              border: `1px solid ${activeBeacon && command.trim() ? C.blue : C.border}`,
              color: C.text, padding: '2px 14px', fontSize: 11, borderRadius: 2,
              cursor: activeBeacon && command.trim() && !sending ? 'pointer' : 'default',
              opacity: !activeBeacon ? 0.3 : 1,
            }}
          >{sending ? '…' : 'Send'}</button>
        </div>
      </div>

      {/* ── Context menu ──────────────────────────────────────── */}
      {ctxMenu && (
        <div onClick={e => e.stopPropagation()} style={{
          position: 'fixed', top: ctxMenu.y, left: ctxMenu.x,
          background: '#2d2d2d', border: `1px solid ${C.border}`,
          borderRadius: 3, zIndex: 9999, minWidth: 170,
          boxShadow: '0 4px 20px rgba(0,0,0,0.6)', overflow: 'hidden',
        }}>
          {CTX_ITEMS.map((item, i) =>
            item === null
              ? <div key={`sep-${i}`} style={{ height: 1, background: C.border, margin: '2px 0' }} />
              : (
                <div key={item.label}
                  onClick={() => {
                    if (item.danger) { killSession(ctxMenu.beacon); return }
                    selectBeacon(ctxMenu.beacon)
                    if (item.pre) setCommand(item.pre)
                    setCtxMenu(null)
                    setTimeout(() => inputRef.current?.focus(), 50)
                  }}
                  style={{ padding: '6px 14px', fontSize: 12, cursor: 'pointer', color: item.danger ? C.red : C.text, display: 'flex', alignItems: 'center', gap: 8 }}
                  onMouseEnter={e => { e.currentTarget.style.background = item.danger ? '#3a1212' : C.rowSel }}
                  onMouseLeave={e => { e.currentTarget.style.background = 'transparent' }}
                >
                  <span style={{ width: 16, fontSize: 13 }}>{item.icon}</span>
                  <span>{item.label}</span>
                </div>
              )
          )}
        </div>
      )}
    </div>
  )
}

import React, { useState } from 'react'
import Operations from './Operations'
import { BeaconStoreProvider } from './BeaconStore'
import BeaconGen from './BeaconGen'

const C = {
  bg:      '#1a1a1a',
  panel:   '#252526',
  border:  '#3c3c3c',
  blue:    '#007acc',
  text:    '#cccccc',
  dim:     '#858585',
  toolbar: '#333333',
}

const TABS = [
  { id: 'operations', label: 'Operations', icon: '⬡' },
  { id: 'beacongen',  label: 'Beacon Gen', icon: '⚡' },
]

interface AlloyNetProps {
  token?: string
}

export default function AlloyNet({ token = '' }: AlloyNetProps) {
  const [activeTab, setActiveTab] = useState('operations')
  const [sessionId, setSessionId]   = useState('')

  return (
    <BeaconStoreProvider>
    <div style={{
      display: 'flex', flexDirection: 'column',
      width: '100%', height: '100%',
      background: C.bg,
      fontFamily: '"Consolas","Courier New",monospace',
      overflow: 'hidden',
    }}>
      {/* ── Tab bar ─────────────────────────────────────────── */}
      <div style={{
        display: 'flex', flexDirection: 'row', alignItems: 'flex-end',
        background: C.panel, borderBottom: `1px solid ${C.border}`,
        padding: '0 8px', height: 32, flexShrink: 0,
      }}>
        {TABS.map(tab => {
          const active = activeTab === tab.id
          return (
            <div
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              style={{
                display: 'flex', alignItems: 'center', gap: 5,
                padding: '5px 14px 4px', fontSize: 12, cursor: 'default',
                color: active ? '#ffffff' : C.dim,
                borderBottom: active ? `2px solid ${C.blue}` : '2px solid transparent',
                marginRight: 2, userSelect: 'none',
              }}
              onMouseEnter={e => { if (!active) e.currentTarget.style.color = C.text }}
              onMouseLeave={e => { if (!active) e.currentTarget.style.color = C.dim }}
            >
              <span style={{ fontSize: 11 }}>{tab.icon}</span>
              <span>{tab.label}</span>
            </div>
          )
        })}
      </div>

      {/* ── Tab content ─────────────────────────────────────── */}
      <div style={{ flex: 1, overflow: 'hidden', minHeight: 0 }}>
        <div style={{ display: activeTab === 'operations' ? 'contents' : 'none' }}><Operations token={token} onSessionId={setSessionId} /></div>
        <div style={{ display: activeTab === 'beacongen'  ? 'contents' : 'none' }}><BeaconGen  token={token} sessionId={sessionId} /></div>
      </div>
    </div>
  </BeaconStoreProvider>
  )
}

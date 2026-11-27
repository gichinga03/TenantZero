import React, { createContext, useContext, useState, ReactNode } from 'react'

export interface SavedBuild {
  id:          string   // 6-char hex
  profileName: string
  format:      string
  filename:    string
  hostedUrl:   string | null  // null = not hosted yet
  createdAt:   string
}

interface BeaconStoreCtx {
  builds:    SavedBuild[]
  addBuild:  (b: SavedBuild) => void
  hostBuild: (id: string, url: string) => void
}

const Ctx = createContext<BeaconStoreCtx>({
  builds: [], addBuild: () => {}, hostBuild: () => {},
})

export function BeaconStoreProvider({ children }: { children: ReactNode }) {
  const [builds, setBuilds] = useState<SavedBuild[]>([])
  function addBuild(b: SavedBuild) { setBuilds(prev => [b, ...prev]) }
  function hostBuild(id: string, url: string) {
    setBuilds(prev => prev.map(b => b.id === id ? { ...b, hostedUrl: url } : b))
  }
  return <Ctx.Provider value={{ builds, addBuild, hostBuild }}>{children}</Ctx.Provider>
}

export function useBeaconStore() { return useContext(Ctx) }

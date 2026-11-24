import React, { useState, useCallback } from 'react';
import Desktop from './Desktop';
import Taskbar from './Taskbar';
import Window from '../components/Window';
import AlloyNet from '../apps/AlloyNet/AlloyNet';
import BloodHound from '../apps/BloodHound/BloodHound';
import FileExplorer from '../apps/FileExplorer/FileExplorer';
import Notepad from '../apps/Notepad/Notepad';

interface AppDef {
  id: string;
  title: string;
  icon: string;
  defaultWidth: number;
  defaultHeight: number;
  defaultX: number;
  defaultY: number;
  component: React.ComponentType<any>;
}

const APP_REGISTRY: AppDef[] = [
  { id: 'alloynet',     title: 'AlloyNet C2',   icon: '/Alloynet.jpeg', defaultWidth: 1100, defaultHeight: 700, defaultX: 80,  defaultY: 40,  component: AlloyNet },
  { id: 'bloodhound',   title: 'BloodHound',    icon: '🐕', defaultWidth: 1000, defaultHeight: 650, defaultX: 120, defaultY: 60,  component: BloodHound },
  { id: 'fileexplorer', title: 'File Explorer', icon: '📁', defaultWidth: 800,  defaultHeight: 500, defaultX: 160, defaultY: 80,  component: FileExplorer },
  { id: 'notepad',      title: 'Notepad',       icon: '📝', defaultWidth: 600,  defaultHeight: 400, defaultX: 200, defaultY: 100, component: Notepad },
];

interface WindowState {
  id: string;
  appId: string;
  minimized: boolean;
  zIndex: number;
}

export default function WindowManager() {
  const [openWindows, setOpenWindows] = useState<WindowState[]>([]);
  const [activeWindowId, setActiveWindowId] = useState<string | null>(null);
  const [zCounter, setZCounter] = useState(100);
  const [notepadFile, setNotepadFile] = useState('');

  const openApp = useCallback((appId: string) => {
    setOpenWindows(prev => {
      const existing = prev.find(w => w.appId === appId);
      if (existing) {
        const newZ = Date.now();
        setActiveWindowId(existing.id);
        setZCounter(newZ);
        return prev.map(w => w.id === existing.id ? { ...w, minimized: false, zIndex: newZ } : w);
      }
      const newZ = Date.now();
      const newWin: WindowState = {
        id: crypto.randomUUID(),
        appId,
        minimized: false,
        zIndex: newZ,
      };
      setActiveWindowId(newWin.id);
      setZCounter(newZ);
      return [...prev, newWin];
    });
  }, []);

  const closeWindow = useCallback((id: string) => {
    setOpenWindows(prev => {
      const remaining = prev.filter(w => w.id !== id);
      const next = remaining.filter(w => !w.minimized).sort((a, b) => b.zIndex - a.zIndex)[0];
      setActiveWindowId(next?.id ?? null);
      return remaining;
    });
  }, []);

  const minimizeWindow = useCallback((id: string) => {
    setOpenWindows(prev => {
      const updated = prev.map(w => w.id === id ? { ...w, minimized: true } : w);
      const next = updated.filter(w => !w.minimized).sort((a, b) => b.zIndex - a.zIndex)[0];
      setActiveWindowId(curr => curr === id ? (next?.id ?? null) : curr);
      return updated;
    });
  }, []);

  const focusWindow = useCallback((id: string) => {
    const newZ = Date.now();
    setZCounter(newZ);
    setActiveWindowId(id);
    setOpenWindows(prev => prev.map(w => w.id === id ? { ...w, zIndex: newZ } : w));
  }, []);

  const onWindowClick = useCallback((id: string) => {
    setOpenWindows(prev => {
      const win = prev.find(w => w.id === id);
      if (!win) return prev;
      if (win.minimized) {
        const newZ = Date.now();
        setActiveWindowId(id);
        return prev.map(w => w.id === id ? { ...w, minimized: false, zIndex: newZ } : w);
      }
      if (id === activeWindowId) {
        const updated = prev.map(w => w.id === id ? { ...w, minimized: true } : w);
        const next = updated.filter(w => !w.minimized).sort((a, b) => b.zIndex - a.zIndex)[0];
        setActiveWindowId(next?.id ?? null);
        return updated;
      }
      focusWindow(id);
      return prev;
    });
  }, [activeWindowId, focusWindow]);

  const openNotepad = useCallback((filePath: string) => {
    setNotepadFile(filePath);
    openApp('notepad');
  }, [openApp]);

  return (
    <div style={{ position: 'fixed', inset: 0, overflow: 'hidden' }}>
      <Desktop onOpenApp={openApp} />

      {openWindows.map(win => {
        const appDef = APP_REGISTRY.find(a => a.id === win.appId);
        if (!appDef) return null;
        const AppComponent = appDef.component;
        return (
          <Window
            key={win.id}
            id={win.id}
            title={appDef.title}
            icon={appDef.icon}
            isActive={win.id === activeWindowId}
            isMinimized={win.minimized}
            defaultWidth={appDef.defaultWidth}
            defaultHeight={appDef.defaultHeight}
            defaultX={appDef.defaultX}
            defaultY={appDef.defaultY}
            onFocus={() => focusWindow(win.id)}
            onClose={() => closeWindow(win.id)}
            onMinimize={() => minimizeWindow(win.id)}
            onMaximize={() => {}}
            style={{ zIndex: win.zIndex }}
          >
            <AppComponent
              onOpenFile={win.appId === 'fileexplorer' ? openNotepad : undefined}
              filePath={win.appId === 'notepad' ? notepadFile : undefined}
            />
          </Window>
        );
      })}

      <Taskbar
        openWindows={openWindows.map(w => ({
          id: w.id,
          title: APP_REGISTRY.find(a => a.id === w.appId)?.title ?? w.appId,
          minimized: w.minimized,
        }))}
        activeWindowId={activeWindowId}
        onWindowClick={onWindowClick}
        onStartClick={() => {}}
      />
    </div>
  );
}

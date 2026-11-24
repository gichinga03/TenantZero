import React, { useState, useEffect, useRef, useCallback } from 'react';

interface WindowProps {
  id: string;
  title: string;
  icon?: string;
  isActive: boolean;
  isMinimized: boolean;
  defaultWidth?: number;
  defaultHeight?: number;
  defaultX?: number;
  defaultY?: number;
  onFocus: () => void;
  onClose: () => void;
  onMinimize: () => void;
  onMaximize: () => void;
  style?: React.CSSProperties;
  children: React.ReactNode;
}

export default function Window({
  id, title, icon, isActive, isMinimized,
  defaultWidth = 900, defaultHeight = 600,
  defaultX = 100, defaultY = 60,
  onFocus, onClose, onMinimize, onMaximize,
  style, children,
}: WindowProps) {
  const [position, setPosition] = useState({ x: defaultX, y: defaultY });
  const [size, setSize] = useState({ width: defaultWidth, height: defaultHeight });
  const [isMaximized, setIsMaximized] = useState(false);
  const [savedBounds, setSavedBounds] = useState<{ x: number; y: number; width: number; height: number } | null>(null);

  // Drag state
  const dragRef = useRef({
    isDragging: false,
    startMouseX: 0,
    startMouseY: 0,
    startPosX: 0,
    startPosY: 0,
  });

  // Resize state
  const resizeRef = useRef({
    isResizing: false,
    startMouseX: 0,
    startMouseY: 0,
    startWidth: 0,
    startHeight: 0,
  });

  const toggleMaximize = useCallback(() => {
    if (!isMaximized) {
      setSavedBounds({ x: position.x, y: position.y, width: size.width, height: size.height });
      setIsMaximized(true);
    } else {
      if (savedBounds) {
        setPosition({ x: savedBounds.x, y: savedBounds.y });
        setSize({ width: savedBounds.width, height: savedBounds.height });
      } else {
        setPosition({ x: defaultX, y: defaultY });
        setSize({ width: defaultWidth, height: defaultHeight });
      }
      setIsMaximized(false);
    }
  }, [isMaximized, position, size, savedBounds, defaultX, defaultY, defaultWidth, defaultHeight]);

  // Drag handlers
  const handleTitlebarMouseDown = useCallback((e: React.MouseEvent) => {
    if ((e.target as HTMLElement).closest('[data-wincontrol]')) return;
    e.preventDefault();
    onFocus();

    if (isMaximized) {
      // Un-maximize, then start dragging
      const restoredWidth = savedBounds?.width ?? defaultWidth;
      const restoredHeight = savedBounds?.height ?? defaultHeight;
      setSavedBounds(null);
      setIsMaximized(false);
      // Position window so click lands at horizontal center of titlebar
      const newX = Math.max(0, e.clientX - restoredWidth / 2);
      const newY = 0;
      setPosition({ x: newX, y: newY });
      setSize({ width: restoredWidth, height: restoredHeight });

      dragRef.current = {
        isDragging: true,
        startMouseX: e.clientX,
        startMouseY: e.clientY,
        startPosX: newX,
        startPosY: newY,
      };
    } else {
      dragRef.current = {
        isDragging: true,
        startMouseX: e.clientX,
        startMouseY: e.clientY,
        startPosX: position.x,
        startPosY: position.y,
      };
    }
  }, [isMaximized, savedBounds, defaultWidth, defaultHeight, position, onFocus]);

  // Resize handler
  const handleResizeMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    resizeRef.current = {
      isResizing: true,
      startMouseX: e.clientX,
      startMouseY: e.clientY,
      startWidth: size.width,
      startHeight: size.height,
    };
  }, [size]);

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (dragRef.current.isDragging) {
        const dx = e.clientX - dragRef.current.startMouseX;
        const dy = e.clientY - dragRef.current.startMouseY;
        const newX = Math.max(0, Math.min(
          dragRef.current.startPosX + dx,
          window.innerWidth - size.width
        ));
        const newY = Math.max(0, Math.min(
          dragRef.current.startPosY + dy,
          window.innerHeight - 48
        ));
        setPosition({ x: newX, y: newY });
      }
      if (resizeRef.current.isResizing) {
        const dx = e.clientX - resizeRef.current.startMouseX;
        const dy = e.clientY - resizeRef.current.startMouseY;
        setSize({
          width: Math.max(400, resizeRef.current.startWidth + dx),
          height: Math.max(300, resizeRef.current.startHeight + dy),
        });
      }
    };

    const onMouseUp = () => {
      dragRef.current.isDragging = false;
      resizeRef.current.isResizing = false;
    };

    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
    return () => {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };
  }, [size.width]);

  const outerStyle: React.CSSProperties = isMaximized
    ? {
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 48,
        width: '100%', height: 'auto',
        display: 'flex', flexDirection: 'column',
        background: '#1e1e1e',
        border: `1px solid ${isActive ? 'rgba(255,255,255,0.2)' : 'rgba(255,255,255,0.1)'}`,
        borderRadius: 0,
        boxShadow: isActive ? '0 8px 32px rgba(0,0,0,0.6)' : '0 4px 16px rgba(0,0,0,0.4)',
        overflow: 'hidden',
        ...style,
      }
    : {
        position: 'absolute',
        left: position.x, top: position.y,
        width: size.width, height: size.height,
        display: isMinimized ? 'none' : 'flex',
        flexDirection: 'column',
        background: '#1e1e1e',
        border: `1px solid ${isActive ? 'rgba(255,255,255,0.2)' : 'rgba(255,255,255,0.1)'}`,
        borderRadius: 8,
        boxShadow: isActive ? '0 8px 32px rgba(0,0,0,0.6)' : '0 4px 16px rgba(0,0,0,0.4)',
        overflow: 'hidden',
        ...style,
      };

  const ctrlBtn = (hoverBg: string): React.CSSProperties => ({
    width: 46, height: 32, border: 'none',
    background: 'transparent', color: '#ffffff',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    cursor: 'default', flexShrink: 0,
  });

  return (
    <div style={outerStyle} onMouseDown={onFocus}>
      {/* Titlebar */}
      <div
        style={{
          height: 32, display: 'flex', flexDirection: 'row',
          alignItems: 'center', background: '#2d2d2d', flexShrink: 0,
          cursor: 'default', userSelect: 'none',
        }}
        onMouseDown={handleTitlebarMouseDown}
        onDoubleClick={toggleMaximize}
      >
        {icon && (
          icon.startsWith('/') || icon.startsWith('http')
            ? <img src={icon} style={{ width: 18, height: 18, objectFit: 'contain', borderRadius: 2, margin: '0 6px 0 10px', flexShrink: 0 }} draggable={false} />
            : <span style={{ fontSize: 14, margin: '0 6px 0 12px' }}>{icon}</span>
        )}
        <span style={{
          color: '#ffffff', fontSize: 13, fontWeight: 500,
          flex: 1, overflow: 'hidden', textOverflow: 'ellipsis',
          whiteSpace: 'nowrap', fontFamily: 'Segoe UI, sans-serif',
          marginLeft: icon ? 0 : 12,
        }}>
          {title}
        </span>

        {/* Window controls */}
        <div data-wincontrol style={{ display: 'flex', flexDirection: 'row' }}>
          <button
            data-wincontrol
            style={ctrlBtn('rgba(255,255,255,0.1)')}
            onMouseOver={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.1)')}
            onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
            onClick={e => { e.stopPropagation(); onMinimize(); }}
          >
            <span style={{ fontSize: 13 }}>&#8211;</span>
          </button>
          <button
            data-wincontrol
            style={ctrlBtn('rgba(255,255,255,0.1)')}
            onMouseOver={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.1)')}
            onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
            onClick={e => { e.stopPropagation(); toggleMaximize(); }}
          >
            <span style={{ fontSize: 11 }}>&#9633;</span>
          </button>
          <button
            data-wincontrol
            style={ctrlBtn('#c42b1c')}
            onMouseOver={e => (e.currentTarget.style.background = '#c42b1c')}
            onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
            onClick={e => { e.stopPropagation(); onClose(); }}
          >
            <span style={{ fontSize: 13 }}>&#10005;</span>
          </button>
        </div>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflow: 'hidden' }}>
        {children}
      </div>

      {/* Resize handle */}
      {!isMaximized && (
        <div
          style={{
            position: 'absolute', bottom: 0, right: 0,
            width: 12, height: 12, cursor: 'se-resize',
            background: 'transparent',
          }}
          onMouseDown={handleResizeMouseDown}
        />
      )}
    </div>
  );
}
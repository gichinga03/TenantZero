import React, { useState, useCallback } from 'react';
import { useAuth } from './hooks/useAuth';
import LoginScreen from './shell/LoginScreen';
import WindowManager from './shell/WindowManager';

const LAB_USER   = 'MERIDIAN\\jsmith';
const LAB_DOMAIN = 'MERIDIAN';

export default function App() {
  const { auth, loginError, login, logout } = useAuth();

  const handleLogin = useCallback(async (username: string, password: string) => {
    await login(username, password);
  }, [login]);

  const handleLock = useCallback(() => {
    logout();
  }, [logout]);

  return (
    <div style={{ position: 'fixed', inset: 0, overflow: 'hidden', background: '#000' }}>
      {!auth
        ? <LoginScreen
            labUser={LAB_USER}
            labDomain={LAB_DOMAIN}
            onLogin={handleLogin}
            loginError={loginError}
          />
        : <WindowManager
            onLock={handleLock}
            token={auth.token}
          />
      }
    </div>
  );
}

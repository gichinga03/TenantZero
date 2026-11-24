import React, { useState, useCallback } from 'react';
import LoginScreen from './shell/LoginScreen';
import WindowManager from './shell/WindowManager';

const LAB_USER = 'MERIDIAN\\jsmith';
const LAB_DOMAIN = 'MERIDIAN';
const CORRECT_PASSWORD = 'Welcome1!';

export default function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [loginError, setLoginError] = useState('');

  const handleLogin = useCallback((username: string, password: string) => {
    if (password === CORRECT_PASSWORD) {
      setIsLoggedIn(true);
      setLoginError('');
    } else {
      setLoginError('The password is incorrect. Try again.');
    }
  }, []);

  const handleLock = useCallback(() => {
    setIsLoggedIn(false);
    setLoginError('');
  }, []);

  return (
    <div style={{ position: 'fixed', inset: 0, overflow: 'hidden', background: '#000' }}>
      {!isLoggedIn
        ? <LoginScreen
            labUser={LAB_USER}
            labDomain={LAB_DOMAIN}
            onLogin={handleLogin}
            loginError={loginError}
          />
        : <WindowManager onLock={handleLock} />
      }
    </div>
  );
}

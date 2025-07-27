
"use client";
import { useState } from 'react';

export default function Home() {
  // Helper to generate UUID v4
  function uuidv4() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = crypto.getRandomValues(new Uint8Array(1))[0] % 16;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }
  const [mode, setMode] = useState('');
  const [step, setStep] = useState(0);
  const [userId, setUserId] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');
  const [pairingCode, setPairingCode] = useState('');
  const [deviceId, setDeviceId] = useState('');
  const [threshold, setThreshold] = useState(2);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [wsMsg, setWsMsg] = useState('');
  const [apiResult, setApiResult] = useState('');
  const [connectedDevices, setConnectedDevices] = useState([]);

  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:40715/api';
  const WS_BASE = process.env.NEXT_PUBLIC_WS_BASE || 'ws://localhost:40715/api/tss/ws';

  // Register user (first device)
  const register = async () => {
    const res = await fetch(`${API_BASE}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setApiResult('Error: Invalid JSON response');
      return;
    }
    setUserId(data.id);
    setDeviceId(data.id);
    setApiResult(JSON.stringify(data, null, 2));
    setToken('demo-token');
    setStep(1);
  };

  // Login user (existing)
  const login = async () => {
    const res = await fetch(`${API_BASE}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setApiResult('Error: Invalid JSON response');
      return;
    }
    setUserId(data.id || '');
    setDeviceId(data.id);
    setToken(data.token || '');
    setApiResult(JSON.stringify(data, null, 2));
    if (data.id && data.token) setStep(1);
  };

  // Start pairing session
  const startPairing = async () => {
    const res = await fetch(`${API_BASE}/pairing/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, device_id: deviceId, threshold })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setApiResult('Error: Invalid JSON response');
      return;
    }
    setPairingCode(data.pairing_code);
    setApiResult(JSON.stringify(data, null, 2));
    setStep(2);
  };

  // Link device (additional device)
  const linkDevice = async () => {
    const res = await fetch(`${API_BASE}/pairing/link`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pairing_code: pairingCode, device_id: deviceId })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setApiResult('Error: Invalid JSON response');
      return;
    }
    setApiResult(JSON.stringify(data, null, 2));
    setStep(3);
  };

  // Complete pairing and keygen
  const completePairing = async () => {
    const res = await fetch(`${API_BASE}/pairing/complete`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pairing_code: pairingCode })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setApiResult('Error: Invalid JSON response');
      return;
    }
    setApiResult(JSON.stringify(data, null, 2));
    setStep(4);
  };

  // Connect WebSocket and send initial device info
  const connectWs = () => {
    setWsMsg('Connecting to WebSocket...');
    const socket = new WebSocket(WS_BASE);
    socket.onopen = () => {
      setWsMsg('WebSocket connected. Sending device info...');
      const did = mode === 'register' || mode === 'login' ? userId : deviceId;
      socket.send(JSON.stringify({ device_id: did, pairingCode }));
    };
    socket.onerror = (e: Event) => {
      // Try to get error message from event
      let errorMsg = 'Unknown error';
      if ('message' in e) {
        // @ts-ignore
        errorMsg = e.message;
      }
      setWsMsg('WebSocket error: ' + errorMsg);
    };
    socket.onclose = () => {
      setWsMsg('WebSocket closed');
    };
    socket.onmessage = (e) => {
      let msgText = e.data;
      try {
        const msg = JSON.parse(e.data);
        if (msg.status) {
          msgText = `Status: ${msg.status} (device_id: ${msg.device_id || ''})`;
        }
        if (msg.connected_devices) {
          setConnectedDevices(msg.connected_devices);
          msgText = 'Connected devices update: ' + msg.connected_devices.join(', ');
        }
      } catch {}
      setWsMsg('WebSocket message: ' + msgText);
    };
    setWs(socket);
  };

  return (
    <div style={{ maxWidth: 400, margin: 'auto', padding: 20 }}>
      <h1>nokey Demo</h1>
      {!mode && (
        <div>
          <button onClick={() => { setMode('register'); setStep(0); }}>Register (First Device)</button>
          <button onClick={() => { setMode('login'); setStep(0); }}>Login (Existing User)</button>
          <button onClick={() => { setMode('link'); setStep(0); }}>Link Device (Other Device)</button>
        </div>
      )}
      <pre style={{ background: '#f4f4f4', padding: 10 }}>{apiResult}</pre>
      {connectedDevices.length > 0 && (
        <div style={{ background: '#e0f7fa', padding: 10, marginBottom: 10 }}>
          <b>Connected Devices:</b>
          <ul>
            {connectedDevices.map((d, i) => <li key={i}>{d}</li>)}
          </ul>
        </div>
      )}
      {mode === 'register' && step === 0 && (
        <div>
          <h2>Register</h2>
          <input placeholder="Username" value={username} onChange={e => setUsername(e.target.value)} /><br />
          <input placeholder="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} /><br />
          <button onClick={register}>Register</button>
        </div>
      )}
      {mode === 'login' && step === 0 && (
        <div>
          <h2>Login</h2>
          <input placeholder="Username" value={username} onChange={e => setUsername(e.target.value)} /><br />
          <input placeholder="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} /><br />
          <button onClick={login}>Login</button>
        </div>
      )}
      {(mode === 'register' || mode === 'login') && step === 1 && (
        <div>
          <h2>Start Pairing</h2>
          <div>User ID: <b>{userId}</b></div>
          <div>Token: <b>{token}</b></div>
          <input placeholder="Threshold" type="number" value={threshold} onChange={e => setThreshold(Number(e.target.value))} /><br />
          <button onClick={startPairing}>Start Pairing</button>
        </div>
      )}
      {(mode === 'register' || mode === 'login') && step === 2 && (
        <div>
          <h2>Pairing Code</h2>
          <div>Pairing Code: <b>{pairingCode}</b></div>
          <button onClick={connectWs}>Connect WebSocket</button>
          <div>WS Message: {wsMsg}</div>
        </div>
      )}
      {mode === 'link' && step === 0 && (
        <div>
          <h2>Link Device</h2>
          <input placeholder="Pairing Code" value={pairingCode} onChange={e => setPairingCode(e.target.value)} /><br />
          <button onClick={() => {
            const uuid = uuidv4();
            setDeviceId(uuid);
            linkDevice();
          }}>Link Device</button>
          {deviceId && <div>Generated Device ID: <b>{deviceId}</b></div>}
        </div>
      )}
      {mode === 'link' && step === 3 && (
        <div>
          <h2>Device Linked</h2>
          <button onClick={connectWs}>Connect WebSocket</button>
          <div>WS Message: {wsMsg}</div>
        </div>
      )}
      {(mode === 'register' || mode === 'login') && step === 2 && (
        <div>
          <h2>Complete Pairing & Keygen</h2>
          <button onClick={completePairing}>Complete & Keygen</button>
        </div>
      )}
      {(mode === 'register' || mode === 'login') && step === 4 && (
        <div>
          <h2>Keygen Complete</h2>
          <button onClick={connectWs}>Connect WebSocket</button>
          <div>WS Message: {wsMsg}</div>
        </div>
      )}
    </div>
  );
}


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
  const [mode, setMode] = useState(''); // 'register', 'login', 'link', 'join-signing'
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

  // Multi-device signing/encryption states
  const [signingSessionId, setSigningSessionId] = useState('');
  const [signingMessage, setSigningMessage] = useState('');
  const [signingResult, setSigningResult] = useState('');
  const [signingStep, setSigningStep] = useState(0); // Device A: 0=idle, 1=session started, 2=sent share, 3=result; Device B: 0=idle, 1=joined, 2=sent share, 3=result

  // Start signing session (Device A)
  const startSigningSession = async () => {
    const res = await fetch(`${API_BASE}/signing/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceId, message: signingMessage })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setSigningResult('Error: Invalid JSON response');
      return;
    }
    setSigningSessionId(data.session_id);
    setSigningResult(JSON.stringify(data, null, 2));
    setSigningStep(1);
  };

  // Join signing session (Device B)
  const joinSigningSession = async () => {
    const res = await fetch(`${API_BASE}/signing/join`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_id: deviceId, session_id: signingSessionId })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setSigningResult('Error: Invalid JSON response');
      return;
    }
    setSigningResult(JSON.stringify(data, null, 2));
    setSigningStep(2);
  };

  // Send key share for signing/encryption
  const sendKeyShareForSigning = async () => {
    const keyShareStr = localStorage.getItem('key_share');
    if (!keyShareStr) {
      setSigningResult('No key share found in localStorage');
      return;
    }
    const keyShare = JSON.parse(keyShareStr);
    const res = await fetch(`${API_BASE}/message/store`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: signingSessionId, device_id: deviceId, key_share: keyShare })
    });
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setSigningResult('Error: Invalid JSON response');
      return;
    }
    setSigningResult(JSON.stringify(data, null, 2));
    setSigningStep(3);
  };

  // Get signing session info
  const getSigningSessionInfo = async () => {
    const res = await fetch(`${API_BASE}/signing/info?session_id=${signingSessionId}`);
    let data;
    try {
      data = await res.json();
    } catch (err) {
      setSigningResult('Error: Invalid JSON response');
      return;
    }
    setSigningResult(JSON.stringify(data, null, 2));
  };

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
  // Connect WebSocket and send device_id as first message, handle key_share
  const connectWs = () => {
    setWsMsg('Connecting to WebSocket...');
    const socket = new WebSocket(WS_BASE);
    socket.onopen = () => {
      setWsMsg('WebSocket connected. Sending device_id...');
      const did = mode === 'register' || mode === 'login' ? userId : deviceId;
      socket.send(JSON.stringify({ device_id: did }));
    };
    socket.onerror = (e: Event) => {
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
        if (msg.type === 'key_share' && msg.device_id) {
          // Store key share in localStorage
          localStorage.setItem('key_share', JSON.stringify(msg.key_share));
          setWsMsg(`[WS] Received key share for device: ${msg.device_id}`);
          // Send acknowledgement
          socket.send(JSON.stringify({ type: 'ack', device_id: msg.device_id }));
          return;
        }
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
          <button onClick={() => { setMode('login'); setStep(0); }}>Login (Device A)</button>
          <button onClick={() => { setMode('link'); setStep(0); }}>Link Device (Other Device)</button>
          <button onClick={() => { setMode('join-signing'); setSigningStep(0); }}>Join for Signing (Device B)</button>
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
      <hr style={{ margin: '30px 0' }} />
      <h2>Multi-Device Signing/Encryption Demo</h2>
      {/* Device A: Login and start signing session */}
      {mode === 'login' && step === 0 && (
        <div>
          <h3>Device A: Login</h3>
          <input placeholder="Username" value={username} onChange={e => setUsername(e.target.value)} /><br />
          <input placeholder="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} /><br />
          <button onClick={login}>Login</button>
        </div>
      )}
      {mode === 'login' && step === 1 && (
        <div>
          <h3>Device A: Start Signing Session</h3>
          <input placeholder="Message to sign/encrypt" value={signingMessage} onChange={e => setSigningMessage(e.target.value)} /><br />
          <button onClick={startSigningSession}>Start Signing Session</button>
        </div>
      )}
      {mode === 'login' && signingStep === 1 && (
        <div>
          <h3>Signing Session Started</h3>
          <div>Session ID: <b>{signingSessionId}</b></div>
          <button onClick={sendKeyShareForSigning}>Send Key Share</button>
        </div>
      )}
      {mode === 'login' && signingStep === 2 && (
        <div>
          <h3>Key Share Sent</h3>
          <button onClick={getSigningSessionInfo}>Refresh Session Info</button>
        </div>
      )}
      {mode === 'login' && signingStep >= 3 && (
        <div>
          <h3>Signing/Encryption Result</h3>
          <pre style={{ background: '#f4f4f4', padding: 10 }}>{signingResult}</pre>
        </div>
      )}

      {/* Device B: Join for signing */}
      {mode === 'join-signing' && signingStep === 0 && (
        <div>
          <h3>Device B: Join Signing Session</h3>
          <input placeholder="Session ID / Pairing Key" value={signingSessionId} onChange={e => setSigningSessionId(e.target.value)} /><br />
          <button onClick={joinSigningSession}>Join Session</button>
        </div>
      )}
      {mode === 'join-signing' && signingStep === 1 && (
        <div>
          <h3>Joined Session</h3>
          <button onClick={sendKeyShareForSigning}>Send Key Share</button>
        </div>
      )}
      {mode === 'join-signing' && signingStep === 2 && (
        <div>
          <h3>Key Share Sent</h3>
          <button onClick={getSigningSessionInfo}>Refresh Session Info</button>
        </div>
      )}
      {mode === 'join-signing' && signingStep >= 3 && (
        <div>
          <h3>Signing/Encryption Result</h3>
          <pre style={{ background: '#f4f4f4', padding: 10 }}>{signingResult}</pre>
        </div>
      )}
    </div>
  );
}

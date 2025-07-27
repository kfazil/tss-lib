# nokey Backend

This backend implements a production-ready, device-linked threshold signing platform using GoFrame and tss-lib.

## Approach & Flow

1. **User Registration**: User creates an account and registers their first device (e.g., mobile phone).
2. **Device Linking**: User adds more devices by generating a pairing code/QR. Other devices join by entering/scanning the code, linking to the user account.
3. **Threshold Setup**: User sets the signing threshold (e.g., 2-of-3 devices required).
4. **TSS Key Generation**: When all devices are linked, the backend coordinates a TSS keygen session using tss-lib. Each device receives a unique key share.
5. **Signing & Encryption**: Data can be signed/encrypted only when the threshold number of devices participate, ensuring no single point of failure.

## API Endpoints
- `/api/register`: Register user
- `/api/device/register`: Register device
- `/api/device/threshold`: Set threshold
- `/api/pairing/start`: Start device pairing session
- `/api/pairing/link`: Link device with pairing code
- `/api/pairing/complete`: Complete pairing and start TSS keygen
- `/api/encrypt`: Encrypt data (stub)
- `/api/decrypt`: Decrypt data (stub)


## Security Notes
- Key shares are distributed securely to each device.
- All cryptographic operations use tss-lib for threshold security.
- In production, use secure channels and persistent storage for key shares.

## Keygen Flow: Frontend & Backend Integration

This project provides a full-stack demo for multi-device threshold key generation using tss-lib.

### How It Works

- **Frontend** (`frontend/`):
  - Users register and link multiple devices (e.g., phone, laptop) via pairing codes/QR.
  - The frontend coordinates the TSS keygen session, connecting devices via WebSocket.
  - Each device receives its own key share after keygen.

- **Backend** (`backend/`):
  - Handles user/device registration, pairing, and threshold setup.
  - Orchestrates the TSS keygen process using tss-lib, distributing key shares to devices.
  - Provides REST API and WebSocket endpoints for device communication.

### Steps to Perform Keygen Flow

1. **Start Backend**
   - Run:
     ```
     cd backend
     go run main.go user.go device.go pairing.go auth.go storage.go tss_ws.go tss_keygen.go
     ```
   - Ensure port 40715 is free (see below for kill command).

2. **Start Frontend**
   - Run:
     ```
     cd frontend
     npm install
     npm run dev
     ```

3. **Register & Link Devices**
   - Register a user and link devices using the frontend UI (pairing code/QR).

4. **Set Threshold**
   - Choose how many devices are required to sign (e.g., 2-of-3).

5. **Initiate Keygen**
   - Once all devices are linked, start the keygen session from the frontend.
   - Devices connect via WebSocket; backend coordinates keygen and distributes shares.

6. **Ready for Signing**
   - After keygen, each device holds a key share. Signing requires threshold devices to participate.

### Requirements

- Go (for backend)
- Node.js & npm (for frontend)
- Multiple devices or browser tabs for demo
- Secure channels recommended for production




## Kill port
sudo kill -9 $(sudo lsof -t -i:40715)

## Run
go run main.go user.go device.go pairing.go auth.go storage.go tss_ws.go tss_keygen.go



## Sign Approach: Practical Multi-Device Flow

### 1. Keygen & Key Share Storage
- User registers and links three devices (e.g., phone, laptop, tablet).
- Each device receives and stores its key share in localStorage (or secure storage).

### 2. Signing Request
- User initiates a signing request from one device (Device A).

### 3. Session Coordination
- Device A starts a signing session; backend creates a unique session ID.
- Device A displays a QR code or short session code (e.g., "ABC123").

### 4. Device B Joins
- Device B opens the app and selects "Join Signing Session."
- Device B scans the QR code or enters the session code.
- Backend matches Device B to the session started by Device A.

### 5. Message Sharing & Protocol
- Both devices load their key shares from localStorage.
- The message to be signed is shared between Device A and Device B.
- Backend routes TSS protocol messages between devices.

### 6. Signature Generation
- When both devices complete the protocol, a valid threshold signature is produced.
- The signature can be used for encryption/decryption as needed.

### 7. Security & User Experience
- No single device can sign alone; any two devices can sign.
- Private key never exists in full on any device.
- User sees progress and completion status on both devices.

**Summary:**
- Device A starts the session and shares a code/QR.
- Device B uses the code/QR to join the same session.
- Backend matches devices by session ID and coordinates the protocol.

This approach ensures secure, user-friendly threshold signing with fault tolerance and privacy.



## signing flow
Practical Flow:

Device A logs in and creates a message to be kept safe.
Device A initiates a signing session for this message.
Device B (any other device with a key share) joins the session.
Both devices participate in TSS signing; the backend coordinates the protocol.
The backend receives the threshold signature.
The backend encrypts the message using the signature (or a derived key).
The encrypted message is stored securely.
To decrypt, two devices must participate in a TSS decryption/signing session.
Why this works:

Only the backend needs the final signature to encrypt/store the message.
Decryption is only possible when two devices participate again, proving threshold access.
Summary:

The backend acts as the coordinator and secure store.
Devices only need to participate in signing/decryption sessions; they don’t need the signature themselves.
This keeps the flow simple and secure.
Would you like to proceed with backend logic for:

Accepting a message, coordinating TSS signing, encrypting, and storing?
Coordinating TSS decryption with two devices?



/** 
after keygen devices are already storing key shares on local storage. we need to connect devices and load their keys from localstorage when performming signing.

user comes and login
do a message sign - add a message to sign
click encrypt message
next we need another device to connect to do this signing part.
we can show a paring code (QR/code) which other device will enter when doing join for signing feature om ui
now connect both devices - acknowledge that both devices are connected via ws.
now perform sign, encrypt and store
later this message can be decrypted by performing silimar signing flow.
we will show a list to encrypted message which user can decrypt if needed.

 */

// UI Steps for signing flow:
Plan:

Add UI for Device A to start a signing session (input message, start session, get session ID).
Add UI for Device B to join a signing session (input session ID/pairing code, join session).
Add UI for both devices to send their key shares to the backend for signing/encryption.
Display signing/encryption result and allow decryption.
I'll start by adding these flows to your page.tsx with clear separation for Device A and Device B actions.
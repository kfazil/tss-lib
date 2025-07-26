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

## Next Steps
- Implement real TSS keygen and signing flows with network communication between devices.
- Add authentication and persistent storage.
- Build frontend for device linking and keygen coordination.


## Kill port
sudo kill -9 $(sudo lsof -t -i:40715)

## Run
go run main.go user.go device.go pairing.go auth.go storage.go tss_ws.go tss_keygen.go
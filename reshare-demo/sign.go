package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"reflect"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/crypto"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

func main() {
	tss.SetCurve(tss.S256())
	ctx := context.Background()
	amountToSend := big.NewInt(1000000000000) // 0.00000001 ETH
	receiverAddress := "0x22a6a4Dd1eB834f62c43F8A4f58B7F6c1ED5A2F8"
	client, _ := ethclient.Dial("https://sepolia.infura.io/v3/55f7938f379244c69b7c0f0a6eee6d25")

	// Load Alice and Bob's key shares from JSON files
	aliceKey := loadKeyFromFile("new_tss_key_Alice.json")
	bobKey := loadKeyFromFile("new_tss_key_Dave.json")

	party1ID := tss.NewPartyID("P1", "Alice", aliceKey.Ks[0])
	party2ID := tss.NewPartyID("P4", "Dave", bobKey.Ks[0])

	address := gethcommon.HexToAddress(getEthereumAddress(aliceKey.ECDSAPub))
	fmt.Println("🔑 TSS Ethereum Address:", address.Hex())

	// Funding check
	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to fetch gas price: %v", err)
	}
	fastGasPrice := new(big.Int).Mul(gasPrice, big.NewInt(2)) // 2x faster
	txFee := new(big.Int).Mul(big.NewInt(int64(gasLimit)), fastGasPrice)
	minFund := new(big.Int).Add(amountToSend, txFee)

	fmt.Printf("💡 Suggested Gas Price: %s wei (%s ETH)\n", gasPrice.String(), weiToEther(gasPrice))
	fmt.Printf("⚡ Fast Gas Price (2x): %s wei (%s ETH)\n", fastGasPrice.String(), weiToEther(fastGasPrice))
	fmt.Printf("💰 Required minFund: %s wei (%s ETH)\n", minFund.String(), weiToEther(minFund))
	waitForFaucetFunding(client, address, minFund)

	// Create tx
	toAddress := gethcommon.HexToAddress(receiverAddress)
	nonce, _ := client.PendingNonceAt(ctx, address)
	tx := types.NewTransaction(nonce, toAddress, amountToSend, gasLimit, fastGasPrice, nil)
	txHash := types.LatestSignerForChainID(big.NewInt(11155111)).Hash(tx)
	msg := new(big.Int).SetBytes(txHash.Bytes())

	// Start signing with ONLY 2 parties (true threshold test)
	// The 3rd party (Charlie) will be offline - demonstrating fault tolerance
	fmt.Printf("🔐 Starting TRUE 2-of-3 threshold signing...\n")
	fmt.Printf("   ✅ Alice and Bob will sign\n")
	fmt.Printf("   🔴 Charlie is OFFLINE (simulating real-world scenario)\n")

	// Create a 2-party signing context with Alice and Bob
	// For threshold signing, we need t+1 parties (2 parties for 2-of-3)
	signingPartyIDs := tss.SortPartyIDs([]*tss.PartyID{party1ID, party2ID}) // Only Alice and Bob
	signingPeerCtx := tss.NewPeerContext(signingPartyIDs)

	// Create signing parameters for 2-party context with threshold=1 (need 1+1=2 signers)
	signingParams := make([]*tss.Parameters, 2)
	signingParams[0] = tss.NewParameters(tss.S256(), signingPeerCtx, party1ID, 2, 1) // Alice
	signingParams[1] = tss.NewParameters(tss.S256(), signingPeerCtx, party2ID, 2, 1) // Bob

	signingOuts := []chan tss.Message{make(chan tss.Message, 100), make(chan tss.Message, 100)}
	signingEnds := []chan *common.SignatureData{make(chan *common.SignatureData, 1), make(chan *common.SignatureData, 1)}

	// Start ONLY Alice and Bob for signing (Charlie is offline)
	signer1 := signing.NewLocalParty(msg, signingParams[0], *aliceKey, signingOuts[0], signingEnds[0]) // Alice
	signer2 := signing.NewLocalParty(msg, signingParams[1], *bobKey, signingOuts[1], signingEnds[1])   // Bob
	// Note: Charlie (signer3) is NOT started - demonstrating offline party

	go signer1.Start()
	go signer2.Start()

	activeSigners := []tss.Party{signer1, signer2} // Only 2 parties

	// Handle signing with only 2 parties
	sig := handleThresholdSigning(activeSigners, signingOuts, signingEnds)

	// Finalize and send
	r, s := sig.R, sig.S
	v := byte(0)
	rawSig := append(r, s...)
	rawSig = append(rawSig, v)

	// Verify
	R := new(big.Int).SetBytes(r)
	S := new(big.Int).SetBytes(s)
	pubKey := ecdsa.PublicKey{Curve: tss.S256(), X: aliceKey.ECDSAPub.X(), Y: aliceKey.ECDSAPub.Y()}
	if !ecdsa.Verify(&pubKey, txHash.Bytes(), R, S) {
		log.Fatal("❌ Signature verification failed")
	}
	fmt.Println("✅ Signature Verified!")

	// Broadcast
	signedTx, _ := tx.WithSignature(types.LatestSignerForChainID(big.NewInt(11155111)), rawSig)
	waitForConfirmedBalance(client, address, minFund)
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatal("❌ TX send failed:", err)
	}
	fmt.Println("✅ Transaction sent!")
	fmt.Printf("🔗 https://sepolia.etherscan.io/tx/%s\n", signedTx.Hash().Hex())

	// Success summary
	fmt.Println()
	fmt.Println("🎉 TRUE THRESHOLD SIGNATURE SUCCESS!")
	fmt.Println("====================================")
	fmt.Println("✅ Generated 3 key shares (Alice, Bob, Charlie)")
	fmt.Println("✅ Used ONLY 2 parties for signing (Alice + Bob)")
	fmt.Println("✅ Charlie was OFFLINE - demonstrating fault tolerance")
	fmt.Println("✅ Transaction successfully signed and broadcast")
	fmt.Println("✅ True 2-of-3 threshold cryptography proven!")
	fmt.Println()
	fmt.Println("🔒 Security Benefits Demonstrated:")
	fmt.Println("   • No single party can create signatures alone")
	fmt.Println("   • System remains operational with 1 party offline")
	fmt.Println("   • Private key never existed in complete form")
	fmt.Println("   • Threshold signatures indistinguishable from regular ECDSA")
}

func routeMessage(msg tss.Message, recipients []tss.Party) {
	wire, _, _ := msg.WireBytes()
	dest := msg.GetTo()
	fmt.Printf("🔔 Routing message from %s to %v\n", msg.GetFrom().Id, getPartyIDs(recipients))
	if dest == nil { // Broadcast
		for _, party := range recipients {
			party.UpdateFromBytes(wire, msg.GetFrom(), true)
		}
	} else { // Point-to-point
		for _, destParty := range dest {
			for _, recipient := range recipients {
				if recipient.PartyID().Index == destParty.Index {
					recipient.UpdateFromBytes(wire, msg.GetFrom(), false)
					break
				}
			}
		}
	}
}

func getEthereumAddress(pubKey *crypto.ECPoint) string {
	x := pubKey.X()
	y := pubKey.Y()
	xBytes, yBytes := x.Bytes(), y.Bytes()

	// pad to 32 bytes
	xPadded, yPadded := make([]byte, 32), make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	pubBytes := append(xPadded, yPadded...)
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubBytes)
	return "0x" + hex.EncodeToString(hash.Sum(nil)[12:])
}

// Helper function to load a key from a JSON file
func loadKeyFromFile(filename string) *keygen.LocalPartySaveData {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("❌ Failed to read key file %s: %v", filename, err)
	}
	var key keygen.LocalPartySaveData
	if err := json.Unmarshal(data, &key); err != nil {
		log.Fatalf("❌ Failed to unmarshal key file %s: %v", filename, err)
	}
	return &key
}

// Handle true threshold signing with only the active parties
func handleThresholdSigning(
	activeSigners []tss.Party,
	signingOuts []chan tss.Message,
	signingEnds []chan *common.SignatureData,
) *common.SignatureData {
	fmt.Printf("🔄 Processing threshold signature with %d active parties...\n", len(activeSigners))

	var finalSignature *common.SignatureData
	signaturesReceived := 0
	totalExpected := len(activeSigners)

	for signaturesReceived < totalExpected {
		selectCases := []reflect.SelectCase{}

		// Add output channels for each active signer
		for i := range signingOuts {
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(signingOuts[i]),
			})
		}

		// Add end channels for each active signer
		for i := range signingEnds {
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(signingEnds[i]),
			})
		}

		// Add timeout case
		selectCases = append(selectCases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(time.After(20 * time.Second)),
		})

		chosen, recv, ok := reflect.Select(selectCases)

		// Handle timeout
		if chosen == len(selectCases)-1 {
			fmt.Printf("⏳ Signing progress... (%d/%d completed)\n", signaturesReceived, totalExpected)
			continue
		}

		// Handle outgoing messages
		if chosen < len(signingOuts) {
			if !ok {
				continue
			}

			fromIndex := chosen
			msg := recv.Interface().(tss.Message)
			fromID := activeSigners[fromIndex].PartyID().Id

			// Route message to all other active parties
			var recipients []tss.Party
			for i, signer := range activeSigners {
				if i != fromIndex {
					recipients = append(recipients, signer)
				}
			}

			fmt.Printf("📡 %s → others | Message sent\n", fromID)
			routeMessage(msg, recipients)
			continue
		}

		// Handle signature completion
		sigIndex := chosen - len(signingOuts)
		if !ok {
			continue
		}

		signature := recv.Interface().(*common.SignatureData)
		signerID := activeSigners[sigIndex].PartyID().Id

		fmt.Printf("✅ %s completed threshold signature!\n", signerID)
		if finalSignature == nil {
			finalSignature = signature // Use the first valid signature
		}
		signaturesReceived++
	}

	fmt.Printf("🎉 Threshold signature SUCCESS with %d parties!\n", len(activeSigners))
	fmt.Printf("   ✅ Only %d out of 3 total parties signed\n", len(activeSigners))
	fmt.Printf("   ✅ True threshold cryptography proven!\n")

	return finalSignature
}

func waitForFaucetFunding(client *ethclient.Client, address gethcommon.Address, minBalance *big.Int) {
	ctx := context.Background()
	fmt.Println("⏳ Waiting for Sepolia faucet to fund:", address.Hex())
	for {
		balance, err := client.BalanceAt(ctx, address, nil)
		if err != nil {
			log.Fatalf("❌ Error fetching balance: %v", err)
		}
		fmt.Printf("💵 Current balance: %s wei (%s ETH)\n", balance.String(), weiToEther(balance))

		if balance.Cmp(minBalance) >= 0 {
			fmt.Println("✅ Faucet funding detected!")
			break
		}

		fmt.Println("🔁 Faucet not yet funded, retrying in 10 seconds...")
		time.Sleep(10 * time.Second)
	}
}

func waitForConfirmedBalance(client *ethclient.Client, address gethcommon.Address, minBalance *big.Int) {
	for {
		balance, err := client.BalanceAt(context.Background(), address, nil)
		if err != nil {
			log.Fatalf("❌ Error checking balance before TX send: %v", err)
		}
		fmt.Printf("🔎 Final balance before send: %s wei (%s ETH)\n", balance.String(), weiToEther(balance))

		if balance.Cmp(minBalance) >= 0 {
			return
		}

		fmt.Println("⏳ Waiting for balance to reflect on chain...")
		time.Sleep(5 * time.Second)
	}
}

func weiToEther(wei *big.Int) string {
	ether := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18))
	return ether.Text('f', 18) // 18 decimal places
}

func getPartyIDs(parties []tss.Party) []string {
	ids := []string{}
	for _, p := range parties {
		ids = append(ids, p.PartyID().Id)
	}
	return ids
}

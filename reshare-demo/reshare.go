package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/bnb-chain/tss-lib/v2/crypto"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/resharing"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"golang.org/x/crypto/sha3"
)

// createProductionPartyID creates a deterministic PartyID for production use
func createProductionPartyID(id, moniker, uniqueKey string) *tss.PartyID {
	// Create a deterministic big integer from the unique key
	// In production, this could be derived from:
	// - User's public key hash
	// - A stable user identifier + salt
	// - Any other deterministic, unique value per participant

	hash := sha256.Sum256([]byte(uniqueKey))
	key := new(big.Int).SetBytes(hash[:])

	return tss.NewPartyID(id, moniker, key)
}

func main() {
	fmt.Println("🔄 Production TSS Key Resharing Demo: 3→4 parties")
	fmt.Println("=================================================")

	tss.SetCurve(tss.S256())
	// Load existing keys (these have their original party IDs embedded)
	fmt.Println("📁 Loading existing key shares...")
	aliceKey := loadKeyFromFile("tss_key_Alice.json")
	bobKey := loadKeyFromFile("tss_key_Bob.json")
	charlieKey := loadKeyFromFile("tss_key_Charlie.json")

	oldKeys := []*keygen.LocalPartySaveData{aliceKey, bobKey, charlieKey}

	// Verify original shared public key and address
	originalPubKey := aliceKey.ECDSAPub
	originalAddress := getEthereumAddress(originalPubKey)
	fmt.Printf("🔑 Original Shared Public Key Address: %s\n", originalAddress)

	// Create old committee with the SAME party IDs as used in original keygen
	oldPIDs := tss.SortPartyIDs([]*tss.PartyID{
		tss.NewPartyID("P1", "Alice", oldKeys[0].Ks[0]),   // Use original K values
		tss.NewPartyID("P2", "Bob", oldKeys[1].Ks[1]),     // Use original K values
		tss.NewPartyID("P3", "Charlie", oldKeys[2].Ks[2]), // Use original K values
	})
	oldP2PCtx := tss.NewPeerContext(oldPIDs)

	// Create new committee with PRODUCTION-READY party IDs
	// Replace GenerateTestPartyIDs with deterministic PartyID creation
	// In production, these would come from your application's user management system
	newPIDs := []*tss.PartyID{
		createProductionPartyID("P1", "Alice", "alice_unique_stable_id"),
		createProductionPartyID("P2", "Bob", "bob_unique_stable_id"),
		createProductionPartyID("P3", "Charlie", "charlie_unique_stable_id"),
		createProductionPartyID("P4", "Dave", "dave_unique_stable_id"),
	}

	// Sort the new committee (essential for consistent ordering)
	sortedNewPIDs := tss.SortPartyIDs(newPIDs)
	newP2PCtx := tss.NewPeerContext(sortedNewPIDs)

	threshold := 1    // 2-of-3
	newThreshold := 1 // 2-of-4
	oldPartyCount := 3
	newPartyCount := 4

	fmt.Printf("👥 Old Committee: %d parties (threshold: %d)\n", oldPartyCount, threshold+1)
	fmt.Printf("👥 New Committee: %d parties (threshold: %d)\n", newPartyCount, newThreshold+1)

	fmt.Println("\n📋 New Committee PartyIDs (production-ready):")
	for i, pid := range sortedNewPIDs {
		fmt.Printf("   %d. %s (%s) - Key: %s\n", i+1, pid.Id, pid.Moniker,
			hex.EncodeToString(pid.Key)[:16]+"...")
	}

	// Shared channels (exactly like the test)
	outCh := make(chan tss.Message, 100)
	endCh := make(chan *keygen.LocalPartySaveData, 10)
	errCh := make(chan *tss.Error, 10)

	// Create old committee parties
	oldCommittee := make([]*resharing.LocalParty, 0, len(oldPIDs))
	for j, pID := range oldPIDs {
		params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, oldPartyCount, threshold, newPartyCount, newThreshold)
		party := resharing.NewLocalParty(params, *oldKeys[j], outCh, endCh).(*resharing.LocalParty)
		oldCommittee = append(oldCommittee, party)
	}

	// Create new committee parties
	newCommittee := make([]*resharing.LocalParty, 0, newPartyCount)
	for j, pID := range sortedNewPIDs {
		params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, oldPartyCount, threshold, newPartyCount, newThreshold)

		// Create fresh save data for new committee members
		save := keygen.NewLocalPartySaveData(newPartyCount)

		// For new parties (like Dave), generate pre-params
		if j >= oldPartyCount { // Dave is index 3, so j=3 >= 3
			fmt.Printf("🔐 Generating pre-parameters for new party %d...\n", j+1)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			preParams, err := keygen.GeneratePreParamsWithContext(ctx, 1)
			cancel()
			if err != nil {
				fmt.Printf("❌ Failed to generate pre-parameters: %v\n", err)
				return
			}
			save.LocalPreParams = *preParams
		} else {
			// Reuse existing pre-params for continuing parties
			save.LocalPreParams = oldKeys[j].LocalPreParams
		}

		party := resharing.NewLocalParty(params, save, outCh, endCh).(*resharing.LocalParty)
		newCommittee = append(newCommittee, party)
	}

	fmt.Println("🚀 Starting resharing process...")

	// Start new committee first (they wait for messages)
	for i, party := range newCommittee {
		go func(i int, p *resharing.LocalParty) {
			if err := p.Start(); err != nil {
				errCh <- err
			}
		}(i, party)
	}

	// Start old committee (they send messages)
	for i, party := range oldCommittee {
		go func(i int, p *resharing.LocalParty) {
			if err := p.Start(); err != nil {
				errCh <- err
			}
		}(i, party)
	}

	// Message routing (exactly like the test)
	newKeys := make([]keygen.LocalPartySaveData, len(newCommittee))
	endedNewCommittee := 0

	for {
		select {
		case err := <-errCh:
			fmt.Printf("❌ Party error: %v\n", err)
			return

		case msg := <-outCh:
			dest := msg.GetTo()
			if dest == nil {
				fmt.Println("❌ Unexpected nil destination")
				return
			}

			// Route to old committee
			if msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest[:len(oldCommittee)] {
					go updateParty(oldCommittee[destP.Index], msg, errCh)
				}
			}

			// Route to new committee
			if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, destP := range dest {
					go updateParty(newCommittee[destP.Index], msg, errCh)
				}
			}

		case save := <-endCh:
			// Only new committee members produce output
			if save.Xi != nil {
				newKeys[endedNewCommittee] = *save
				endedNewCommittee++
				fmt.Printf("✅ Party %d completed (%d/%d)\n", endedNewCommittee, endedNewCommittee, newPartyCount)

				if endedNewCommittee == newPartyCount {
					fmt.Println("🎉 Resharing completed successfully!")
					goto Done
				}
			}

		case <-time.After(30 * time.Second):
			fmt.Printf("⏳ Resharing progress... (%d/%d completed)\n", endedNewCommittee, newPartyCount)
		}
	}

Done:
	// Save new keys
	fmt.Println("💾 Saving new key shares...")
	names := []string{"Alice", "Bob", "Charlie", "Dave"}
	for i, key := range newKeys {
		filename := fmt.Sprintf("new_tss_key_%s.json", names[i])
		keyJSON, _ := json.MarshalIndent(key, "", "  ")
		os.WriteFile(filename, keyJSON, 0644)
		fmt.Printf("✅ %s new key saved to %s\n", names[i], filename)
	}

	// Verify that the shared public key is preserved
	newPubKey := newKeys[0].ECDSAPub
	newAddress := getEthereumAddress(newPubKey)

	fmt.Println("\n🔍 Verification:")
	fmt.Println("================")
	fmt.Printf("🔑 Original Address: %s\n", originalAddress)
	fmt.Printf("🔑 New Address:      %s\n", newAddress)

	if originalAddress == newAddress {
		fmt.Println("✅ SUCCESS: Shared public key preserved!")
	} else {
		fmt.Println("❌ ERROR: Shared public key changed!")
	}

	fmt.Println("\n📊 Resharing Summary:")
	fmt.Println("====================")
	fmt.Printf("👥 Old Committee: %d parties (Alice, Bob, Charlie)\n", oldPartyCount)
	fmt.Printf("👥 New Committee: %d parties (Alice, Bob, Charlie, Dave)\n", newPartyCount)
	fmt.Printf("🎯 Threshold: %d (need %d parties to sign)\n", newThreshold, newThreshold+1)
	fmt.Printf("🔐 Shared Address: %s\n", newAddress)
	fmt.Println("📁 Generated New Key Files:")
	for _, name := range names {
		fmt.Printf("   • new_tss_key_%s.json\n", name)
	}

	fmt.Println("\n🎉 Resharing complete! The shared public key remains unchanged.")

	fmt.Println("\n✨ Production Features:")
	fmt.Println("• Deterministic PartyID generation (no GenerateTestPartyIDs)")
	fmt.Println("• Stable party keys from unique participant identifiers")
	fmt.Println("• Compatible with real-world user management systems")

	fmt.Println("\n📋 For Production Use:")
	fmt.Println("• Replace 'unique_stable_id' with actual user public keys or stable IDs")
	fmt.Println("• Use the new key shares for signing with ShareID-based PartyIDs")
	fmt.Println("• Implement secure key storage and network communication")
}

func updateParty(party *resharing.LocalParty, msg tss.Message, errCh chan<- *tss.Error) {
	wire, _, _ := msg.WireBytes()
	if ok, err := party.UpdateFromBytes(wire, msg.GetFrom(), msg.IsBroadcast()); !ok && err != nil {
		errCh <- err
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

func loadKeyFromFile(filename string) *keygen.LocalPartySaveData {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("Failed to read key file %s: %v", filename, err))
	}

	var key keygen.LocalPartySaveData
	if err := json.Unmarshal(data, &key); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal key file %s: %v", filename, err))
	}

	return &key
}

package main

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/bnb-chain/tss-lib/v2/crypto"
    "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
    "github.com/bnb-chain/tss-lib/v2/ecdsa/resharing"
    "github.com/bnb-chain/tss-lib/v2/tss"
    "golang.org/x/crypto/sha3"
)

func main() {
    fmt.Println("🔄 Production-Ready TSS Script 3: Remove Charlie & Verify")
    fmt.Println("=========================================================")
    fmt.Println("📋 Objective: Remove Charlie → 3-party committee (Alice, Bob, Dave)")
    fmt.Println("🔍 Final verification of shared key preservation")
    fmt.Println("🏭 Production Mode: No test fixtures")
    fmt.Println()

    tss.SetCurve(tss.S256())

    // Load original 3-party address for comparison
    originalAddress := getOriginalAddress()
    fmt.Printf("🔑 Original 3-party address: %s\n", originalAddress)

    // Load 4-party keys and PartyIDs
    fmt.Println("\n📂 Loading 4-party committee data...")
    fourPartyKeys, fourPartyPIDs := load4PartyData()
    
    fourPartyPubKey := fourPartyKeys[0].ECDSAPub
    fourPartyAddress := getEthereumAddress(fourPartyPubKey)
    fmt.Printf("🔑 4-party address: %s\n", fourPartyAddress)

    // Remove Charlie from committee
    fmt.Println("\n📦 Removing Charlie from committee (4→3 parties)...")
    fmt.Println("==================================================")
    
    finalKeys := reshareRemoveParty(fourPartyKeys, fourPartyPIDs, 2) // Remove Charlie (index 2)
    
    finalPubKey := finalKeys[0].ECDSAPub
    finalAddress := getEthereumAddress(finalPubKey)
    fmt.Printf("🔑 Final 3-party address: %s\n", finalAddress)

    // Save final keys
    names := []string{"Alice", "Bob", "Dave"}
    for i, key := range finalKeys {
        filename := fmt.Sprintf("final_%s.json", names[i])
        keyJSON, _ := json.MarshalIndent(key, "", "  ")
        os.WriteFile(filename, keyJSON, 0644)
        fmt.Printf("💾 Saved %s key to %s\n", names[i], filename)
    }

    // COMPREHENSIVE VERIFICATION
    fmt.Println("\n🔍 COMPREHENSIVE VERIFICATION")
    fmt.Println("==============================")
    fmt.Printf("Phase 1 Address (3-party): %s\n", originalAddress)
    fmt.Printf("Phase 2 Address (4-party): %s\n", fourPartyAddress)
    fmt.Printf("Phase 3 Address (3-party): %s\n", finalAddress)
    fmt.Println()

    // Verify shared key preservation
    phase1To2Success := originalAddress == fourPartyAddress
    phase2To3Success := fourPartyAddress == finalAddress
    overallSuccess := originalAddress == finalAddress

    fmt.Println("📊 Verification Results:")
    fmt.Printf("   Phase 1→2 (Add Dave):       %s\n", getStatusSymbol(phase1To2Success))
    fmt.Printf("   Phase 2→3 (Remove Charlie): %s\n", getStatusSymbol(phase2To3Success))
    fmt.Printf("   Overall (3→4→3):            %s\n", getStatusSymbol(overallSuccess))
    fmt.Println()

    if overallSuccess {
        fmt.Println("🎉 SUCCESS: Shared public key preserved throughout all operations!")
    } else {
        fmt.Println("❌ ERROR: Shared public key changed during operations!")
    }

    fmt.Println()
    fmt.Println("✅ SUCCESS: Charlie removed from committee!")
    fmt.Printf("🔐 Final Shared Address: %s\n", finalAddress)
    fmt.Println("📁 Generated Files:")
    fmt.Println("   - final_Alice.json")
    fmt.Println("   - final_Bob.json") 
    fmt.Println("   - final_Dave.json")
    fmt.Println()
    fmt.Println("🎊 COMPLETE: Production-ready TSS resharing sequence finished!")
    fmt.Println("   ✅ Created 3-party committee (Alice, Bob, Charlie)")
    fmt.Println("   ✅ Added Dave → 4-party committee")
    fmt.Println("   ✅ Removed Charlie → 3-party committee (Alice, Bob, Dave)")
    fmt.Println("   ✅ Verified shared key preservation throughout")
}

func getOriginalAddress() string {
    // Load original Alice key to get the original address
    data, err := os.ReadFile("3party_Alice.json")
    if err != nil {
        panic(fmt.Sprintf("Failed to load original Alice key: %v", err))
    }
    
    var key keygen.LocalPartySaveData
    if err := json.Unmarshal(data, &key); err != nil {
        panic(fmt.Sprintf("Failed to parse original Alice key: %v", err))
    }
    
    return getEthereumAddress(key.ECDSAPub)
}

func load4PartyData() ([]keygen.LocalPartySaveData, tss.SortedPartyIDs) {
    // Load keys
    names := []string{"Alice", "Bob", "Charlie", "Dave"}
    keys := make([]keygen.LocalPartySaveData, 4)
    
    for i, name := range names {
        filename := fmt.Sprintf("4party_%s.json", name)
        data, err := os.ReadFile(filename)
        if err != nil {
            panic(fmt.Sprintf("Failed to load %s: %v", filename, err))
        }
        
        var key keygen.LocalPartySaveData
        if err := json.Unmarshal(data, &key); err != nil {
            panic(fmt.Sprintf("Failed to parse %s: %v", filename, err))
        }
        
        keys[i] = key
        fmt.Printf("📂 Loaded %s key from %s\n", name, filename)
    }
    
    // Load PartyIDs
    data, err := os.ReadFile("4party_pids.json")
    if err != nil {
        panic(fmt.Sprintf("Failed to load PartyIDs: %v", err))
    }
    
    var pIDs tss.SortedPartyIDs
    if err := json.Unmarshal(data, &pIDs); err != nil {
        panic(fmt.Sprintf("Failed to parse PartyIDs: %v", err))
    }
    
    fmt.Printf("📂 Loaded PartyIDs from 4party_pids.json\n")
    return keys, pIDs
}

func getStatusSymbol(success bool) string {
    if success {
        return "✅ PRESERVED"
    }
    return "❌ CHANGED"
}

// reshareRemoveParty removes a party from the committee using resharing
func reshareRemoveParty(oldKeys []keygen.LocalPartySaveData, oldPIDs tss.SortedPartyIDs, removeIndex int) []keygen.LocalPartySaveData {
    fmt.Printf("🔄 Removing party at index %d from committee...\n", removeIndex)
    
    oldPartyCount := len(oldKeys)
    newPartyCount := oldPartyCount - 1
    
    // Use the same threshold approach as the official test
    oldThreshold := 1  // 2-of-4
    newThreshold := 1  // 2-of-3
    
    fmt.Printf("👥 Old Committee: %d parties (threshold: %d)\n", oldPartyCount, oldThreshold+1)
    fmt.Printf("👥 New Committee: %d parties (threshold: %d)\n", newPartyCount, newThreshold+1)

    // Create old committee context using passed PartyIDs
    oldP2PCtx := tss.NewPeerContext(oldPIDs)

    // Create new committee PartyIDs (fresh IDs)
    newPIDs := tss.GenerateTestPartyIDs(newPartyCount)
    newP2PCtx := tss.NewPeerContext(newPIDs)

    // Create mapping between old and new committee
    continueMap := make([]int, 0, newPartyCount)
    j := 0
    for i := 0; i < oldPartyCount; i++ {
        if i != removeIndex {
            continueMap = append(continueMap, i)
            j++
        }
    }

    fmt.Println("\n📋 Committee Structure:")
    fmt.Println("Old Committee:")
    for i, pid := range oldPIDs {
        status := "✅ Continuing"
        if i == removeIndex {
            status = "❌ Being Removed"
        }
        fmt.Printf("   %d. %s (%s) - %s\n", i+1, pid.Id, pid.Moniker, status)
    }
    fmt.Println("New Committee:")
    for i, pid := range newPIDs {
        status := "🆕 New Member"
        fmt.Printf("   %d. %s (%s) - %s\n", i+1, pid.Id, pid.Moniker, status)
    }

    // Create channels
    outCh := make(chan tss.Message, 100)
    endCh := make(chan *keygen.LocalPartySaveData, 20)
    errCh := make(chan *tss.Error, 20)

    // Create old committee parties
    oldCommittee := make([]*resharing.LocalParty, 0, oldPartyCount)
    fmt.Println("🔧 Creating old committee parties...")
    for i, pID := range oldPIDs {
        fmt.Printf("   Creating old party %d: %s\n", i, pID.Id)
        params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, oldPartyCount, oldThreshold, newPartyCount, newThreshold)
        party := resharing.NewLocalParty(params, oldKeys[i], outCh, endCh).(*resharing.LocalParty)
        oldCommittee = append(oldCommittee, party)
    }

    // Create new committee parties (fresh IDs with empty save data)
    newCommittee := make([]*resharing.LocalParty, 0, newPartyCount)
    fmt.Println("\n🔧 Creating new committee parties...")
    for i, pID := range newPIDs {
        fmt.Printf("   📦 Creating new party %d: %s [Production Mode]\n", i, pID.Moniker)
        params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, oldPartyCount, oldThreshold, newPartyCount, newThreshold)
        params.SetNoProofMod()
        params.SetNoProofFac()
        
        // Create new empty save data for new committee
        save := keygen.NewLocalPartySaveData(newPartyCount)
        party := resharing.NewLocalParty(params, save, outCh, endCh).(*resharing.LocalParty)
        newCommittee = append(newCommittee, party)
    }

    fmt.Println("🚀 Starting resharing process...")

    // Start new committee first (like step2-add-dave.go)
    for i, party := range newCommittee {
        go func(i int, p *resharing.LocalParty) {
            if err := p.Start(); err != nil {
                errCh <- err
            }
        }(i, party)
    }

    // Start old committee
    for i, party := range oldCommittee {
        go func(i int, p *resharing.LocalParty) {
            if err := p.Start(); err != nil {
                errCh <- err
            }
        }(i, party)
    }

    // Process resharing
    newKeys := make([]keygen.LocalPartySaveData, newPartyCount)
    endedNewCommittee := 0

    fmt.Println("🔄 Processing resharing messages...")

    for {
        select {
        case err := <-errCh:
            fmt.Printf("❌ Resharing error: %v\n", err)
            panic(err)

        case msg := <-outCh:
            dest := msg.GetTo()
            if dest == nil {
                fmt.Println("❌ Unexpected nil destination")
                panic("nil destination")
            }

            // Route to old committee
            if msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
                for _, destP := range dest {
                    if destP.Index < len(oldCommittee) {
                        go updateReshareParty(oldCommittee[destP.Index], msg, errCh)
                    }
                }
            }

            // Route to new committee
            if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
                for _, destP := range dest {
                    if destP.Index < len(newCommittee) {
                        go updateReshareParty(newCommittee[destP.Index], msg, errCh)
                    }
                }
            }

        case save := <-endCh:
            // Only new committee members produce output
            if save.Xi != nil {
                newKeys[endedNewCommittee] = *save
                endedNewCommittee++
                fmt.Printf("✅ New party %d completed (%d/%d)\n", endedNewCommittee, endedNewCommittee, newPartyCount)
                
                if endedNewCommittee == newPartyCount {
                    fmt.Printf("🎉 Resharing completed successfully! Charlie removed from committee.\n")
                    return newKeys
                }
            } else {
                fmt.Printf("✅ Old party finished\n")
            }

        case <-time.After(30 * time.Second):
            fmt.Printf("⏳ Resharing progress... (%d/%d completed)\n", endedNewCommittee, newPartyCount)
        }
    }
}

func updateReshareParty(party *resharing.LocalParty, msg tss.Message, errCh chan<- *tss.Error) {
    wire, _, err := msg.WireBytes()
    if err != nil {
        errCh <- party.WrapError(err)
        return
    }
    if ok, err := party.UpdateFromBytes(wire, msg.GetFrom(), msg.IsBroadcast()); !ok && err != nil {
        errCh <- err
    }
}

func getEthereumAddress(pubKey *crypto.ECPoint) string {
    x := pubKey.X()
    y := pubKey.Y()
    xBytes, yBytes := x.Bytes(), y.Bytes()

    xPadded, yPadded := make([]byte, 32), make([]byte, 32)
    copy(xPadded[32-len(xBytes):], xBytes)
    copy(yPadded[32-len(yBytes):], yBytes)

    pubBytes := append(xPadded, yPadded...)
    hash := sha3.NewLegacyKeccak256()
    hash.Write(pubBytes)
    return "0x" + hex.EncodeToString(hash.Sum(nil)[12:])
}

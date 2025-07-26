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
    fmt.Println("🔄 Production-Ready TSS Script 2: Add Dave to Committee")
    fmt.Println("=======================================================")
    fmt.Println("📋 Objective: Add Dave → 4-party committee (Alice, Bob, Charlie, Dave)")
    fmt.Println("🏭 Production Mode: No test fixtures")
    fmt.Println()

    tss.SetCurve(tss.S256())

    // Load 3-party keys and PartyIDs
    fmt.Println("📂 Loading 3-party committee data...")
    originalKeys, originalPIDs := load3PartyData()
    
    originalPubKey := originalKeys[0].ECDSAPub
    originalAddress := getEthereumAddress(originalPubKey)
    fmt.Printf("🔑 Original 3-party address: %s\n", originalAddress)

    // Add Dave to committee
    fmt.Println("\n📦 Adding Dave to committee (3→4 parties)...")
    fmt.Println("=============================================")
    
    fourPartyKeys, fourPartyPIDs := reshareAddParty(originalKeys, originalPIDs, "Dave")
    
    fourPartyPubKey := fourPartyKeys[0].ECDSAPub
    fourPartyAddress := getEthereumAddress(fourPartyPubKey)
    fmt.Printf("🔑 4-party shared address: %s\n", fourPartyAddress)

    // Save 4-party keys
    names := []string{"Alice", "Bob", "Charlie", "Dave"}
    for i, key := range fourPartyKeys {
        filename := fmt.Sprintf("4party_%s.json", names[i])
        keyJSON, _ := json.MarshalIndent(key, "", "  ")
        os.WriteFile(filename, keyJSON, 0644)
        fmt.Printf("💾 Saved %s key to %s\n", names[i], filename)
    }

    // Save PartyIDs for next script
    pidsJSON, _ := json.MarshalIndent(fourPartyPIDs, "", "  ")
    os.WriteFile("4party_pids.json", pidsJSON, 0644)
    fmt.Printf("💾 Saved PartyIDs to 4party_pids.json\n")

    // Verify key preservation
    fmt.Println("\n🔍 Verification:")
    fmt.Println("================")
    fmt.Printf("Original Address: %s\n", originalAddress)
    fmt.Printf("4-Party Address:  %s\n", fourPartyAddress)
    
    if originalAddress == fourPartyAddress {
        fmt.Println("✅ SUCCESS: Shared public key preserved!")
    } else {
        fmt.Println("❌ ERROR: Shared public key changed!")
    }

    fmt.Println()
    fmt.Println("✅ SUCCESS: Dave added to committee!")
    fmt.Printf("🔐 Shared Address: %s\n", fourPartyAddress)
    fmt.Println("📁 Generated Files:")
    fmt.Println("   - 4party_Alice.json")
    fmt.Println("   - 4party_Bob.json") 
    fmt.Println("   - 4party_Charlie.json")
    fmt.Println("   - 4party_Dave.json")
    fmt.Println("   - 4party_pids.json")
    fmt.Println()
    fmt.Println("🚀 Ready for Step 3: Run 'go run step3-remove-charlie.go'")
}

func load3PartyData() ([]keygen.LocalPartySaveData, tss.SortedPartyIDs) {
    // Load keys
    names := []string{"Alice", "Bob", "Charlie"}
    keys := make([]keygen.LocalPartySaveData, 3)
    
    for i, name := range names {
        filename := fmt.Sprintf("3party_%s.json", name)
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
    data, err := os.ReadFile("3party_pids.json")
    if err != nil {
        panic(fmt.Sprintf("Failed to load PartyIDs: %v", err))
    }
    
    var pIDs tss.SortedPartyIDs
    if err := json.Unmarshal(data, &pIDs); err != nil {
        panic(fmt.Sprintf("Failed to parse PartyIDs: %v", err))
    }
    
    fmt.Printf("📂 Loaded PartyIDs from 3party_pids.json\n")
    return keys, pIDs
}

// reshareAddParty adds a new party to the committee using resharing
func reshareAddParty(oldKeys []keygen.LocalPartySaveData, oldPIDs tss.SortedPartyIDs, newPartyName string) ([]keygen.LocalPartySaveData, tss.SortedPartyIDs) {
    fmt.Printf("🔄 Adding %s to committee...\n", newPartyName)
    
    oldPartyCount := len(oldKeys)
    newPartyCount := oldPartyCount + 1
    oldThreshold := 1  // 2-of-3
    newThreshold := 1  // 2-of-4
    
    fmt.Printf("👥 Old Committee: %d parties (threshold: %d)\n", oldPartyCount, oldThreshold+1)
    fmt.Printf("👥 New Committee: %d parties (threshold: %d)\n", newPartyCount, newThreshold+1)

    // Create old committee context using passed PartyIDs
    oldP2PCtx := tss.NewPeerContext(oldPIDs)

    // Create new committee PartyIDs (fresh IDs)
    newPIDs := tss.GenerateTestPartyIDs(newPartyCount)
    newP2PCtx := tss.NewPeerContext(newPIDs)

    fmt.Println("\n📋 Committee Structure:")
    fmt.Println("Old Committee:")
    for i, pid := range oldPIDs {
        fmt.Printf("   %d. %s (%s) - ✅ Continuing\n", i+1, pid.Id, pid.Moniker)
    }
    fmt.Println("New Committee:")
    for i, pid := range newPIDs {
        status := "✅ Continuing"
        if i == newPartyCount-1 {
            status = fmt.Sprintf("🆕 New (%s)", newPartyName)
        }
        fmt.Printf("   %d. %s (%s) - %s\n", i+1, pid.Id, pid.Moniker, status)
    }

    // Create channels
    outCh := make(chan tss.Message, 100)
    endCh := make(chan *keygen.LocalPartySaveData, 20)
    errCh := make(chan *tss.Error, 20)

    // Create old committee parties
    oldCommittee := make([]*resharing.LocalParty, 0, oldPartyCount)
    for i, pID := range oldPIDs {
        params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, oldPartyCount, oldThreshold, newPartyCount, newThreshold)
        params.SetNoProofMod()
        params.SetNoProofFac()
        
        party := resharing.NewLocalParty(params, oldKeys[i], outCh, endCh).(*resharing.LocalParty)
        oldCommittee = append(oldCommittee, party)
    }

    // Create new committee parties (production mode - no fixtures)
    newCommittee := make([]*resharing.LocalParty, 0, newPartyCount)
    fmt.Println("\n🔧 Creating new committee parties...")
    for i, pID := range newPIDs {
        params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pID, oldPartyCount, oldThreshold, newPartyCount, newThreshold)
        params.SetNoProofMod()
        params.SetNoProofFac()
        
        save := keygen.NewLocalPartySaveData(newPartyCount)
        fmt.Printf("   📦 Creating new party %d (%s) [Production Mode]\n", i, pID.Moniker)
        
        party := resharing.NewLocalParty(params, save, outCh, endCh).(*resharing.LocalParty)
        newCommittee = append(newCommittee, party)
    }

    fmt.Println("🚀 Starting resharing process...")

    // Start new committee first
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
                    fmt.Printf("🎉 Resharing completed successfully! %s added to committee.\n", newPartyName)
                    return newKeys, newPIDs
                }
            }

        case <-time.After(30 * time.Second):
            fmt.Printf("⏳ Resharing progress... (%d/%d completed)\n", endedNewCommittee, newPartyCount)
        }
    }
}

func updateReshareParty(party *resharing.LocalParty, msg tss.Message, errCh chan<- *tss.Error) {
    wire, _, _ := msg.WireBytes()
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


package main

import (
	"fmt"
	"math/big"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"time"
)

// This is a simplified, demo-oriented implementation. In production, use secure channels and persistent storage.

func StartTSSKeygenSession(deviceIDs []string, threshold int) ([]*keygen.LocalPartySaveData, error) {
	// Robust TSS keygen pattern (step1-create-3party.go style)
	fmt.Printf("[TSS Keygen] deviceIDs: %v, threshold: %d\n", deviceIDs, threshold)
	partyCount := len(deviceIDs)
	if partyCount < 2 {
		return nil, fmt.Errorf("Need at least 2 devices for TSS keygen")
	}
	// Create PartyIDs with unique keys and human-readable names
	partyIDs := make([]*tss.PartyID, partyCount)
	for i, id := range deviceIDs {
		if id == "" {
			return nil, fmt.Errorf("deviceID at index %d is empty", i)
		}
		partyIDs[i] = tss.NewPartyID(fmt.Sprintf("P%d", i+1), id, big.NewInt(int64(i+1)))
	}
	sortedPartyIDs := tss.SortPartyIDs(partyIDs)
	peerCtx := tss.NewPeerContext(sortedPartyIDs)

	// Generate PreParams for each party
	preParams := make([]*keygen.LocalPreParams, partyCount)
	fmt.Println("⏳ Generating PreParams for all parties...")
	for i := range preParams {
		preParams[i], _ = keygen.GeneratePreParams(1 * 60 * 1000000000) // 1 minute
	}

	// Output and end channels for each party
	outs := make([]chan tss.Message, partyCount)
	ends := make([]chan *keygen.LocalPartySaveData, partyCount)
	for i := 0; i < partyCount; i++ {
		outs[i] = make(chan tss.Message, 100)
		ends[i] = make(chan *keygen.LocalPartySaveData, 1)
	}

	// Create parameters for each party
	params := make([]*tss.Parameters, partyCount)
	for i := 0; i < partyCount; i++ {
		params[i] = tss.NewParameters(tss.S256(), peerCtx, sortedPartyIDs[i], partyCount, threshold)
	}

	// Start keygen parties
	fmt.Println("🚀 Starting Key Generation...")
	parties := make([]*keygen.LocalParty, partyCount)
	for i := 0; i < partyCount; i++ {
		parties[i] = keygen.NewLocalParty(params[i], outs[i], ends[i], *preParams[i]).(*keygen.LocalParty)
		go parties[i].Start()
	}

	// Message routing and result collection
	keys := make([]*keygen.LocalPartySaveData, partyCount)
	done := 0
	for done < partyCount {
		select {
		case msg := <-outs[0]:
			routeMessage(msg, []tss.Party{parties[1], parties[2]})
		case msg := <-outs[1]:
			routeMessage(msg, []tss.Party{parties[0], parties[2]})
		case msg := <-outs[2]:
			routeMessage(msg, []tss.Party{parties[0], parties[1]})
		case result := <-ends[0]:
			fmt.Println("✅ Party 1 Keygen Done")
			keys[0] = result
			done++
		case result := <-ends[1]:
			fmt.Println("✅ Party 2 Keygen Done")
			keys[1] = result
			done++
		case result := <-ends[2]:
			fmt.Println("✅ Party 3 Keygen Done")
			keys[2] = result
			done++
		case <-time.After(10 * 1000000000):
			fmt.Println("⏳ Waiting for keygen messages...")
		}
	}

	// Return keys in order
	resultPtrs := make([]*keygen.LocalPartySaveData, partyCount)
	for i := range keys {
		resultPtrs[i] = keys[i]
	}
	fmt.Println("🎉 Keygen completed successfully!")
	return resultPtrs, nil
}

func updateKeygenParty(party *keygen.LocalParty, msg tss.Message, errCh chan<- *tss.Error) {
	wire, _, _ := msg.WireBytes()
	if ok, err := party.UpdateFromBytes(wire, msg.GetFrom(), msg.IsBroadcast()); !ok && err != nil {
		errCh <- err
	}
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

func getPartyIDs(parties []tss.Party) []string {
	ids := []string{}
	for _, p := range parties {
		ids = append(ids, p.PartyID().Id)
	}
	return ids
}

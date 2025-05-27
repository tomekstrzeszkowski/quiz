package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"example.com/voting/receiver"
	sdr "example.com/voting/sender"
	"example.com/voting/utils"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	golog.SetAllLoggers(golog.LevelError)

	listenF := flag.Int("l", 0, "wait for incoming connections")
	seedF := flag.Int64("seed", 0, "set random seed for id generation")
	Nick := flag.String("n", "", "set nickname")
	flag.Parse()
	if *listenF == 0 {
		log.Fatal("Please provide a port to bind (-l)")
	}
	host, kademliaDHT, _ := utils.MakeEnhancedHost(ctx, *listenF, false, *seedF)
	defer host.Close()
	defer kademliaDHT.Close()
	fmt.Printf("Host ID: %s\n", host.ID())
	fmt.Printf("Host Addresses: %v\n", host.Addrs())

	receiver := receiver.Receiver{Cancel: cancel}
	inputChan := make(chan bool, 1)
	if *listenF == 10000 {
		receiver.StartListening(ctx, host, inputChan)
		receiver.HandleKeyboard(ctx, inputChan)
		subscription, err := host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			defer subscription.Close()
			for {
				select {
				case evt := <-subscription.Out():
					connectEvent := evt.(event.EvtPeerConnectednessChanged)
					switch connectEvent.Connectedness {
					case network.Connected:
						//fmt.Printf("Peer connected: %s\n", connectEvent.Peer)
					case network.NotConnected:
						for i, sender := range receiver.Senders {
							if sender.ID == connectEvent.Peer {
								fmt.Printf("Peer disconnected: %s (%s)\n", sender.Nick, connectEvent.Peer)
								receiver.Senders = append(receiver.Senders[:i], receiver.Senders[i+1:]...)
							}
						}
					}
				}
			}
		}()
	}
	peerChan := utils.InitMDNS(host, "tstrz-voting-p2p-app-v1.0.0")
	for {
		peer := <-peerChan // will block until we discover a peer
		// maybe it should skip connect to sender if connection already exists (duplicated connections?)
		// if peer.ID <= host.ID() {
		// 	// if other end peer id greater than us, don't connect to it, just wait for it to connect us
		// 	fmt.Println("Found peer:", peer, " id is greater than us, wait for it to connect to us")
		// 	continue
		// }
		fmt.Println("Found peer:", peer, ", connecting")
		if *listenF == 10000 {
			<-ctx.Done()
		}
		sender, _ := sdr.CreateAndConnectSender(ctx, &host, peer, *Nick)
		if sender == nil {
			continue
		}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Exiting.")
				return
			case <-ticker.C:
				reply := sender.AskForStep()
				switch reply {
				case fmt.Sprintf("%s\n", utils.Name):
					if len(sender.Nick) == 0 {
						log.Println("Give me your name")
						sender.Nick = utils.GetText()
					} else {
						log.Printf("Hello %s, ready for voting?", sender.Nick)
					}
					sender.SendName()
				case fmt.Sprintf("%s\n", utils.WaitForVoting):
				case fmt.Sprintf("%s\n", utils.Voting):
					question := sender.GetQuestion()
					log.Println(question)
					sender.Votes = append(sender.Votes, utils.GetVote())
					sender.SendVote()
				case fmt.Sprintf("%s\n", utils.WaitingForVotes):
					//log.Println("Waiting for votes")
				case fmt.Sprintf("%s\n", utils.Summary):
					log.Println("SUMMARY")
					log.Println(sender.GetSummary())
				default:
					log.Printf("Unknown step %s", reply)
					cancel()
				}
			}
		}
	}
}

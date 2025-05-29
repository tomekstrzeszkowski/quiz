package receiver

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	sdr "example.com/voting/sender"
	"example.com/voting/utils"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Receiver struct {
	Senders []*sdr.Sender
	Cancel  context.CancelFunc
	Votes   []utils.Vote
}

func (r *Receiver) AddSender(s *sdr.Sender) {
	for i, candidate := range r.Senders {
		if candidate.ID == s.ID {
			r.Senders[i] = s
			return
		}
	}
	r.Senders = append(r.Senders, s)
}
func (r *Receiver) FindSender(peerID peer.ID) *sdr.Sender {
	for _, candidate := range r.Senders {
		if candidate.ID == peerID {
			return candidate
		}
	}
	return nil
}
func (r *Receiver) read(s network.Stream) (string, error) {
	buf := bufio.NewReader(s)
	return buf.ReadString('\n')
}
func (r *Receiver) Write(s network.Stream, str string) (int, error) {
	return s.Write([]byte(str + "\n"))
}

func (r *Receiver) HasVoted(sender *sdr.Sender) bool {
	log.Printf("sender %s", r.Senders[0].Step)
	if len(r.Senders) == 1 {
		return false
	}
	senderLenVotes := len(sender.Votes)
	for _, other := range r.Senders {
		if other.ID != sender.ID && len(other.Votes) > senderLenVotes {
			return false
		}
	}
	return true
}
func (r *Receiver) GetSummary() string {
	var sumOfVotes []float64
	lenOfSenders := float64(len(r.Senders))
	var summary string = "\n"
	for _, sender := range r.Senders {
		var votes string = ""
		for j := range sender.Votes {
			verbose := *sender.GetVerboseVote(j)
			if len(sumOfVotes) < j+1 {
				sumOfVotes = append(sumOfVotes, 0)
			}
			sumOfVotes[j] += float64(verbose)
			votes += fmt.Sprintf("%4s", strconv.Itoa(verbose)) + " | "
		}
		summary += fmt.Sprintf("%10s | %s\n", sender.Nick, votes)
	}
	var votes string = ""
	for _, voteSum := range sumOfVotes {
		votes += fmt.Sprintf("%4s", strconv.FormatFloat(voteSum/lenOfSenders, 'f', -1, 64)) + " | "
	}
	summary += fmt.Sprintf("%10s | %s\n", "Average", votes)
	votes = ""
	for _, vote := range r.Votes {
		verbose := *utils.FindVerboseForVote(vote)
		votes += fmt.Sprintf("%4s", strconv.FormatFloat(float64(verbose), 'f', -1, 64)) + " | "
	}
	return summary + fmt.Sprintf("%10s | %s\n", "Answer", votes)

}
func (r *Receiver) handleKeyboardInput(ctx context.Context, programmaticInput <-chan bool) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return
		case <-programmaticInput:
			r.Votes = append(r.Votes, utils.GetVote())
			log.Println(r.GetSummary())
			for _, sender := range r.Senders {
				sender.Step = utils.Summary
			}
		default:
			if scanner.Scan() {
				input := strings.TrimSpace(strings.ToLower(scanner.Text()))
				switch input {
				case "quiz":
					question := utils.GetText()
					log.Println("State changed to: Voting")
					for _, sender := range r.Senders {
						sender.Question = question
						sender.Step = utils.Voting
					}
				case "answer":
					r.Votes = append(r.Votes, utils.GetVote())
				case "all":
					log.Println(r.GetSummary())
				case "q":
					log.Println("Quit")
					r.Cancel()
				}
			}
		}
	}
}
func (r *Receiver) HandleConnectedPeers(host host.Host) {
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
					for i, sender := range r.Senders {
						if sender.ID == connectEvent.Peer {
							fmt.Printf("Peer disconnected: %s (%s)\n", sender.Nick, connectEvent.Peer)
							r.Senders = append(r.Senders[:i], r.Senders[i+1:]...)
						}
					}
				}
			}
		}
	}()
}
func (r *Receiver) HandleKeyboard(ctx context.Context, inputChan chan bool) {
	go r.handleKeyboardInput(ctx, inputChan)
}
func (r *Receiver) StartListening(ctx context.Context, host host.Host, inputChan chan bool) {
	fullAddr := utils.GetHostAddress(host)
	log.Printf("I am %s\n", fullAddr)
	host.SetStreamHandler("/step/1.0.0", func(s network.Stream) {
		remotePeerID := s.Conn().RemotePeer()
		sender := r.FindSender(remotePeerID)
		if sender == nil {
			r.AddSender(&sdr.Sender{
				ID: remotePeerID,
			})
			sender = r.FindSender(remotePeerID)
		}
		if len(sender.Nick) == 0 {
			sender.Step = utils.Name
		} else if sender.Step == utils.Name {
			sender.Step = utils.WaitForVoting
		}
		sender.SendStep(s, r)
	})
	host.SetStreamHandler("/voting/1.0.0", func(s network.Stream) {
		sender := r.FindSender(s.Conn().RemotePeer())
		vote, _ := r.read(s)
		voteValue, _ := strconv.Atoi(strings.ReplaceAll(vote, "\n", ""))
		sender.Votes = append(sender.Votes, utils.Vote(voteValue))
		verboseVote := sender.GetVerboseVote(-1)
		log.Printf("%s's vote is %d", (*sender).Nick, *verboseVote)
		sender.Step = utils.WaitingForVotes
		sender.SendStep(s, r)
		votesLen := len(sender.Votes)
		roundCompleted := true
		for _, other := range r.Senders {
			roundCompleted = roundCompleted && votesLen == len(other.Votes)
		}
		if roundCompleted {
			log.Println("Round is completed!")
			inputChan <- true
			log.Println(r.GetSummary())
		}
	})
	host.SetStreamHandler("/name/1.0.0", func(s network.Stream) {
		remotePeerID := s.Conn().RemotePeer()
		nick, err := r.read(s)
		nick = strings.ReplaceAll(nick, "\n", "")
		sender := r.FindSender(remotePeerID)
		sender.Nick = nick
		log.Printf("%s has joined! (%d)", nick, len(r.Senders))
		if err != nil {
			log.Println(err)
			s.Reset()
		} else {
			sender.Step = utils.WaitForVoting
			sender.SendStep(s, r)
		}
	})
	host.SetStreamHandler("/question/1.0.0", func(s network.Stream) {
		remotePeerID := s.Conn().RemotePeer()
		sender := r.FindSender(remotePeerID)
		sender.SendQuestion(s, r)
	})
	host.SetStreamHandler("/summary/1.0.0", func(s network.Stream) {
		remotePeerID := s.Conn().RemotePeer()
		sender := r.FindSender(remotePeerID)
		sender.SendSummary(s, r, r.GetSummary())
		sender.Step = utils.WaitForVoting
	})
	log.Println("listening for connections")
	log.Printf("Now run \"./voting -l 10001\"")
}

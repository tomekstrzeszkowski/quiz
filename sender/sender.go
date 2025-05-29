package sender

import (
	"context"
	"io"
	"log"
	"strconv"

	"example.com/quiz/utils"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

type receiver interface {
	Write(network.Stream, string) (int, error)
}

type Sender struct {
	ID       peer.ID
	Votes    []utils.Vote
	Nick     string
	Host     *host.Host
	Info     *peer.AddrInfo
	Step     utils.HandshakeStatus
	Question string
}

func CreateSender(ha *host.Host, targetPeer string, nick string) (*Sender, error) {
	fullAddr := utils.GetHostAddress(*ha)
	log.Printf("I'm %s\n", fullAddr)
	maddr, err := ma.NewMultiaddr(targetPeer)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	(*ha).Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	log.Println("sender opening stream")
	return &Sender{
		ID:   info.ID,
		Nick: nick,
		Host: ha,
		Info: info,
	}, nil
}
func CreateAndConnectSender(ctx context.Context, ha *host.Host, info peer.AddrInfo, nick string) (*Sender, error) {
	fullAddr := utils.GetHostAddress(*ha)
	log.Printf("I'm %s\n", fullAddr)
	if err := (*ha).Connect(ctx, info); err != nil {
		log.Println("Connection failed:", err)
		return nil, err
	}
	(*ha).Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	log.Println("sender opening stream")
	return &Sender{
		ID:   info.ID,
		Nick: nick,
		Host: ha,
		Info: &info,
	}, nil
}

func (s *Sender) SendStep(stream network.Stream, receiver receiver) {
	receiver.Write(stream, string(s.Step))
	stream.Close()
}
func (s *Sender) SendQuestion(stream network.Stream, receiver receiver) {
	receiver.Write(stream, string(s.Question))
	stream.Close()
}
func (s *Sender) SendSummary(stream network.Stream, receiver receiver, summary string) {
	receiver.Write(stream, summary)
	stream.Close()
}

func (s *Sender) GetCurrentVote() utils.Vote {
	return s.Votes[len(s.Votes)-1]
}
func (s *Sender) GetVerboseVote(index int) *int {
	var vote utils.Vote
	if index == -1 {
		vote = s.GetCurrentVote()
	} else {
		vote = s.Votes[index]
	}
	return utils.FindVerboseForVote(vote)
}
func (s *Sender) SendVote() {
	stream, err := (*s.Host).NewStream(context.Background(), (*s.Info).ID, "/voting/1.0.0")
	defer stream.Close()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Sender sending vote")
	_, err = stream.Write([]byte(strconv.Itoa(int(s.GetCurrentVote())) + "\n"))
	if err != nil {
		log.Println(err)
		return
	}
	out, err := io.ReadAll(stream)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("read reply %q\n", out)
}
func (s *Sender) AskForStep() string {
	stream, err := (*s.Host).NewStream(context.Background(), (*s.Info).ID, "/step/1.0.0")
	defer stream.Close()
	if err != nil {
		log.Println(err)
		return string(utils.Error)
	}
	stream.Write([]byte("\n"))
	out, _ := io.ReadAll(stream)
	return string(out[:])
}
func (s *Sender) SendName() string {
	stream, err := (*s.Host).NewStream(context.Background(), (*s.Info).ID, "/name/1.0.0")
	defer stream.Close()
	if err != nil {
		log.Println(err)
		return string(utils.Error)
	}
	stream.Write([]byte(s.Nick + "\n"))
	out, _ := io.ReadAll(stream)
	return string(out[:])
}
func (s *Sender) GetQuestion() string {
	stream, err := (*s.Host).NewStream(context.Background(), (*s.Info).ID, "/question/1.0.0")
	defer stream.Close()
	if err != nil {
		log.Println(err)
		return string(utils.Error)
	}
	out, _ := io.ReadAll(stream)
	return string(out[:])
}
func (s *Sender) GetSummary() string {
	stream, err := (*s.Host).NewStream(context.Background(), (*s.Info).ID, "/summary/1.0.0")
	defer stream.Close()
	if err != nil {
		log.Println(err)
		return string(utils.Error)
	}
	out, _ := io.ReadAll(stream)
	return string(out[:])
}

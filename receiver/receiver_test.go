package receiver

import (
	"testing"

	sdr "example.com/quiz/sender"
	"example.com/quiz/utils"
)

func TestAddingSender(t *testing.T) {
	sender := sdr.Sender{}
	sender.Votes = append(sender.Votes, utils.One)
	sender.Nick = "Max"
	receiver := Receiver{}
	receiver.AddSender(&sender)
}

package sender

import (
	"testing"

	"example.com/quiz/utils"
)

func TestSetVote(t *testing.T) {
	sender := Sender{}
	sender.Votes = append(sender.Votes, utils.One, utils.Two)
}

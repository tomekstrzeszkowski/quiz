package sender

import (
	"testing"

	"example.com/voting/utils"
)

func TestSetVote(t *testing.T) {
	sender := Sender{}
	sender.Votes = append(sender.Votes, utils.One, utils.Twelve)
}

package utils

type Vote int8

const (
	One Vote = iota
	Two
	Three
	Four
)

var VoteVerboseToVote = map[int]Vote{
	1: One,
	2: Two,
	3: Three,
	4: Four,
}

type HandshakeStatus string

const (
	Name            HandshakeStatus = "NAME"
	WaitForVoting   HandshakeStatus = "WAIT_FOR_VOTING"
	Voting          HandshakeStatus = "VOTING"
	Error           HandshakeStatus = "ERROR"
	WaitingForVotes HandshakeStatus = "WAITING"
	Summary         HandshakeStatus = "SUMMARY"
)

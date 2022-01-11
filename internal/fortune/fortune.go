package fortune

import (
	"math/rand"
	"strings"
	"time"
)

type Fortune struct {
	Quote  string
	Author string
}

var fortunes = []Fortune{
	{
		"The trick is to fix the problem you have, rather than the problem you want.",
		"Bram Cohen",
	},
	{
		"As a rule, software systems do not work well until they have been used, and have failed repeatedly, in real applications.",
		"Dave Parnas",
	},
	{
		"The most effective debugging tool is still careful thought, coupled with judiciously placed print statements.",
		"Brian Kernighan",
	},
	{
		"Duplication is far cheaper than the wrong abstraction.",
		"Sandi Metz",
	},
	{
		"First, solve the problem. Then, write the code.",
		"John Johnson",
	},
	{
		"In programming the hard part isn't solving problems, but deciding what problems to solve.",
		"Paul Graham",
	},
	{
		"Debugging is twice as hard as writing the code in the first place. Therefore, if you write the code as cleverly as possible, you are, by definition, not smart enough to debug it.",
		"Brian Kernighan",
	},
	{
		"Clear is better than clever.",
		"Dave Cheney",
	},
	{
		"Don't be a boilerplate programmer. Instead, build tools for users and other programmers. Take historical note of textile and steel industries: do you want to build machines and tools, or do you want to operate those machines?",
		"Ras Bodik",
	},
	{
		"Don't program defensively.",
		"Erlang style guide",
	},
	{
		"Hofstadter's Law: It always takes longer than you expect, even when you take into account Hofstadter's Law.",
		"Douglas Hofstadter",
	},
	{
		"One of my most productive days was throwing away 1000 lines of code.",
		"Ken Thompson",
	},
	{
		"The cost of adding a feature isn't just the time it takes to code it. The cost also includes the addition of an obstacle to future expansion. The trick is to pick the features that don't fight each other.",
		"John Carmack",
	},
	{
		"As a programmer, it is your job to put yourself out of business. What you do today can be automated tomorrow.",
		"Doug Mcllroy",
	},
	{
		"It is not that uncommon for the cost of an abstraction to outweigh the benefit it delivers. Kill one today!",
		"John Carmack",
	},
	{
		"A distributed system is one in which the failure of a computer you didn't even know existed can render your own computer unusable.",
		"Leslie Lamport",
	},
	{
		"The best performance improvement is the transition from the nonworking state to the working state",
		"John Ousterhout",
	},
	{
		"Without requirements or design, programming is the art of adding bugs to an empty text file.",
		"Louis Srygley",
	},
	{
		"There are two methods in software design. One is to make the program so simple, there are obviously no errors. The other is to make it so complicated, there are no obvious errors.",
		"Tony Hoare",
	},
}

// Pretty creates a pretty string representation of the fortune with the quote and author.
//
// maxLineLen is the maximum length a line in the quote is allowed to be. This is used
// to nicely split the quote into multiple lines while preserving whole words.
// If zero, the default max length is 80.
func (f Fortune) Pretty(maxLineLen int) string {
	if maxLineLen == 0 {
		// Default to 80 for good enough, 0 wouldn't make sense anyway
		maxLineLen = 80
	}

	var sb strings.Builder
	sb.WriteByte('"')
	lineLen := 0

	// Go through each word and construct lines based on termWidth
	for _, word := range strings.Split(f.Quote, " ") {
		// Prevent line from going over maxLineLength, if it's just under that's ok.
		if lineLen+len(word) >= maxLineLen {
			sb.WriteByte('\n')
			lineLen = 0
		}
		if lineLen > 0 {
			sb.WriteByte(' ')
			lineLen++
		}
		sb.WriteString(word)
		lineLen += len(word)
	}

	sb.WriteByte('"')
	sb.WriteString("\n\t-- ")
	sb.WriteString(f.Author)
	sb.WriteByte('\n')
	return sb.String()
}

func Random() Fortune {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Return a random fortune between [0, len)
	return fortunes[r.Intn(len(fortunes))]
}

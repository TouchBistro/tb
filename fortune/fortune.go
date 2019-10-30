package fortune

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const termWidth = 80

type Fortune struct {
	Quote  string
	Author string
}

var fortunes = []Fortune{
	Fortune{
		"The trick is to fix the problem you have, rather than the problem you want.",
		"Bram Cohen",
	},
	Fortune{
		"As a rule, software systems do not work well until they have been used, and have failed repeatedly, in real applications.",
		"Dave Parnas",
	},
	Fortune{
		"The most effective debugging tool is still careful thought, coupled with judiciously placed print statements.",
		"Brian Kernighan",
	},
	Fortune{
		"Duplication is far cheaper than the wrong abstraction.",
		"Sandi Metz",
	},
	Fortune{
		"First, solve the problem. Then, write the code.",
		"John Johnson",
	},
	Fortune{
		"In programming the hard part isn’t solving problems, but deciding what problems to solve.",
		"Paul Graham",
	},
	Fortune{
		"Debugging is twice as hard as writing the code in the first place. Therefore, if you write the code as cleverly as possible, you are, by definition, not smart enough to debug it.",
		"Brian Kernighan",
	},
	Fortune{
		"Clear is better than clever.",
		"Dave Cheney",
	},
	Fortune{
		"Don't be a boilerplate programmer. Instead, build tools for users and other programmers. Take historical note of textile and steel industries: do you want to build machines and tools, or do you want to operate those machines?",
		"Ras Bodik",
	},
	// Fortune{
	// 	"Alan Kay is a cuck",
	// 	"Omar Sabry",
	// },
	Fortune{
		"Don't program defensively.",
		"Erlang style guide",
	},
	Fortune{
		"Hofstadter's Law: It always takes longer than you expect, even when you take into account Hofstadter's Law.",
		"Douglas Hofstadter",
	},
	Fortune{
		"One of my most productive days was throwing away 1000 lines of code.",
		"Ken Thompson",
	},
	Fortune{
		"The cost of adding a feature isn't just the time it takes to code it. The cost also includes the addition of an obstacle to future expansion. The trick is to pick the features that don't fight each other.",
		"John Carmack",
	},
	Fortune{
		"As a programmer, it is your job to put yourself out of business. What you do today can be automated tomorrow.",
		"Doug Mcllroy",
	},
	Fortune{
		"It is not that uncommon for the cost of an abstraction to outweigh the benefit it delivers. Kill one today!",
		"John Carmack",
	},
	Fortune{
		"A distributed system is one in which the failure of a computer you didn't even know existed can render your own computer unusable.",
		"Leslie Lamport",
	},
	Fortune{
		"The best performance improvement is the transition from the nonworking state to the working state",
		"John Ousterhout",
	},
	Fortune{
		"Without requirements or design, programming is the art of adding bugs to an empty text file.",
		"Louis Srygley",
	},
}

func (f Fortune) String() string {
	words := strings.Split(f.Quote, " ")
	lines := make([]string, 0)

	var currLine strings.Builder

	for i, word := range words {
		if currLine.Len()+len(word) >= termWidth {
			lines = append(lines, currLine.String())
			var nextLine strings.Builder
			currLine = nextLine
		}

		currLine.WriteString(word)

		if i == len(words)-1 {
			lines = append(lines, currLine.String())
		} else {
			currLine.WriteString(" ")
		}
	}

	quote := strings.Join(lines, "\n")
	return fmt.Sprintf("\"%s\"\n\t-- %s\n", quote, f.Author)
}

func Random() Fortune {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	idx := r.Intn(len(fortunes))
	return fortunes[idx]
}
